package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* PipelineRunSpec defines specs of a pipeline run */
type PipelineRunSpec struct {
	// +kubebuilder:validation:Required
	PipelineName string `json:"pipelineName"`
	// +kubebuilder:validation:Required
	VersionPattern string `json:"versionPattern"`
	// +kubebuilder:validation:Optional
	Description *string `json:"description"`
	// +kubebuilder:validation:Optional
	ParentRun *string `json:"parentRun"`
	// +kubebuilder:validation:Required
	InputPipes []*string `json:"inputPipes"`
}

// PipelineRunStatus defines the observed state of a pipeline run
type PipelineRunStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	// +kubebuilder:validation:Optional
	PipelineVersion *string `json:"pipelineVersion"`
	// +kubebuilder:validation:Optional
	PipelineStructure *PipelineStructure `json:"pipelineStructure"`
	NumStepsActive    int                `json:"numStepsActive"`
	NumStepsSucceeded int                `json:"numStepsSucceeded"`
	NumStepsFailed    int                `json:"numStepsFailed"`
	NumStepsTotal     int                `json:"numStepsTotal"`
	State             *string            `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pr,singular=pipelinerun
//+kubebuilder:printcolumn:name="Pipeline",type="string",JSONPath=`.spec.pipelineName`
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=`.status.pipelineVersion`
//+kubebuilder:printcolumn:name="Active",type="integer",JSONPath=`.status.numStepsActive`
//+kubebuilder:printcolumn:name="Success",type="integer",JSONPath=`.status.numStepsSucceeded`
//+kubebuilder:printcolumn:name="Failed",type="integer",JSONPath=`.status.numStepsFailed`
//+kubebuilder:printcolumn:name="Total",type="integer",JSONPath=`.status.numStepsTotal`
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=`.status.state`

// PipelineRun is the Schema for the pipelines runs
type PipelineRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineRunSpec   `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineRunList contains a list of PipelineRuns
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineRun{}, &PipelineRunList{})
}
