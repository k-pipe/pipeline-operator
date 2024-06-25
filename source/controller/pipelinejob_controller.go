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
	"sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	log := Logger(ctx, req, "PJ")
	defer LoggingDone(log)

	// get the pipeline job by name from request
	pj, result, err := r.loadResource(ctx, log, req.NamespacedName)
	if result != nil {
		return *result, err
	}

	// create job if it does not exist, yet
	j, err := r.GetJob(ctx, req.NamespacedName)
	if err != nil {
		return r.failed(ctx, "Failed to get PipelineJob", pj, r.Recorder), err
	}
	if j == nil {
		return r.createJob(ctx, log, pj)
	}

	// update job status if changed
	if result, err = r.updatedJobStatus(ctx, log, pj, j); result != nil {
		return *result, err
	}
	log("End of reconcile")
	return ctrl.Result{}, nil
}

func (r *PipelineJobReconciler) createJob(ctx context.Context, log func(string, ...interface{}), pj *pipelinev1.PipelineJob) (ctrl.Result, error) {
	j, err := r.CreateJob(ctx, log, pj)
	if err != nil {
		return r.failed(ctx, "Failed to create Job", pj, r.Recorder), err
	}

	state := "Job created"
	pj.Status.State = &state
	if r.SetPipelineJobStatus(ctx, log, pj, JobCreated, metav1.ConditionTrue, "Created Job: "+j.Name) != nil {
		return r.failed(ctx, "Failed to set JobCreated status: "+j.Name, pj, r.Recorder), err
	}
	r.Recorder.Event(pj, "Normal", "Reconciliation", "Created Job: "+j.Name)

	// changes to state have been made, return empty result to stop current reconciliation iteration
	return ctrl.Result{}, nil
}

func (r *PipelineJobReconciler) loadResource(ctx context.Context, log func(string, ...interface{}), name types.NamespacedName) (*pipelinev1.PipelineJob, *ctrl.Result, error) {
	pj, err := r.GetPipelineJob(ctx, name)
	if pj == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log("PipelineJob resource not found. Ignoring since object has been deleted")
		return nil, &ctrl.Result{}, nil
	}
	if err != nil {
		// any other error will be logged
		res := r.failed(ctx, "Failed to get PipelineJob", pj, r.Recorder)
		return nil, &res, err
	}
	// return nil result to indicate that reconciliation can proceed
	return pj, nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineJob{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

// called whenever an error occurred, to create an error event
func (r *PipelineJobReconciler) failed(ctx context.Context, errormessage string, pj *pipelinev1.PipelineJob, recorder record.EventRecorder) ctrl.Result {
	errState := "Error (" + errormessage + ")"
	pj.Status.State = &errState
	if err := r.Status().Update(ctx, pj); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update state to "+errormessage)
	}
	recorder.Event(pj, "Warning", "ReconciliationError", errormessage)
	return ctrl.Result{}
}

func (r *PipelineJobReconciler) updatedJobStatus(ctx context.Context, log func(string, ...interface{}), pj *pipelinev1.PipelineJob, j *batchv1.Job) (*ctrl.Result, error) {
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
			res := r.failed(ctx, "Failed to get PipelineRun resource for updating JobSucceeded status", pj, r.Recorder)
			return &res, err
		}
		if err := r.SetPipelineRunStatus(ctx, log, pr, StepStatus(pj.Spec.StepId), newSucceededState, message); err != nil {
			res := r.failed(ctx, "Failed to set PipelineRun status", pj, r.Recorder)
			return &res, err
		}

		// then set it on PipelineJob
		var state string
		switch newSucceededState {
		case metav1.ConditionTrue:
			state = "Done"
		case metav1.ConditionFalse:
			state = "Failed"
		case metav1.ConditionUnknown:
			state = "Created"
		}
		pj.Status.State = &state
		if r.SetPipelineJobStatus(ctx, log, pj, JobSucceeded, newSucceededState, message) != nil {
			res := r.failed(ctx, "Failed to set PipelineJob succeeded status", pj, r.Recorder)
			return &res, err
		}

		// finally record an event if successful
		r.Recorder.Event(pj, "Normal", "Reconciliation", message)

		res := ctrl.Result{}
		return &res, nil
	}

	// return nil result to indicate that reconciliation can proceed
	return nil, nil
}
