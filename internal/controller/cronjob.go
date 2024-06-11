package controller

// My version v2

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

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
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

/*
create cronjob object according to provided schedule in range
*/
func (r *PipelineScheduleReconciler) CreateCronJob(ctx context.Context, log func(string, ...interface{}), schedule *pipelinev1.PipelineSchedule, sir pipelinev1.ScheduleInRange) error {

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
		return err
	}

	// create the cronjob
	log(
		"Creating a new Cronjob",
		"CronJob.Namespace", cj.Namespace,
		"CronJob.Name", cj.Name,
	)
	return CreateOrUpdate(r, r, ctx, log, cj, &batchv1.CronJob{})
}

func (r *PipelineScheduleReconciler) DeleteCronJob(
	ctx context.Context,
	log func(string, ...interface{}),
	cj *batchv1.CronJob,
) error {
	log(
		"Deleting the Cronjob",
		"CronJob.Namespace", cj.Namespace,
		"CronJob.Name", cj.Name,
	)

	return r.Delete(ctx, cj)
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
