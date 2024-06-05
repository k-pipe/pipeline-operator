package controller

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// status flags
	VersionDetermined string = "VersionDetermined"
	StructureLoaded   string = "StructureLoaded"
	Paused            string = "Paused"
	Terminated        string = "Terminated"
	Failed            string = "Failed"
	Succeeded         string = "Succeeded"
)

// Gets a pipeline schedule object by name from api server, returns nil,nil if not found
func (r *PipelineRunReconciler) GetPipelineRun(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineRun, error) {
	res := &pipelinev1.PipelineRun{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

// Sets status condition of the pipeline run (from PipelineRun reconciliation)
func (r *PipelineRunReconciler) SetPipelineRunStatus(ctx context.Context, pr *pipelinev1.PipelineRun, statusType string, status metav1.ConditionStatus, message string) error {
	return SetStatusCondition(r.Status(), ctx, pr, &pr.Status.Conditions, statusType, status, message)
}

// Sets status condition of the pipeline run (from PipelineJob reconciliation)
func (r *PipelineJobReconciler) SetPipelineRunStatus(ctx context.Context, pr *pipelinev1.PipelineRun, statusType string, status metav1.ConditionStatus, message string) error {
	return SetStatusCondition(r.Status(), ctx, pr, &pr.Status.Conditions, statusType, status, message)
}

func (r *PipelineRunReconciler) DeterminePipelineVersion(ctx context.Context, pr *pipelinev1.PipelineRun) error {
	version := "1.0.0"
	pr.Status.PipelineVersion = &version
	// TODO
	return nil
}
