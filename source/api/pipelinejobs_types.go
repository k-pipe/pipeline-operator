package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* JobSpec encapsulates the details of the kubernetes job wrapped by PipelineJob */
type JobSpec struct {
	// +kubebuilder:validation:Required
	Image string `json:"image"`
}

/* PipelineJobSpec defines the specs of a PipelineStep */
type PipelineJobSpec struct {
	// +kubebuilder:validation:Required
	PipelineName string `json:"pipelineName"`
	// +kubebuilder:validation:Required
	PipelineVersion string `json:"pipelineVersion"`
	// +kubebuilder:validation:Required
	PipelineRun string `json:"pipelineRun"`
	// +kubebuilder:validation:Optional
	Description *string `json:"description"`
	// +kubebuilder:validation:Required
	InputPipes []*string `json:"inputPipes"`
	// +kubebuilder:validation:Required
	OutputPipes []*string `json:"outputPipes"`
	// +kubebuilder:validation:Required
	JobSpec *JobSpec `json:"jobSpec"`
}

// ScheduleStatus defines the observed state of Schedule
type PipelineJobStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pj,singular=pipelinejob

// Pipeline is the Schema for the pipelines API
type PipelineJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineJobSpec   `json:"spec,omitempty"`
	Status PipelineJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineJob{}, &PipelineJobList{})
}