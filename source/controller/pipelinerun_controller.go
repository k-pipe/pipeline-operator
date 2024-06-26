/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
)

const SUCCESS_STATUS_PREFIX = "success-"
const PVC_STATUS_PREFIX = "pvc-"

// PipelineRunReconciler reconciles a PipelineRun object
type PipelineRunReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelineruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelineruns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PipelineRun object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PipelineRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := Logger(ctx, req, "PR")
	defer LoggingDone(log)

	// get the pipeline run by name from request
	pr, result, err := r.loadResource(ctx, log, req.NamespacedName)
	if result != nil || err != nil {
		return *result, err
	}

	// determine pipeline version if not set, yet
	if (pr.Status.PipelineVersion == nil) || !isTrue(pr, VersionDetermined) {
		return r.determinePipelineVersion(ctx, log, pr)
	}

	// load pipeline structure if not set, yet
	if (pr.Status.PipelineStructure == nil) || !isTrue(pr, StructureLoaded) {
		return r.storePipelineStructure(ctx, log, pr)
	}

	// if not paused nor terminated ...
	if !(isTrue(pr, Paused) || isTrue(pr, Terminated)) {
		if result, err := r.startStartableStep(ctx, log, pr); result != nil || err != nil {
			return *result, err
		}

		if result, err = r.startStartableStep(ctx, log, pr); result != nil || err != nil {
			return *result, err
		}
	}

	// update step statistics
	if result, err = r.updateStepStatistics(ctx, pr); result != nil || err != nil {
		return *result, err
	}

	// delete unneeded volumes
	if result, err = r.removeUnneededPipelineJob(ctx, log, pr); result != nil || err != nil {
		return *result, err
	}

	// set state to succeeded or failed
	if result, err = r.determineTerminalRunState(ctx, pr); result != nil || err != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) loadResource(ctx context.Context, log func(string, ...interface{}), name types.NamespacedName) (*pipelinev1.PipelineRun, *ctrl.Result, error) {
	pr, err := r.GetPipelineRun(ctx, name)
	if pr == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log("PipelineRun resource not found. Ignoring since object has been deleted")
		return nil, &ctrl.Result{}, nil
	}
	if err != nil {
		// any other error will be logged
		res := r.failed(ctx, "Failed to get PipelineRun", err, pr, r.Recorder)
		return nil, &res, err
	}
	// return nil result to indicate that reconciliation can proceed
	return pr, nil, nil
}

func (r *PipelineRunReconciler) determinePipelineVersion(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun) (ctrl.Result, error) {
	log("Determining pipeline version")
	err := r.DeterminePipelineVersion(ctx, pr)
	if err != nil {
		// any other error will be logged
		return r.failed(ctx, "Failed to determine which pipeline version to run", err, pr, r.Recorder), err
	}
	state := VersionDetermined
	pr.Status.State = &state
	if err = r.SetPipelineRunStatus(ctx, log, pr, state, v1.ConditionTrue, "Pipeline version used for run: "+*pr.Status.PipelineVersion); err != nil {
		return r.failed(ctx, "Failed to set PipelineRun status", err, pr, r.Recorder), err
	}
	r.Recorder.Event(pr, "Normal", "Reconciliation", "Pipeline version determined: "+*pr.Status.PipelineVersion)
	// changes to state have been made, return empty result to stop current reconciliation iteration
	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) storePipelineStructure(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun) (ctrl.Result, error) {
	pipelineDefinition := getPipelineId(*pr)
	pd, err := GetPipelineDefinition(r, ctx, types.NamespacedName{Name: pipelineDefinition, Namespace: pr.Namespace})
	if err != nil {
		return r.failed(ctx, "Failed to load pipeline definition", err, pr, r.Recorder), err
	}
	if pd == nil {
		return r.failed(ctx, "No such pipeline definition", err, pr, r.Recorder), err
	}
	// make a deep copy
	structure := pd.Spec.PipelineStructure.DeepCopy()
	// then remove all configs
	for _, pjs := range structure.JobSteps {
		pjs.Config = nil
	}
	message := fmt.Sprintf("Pipeline structure loaded: %d steps, %d sub-pipelines, %d pipes", len(structure.JobSteps), len(structure.SubPipelines), len(structure.Pipes))
	pr.Status.PipelineStructure = structure
	pr.Status.NumStepsTotal = len(structure.JobSteps) + len(structure.SubPipelines)
	state := StructureLoaded
	pr.Status.State = &state
	if err = r.SetPipelineRunStatus(ctx, log, pr, state, v1.ConditionTrue, message); err != nil {
		return r.failed(ctx, "Failed to set PipelineRun status", err, pr, r.Recorder), err
	}
	r.Recorder.Event(pr, "Normal", "Reconciliation", message)
	// changes to state have been made, return empty result to stop current reconciliation iteration
	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) startStartableStep(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun) (*ctrl.Result, error) {
	if step := findNextStartableStep(pr); step != nil {
		log("Starting step: " + step.Id)
		jobName := r.ConstructPipelineJobName(pr, step.Id)
		if !isPVCActive(pr, step.Id) {
			// create pvc
			var volumeSize int64 = 10 // TODO replace by resource specs
			if _, err := r.CreatePersistentVolumeClaim(ctx, log, pr, jobName, volumeSize); err != nil {
				result := r.failed(ctx, "Failed to create PersistentVolume", err, pr, r.Recorder)
				return &result, err
			}
			// set volume flag to true
			if err := r.setPVCStatus(ctx, log, pr, step.Id, v1.ConditionTrue, "pvc was created"); err != nil {
				result := r.failed(ctx, "Failed to set pvc flag active", err, pr, r.Recorder)
				return &result, err
			}
		}
		state := "Started " + step.Id
		pr.Status.State = &state
		if err := r.SetPipelineRunStatus(ctx, log, pr, StepStatus(step.Id), v1.ConditionUnknown, "Started step: "+step.Id); err != nil {
			result := r.failed(ctx, "Failed to update PipelineRunStatus for job step "+step.Id, err, pr, r.Recorder)
			return &result, err
		}
		if err := r.CreatePipelineJob(ctx, log, pr, jobName, step); err != nil {
			result := r.failed(ctx, "Failed to create PipelineJob for step "+step.Id, err, pr, r.Recorder)
			return &result, err
		}
		r.Recorder.Event(pr, "Normal", "PipelineExecution", "Created PipelineJob: "+step.Id)
		// changes to state have been made, return empty result to stop current reconciliation iteration
		return &ctrl.Result{}, nil
	}
	// return nil result to indicate that reconciliation can proceed
	return nil, nil
}

// find a job step that is startable (i.e. not active yet and all input steps have succeeded)
func findNextStartableStep(pr *pipelinev1.PipelineRun) *pipelinev1.PipelineJobStepSpec {
	for _, step := range pr.Status.PipelineStructure.JobSteps {
		if !isActive(pr, step.Id) && allInputsSucceeded(pr, step.Id) {
			return step
		}
	}
	return nil
}

// check all input steps (those at the other end of a pipe that has the given step as target), whether they succeeded
func allInputsSucceeded(pr *pipelinev1.PipelineRun, step string) bool {
	for _, pipe := range pr.Status.PipelineStructure.Pipes {
		if (pipe.To.StepId == step) && !hasSucceeded(pr, pipe.From.StepId) {
			return false
		}
	}
	return true
}

// check all dependent steps (those at the other end of a pipe that has the given step as source), whether they succeeded
func allOutputsSucceeded(pr *pipelinev1.PipelineRun, step string) bool {
	for _, pipe := range pr.Status.PipelineStructure.Pipes {
		if (pipe.From.StepId == step) && !hasSucceeded(pr, pipe.To.StepId) {
			return false
		}
	}
	return true
}

func (r *PipelineRunReconciler) updateStepStatistics(ctx context.Context, pr *pipelinev1.PipelineRun) (*ctrl.Result, error) {
	// count steps per success status
	count := map[v1.ConditionStatus]int{}
	for _, condition := range pr.Status.Conditions {
		if strings.HasPrefix(condition.Type, SUCCESS_STATUS_PREFIX) {
			count[condition.Status]++
		}
	}
	// update counts if needed
	if (pr.Status.NumStepsActive != count[v1.ConditionUnknown]) || (pr.Status.NumStepsSucceeded != count[v1.ConditionTrue]) || (pr.Status.NumStepsFailed != count[v1.ConditionFalse]) {
		message := "Step statistics has changed: " + strconv.Itoa(pr.Status.NumStepsActive) + "/" + strconv.Itoa(pr.Status.NumStepsSucceeded) + "/" + strconv.Itoa(pr.Status.NumStepsFailed) + " --> " + strconv.Itoa(count[v1.ConditionUnknown]) + "/" + strconv.Itoa(count[v1.ConditionTrue]) + "/" + strconv.Itoa(count[v1.ConditionFalse])
		pr.Status.NumStepsActive = count[v1.ConditionUnknown]
		pr.Status.NumStepsSucceeded = count[v1.ConditionTrue]
		pr.Status.NumStepsFailed = count[v1.ConditionFalse]
		if err := r.Status().Update(ctx, pr); err != nil {
			result := r.failed(ctx, "Failed to update step statistics of PipelineRun", err, pr, r.Recorder)
			return &result, err
		}
		r.Recorder.Event(pr, "Normal", "PipelineExecution", message)
		// changes to state have been made, return empty result to stop current reconciliation iteration
		return &ctrl.Result{}, nil
	}

	// return nil result to indicate that reconciliation can proceed
	return nil, nil
}

func (r *PipelineRunReconciler) removeUnneededPipelineJob(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun) (*ctrl.Result, error) {
	for _, step := range pr.Status.PipelineStructure.JobSteps {
		if hasSucceeded(pr, step.Id) && isPVCActive(pr, step.Id) && allOutputsSucceeded(pr, step.Id) {
			jobName := r.ConstructPipelineJobName(pr, step.Id)

			// delete the job, otherwise pvc will still be bound
			if err := r.DeletePipelineJob(ctx, log, pr, jobName); err != nil {
				result := r.failed(ctx, "Failed to delete PipelineJob", err, pr, r.Recorder)
				return &result, err
			}

			// delete the volume, it will not be needed anymore
			if err := r.DeletePersistentVolumeClaim(ctx, log, pr, jobName); err != nil {
				result := r.failed(ctx, "Failed to delete PersistentVolumeClaim", err, pr, r.Recorder)
				return &result, err
			}

			// set volume flag to false
			if err := r.setPVCStatus(ctx, log, pr, step.Id, v1.ConditionFalse, "pvc was deleted"); err != nil {
				result := r.failed(ctx, "Failed to set pvc flag inactive", err, pr, r.Recorder)
				return &result, err
			}

			// changes to state have been made, return empty result to stop current reconciliation iteration
			return &ctrl.Result{}, nil
		}
	}

	// return nil result to indicate that reconciliation can proceed
	return nil, nil
}

func (r *PipelineRunReconciler) determineTerminalRunState(ctx context.Context, pr *pipelinev1.PipelineRun) (*ctrl.Result, error) {
	allSucceeded := true
	someFailed := false
	for _, step := range pr.Status.PipelineStructure.JobSteps {
		if !hasSucceeded(pr, step.Id) {
			allSucceeded = false
		}
		if hasFailed(pr, step.Id) {
			someFailed = true
		}
	}
	oldState := *pr.Status.State
	newState := oldState
	if allSucceeded {
		newState = Succeeded
	}
	if someFailed {
		newState = Failed
	}
	if oldState != newState {
		pr.Status.State = &newState
		if err := r.Status().Update(ctx, pr); err != nil {
			result := r.failed(ctx, "Failed to update state of PipelineRun", err, pr, r.Recorder)
			return &result, err
		}
		message := "Pipeline has terminated with result: " + newState
		r.Recorder.Event(pr, "Normal", "PipelineRunTerminated", message)

		// changes to state have been made, return empty result to stop current reconciliation iteration
		return &ctrl.Result{}, nil
	}

	// return nil result to indicate that reconciliation can proceed
	return nil, nil
}

func isTrue(pr *pipelinev1.PipelineRun, condition string) bool {
	return meta.IsStatusConditionPresentAndEqual(pr.Status.Conditions, condition, v1.ConditionTrue)
}

func isFalse(pr *pipelinev1.PipelineRun, condition string) bool {
	return meta.IsStatusConditionPresentAndEqual(pr.Status.Conditions, condition, v1.ConditionFalse)
}

func isActive(pr *pipelinev1.PipelineRun, stepId string) bool {
	return meta.FindStatusCondition(pr.Status.Conditions, StepStatus(stepId)) != nil
}

func hasSucceeded(pr *pipelinev1.PipelineRun, stepId string) bool {
	return isTrue(pr, StepStatus(stepId))
}

func hasFailed(pr *pipelinev1.PipelineRun, stepId string) bool {
	return isFalse(pr, StepStatus(stepId))
}

func StepStatus(stepId string) string {
	return SUCCESS_STATUS_PREFIX + stepId
}

func isPVCActive(pr *pipelinev1.PipelineRun, stepId string) bool {
	return isTrue(pr, PVCStatus(stepId))
}

func PVCStatus(stepId string) string {
	return PVC_STATUS_PREFIX + stepId
}

func (r *PipelineRunReconciler) setPVCStatus(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun, stepId string, state v1.ConditionStatus, message string) error {
	return r.SetPipelineRunStatus(ctx, log, pr, PVCStatus(stepId), state, message)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineRun{}).
		//Owns(&pipelinev1.PipelineJob{}).
		//Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}

// called whenever an error occurred, to create an error event
func (r *PipelineRunReconciler) failed(ctx context.Context, errormessage string, err error, pr *pipelinev1.PipelineRun, recorder record.EventRecorder) ctrl.Result {
	if err != nil {
		errormessage = errormessage + ": " + err.Error()
	}
	errState := "Error (" + errormessage + ")"
	pr.Status.State = &errState
	if err := r.Status().Update(ctx, pr); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update state to "+errormessage)
	}
	recorder.Event(pr, "Warning", "ReconciliationError", errormessage)
	return ctrl.Result{}
}

func (r *PipelineRunReconciler) ConstructPipelineJobName(pr *pipelinev1.PipelineRun, stepId string) string {
	return pr.Name + "-" + stepId
}
