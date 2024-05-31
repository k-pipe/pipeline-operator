package tests

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* PipelineRunSpec defines specs of a pipeline run */
type PipelineRunSpec struct {
	// +kubebuilder:validation:Required
	PipelineName string `json:"pipelineName"`
	// +kubebuilder:validation:Required
	PipelineVersion string `json:"pipelineVersion"`
	// +kubebuilder:validation:Optional
	Description *string `json:"description"`
	// +kubebuilder:validation:Optional
	ParentRun *string `json:"parentRun"`
	// +kubebuilder:validation:Required
	InputPipes []*string `json:"inputPipes"`
}

// ScheduleStatus defines the observed state of Schedule
type PipelineRunStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pj,singular=pipelinerun

// Pipeline is the Schema for the pipelines API
type PipelineRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineRunSpec   `json:"spec,omitempty"`
	Status PipelineRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineRun{}, &PipelineRunList{})
}
