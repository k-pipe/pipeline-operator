package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
// status flags
)

// Gets a pipeline schedule object by name from api server, returns nil,nil if not found
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

// Sets the status condition of the pipeline schedule to available initially, i.e. if no condition exists yet.
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
func (r *PipelineRunReconciler) CreatePipelineJob(ctx context.Context, pr *pipelinev1.PipelineRun, spec *pipelinev1.PipelineJobStepSpec) error {
	//log := log.FromContext(ctx)

	// the labels to be attached to cron job
	/*	cjLabels := map[string]string{
			"app.kubernetes.io/name":       "PipelineSchedule",
			"app.kubernetes.io/instance":   spec.Id,
			"app.kubernetes.io/version":    "v1",
			"app.kubernetes.io/part-of":    "pipeline-operator",
			"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
		}
		jobLabels := map[string]string{
			"app.kubernetes.io/name":       "PipelineSchedule",
			"app.kubernetes.io/instance":   spec.Id,
			"app.kubernetes.io/version":    "v1",
			"app.kubernetes.io/part-of":    "pipeline-operator",
			"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
		}
		podLabels := map[string]string{
			"app.kubernetes.io/name":       "PipelineSchedule",
			"app.kubernetes.io/instance":   spec.Id,
			"app.kubernetes.io/version":    "v1",
			"app.kubernetes.io/part-of":    "pipeline-operator",
			"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
		}
	*/
	// define the cronjob object
	/*	cj := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      schedule.Name,
				Namespace: schedule.Namespace,
				Labels:    cjLabels,
			},
			Spec: batchv1.CronJobSpec{
				Schedule:                sir.CronSpec,
				TimeZone:                sir.TimeZone,
				StartingDeadlineSeconds: nil,
				ConcurrencyPolicy:       batchv1.ForbidConcurrent,
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: jobLabels,
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: podLabels,
							},
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyNever,
								Containers: []corev1.Container{{
									Image:           "busybox",
									Name:            schedule.Name,
									ImagePullPolicy: corev1.PullIfNotPresent,
								}},
							},
						},
					},
				},
			},
		}

		// Set the ownerRef for the CronJob
		// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
		if err := ctrl.SetControllerReference(pr, cj, r.Scheme); err != nil {
			log.Error(err, "failed to set controller owner reference")
			return err
		}

		// create the cronjob
		log.Info(
			"Creating a new Cronjob",
			"CronJob.Namespace", cj.Namespace,
			"CronJob.Name", cj.Name,
		)
		err := r.Create(ctx, cj)
		if err != nil {
			log.Error(
				err, "Failed to create new CronJob",
				"CronJob.Namespace", schedule.Namespace,
				"CronJob.Name", schedule.Name,
			)
		}*/

	return nil
}
