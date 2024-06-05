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
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
)

// PipelineJobReconciler reconciles a PipelineJob object
type PipelineJobReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelinejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelinejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelinejobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PipelineJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PipelineJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("========================= Starting reconciliation (PipelineJob) =========================")

	// get the pipeline run by name from request
	pj, err := r.GetPipelineJob(ctx, req.NamespacedName)
	if pj == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log.Info("PipelineJob resource not found. Ignoring since object has been deleted")
		return ctrl.Result{}, nil
	}
	if err != nil {
		// any other error will be logged
		return failed("Failed to get PipelineJob", pj, r.Recorder), err
	}

	// create job if it does not exist, yet
	j, err := r.GetJob(ctx, req.NamespacedName)
	if j == nil {
		log.Info("Creating Job resource")
		j, err = r.CreateJob(ctx, pj)
		if err != nil {
			// any other error will be logged
			return failed("Failed to create Job", j, r.Recorder), err
		}

		if r.SetPipelineJobStatus(ctx, pj, JobCreated, metav1.ConditionTrue, "Created Job: "+j.Name) != nil {
			return failed("Failed to create Job: "+j.Name, j, r.Recorder), err
		}
		r.Recorder.Event(pj, "Normal", "Reconciliation", "Created Job: "+j.Name)
	}
	if err != nil {
		// any other error will be logged
		return failed("Failed to get PipelineJob", pj, r.Recorder), err
	}

	// collect the job status into one variable
	newSucceededState := metav1.ConditionUnknown

	// align with job status "failed"
	if isTrueInJob(j, batchv1.JobFailed) {
		newSucceededState = metav1.ConditionFalse
	}
	// align with job status "complete"
	if isTrueInJob(j, batchv1.JobComplete) {
		newSucceededState = metav1.ConditionTrue
	}

	// check if changed
	oldSucceededStateCondition := meta.FindStatusCondition(pj.Status.Conditions, JobSucceeded)
	var oldSucceededState metav1.ConditionStatus
	if oldSucceededStateCondition == nil {
		oldSucceededState = "NotSet"
	} else {
		oldSucceededState = oldSucceededStateCondition.Status
	}
	if newSucceededState != oldSucceededState {
		message := "JobSucceeded state has changed: " + string(oldSucceededState) + " -> " + string(newSucceededState)

		// first set on PipelineRun (to make sure to retry this in case it fails)
		pr, err := r.GetPipelineRun(ctx, types.NamespacedName{Name: pj.Spec.PipelineRun, Namespace: pj.Namespace})
		if err != nil || pr == nil {
			return failed("Failed to get PipelineRun resource for updating JobSucceeded status", j, r.Recorder), err
		}
		r.SetPipelineRunStatus(ctx, pr, StepStatus(pj.Spec.StepId), newSucceededState, message)

		// last set it on PipelineJob
		if r.SetPipelineJobStatus(ctx, pj, JobSucceeded, newSucceededState, message) != nil {
			return failed("Failed to set PipelineJob succeeded status", j, r.Recorder), err
		}

		// finally record an event if successful
		r.Recorder.Event(pj, "Normal", "Reconciliation", message)
	}
	log.Info("========================= Terminated reconciliation (PipelineJob) =========================")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineJob{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
