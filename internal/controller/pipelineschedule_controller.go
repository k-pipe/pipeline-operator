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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
)

// PipelineScheduleReconciler reconciles a PipelineSchedule object
type PipelineScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
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
	log := log.FromContext(ctx)
	log.Info("starting reconciliation (pipelineschedule v2)")

	// TODO make this configurable!
	requeue := ctrl.Result{RequeueAfter: time.Minute}
	noResult := ctrl.Result{}

	// get the pipeline schedule by name from request
	ps, err := r.GetPipelineSchedule(ctx, req.NamespacedName)
	if ps == nil {
		// not found, this may happen when a resource is deleted, just end the reconciliation
		log.Info("PipelineSchedule resource not found. Ignoring since object has been deleted")
		return requeue, nil
	}
	if err != nil {
		// any other error will be logged
		log.Error(err, "Failed to get PipelineSchedule")
		return noResult, err
	}

	// determine which schedule is expected (this depends on current time)
	sir, err := r.GetExpectedScheduleInRange(ctx, *ps)
	if err != nil {
		log.Error(err, "failed to determine expected schedule")
		return noResult, err
	}

	// get the cronjob with identical name
	cj, err := r.GetCronJob(ctx, req.NamespacedName)
	if err != nil {
		log.Error(err, "failed to get cronjob from API")
		return noResult, err
	}

	// if state is consistent with expectation, end conciliation
	if consistent, message := stateConsistent(sir, cj); consistent {
		// set "UpdateRequired" to false
		err = r.SetUpdateRequiredStatus(ctx, ps, v1.ConditionFalse, message)
		if err != nil {
			log.Error(err, "failed to clear UpdateRequiredStatus")
			return noResult, err
		}
		// end reconcilition
		return requeue, nil
	} else {
		// set UpdateRequired to true
		err := r.SetUpdateRequiredStatus(ctx, ps, v1.ConditionTrue, message)
		if err != nil {
			log.Error(err, "failed to set UpdateRequiredStatus")
			return noResult, err
		}
	}

	// if no schedule in range, delete cronjob
	var message string
	if sir == nil {
		log.Info("No schedule in current time range expected, deleting CronJob")
		err := r.DeleteCronJob(ctx, cj)
		if err != nil {
			log.Error(err, "failed to delete CronJob")
			return noResult, err
		}
		message = "CronJob has been deleted"
	} else {
		// if no cronjob exists, create one
		if cj == nil {
			err := r.CreateCronJob(ctx, ps, *sir)
			if err != nil {
				log.Error(err, "failed to create CronJob")
				return noResult, err
			}
			message = "CronJob has been created"
		} else {
			// update existing cronjob with new schedule in range
			err := r.UpdateCronJob(ctx, *sir, cj)
			if err != nil {
				log.Error(err, "failed to update CronJob")
				return noResult, err
			}
			message = "CronJob has been updated"
		}
	}

	// clear UpdateRequired status again
	err = r.SetUpdateRequiredStatus(ctx, ps, v1.ConditionFalse, message)
	if err != nil {
		log.Error(err, "failed to clear UpdateRequiredStatus")
		return noResult, err
	}

	log.Info("done with reconciliation")
	return requeue, nil
}

// determine if given ScheduleInRange is consistent with CronJob
func stateConsistent(sir *pipelinev1.ScheduleInRange, cj *batchv1.CronJob) (bool, string) {
	if (sir == nil) && (cj != nil) {
		// one is nil, the other isn't --> inconsistent state
		return false, "CronJob should exist, but doesn't"
	}
	if (sir != nil) && (cj == nil) {
		// one is nil, the other isn't --> inconsistent state
		return false, "CronJob exists, but shouldn't"
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineSchedule{}).
		Complete(r)
}
