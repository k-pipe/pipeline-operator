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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
)

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
	log.Info("Starting reconciliation (PipelineRun)")

	// get the pipeline run by name from request
	pr, err := r.GetPipelineRun(ctx, req.NamespacedName)
	if pr == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log.Info("PipelineRun resource not found. Ignoring since object has been deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		// any other error will be logged
		return failed("Failed to get PipelineRun", pr, r.Recorder), err
	}

	// determine pipeline version if not set, yet
	if (pr.Status.PipelineVersion == nil) || !isTrue(pr, VersionDetermined) {
		log.Info("Determining pipeline version")
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
	}

	// load pipeline structure if not set, yet
	if (pr.Status.PipelineStructure == nil) || !isTrue(pr, StructureLoaded) {
		pipelineDefinition := pr.Spec.PipelineName + "-" + *pr.Status.PipelineVersion
		pd, err := GetPipelineDefinition(r, ctx, types.NamespacedName{Name: pipelineDefinition, Namespace: pr.Namespace})
		if err != nil {
			return failed("Failed to load pipeline definition: "+pipelineDefinition, pr, r.Recorder), err
		}
		if pd == nil {
			return failed("No such pipeline definition: "+pipelineDefinition, pr, r.Recorder), err
		}
		pr.Status.PipelineStructure = &pd.Spec.PipelineStructure
		message := fmt.Sprintf("Pipeline structure loaded: %d steps, %d pipes", len(pr.Status.PipelineStructure.Steps), len(pr.Status.PipelineStructure.Pipes))
		state := StructureLoaded
		pr.Status.State = &state
		if err = r.SetPipelineRunStatus(ctx, pr, state, v1.ConditionTrue, message); err != nil {
			return failed("Failed to set PipelineRun status", pr, r.Recorder), err
		}
		r.Recorder.Event(pr, "Normal", "Reconciliation", message)
	}

	// if not paused nor terminated ...
	if !(isTrue(pr, Paused) || isTrue(pr, Terminated)) {
		log.Info("Updating job states")
		//  updade running step state
		// TODO

		// find steps that can be started
		startable := findStartableSteps(pr)
		if len(startable) > 0 {
			var ids []string
			for _, s := range startable {
				ids = append(ids, s.Id)
			}
			log.Info(fmt.Sprintf("%d steps are startable", len(startable)), "startable", strings.Join(ids, ","))
			for _, step := range findStartableSteps(pr) {
				log.Info("Starting step: " + step.Id)
				if err := startStep(ctx, pr, step); err != nil {
					return failed("Failed to create job for step: "+step.Id, pr, r.Recorder), err
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// find all job steps that are startable (i.e. not active yet and all input steps have succeeded)
func findStartableSteps(pr *pipelinev1.PipelineRun) []*pipelinev1.PipelineStepSpec {
	var res []*pipelinev1.PipelineStepSpec
	for _, step := range pr.Status.PipelineStructure.Steps {
		if !isActive(pr, step.Id) && (step.Type == STEP_TYPE_JOB) {
			if allInputsSucceeded(pr, step.Id) {
				res = append(res, step)
			}
		}
	}
	return res
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

func startStep(ctx context.Context, pr *pipelinev1.PipelineRun, step *pipelinev1.PipelineStepSpec) error {
	//var j interface{}
	//err := json.Unmarshal(step.Specification, &j)
	//fmt.Println(err)
	// TODO
	fmt.Println(step.Specification)

	return nil
}

func isTrue(pr *pipelinev1.PipelineRun, condition string) bool {
	return meta.IsStatusConditionPresentAndEqual(pr.Status.Conditions, condition, v1.ConditionTrue)
}

func isActive(pr *pipelinev1.PipelineRun, stepId string) bool {
	return meta.FindStatusCondition(pr.Status.Conditions, stepStatus(stepId)) != nil
}

func hasSucceeded(pr *pipelinev1.PipelineRun, stepId string) bool {
	return isTrue(pr, stepStatus(stepId))
}

func stepStatus(stepId string) string {
	return "success-" + stepId
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineRun{}).
		Owns(&pipelinev1.PipelineJob{}).
		Complete(r)
}
