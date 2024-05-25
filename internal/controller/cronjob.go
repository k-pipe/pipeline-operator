package controller

// My version v2

import (
	"context"
	"fmt"

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

/*
create cronjob object according to expected schedule in range corresponding to current time, returns nil if there is

	no matching schedule range
*/
func (r *PipelineScheduleReconciler) CronJob(
	ctx context.Context, req ctrl.Request,
	schedule *pipelinev1.PipelineSchedule,
) (*batchv1.CronJob, error) {
	log := log.FromContext(ctx)

	// get the first matching schedule range
	scheduleRange, err := r.GetExpectedScheduleInRange(ctx, req, schedule)
	if err != nil {
		log.Error(err, "failed to get expected schedule in range corresponding to current time")
		return nil, err
	}

	// if no schedule range matches, return nil
	if scheduleRange == nil {
		return nil, nil
	}

	// the labels to be attached to cron job
	cjLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   schedule.Name,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
		PipeLineVersionPatternLabel:    scheduleRange.VersionPattern,
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

	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedule.Name,
			Namespace: schedule.Namespace,
			Labels:    cjLabels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                scheduleRange.CronSpec,
			TimeZone:                scheduleRange.TimeZone,
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

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(schedule, cj, r.Scheme); err != nil {
		log.Error(err, "failed to set controller owner reference")
		return nil, err
	}

	return cj, nil
}

func (r *PipelineScheduleReconciler) CronJobIfNotExist(
	ctx context.Context, req ctrl.Request,
	schedule *pipelinev1.PipelineSchedule,
) (bool, error) {
	log := log.FromContext(ctx)

	cj := &batchv1.CronJob{}

	err := r.Get(ctx, types.NamespacedName{Name: schedule.Name, Namespace: schedule.Namespace}, cj)
	if err != nil && apierrors.IsNotFound(err) {
		cj, err := r.CronJob(ctx, req, schedule)
		if err != nil {
			log.Error(err, "Failed to define new Cronjob resource for PipelineSchedule")

			err = r.SetPSCondition(
				ctx, req, schedule, ScheduleAvailable,
				fmt.Sprintf("Failed to create CronJob for PipelineSchedule (%s): (%s)", schedule.Name, err),
			)
			if err != nil {
				return false, err
			}
		}

		log.Info(
			"Creating a new CronJob",
			"ConJob.Namespace", cj.Namespace,
			"CronJob.Name", cj.Name,
		)

		err = r.Create(ctx, cj)
		if err != nil {
			log.Error(
				err, "Failed to create new CronJob",
				"CronJob.Namespace", cj.Namespace,
				"CronJob.Name", cj.Name,
			)

			return false, err
		}

		err = r.GetPipelineSchedule(ctx, req, schedule)
		if err != nil {
			log.Error(err, "Failed to re-fetch TDSet")
			return false, err
		}

		err = r.SetPSCondition(
			ctx, req, schedule, ScheduleProgressing,
			fmt.Sprintf("Created Cronjob for the PipelineSchedule: (%s)", schedule.Name),
		)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	if err != nil {
		log.Error(err, "Failed to get CronJob")

		return false, err
	}

	return false, nil
}

func (r *PipelineScheduleReconciler) UpdateCronJob(
	ctx context.Context, req ctrl.Request,
	schedule *pipelinev1.PipelineSchedule,
) error {
	log := log.FromContext(ctx)

	cj := &batchv1.CronJob{}

	err := r.Get(ctx, types.NamespacedName{Name: schedule.Name, Namespace: schedule.Namespace}, cj)
	if err != nil {
		log.Error(err, "Failed to get CronJob")

		return err
	}

	scheduleRange, err := r.GetExpectedScheduleInRange(ctx, req, schedule)
	if err != nil {
		log.Error(err, "failed to get expected schedule range")

		return err
	}
	// TODO test if no range, then delete cj

	if equalSpecs(scheduleRange, cj) {
		return nil
	}

	log.Info(
		"Updating a Cronjob specs",
		"CronJob.Namespace", cj.Namespace,
		"CronJob.Name", cj.Name,
	)
	cj.Spec.TimeZone = scheduleRange.TimeZone
	cj.Spec.Schedule = scheduleRange.CronSpec
	cj.ObjectMeta.Labels[PipeLineVersionPatternLabel] = scheduleRange.VersionPattern
	// TODO add version pattern also in environment

	err = r.Update(ctx, cj)
	if err != nil {
		log.Error(
			err, "Failed to update CronJob",
			"CronJob.Namespace", cj.Namespace,
			"CronJob.Name", cj.Name,
		)

		err = r.GetPipelineSchedule(ctx, req, schedule)
		if err != nil {
			log.Error(err, "Failed to re-fetch TDSet")
			return err
		}

		err = r.SetPSCondition(
			ctx, req, schedule, ScheduleProgressing,
			fmt.Sprintf("Failed to update CronJob for the PipelineSchedule (%s): (%s)", schedule.Name, err),
		)
		if err != nil {
			return err
		}

		return nil
	}

	// TODO should we really always refetch?
	err = r.GetPipelineSchedule(ctx, req, schedule)
	if err != nil {
		log.Error(err, "Failed to re-fetch TDSet")
		return err
	}

	err = r.SetPSCondition(
		ctx, req, schedule, ScheduleProgressing,
		fmt.Sprintf("Updated CronJob for the PipelineSchedule (%s)", schedule.Name),
	)
	if err != nil {
		return err
	}

	return nil
}

func equalSpecs(scheduleRange *pipelinev1.ScheduleInRange, cj *batchv1.CronJob) bool {
	if (scheduleRange.TimeZone == nil) != (cj.Spec.TimeZone == nil) {
		fmt.Sprintf("Change in timezone existence")
		return false
	}
	if (scheduleRange != nil) && (*scheduleRange.TimeZone != *cj.Spec.TimeZone) {
		fmt.Sprintf("Change in timezone value")
		return false
	}
	if scheduleRange.CronSpec != cj.Spec.Schedule {
		fmt.Sprintf("Change in cron spec")
		return false
	}
	if scheduleRange.VersionPattern != cj.ObjectMeta.Labels[PipeLineVersionPatternLabel] {
		fmt.Sprintf("Change in cron version pattenr")
		return false
	}
	return true
}
