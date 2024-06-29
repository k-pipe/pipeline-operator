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
func (r *PipelineDefinitionReconciler) GetServiceAccount(ctx context.Context, name types.NamespacedName) (*corev1.ServiceAccount, error) {
	res := &corev1.ServiceAccount{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

/*
create ConfigMap
*/
func (r *PipelineDefinitionReconciler) CreateServiceAccount(ctx context.Context, log func(string, ...interface{}), pd *pipelinev1.PipelineDefinition, name types.NamespacedName, annotatationLabel string, annotationValue string) (*corev1.ServiceAccount, error) {

	// the labels to be attached to pvc
	labels := map[string]string{
		"app.kubernetes.io/name":       "ServiceAccount",
		"app.kubernetes.io/instance":   name.Name,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}

	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name, // claim gets same name as volume it claims
			Namespace: name.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				annotatationLabel: annotationValue,
			},
		},
	}
	if err := ctrl.SetControllerReference(pd, &sa, r.Scheme); err != nil {
		return nil, err
	}
	err := CreateOrUpdate(r, r, ctx, log, &sa, &corev1.ServiceAccount{})
	if err != nil {
		return nil, err
	}

	return &sa, nil
}

func (r *PipelineDefinitionReconciler) addOwnership(ctx context.Context, l func(string, ...interface{}), sa *corev1.ServiceAccount, pd *pipelinev1.PipelineDefinition) (bool, error) {
	for _, or := range sa.ObjectMeta.OwnerReferences {
		if (or.Kind == "PipelineDefinition") && (or.Name == pd.Name) {
			return false, nil
		}
	}
	if err := ctrl.SetControllerReference(pd, sa, r.Scheme); err != nil {
		return false, err
	}
	if err := r.Update(ctx, sa); err != nil {
		return false, err
	}
	return true, nil
}
