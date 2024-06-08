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

const STATUS_PREFIX = "success-"

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
	log := log.FromContext(ctx)
	log.Info("========================= Starting reconciliation (PipelineRun) =========================")

	// get the pipeline run by name from request
	pr, result, err := r.loadResource(ctx, req.NamespacedName)
	if result != nil {
		return *result, err
	}

	// determine pipeline version if not set, yet
	if (pr.Status.PipelineVersion == nil) || !isTrue(pr, VersionDetermined) {
		return r.determinePipelineVersion(ctx, pr)
	}

	// load pipeline structure if not set, yet
	if (pr.Status.PipelineStructure == nil) || !isTrue(pr, StructureLoaded) {
		return r.storePipelineStructure(ctx, pr)
	}

	// if not paused nor terminated ...
	if !(isTrue(pr, Paused) || isTrue(pr, Terminated)) {
		result, err := r.startStartableStep(ctx, pr)
		if result != nil {
			return *result, err
		}
	}

	// update step statistics
	result, err = r.updateStepStatistics(ctx, pr)
	if result != nil {
		return *result, err
	}

	log.Info("========================= Terminated reconciliation (PipelineRun) =========================")
	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) loadResource(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineRun, *ctrl.Result, error) {
	pr, err := r.GetPipelineRun(ctx, name)
	if pr == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log.FromContext(ctx).Info("PipelineRun resource not found. Ignoring since object has been deleted")
		return nil, &ctrl.Result{}, nil
	}
	if err != nil {
		// any other error will be logged
		res := failed("Failed to get PipelineRun", pr, r.Recorder)
		return nil, &res, err
	}
	// return nil result to indicate that reconciliation can proceed
	return pr, nil, nil
}

func (r *PipelineRunReconciler) determinePipelineVersion(ctx context.Context, pr *pipelinev1.PipelineRun) (ctrl.Result, error) {
	log.FromContext(ctx).Info("Determining pipeline version")
	err := r.DeterminePipelineVersion(ctx, pr)
	if err != nil {
		// any other error will be logged
		return failed("Failed to determine which pipeline version to run", pr, r.Recorder), err
	}
	state := VersionDetermined
	pr.Status.State = &state
	if err = r.SetPipelineRunStatus(ctx, pr, state, v1.ConditionTrue, "Pipeline version used for run: "+*pr.Status.PipelineVersion); err != nil {
		return failed("Failed to set PipelineRun status", pr, r.Recorder), err
	}
	r.Recorder.Event(pr, "Normal", "Reconciliation", "Pipeline version determined: "+*pr.Status.PipelineVersion)
	// changes to state have been made, return empty result to stop current reconciliation iteration
	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) storePipelineStructure(ctx context.Context, pr *pipelinev1.PipelineRun) (ctrl.Result, error) {
	pipelineDefinition := pr.Spec.PipelineName + "-" + *pr.Status.PipelineVersion
	pd, err := GetPipelineDefinition(r, ctx, types.NamespacedName{Name: pipelineDefinition, Namespace: pr.Namespace})
	if err != nil {
		return failed("Failed to load pipeline definition: "+pipelineDefinition, pr, r.Recorder), err
	}
	if pd == nil {
		return failed("No such pipeline definition: "+pipelineDefinition, pr, r.Recorder), err
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
	if err = r.SetPipelineRunStatus(ctx, pr, state, v1.ConditionTrue, message); err != nil {
		return failed("Failed to set PipelineRun status", pr, r.Recorder), err
	}
	r.Recorder.Event(pr, "Normal", "Reconciliation", message)
	// changes to state have been made, return empty result to stop current reconciliation iteration
	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) startStartableStep(ctx context.Context, pr *pipelinev1.PipelineRun) (*ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Looking for steps that can be started")
	if step := findNextStartableStep(pr); step != nil {
		log.Info("Step is startable: " + step.Id)
		log.Info("Starting step: " + step.Id)
		if err := r.CreatePipelineJob(ctx, pr, step); err != nil {
			result := failed("Failed to create job for step: "+step.Id, pr, r.Recorder)
			return &result, err
		}
		if err := r.SetPipelineRunStatus(ctx, pr, StepStatus(step.Id), v1.ConditionUnknown, "Started step: "+step.Id); err != nil {
			result := failed("Failed to update PipelineRunStatus for job step: "+step.Id, pr, r.Recorder)
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

func (r *PipelineRunReconciler) updateStepStatistics(ctx context.Context, pr *pipelinev1.PipelineRun) (*ctrl.Result, error) {
	// count steps per success status
	count := map[v1.ConditionStatus]int{}
	for _, condition := range pr.Status.Conditions {
		if strings.HasPrefix(condition.Type, STATUS_PREFIX) {
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
			result := failed("Failed to update step statistics of PipelineRun", pr, r.Recorder)
			return &result, err
		}
		r.Recorder.Event(pr, "Normal", "PipelineExecution", message)
		// changes to state have been made, return empty result to stop current reconciliation iteration
		return &ctrl.Result{}, nil
	}
	// return nil result to indicate that reconciliation can proceed
	return nil, nil
}

func isTrue(pr *pipelinev1.PipelineRun, condition string) bool {
	return meta.IsStatusConditionPresentAndEqual(pr.Status.Conditions, condition, v1.ConditionTrue)
}

func isActive(pr *pipelinev1.PipelineRun, stepId string) bool {
	return meta.FindStatusCondition(pr.Status.Conditions, StepStatus(stepId)) != nil
}

func hasSucceeded(pr *pipelinev1.PipelineRun, stepId string) bool {
	return isTrue(pr, StepStatus(stepId))
}

func StepStatus(stepId string) string {
	return STATUS_PREFIX + stepId
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineRun{}).
		Owns(&pipelinev1.PipelineJob{}).
		Complete(r)
}
