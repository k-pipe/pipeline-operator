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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"time"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO add validation webhook according to this: https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation

// PipelineScheduleReconciler reconciles a PipelineSchedule object
type PipelineScheduleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelineschedules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelineschedules/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelineschedules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PipelineSchedule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PipelineScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := Logger(ctx, req, "PS")
	defer LoggingDone(log)

	// TODO make this configurable!
	requeue := ctrl.Result{RequeueAfter: time.Minute}

	// get the pipeline schedule by name from request
	ps, err := r.GetPipelineSchedule(ctx, req.NamespacedName)
	if ps == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log("PipelineSchedule resource not found. Ignoring since object has been deleted")
		return requeue, nil
	}
	if err != nil {
		// any other error will be logged
		return failed("Failed to get PipelineSchedule", ps, r.Recorder), err
	}

	// determine which schedule is expected (this depends on current time)
	sir, err := r.GetExpectedScheduleInRange(ctx, *ps)
	if err != nil {
		return failed("Failed to determine expected schedule", ps, r.Recorder), err
	}

	// get the cronjob with identical name
	cj, err := r.GetCronJob(ctx, req.NamespacedName)
	if err != nil {
		return failed("Failed to get cronjob from API", ps, r.Recorder), err
	}

	// if state is consistent with expectation, end conciliation
	consistent, message := stateConsistent(sir, cj)
	if consistent {
		// is consistent, set "UpToDate" to true
		if err = r.SetUpToDateStatus(ctx, log, ps, v1.ConditionTrue, message); err != nil {
			return failed("Failed to set UpToDateStatus", ps, r.Recorder), err
		}
		// end reconcilition
		//log.Info(prefix + "Nothing to reconcile")
		return requeue, nil
	} else {
		log("Reason for reconciliation: " + message)
		// copy selected schedule in range data to status (for additionalPrinterColumns), this will be written together with UpToDateStatus
		r.SetStatus(ps, sir)

		// set UpToDate to false
		if err := r.SetUpToDateStatus(ctx, log, ps, v1.ConditionFalse, message); err != nil {
			return failed("Failed to clear UpToDateStatus", ps, r.Recorder), err
		}
	}

	// if no schedule in range, delete cronjob
	var event string
	if sir == nil {
		log("No schedule in current time range expected, deleting CronJob")
		if err := r.DeleteCronJob(ctx, log, cj); err != nil {
			return failed("Failed to delete CronJob", ps, r.Recorder), err
		}
		event = "CronJob has been deleted"
	} else {
		// if no cronjob exists, create one
		if cj == nil {
			if err := r.CreateCronJob(ctx, log, ps, *sir); err != nil {
				return failed("Failed to create CronJob", ps, r.Recorder), err
			}
			event = "CronJob has been created"
		} else {
			// update existing cronjob with new schedule in range
			if err := r.UpdateCronJob(ctx, *sir, cj); err != nil {
				return failed("Failed to update CronJob", ps, r.Recorder), err
			}
			event = "CronJob has been updated"
		}
	}

	// register an event for the executed action
	r.Recorder.Event(ps, "Normal", "Reconciliation", event+" ("+message+")")

	// refetch the pipeline schedule object
	if err := r.Get(ctx, NameSpacedName(ps), ps); err != nil {
		return failed("Failed to re-fetch PipelineSchedule", ps, r.Recorder), err
	}

	// set UpToDate status
	if err = r.SetUpToDateStatus(ctx, log, ps, v1.ConditionTrue, event); err != nil {
		return failed("Failed to set UpToDateStatus", ps, r.Recorder), err
	}

	return requeue, nil
}

// determine if given ScheduleInRange is consistent with CronJob
func stateConsistent(sir *pipelinev1.ScheduleInRange, cj *batchv1.CronJob) (bool, string) {
	if (sir == nil) && (cj != nil) {
		// one is nil, the other isn't --> inconsistent state
		return false, "CronJob exists, but shouldn't"
	}
	if (sir != nil) && (cj == nil) {
		// one is nil, the other isn't --> inconsistent state
		return false, "CronJob should exist, but doesn't"
	}
	if sir == nil {
		// both are nil --> consistent state
		return true, "no CronJob exists - as expected"
	}
	// check time zone
	if (sir.TimeZone == nil) != (cj.Spec.TimeZone == nil) {
		// one is nil, the other isn't --> inconsistent state
		return false, "time zone specs existence has changed"
	}
	if (sir.TimeZone != nil) && (*sir.TimeZone != *cj.Spec.TimeZone) {
		// inconsistent timezone strings
		return false, "time zone value has changed"
	}
	if sir.CronSpec != cj.Spec.Schedule {
		// inconsistent schedule strings
		return false, "schedule value has changed"
	}
	if sir.VersionPattern != cj.ObjectMeta.Labels[PipeLineVersionPatternLabel] {
		// inconsistent version pattern strings
		return false, "version pattern has changed"
	}
	return true, "CronJob parameters match expectation"
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineSchedule{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}
