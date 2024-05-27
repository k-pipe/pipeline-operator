package controller

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Defines condition status of a pipeline schedule.
type PipelineScheduleStatus string

const (
	ScheduleAvailable   PipelineScheduleStatus = "Available"
	ScheduleProgressing PipelineScheduleStatus = "Progressing"
	ScheduleDegraded    PipelineScheduleStatus = "Degraded"
)

// Gets the pipeline schedule status from api server.
func (r *PipelineScheduleReconciler) GetPipelineSchedule(ctx context.Context, req ctrl.Request, schedule *pipelinev1.PipelineSchedule) error {
	err := r.Get(ctx, req.NamespacedName, schedule)
	if err != nil {
		return err
	}

	return nil
}

// Sets the status condition of the pipeline schedule to available initially, i.e. if no condition exists yet.
func (r *PipelineScheduleReconciler) SetInitialPSCondition(ctx context.Context, req ctrl.Request, schedule *pipelinev1.PipelineSchedule) error {
	if schedule.Status.Conditions != nil || len(schedule.Status.Conditions) != 0 {
		return nil
	}

	err := r.SetPSCondition(ctx, req, schedule, ScheduleAvailable, "Starting reconciliation")

	return err
}

// Sets the status condition of the pipelineschedule.
func (r *PipelineScheduleReconciler) SetPSCondition(
	ctx context.Context, req ctrl.Request,
	schedule *pipelinev1.PipelineSchedule, condition PipelineScheduleStatus,
	message string,
) error {
	log := log.FromContext(ctx)

	meta.SetStatusCondition(
		&schedule.Status.Conditions,
		metav1.Condition{
			Type:    string(condition),
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: message,
		},
	)

	if err := r.Status().Update(ctx, schedule); err != nil {
		log.Error(err, "Failed to update PipelineSchedule status")

		return err
	}

	if err := r.Get(ctx, req.NamespacedName, schedule); err != nil {
		log.Error(err, "Failed to re-fetch PipelineSchedule")

		return err
	}

	return nil
}

// Get the expected ScheduleInRange depending on the current time
func (r *PipelineScheduleReconciler) GetExpectedScheduleInRange(ctx context.Context, req ctrl.Request, ps *pipelinev1.PipelineSchedule) (*pipelinev1.ScheduleInRange, error) {
	log := log.FromContext(ctx)

	if ps.Spec.Schedules != nil && len(ps.Spec.Schedules) != 0 {
		now := time.Now()

		log.Info("determined current time", "time", now)

		for _, r := range ps.Spec.Schedules {
			if isBefore(now, r.Before) && isAfter(now, r.After) {
				return r, nil
			}
		}
	}

	return nil, nil
}

// if before is nil, always return true, otherwise return if now is before it
func isBefore(now time.Time, before *string) bool {
	if before == nil {
		return true
	}
	t, err := time.Parse(time.RFC3339, *before)
	if err != nil {
		// TODO handle date format error
		return false
	}
	return now.Before(t)
}

// if after is nil, always return true, otherwise return if now is after it
func isAfter(now time.Time, after *string) bool {
	if after == nil {
		return true
	}
	t, err := time.Parse(time.RFC3339, *after)
	if err != nil {
		// TODO handle date format error
		return false
	}
	return now.After(t)
}
