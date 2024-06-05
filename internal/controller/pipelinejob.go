package controller

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// status flags
	JobCreated   string = "JobCreated"
	JobSucceeded string = "JobSucceeded"
)

// Gets a pipeline job object by name from api server, returns nil,nil if not found
func (r *PipelineJobReconciler) GetPipelineJob(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineJob, error) {
	res := &pipelinev1.PipelineJob{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

// Gets a pipeline run object by name from api server, returns nil,nil if not found
func (r *PipelineJobReconciler) GetPipelineRun(ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineRun, error) {
	res := &pipelinev1.PipelineRun{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

// Sets a status condition of the pipeline job
func (r *PipelineJobReconciler) SetPipelineJobStatus(ctx context.Context, pj *pipelinev1.PipelineJob, statusType string, status metav1.ConditionStatus, message string) error {
	return SetStatusCondition(r.Status(), ctx, pj, &pj.Status.Conditions, statusType, status, message)
}

/*
create PipelineJob provided spec
*/
func (r *PipelineRunReconciler) CreatePipelineJob(ctx context.Context, pr *pipelinev1.PipelineRun, spec *pipelinev1.PipelineJobStepSpec) error {
	log := log.FromContext(ctx)

	// create the input pipes names
	var inputPipes []string
	for i, pipe := range pr.Status.PipelineStructure.Pipes {
		if pipe.To.StepId == spec.Id {
			inputPipes = append(inputPipes, pr.Name+"-"+strconv.Itoa(i))
		}
	}

	// create the output pipes names
	var outputPipes []string
	for i, pipe := range pr.Status.PipelineStructure.Pipes {
		if pipe.From.StepId == spec.Id {
			outputPipes = append(outputPipes, pr.Name+"-"+strconv.Itoa(i))
		}
	}

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
			InputPipes:  inputPipes,
			OutputPipes: outputPipes,
			JobSpec:     spec.JobSpec.DeepCopy(),
			PipelineRun: pr.Name,
			StepId:      spec.Id,
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
	err := CreateOrUpdate(r, r, ctx, pj, &pipelinev1.PipelineJob{})
	if err != nil {
		log.Error(
			err, "Failed to create new PipelineJob",
			"PipelineJob.Namespace", pj.Namespace,
			"PipelineJob.Name", pj.Name,
		)
		return err
	}

	return nil
}

func isTrueInPipelineJob(j *pipelinev1.PipelineJob, condition string) bool {
	return meta.IsStatusConditionPresentAndEqual(j.Status.Conditions, condition, metav1.ConditionTrue)
}
