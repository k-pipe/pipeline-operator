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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	ps := &pipelinev1.PipelineSchedule{}
	fmt.Println("=======================================================================")
	fmt.Println("PipelineSchedule before Get:")
	fmt.Println(ps)
	fmt.Println("=======================================================================")

	// Get the pipeline schedule
	err := r.GetPipelineSchedule(ctx, req, ps)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("PipelineSchedule resource not found. Ignoring since object must be deleted")

			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get PipelineSchedule")

		return ctrl.Result{}, err
	}
	fmt.Println("=======================================================================")
	fmt.Println("PipelineSchedule after Get:")
	fmt.Println(ps)
	fmt.Println("=======================================================================")

	// Try to set initial condition status
	err = r.SetInitialPSCondition(ctx, req, ps)
	if err != nil {
		log.Error(err, "failed to set initial condition")

		return ctrl.Result{}, err
	}
	fmt.Println("=======================================================================")
	fmt.Println("PipelineSchedule after SetInitialCondition:")
	fmt.Println(ps)
	fmt.Println("=======================================================================")

	// Deployment if not exist
	ok, err := r.CronJobIfNotExist(ctx, req, ps)
	if err != nil {
		log.Error(err, "failed to deploy cronjob for PipelineSchedule")
		return ctrl.Result{}, err
	}

	if ok {
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Update deployment replica if mis matched.
	err = r.UpdateCronJob(ctx, req, ps)
	if err != nil {
		log.Error(err, "failed to update CronJob for PipelineSchedule")

		return ctrl.Result{}, err
	}

	requeueIntervalMinutes := DefaultReconciliationInterval

	log.Info("ending reconciliation")

	// TODO this appears to be a bad use of duration, rewrite this!
	return ctrl.Result{RequeueAfter: time.Duration(time.Minute * time.Duration(requeueIntervalMinutes))}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineSchedule{}).
		Complete(r)
}
