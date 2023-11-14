/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Type validation
type RunnerOptions struct {
	// RunnerType, one of 'project_type', 'group_type', 'instance_type'
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=project_type;group_type;instance_type
	RunnerType      *string   `json:"runnerType"`
	GroupID         *int      `json:"groupID,omitempty"`
	ProjectID       *int      `json:"projectID,omitempty"`
	Description     *string   `json:"description,omitempty"`
	Paused          *bool     `json:"paused,omitempty"`
	Locked          *bool     `json:"locked,omitempty"`
	RunUntagged     *bool     `json:"runUntagged,omitempty"`
	TagList         *[]string `json:"tagList"`
	AccessLevel     *string   `json:"accessLevel,omitempty"`
	MaximumTimeout  *int      `json:"maximumTimeout,omitempty"`
	MaintenanceNote *string   `json:"maintenanceNote,omitempty"`
}

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	GitlabBaseUrl string        `json:"gitlabBaseUrl,omitempty"`
	RunnerOptions RunnerOptions `json:"runnerOptions"`
	EnableFor     []string      `json:"enableFor,omitempty"`
}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[-1].type`,description="status represesnts current runner state eg. Alive, Destroing, Provisioning"
// Runner is the Schema for the runners API
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}
