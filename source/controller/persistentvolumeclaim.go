package controller

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Gets a PersistentVolumeClaim object by name from api server, returns nil,nil if not found
func (r *PipelineRunReconciler) GetPersistentVolumeClaim(ctx context.Context, name types.NamespacedName) (*corev1.PersistentVolumeClaim, error) {
	res := &corev1.PersistentVolumeClaim{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

/*
create PersistentVolumeClaim
*/
func (r *PipelineRunReconciler) CreatePersistentVolumeClaim(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun, volumeName string, sizeInGB int64) (*corev1.PersistentVolumeClaim, error) {

	// the labels to be attached to pvc
	labels := map[string]string{
		"app.kubernetes.io/name":       "Pipeline-PVC",
		"app.kubernetes.io/instance":   volumeName,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}

	//storageClass := "standard-rwo"
	size := resource.NewQuantity(sizeInGB*(1<<30), resource.BinarySI)
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      volumeName, // claim gets same name as volume it claims
			Namespace: pr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce /*, corev1.ReadOnlyMany*/},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *size,
				},
			},
			StorageClassName: env("STORAGE_CLASS"),
		},
	}
	if err := ctrl.SetControllerReference(pr, &pvc, r.Scheme); err != nil {
		return nil, err
	}

	err := CreateOrUpdate(r, r, ctx, log, &pvc, &corev1.PersistentVolumeClaim{})
	if err != nil {
		return nil, err
	}

	return &pvc, nil
}

/*
delete PersistentVolumeClaim
*/
func (r *PipelineRunReconciler) DeletePersistentVolumeClaim(ctx context.Context, log func(string, ...interface{}), pr *pipelinev1.PipelineRun, pvcName string) error {
	pvc, err := r.GetPersistentVolumeClaim(ctx, types.NamespacedName{Namespace: pr.Namespace, Name: pvcName})
	if err != nil {
		return err
	}
	if pvc == nil {
		log(
			"PersistentVolumeClaim was already gone",
			"PersistentVolumeClaim.Namespace", pr.Namespace,
			"PersistentVolumeClaim.Name", pvcName,
		)
		return nil
	}
	log(
		"Deleting the PersistentVolumeClaim",
		"PersistentVolumeClaim.Namespace", pr.Namespace,
		"PersistentVolumeClaim.Name", pvcName,
	)
	return r.Delete(ctx, pvc)
}
