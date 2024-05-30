package controller

// My version v2

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// label for storing the version pattern in cronjob resource
	PipeLineVersionPatternLabel = "k-pipe.cloud/pipeline-version-pattern"
)

// Get the cronjob of specified name, returns nil if there is none
func (r *PipelineScheduleReconciler) GetCronJob(ctx context.Context, name types.NamespacedName) (*batchv1.CronJob, error) {
	res := &batchv1.CronJob{}
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

/*
create cronjob object according to provided schedule in range
*/
func (r *PipelineScheduleReconciler) CreateCronJob(ctx context.Context, schedule *pipelinev1.PipelineSchedule, sir pipelinev1.ScheduleInRange) error {
	log := log.FromContext(ctx)

	// the labels to be attached to cron job
	cjLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   schedule.Name,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
		PipeLineVersionPatternLabel:    sir.VersionPattern,
	}
	jobLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   schedule.Name,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}
	podLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   schedule.Name,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}

	// define the cronjob object
	cj := &batchv1.CronJob{
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
	if err := ctrl.SetControllerReference(schedule, cj, r.Scheme); err != nil {
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
	}

	return nil
}

func (r *PipelineScheduleReconciler) DeleteCronJob(
	ctx context.Context,
	cj *batchv1.CronJob,
) error {
	log := log.FromContext(ctx)

	log.Info(
		"Deleting the Cronjob",
		"CronJob.Namespace", cj.Namespace,
		"CronJob.Name", cj.Name,
	)

	err := r.Delete(ctx, cj)
	if err != nil {
		log.Error(
			err, "Failed to delete CronJob",
			"CronJob.Namespace", cj.Namespace,
			"CronJob.Name", cj.Name,
		)
	}
	return err
}

func (r *PipelineScheduleReconciler) UpdateCronJob(
	ctx context.Context,
	sir pipelinev1.ScheduleInRange,
	cj *batchv1.CronJob,
) error {
	log := log.FromContext(ctx)

	log.Info(
		"Updating the Cronjob specs",
		"CronJob.Namespace", cj.Namespace,
		"CronJob.Name", cj.Name,
	)
	cj.Spec.TimeZone = sir.TimeZone
	cj.Spec.Schedule = sir.CronSpec
	cj.ObjectMeta.Labels[PipeLineVersionPatternLabel] = sir.VersionPattern
	// TODO add version pattern also in environment

	err := r.Update(ctx, cj)
	if err != nil {
		log.Error(
			err, "Failed to update CronJob",
			"CronJob.Namespace", cj.Namespace,
			"CronJob.Name", cj.Name,
		)
	}
	return err
}
