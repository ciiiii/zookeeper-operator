/*
Copyright 2022.

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

type ClusterRestoreStatus string

const (
	RestoreStatusPending   ClusterRestoreStatus = "pending"
	RestoreStatusRunning   ClusterRestoreStatus = "running"
	RestoreStatusCompleted ClusterRestoreStatus = "completed"
	RestoreStatusRestarted ClusterRestoreStatus = "restarted"
	RestoreStatusFailed    ClusterRestoreStatus = "failed"
)

// ZooKeeperRestoreSpec defines the desired state of ZooKeeperRestore
type ZooKeeperRestoreSpec struct {
	Image string `json:"image,omitempty"`
	// +optional
	// +kubebuilder:default=false
	RolloutRestart bool `json:"rolloutRestart,omitempty"`
	// +kubebuilder:required
	Source *StroageLocation `json:"source,omitempty"`
	// +kubebuilder:required
	Target *ZooKeeperInstance `json:"target,omitempty"`
}

// ZooKeeperRestoreStatus defines the observed state of ZooKeeperRestore
type ZooKeeperRestoreStatus struct {
	Status  ClusterRestoreStatus `json:"status,omitempty"`
	Message string               `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ZooKeeperRestore is the Schema for the zookeeperrestores API
// +kubebuilder:resource:shortName="zkrestore",scope="Namespaced"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ZooKeeperRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZooKeeperRestoreSpec   `json:"spec,omitempty"`
	Status ZooKeeperRestoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ZooKeeperRestoreList contains a list of ZooKeeperRestore
type ZooKeeperRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZooKeeperRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZooKeeperRestore{}, &ZooKeeperRestoreList{})
}
