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
	"strings"

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
		return r.failed(ctx, "could not extract config from pipeline definition", err, pd, r.Recorder), err
	}

	// update configmap with same name as pipeline definition
	if result, err := r.updateConfigMap(ctx, log, pd, conf, req.NamespacedName); result != nil {
		return *result, err
	}

	// create service accounts
	if result, err := r.updateServiceAccount(ctx, log, pd); result != nil {
		return *result, err
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
		res := r.failed(ctx, "Failed to get PipelineJob", err, pj, r.Recorder)
		return nil, &res, err
	}
	// return nil result to indicate that reconciliation can proceed
	return pj, nil, nil
}

// called whenever an error occurred, to create an error event
func (r *PipelineDefinitionReconciler) failed(ctx context.Context, errormessage string, err error, pd *pipelinev1.PipelineDefinition, recorder record.EventRecorder) ctrl.Result {
	if err != nil {
		errormessage = errormessage + ": " + err.Error()
	}
	//errState := "Error (" + errormessage + ")"
	//pd.Status.State = &errState # TODO add state
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
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}

func (r *PipelineDefinitionReconciler) updateConfigMap(ctx context.Context, log func(string, ...interface{}), pd *pipelinev1.PipelineDefinition, conf map[string]string, name types.NamespacedName) (*ctrl.Result, error) {
	// get configmap if it exists
	cm, err := r.GetConfigMap(ctx, name)
	if err != nil {
		res := r.failed(ctx, "Failed to get ConfigMap", err, pd, r.Recorder)
		return &res, err
	}
	if cm == nil {
		// none exists, create it
		if _, err := r.CreateConfigMap(ctx, log, pd, pd.Name, conf); err != nil {
			res := r.failed(ctx, "Failed to create ConfigMap", err, pd, r.Recorder)
			return &res, err
		}
		r.Recorder.Event(pd, "Normal", "Reconciliation", "ConfigMap created")

		// changes made end reconciliation iteration
		return &ctrl.Result{}, nil
	} else {
		if reflect.DeepEqual(conf, cm.Data) {
			// continue reconciliation
			return nil, nil
		}
		// not equal, update it
		if err := r.UpdateConfigMap(ctx, log, cm, conf); err != nil {
			res := r.failed(ctx, "Failed to update ConfigMap", err, pd, r.Recorder)
			return &res, err
		}
		r.Recorder.Event(pd, "Normal", "Reconciliation", "ConfigMap updated")

		// changes made end reconciliation iteration
		return &ctrl.Result{}, nil
	}
}

func (r *PipelineDefinitionReconciler) updateServiceAccount(ctx context.Context, log func(string, ...interface{}), pd *pipelinev1.PipelineDefinition) (*ctrl.Result, error) {
	// check all service accounts defined in job steps
	for _, step := range pd.Spec.PipelineStructure.JobSteps {
		name := step.JobSpec.ServiceAccountName
		if len(name) > 0 {
			namespacedName := types.NamespacedName{Namespace: pd.Namespace, Name: name}
			// check if a service account already exists
			sa, err := r.GetServiceAccount(ctx, namespacedName)
			if err != nil {
				res := r.failed(ctx, "Failed to get service account", err, pd, r.Recorder)
				return &res, err
			}
			// if sa does not exist, create it
			if sa == nil {
				annotationLabel := "iam.gke.io/gcp-service-account" // TODO in config
				//annotationTemplate := "{name}@breuni-team-admin-{namespace}.iam.gserviceaccount.com" // TODO in config
				annotationTemplate := "bucket-lister-sa@breuninger-pipeline-processing.iam.gserviceaccount.com"
				if _, err := r.CreateServiceAccount(ctx, log, pd, namespacedName, annotationLabel, resolve(annotationTemplate, pd.Namespace, name)); err != nil {
					res := r.failed(ctx, "Failed to create service account", err, pd, r.Recorder)
					return &res, err
				}
				r.Recorder.Event(pd, "Normal", "Reconciliation", "Created service account "+name)

				// changes made: end reconciliation iteration
				return &ctrl.Result{}, nil
			}
			// check if pd is already registered as owner
			added, err := r.addOwnership(ctx, log, sa, pd)
			if err != nil {
				res := r.failed(ctx, "Failed to add ownership to service account", err, pd, r.Recorder)
				return &res, err
			}
			if added {
				r.Recorder.Event(pd, "Normal", "Reconciliation", "Added ownership reference to service account "+name)

				// changes made: end reconciliation iteration
				return &ctrl.Result{}, nil
			}
		}
	}

	// nothing done, continue reconciliation
	return nil, nil
}

func resolve(template string, namespace string, name string) string {
	res := template
	res = strings.ReplaceAll(res, "{namespace}", namespace)
	res = strings.ReplaceAll(res, "{name}", name)
	return res
}
