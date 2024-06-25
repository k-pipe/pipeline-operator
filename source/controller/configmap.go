package controller

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Gets a ConfigMap object by name from api server, returns nil,nil if not found
func (r *PipelineDefinitionReconciler) GetConfigMap(ctx context.Context, name types.NamespacedName) (*corev1.ConfigMap, error) {
	res := &corev1.ConfigMap{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

/*
create ConfigMap
*/
func (r *PipelineDefinitionReconciler) CreateConfigMap(ctx context.Context, log func(string, ...interface{}), pd *pipelinev1.PipelineDefinition, configmapName string, data map[string]string) (*corev1.ConfigMap, error) {

	// the labels to be attached to pvc
	labels := map[string]string{
		"app.kubernetes.io/name":       "Pipeline-ConfigMap",
		"app.kubernetes.io/instance":   configmapName,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}

	immutable := false
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName, // claim gets same name as volume it claims
			Namespace: pd.Namespace,
			Labels:    labels,
		},
		Immutable: &immutable,
		Data:      data,
	}
	if err := ctrl.SetControllerReference(pd, &cm, r.Scheme); err != nil {
		return nil, err
	}

	err := CreateOrUpdate(r, r, ctx, log, &cm, &corev1.ConfigMap{})
	if err != nil {
		return nil, err
	}

	return &cm, nil
}

/*
update ConfigMap
*/
func (r *PipelineDefinitionReconciler) UpdateConfigMap(ctx context.Context, log func(string, ...interface{}), cm *corev1.ConfigMap, data map[string]string) error {
	log(
		"Updating the ConfigMap",
		"CronJob.Namespace", cm.Namespace,
		"CronJob.Name", cm.Name,
	)
	cm.Data = data
	return r.Update(ctx, cm)
}
