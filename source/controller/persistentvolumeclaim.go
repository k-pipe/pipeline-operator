package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (r *PipelineRunReconciler) CreatePersistentVolumeClaim(ctx context.Context, pr *pipelinev1.PipelineRun, volumeName string, sizeInGB int64) (*corev1.PersistentVolumeClaim, error) {
	log := log.FromContext(ctx)

	// the labels to be attached to pvc
	labels := map[string]string{
		"app.kubernetes.io/name":       "Pipeline-PVC",
		"app.kubernetes.io/instance":   volumeName,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}

	storageClass := "standard"
	size := resource.NewQuantity(sizeInGB*(1<<30), resource.BinarySI)
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      volumeName, // claim gets same name as volume it claims
			Namespace: pr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *size,
				},
			},
			StorageClassName: &storageClass,
		},
	}
	if err := ctrl.SetControllerReference(pr, pr, r.Scheme); err != nil {
		log.Error(err, "failed to set controller owner reference")
		return nil, err
	}

	err := CreateOrUpdate(r, r, ctx, &pvc, &corev1.PersistentVolumeClaim{})
	if err != nil {
		log.Error(
			err, "Failed to create new PersistentVolumeClaim: "+pvc.Name,
			"PersistentVolumeClaim.Namespace", pvc.Namespace,
			"PersistentVolumeClaim.Name", pvc.Name,
		)
		return nil, err
	}

	return &pvc, nil
}
