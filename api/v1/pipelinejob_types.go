package v1

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* JobSpec encapsulates the details of the kubernetes job wrapped by PipelineJob */
type JobSpec struct {
	// +kubebuilder:validation:Required
	Image string `json:"image"`
	// +kubebuilder:validation:Optional
	Command []string `json:"command,omitempty"`
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`
	// +kubebuilder:validation:Optional
	WorkingDir string `json:"workingDir,omitempty"`
	// +kubebuilder:validation:Optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// +kubebuilder:validation:Optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds"`
	// +kubebuilder:validation:Optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds"`
	// +kubebuilder:validation:Optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished"`
	// +kubebuilder:validation:Optional
	BackoffLimit *int32 `json:"backoffLimit"`
	// +kubebuilder:validation:Optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Specification json.RawMessage `json:"specification,omitempty"`
}

/* InputPipe defines source and target name of input pipe file */
type InputPipe struct {
	// +kubebuilder:validation:Required
	Volume string `json:"volume"`
	// +kubebuilder:validation:Required
	MountPath string `json:"mountPath"`
	// +kubebuilder:validation:Required
	SourceFile string `json:"sourceFile"`
	// +kubebuilder:validation:Required
	TargetFile string `json:"targetFile"`
}

/* PipelineJobSpec defines the specs of a PipelineStep */
type PipelineJobSpec struct {
	// +kubebuilder:validation:Required
	Id string `json:"id"`
	// +kubebuilder:validation:Required
	PipelineRun string `json:"pipelineRun"`
	// +kubebuilder:validation:Required
	PipelineDefinition string `json:"pipelineDefinition"`
	// +kubebuilder:validation:Required
	StepId string `json:"stepId"`
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`
	// +kubebuilder:validation:Optional
	Inputs []InputPipe `json:"inputVolumes,omitempty"`
	// +kubebuilder:validation:Required
	JobSpec *JobSpec `json:"jobSpec"`
}

// ScheduleStatus defines the observed state of Schedule
type PipelineJobStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	// +kubebuilder:validation:Optional
	State *string `json:"state"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=pj,singular=pipelinejob
//+kubebuilder:printcolumn:name="Step",type="string",JSONPath=`.spec.stepId`
//+kubebuilder:printcolumn:name="Image",type="string",JSONPath=`.spec.jobSpec.image`
//+kubebuilder:printcolumn:name="State",type="string",JSONPath=`.status.state`

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
