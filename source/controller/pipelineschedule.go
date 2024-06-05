package controller

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"time"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UpToDate string = "UpToDate"
)

// Gets a pipeline schedule object by name from api server, returns nil,nil if not found
func (r *PipelineScheduleReconciler) GetPipelineSchedule(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineSchedule, error) {
	res := &pipelinev1.PipelineSchedule{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

// Sets the status condition of the pipeline schedule to available initially, i.e. if no condition exists yet.
func (r *PipelineScheduleReconciler) SetUpToDateStatus(ctx context.Context, ps *pipelinev1.PipelineSchedule, status metav1.ConditionStatus, message string) error {
	return SetStatusCondition(r.Status(), ctx, ps, &ps.Status.Conditions, UpToDate, status, message)
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

func (r *PipelineScheduleReconciler) SetStatus(ps *pipelinev1.PipelineSchedule, sir *pipelinev1.ScheduleInRange) {
	if sir == nil {
		ps.Status.VersionPattern = ""
		ps.Status.CronSpec = ""
		ps.Status.TimeZone = ""
	} else {
		ps.Status.VersionPattern = sir.VersionPattern
		ps.Status.CronSpec = sir.CronSpec
		if sir.TimeZone == nil {
			ps.Status.TimeZone = ""
		} else {
			ps.Status.TimeZone = *sir.TimeZone
		}
	}
}
