package v1

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* PipelineStep defines details of a single pipeline step */
type PipelineStepSpec struct {
	// +kubebuilder:validation:Required
	Id string `json:"id"`
	// enum: INPUT, JOB, PIPELINE, BATCHED, OUTPUT
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// +kubebuilder:validation:Optional
	Description *string `json:"description"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Specification json.RawMessage `json:"specification,omitempty"`
	// +kubebuilder:validation:Required
	Inputs []string `json:"inputs"`
}

/* PipelinePipe defines details of a pipe connection between two pipeline steps */
type PipelinePipe struct {
	// +kubebuilder:validation:Required
	From PipeConnector `json:"from"`
	// +kubebuilder:validation:Required
	To PipeConnector `json:"to"`
}

/* PipelineConnector defines step and pipe name at either end of a pipe */
type PipeConnector struct {
	// +kubebuilder:validation:Required
	StepId string `json:"stepId"`
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

/* PipelineStructure holds the information of steps and pipes that constitute a pipeline */
type PipelineStructure struct {
	// +kubebuilder:validation:Required
	Steps []*PipelineStepSpec `json:"steps,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	// +kubebuilder:validation:Required
	Pipes []*PipelinePipe `json:"pipes,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

/* PipelineDefinitionSpec holds the definition of the pipeline structure, the configuration of steps, and meta information */
type PipelineDefinitionSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// +kubebuilder:validation:Optional
	Description *string `json:"description"`
	// +kubebuilder:validation:Optional
	PlantUML *string `json:"plantUML"`
	// +kubebuilder:validation:Required
	PipelineStructure PipelineStructure `json:"pipelineStructure"`
	// +kubebuilder:validation:Required
	StepConfiguration map[string]string `json:"configuration"`
}

// ScheduleStatus defines the observed state of Schedule
type PipelineDefinitionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pd,singular=pipelinedefinition

// Pipeline is the Schema for the pipelines API
type PipelineDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineDefinitionSpec   `json:"spec,omitempty"`
	Status PipelineDefinitionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineDefinition{}, &PipelineDefinitionList{})
}
