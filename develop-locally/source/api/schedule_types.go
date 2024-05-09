/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScheduleInRange defines cron schedule and version pattern for a specified date range.
type ScheduleInRange struct {
	// +kubebuilder:validation:Optional
	After string `json:"after"`
	// +kubebuilder:validation:Optional
	Before string `json:"before"`
	// posix cron spec regex, see: https://www.codeproject.com/Tips/5299523/Regex-for-Cron-Expressions
	// +kubebuilder:validation:Pattern:=[0-9]*
	// +kubebuilder:validation:Required
	CronSpec string `json:"cronSpec"`
	// semver 2 regex, see https://semver.org/
	// +kubebuilder:validation:Pattern:=^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$
	// +kubebuilder:validation:Required
	VersionPattern string `json:"versionPattern"`
}

// ScheduleSpec defines the desired state of Schedule
type ScheduleSpec struct {
	// +kubebuilder:validation:Required
	Pipeline string `json:"pipeline"`
	// +kubebuilder:validation:Required
	Schedules []*ScheduleInRange `json:"schedules"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1440
	UpdateIntervalMinutes int32 `json:"updateIntervalMinutes"`
}

// ScheduleStatus defines the observed state of Schedule
type ScheduleStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TDSet is the Schema for the tdsets API
type Schedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduleSpec   `json:"spec,omitempty"`
	Status ScheduleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TDSetList contains a list of TDSet
type ScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Schedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Schedule{}, &ScheduleList{})
}
