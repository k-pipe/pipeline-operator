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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PipelineDefinitionReconciler reconciles a PipelineDefinition object
type PipelineDefinitionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelinedefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelinedefinitions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pipeline.k-pipe.cloud,resources=pipelinedefinitions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PipelineDefinition object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *PipelineDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := Logger(ctx, req, "PD")
	defer LoggingDone(log)

	// get the pipeline definition by name from request
	pd, result, err := r.loadResource(ctx, log, req.NamespacedName)
	if result != nil {
		return *result, err
	}

	// extract desired configmap from definition
	conf, err := extractConfigAsMap(pd.Spec.PipelineStructure)
	if err != nil {
		return r.failed(ctx, "could not extract config from pipeline definition", pd, r.Recorder), err
	}

	// get configmap if it exists
	cm, err := r.GetConfigMap(ctx, req.NamespacedName)
	if err != nil {
		return r.failed(ctx, "Failed to get ConfigMap", pd, r.Recorder), err
	}
	if cm == nil {
		// none exists, create it
		if _, err := r.CreateConfigMap(ctx, log, pd, pd.Name, conf); err != nil {
			return r.failed(ctx, "Failed to create ConfigMap", pd, r.Recorder), err
		}
		r.Recorder.Event(pd, "Normal", "Reconciliation", "ConfigMap created")
	} else {
		// compare, if not equal, update it
		if !reflect.DeepEqual(conf, cm.Data) {
			if err := r.UpdateConfigMap(ctx, log, cm, conf); err != nil {
				return r.failed(ctx, "Failed to update ConfigMap", pd, r.Recorder), err
			}
			r.Recorder.Event(pd, "Normal", "Reconciliation", "ConfigMap updated")
		}
	}

	// reconciliation done
	return ctrl.Result{}, nil
}

func extractConfigAsMap(structure pipelinev1.PipelineStructure) (map[string]string, error) {
	res := map[string]string{}
	for _, jobStep := range structure.JobSteps {
		value, err := json.Marshal(&jobStep.Config)
		if err != nil {
			return nil, err
		}
		res[jobStep.Id] = string(value)
	}
	return res, nil
}

func (r *PipelineDefinitionReconciler) loadResource(ctx context.Context, log func(string, ...interface{}), name types.NamespacedName) (*pipelinev1.PipelineDefinition, *ctrl.Result, error) {
	pj, err := GetPipelineDefinition(r, ctx, name)
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

// called whenever an error occurred, to create an error event
func (r *PipelineDefinitionReconciler) failed(ctx context.Context, errormessage string, pd *pipelinev1.PipelineDefinition, recorder record.EventRecorder) ctrl.Result {
	//errState := "Error (" + errormessage + ")"
	//pd.Status.State = &errState
	if err := r.Status().Update(ctx, pd); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update state to "+errormessage)
	}
	recorder.Event(pd, "Warning", "ReconciliationError", errormessage)
	return ctrl.Result{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("pipeline-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineDefinition{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
