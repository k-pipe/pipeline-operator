package controller

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigMapCreated string = "ConfigMapCreated"
)

// Gets a pipeline schedule object by name from api server, returns nil,nil if not found
func GetPipelineDefinition(r client.Reader, ctx context.Context, name types.NamespacedName) (*pipelinev1.PipelineDefinition, error) {
	res := &pipelinev1.PipelineDefinition{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

// Sets the status condition of the pipeline schedule to available initially, i.e. if no condition exists yet.
func (r *PipelineDefinitionReconciler) SetConfigMapCreatedStatus(ctx context.Context, log func(string, ...interface{}), pd *pipelinev1.PipelineDefinition, status metav1.ConditionStatus, message string) error {
	return SetStatusCondition(r.Status(), ctx, log, pd, &pd.Status.Conditions, ConfigMapCreated, status, message)
}
