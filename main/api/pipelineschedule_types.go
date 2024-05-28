package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/* ScheduleInRange defines cron schedule and version pattern for a specified date range. */
type ScheduleInRange struct {
	// +kubebuilder:validation:Optional
	After *string `json:"after"`
	// +kubebuilder:validation:Optional
	Before *string `json:"before"`
	// https://stackoverflow.com/questions/14203122/create-a-regular-expression-for-cron-statement
	// +kubebuilder:validation:Pattern:=^.*$
	// +kubebuilder:validation:Required
	CronSpec string `json:"cronSpec"`
	// semver 2 regex, see https://semver.org/
	// +kubebuilder:validation:Required
	VersionPattern string `json:"versionPattern"`
	// timezone to be used for cronjob
	// +kubebuilder:validation:Optional
	TimeZone *string `json:"timeZone"`
}

/* ScheduleSpec defines the desired state of Schedule */
type PipelineScheduleSpec struct {
	// +kubebuilder:validation:Required
	PipelineName string `json:"pipelineName"`
	// +kubebuilder:validation:Required
	Schedules []*ScheduleInRange `json:"schedules"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1440
	UpdateIntervalMinutes int32 `json:"updateIntervalMinutes"`
}

// ScheduleStatus defines the observed state of Schedule
type PipelineScheduleStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=ps,singular=pipelineschedule

// Pipeline is the Schema for the pipelines API
type PipelineSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineScheduleSpec   `json:"spec,omitempty"`
	Status PipelineScheduleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineSchedule{}, &PipelineScheduleList{})
}
