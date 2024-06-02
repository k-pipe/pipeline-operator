package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapCreated string = "ConfigMapCreated"

	// Step Types
	STEP_TYPE_JOB = "JOB"
)

// Gets a pipeline schedule object by name from api server, returns nil,nil if not found
func GetPipelineDefinition(r client.Reader, ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineDefinition, error) {
	res := &pipelinev1.PipelineDefinition{}
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
func (r *PipelineDefinitionReconciler) SetPipelineDefinitionStatus(ctx context.Context, pd *pipelinev1.PipelineDefinition, status metav1.ConditionStatus, message string) error {
	log := log.FromContext(ctx)

	if meta.IsStatusConditionPresentAndEqual(pd.Status.Conditions, ConfigMapCreated, status) {
		// no change in status
		return nil
	}

	// set the status condition
	meta.SetStatusCondition(
		&pd.Status.Conditions,
		metav1.Condition{
			Type:    ConfigMapCreated,
			Status:  status,
			Reason:  "Reconciling",
			Message: message,
		},
	)

	if err := r.Status().Update(ctx, pd); err != nil {
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
