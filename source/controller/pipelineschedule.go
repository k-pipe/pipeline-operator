package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UpdateRequired string = "UpdateRequired"
)

// Gets a pipeline schedule object by name from api server, returns nil,nil if not found
func (r *PipelineScheduleReconciler) GetPipelineSchedule(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineSchedule, error) {
	res := &pipelinev1.PipelineSchedule{}
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
func (r *PipelineScheduleReconciler) SetUpdateRequiredStatus(ctx context.Context, ps *pipelinev1.PipelineSchedule, status metav1.ConditionStatus, message string) error {
	log := log.FromContext(ctx)

	if meta.IsStatusConditionPresentAndEqual(ps.Status.Conditions, UpdateRequired, status) {
		// no change in status
		return nil
	}

	// set the status condition
	meta.SetStatusCondition(
		&ps.Status.Conditions,
		metav1.Condition{
			Type:    UpdateRequired,
			Status:  status,
			Reason:  "Reconciling",
			Message: message,
		},
	)

	if err := r.Status().Update(ctx, ps); err != nil {
		log.Error(err, "Failed to update PipelineSchedule status")
		return err
	}

	if err := r.Get(ctx, types.NamespacedName{Name: ps.Name, Namespace: ps.Namespace}, ps); err != nil {
		log.Error(err, "Failed to re-fetch PipelineSchedule")
		return err
	}

	return nil
}

// Get the expected ScheduleInRange depending on the current time, returns nil if no ScheduleRange matches
func (r *PipelineScheduleReconciler) GetExpectedScheduleInRange(ctx context.Context, ps pipelinev1.PipelineSchedule) (*pipelinev1.ScheduleInRange, error) {
	if ps.Spec.Schedules != nil && len(ps.Spec.Schedules) != 0 {
		now := time.Now()

		//log.Info("determined current time", "time", now)

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
