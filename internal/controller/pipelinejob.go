package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
// status flags
)

// Gets a pipeline job object by name from api server, returns nil,nil if not found
func (r *PipelineJobReconciler) GetPipelineJob(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineJob, error) {
	res := &pipelinev1.PipelineJob{}
	err := r.Get(ctx, name, res)
	if err != nil {
		// no result will be returned in case of error
		res = nil
		if apierrors.IsNotFound(err) {
			// not found is not considered an error, we simply return nil,nil in that case
			err = nil
		}
	}
	return res, err
}

// Sets a status condition of the pipeline job
func (r *PipelineJobReconciler) SetPipelineJobStatus(ctx context.Context, pr *pipelinev1.PipelineJob, statusType string, status metav1.ConditionStatus, message string) error {
	log := log.FromContext(ctx)

	if meta.IsStatusConditionPresentAndEqual(pr.Status.Conditions, statusType, status) {
		// no change in status
		return nil
	}

	// set the status condition
	meta.SetStatusCondition(
		&pr.Status.Conditions,
		metav1.Condition{
			Type:    statusType,
			Status:  status,
			Reason:  "Reconciling",
			Message: message,
		},
	)

	if err := r.Status().Update(ctx, pr); err != nil {
		log.Error(err, "Failed to update PipelineRun status")
		return err
	}
	// refetch should not be needed, in fact can be problematic...
	//if err := r.Get(ctx, types.NamespacedName{Name: pr.Name, Namespace: pr.Namespace}, pr); err != nil {
	//	log.Error(err, "Failed to re-fetch PipelineRun")
	//	return err
	//}

	return nil
}

/*
create PipelineJob provided spec
*/
func (r *PipelineJobReconciler) CreatePipelineJob(ctx context.Context, pr *pipelinev1.PipelineRun, spec *pipelinev1.PipelineJobStepSpec) error {
	log := log.FromContext(ctx)

	// the labels to be attached to job
	jobLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   spec.Id,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}
	// the labels to be attached to the pod
	jobName := pr.Name + "-" + spec.Id
	// define the job object
	pj := &pipelinev1.PipelineJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pr.Namespace,
			Labels:    jobLabels,
		},
		Spec: pipelinev1.PipelineJobSpec{
			Id:          jobName,
			Description: spec.Description,
		},
	}

	// Set the ownerRef for the PipelineJob
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(pr, pj, r.Scheme); err != nil {
		log.Error(err, "failed to set controller owner reference")
		return err
	}

	// create the cronjob
	log.Info(
		"Creating a new PipelineJob",
		"PipelineJob.Namespace", pj.Namespace,
		"PipelineJob.Name", pj.Name,
	)
	err := r.Create(ctx, pj)
	if err != nil {
		log.Error(
			err, "Failed to create new PipelineJob",
			"PipelineJob.Namespace", pj.Namespace,
			"PipelineJob.Name", pj.Name,
		)
	}

	return nil
}
