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
	Conditions        []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	PipelineVersion   string             `json:"pipelineVersion"`
	numStepsCreated   int                `json:"numStepsCreated"`
	numStepsSucceeded int                `json:"numStepsSucceeded"`
	numStepsFailed    int                `json:"numStepsFailed"`
	numStepsTotal     int                `json:"numStepsTotal"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pr,singular=pipelinerun

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
