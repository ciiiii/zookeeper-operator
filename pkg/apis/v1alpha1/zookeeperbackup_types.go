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

type ClusterBackupMode string
type ClusterBackupStatus string

const (
	BackupOnce     ClusterBackupMode = "once"
	BackupSchedule ClusterBackupMode = "schedule"

	BackupStatusPending   ClusterBackupStatus = "pending"
	BackupStatusRunning   ClusterBackupStatus = "running"
	BackupStatusSuspend   ClusterBackupStatus = "suspend"
	BackupStatusCompleted ClusterBackupStatus = "completed"
	BackupStatusFailed    ClusterBackupStatus = "failed"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZooKeeperBackupSpec defines the desired state of ZooKeeperBackup
type ZooKeeperBackupSpec struct {
	// +optional
	// +kubebuilder:default="once"
	Mode ClusterBackupMode `json:"mode,omitempty"`
	// +optional
	Schedule string `json:"schedule,omitempty"`
	// +optional
	// +kubebuilder:default=false
	Suspend bool   `json:"suspend,omitempty"`
	Image   string `json:"image,omitempty"`
	// +kubebuilder:required
	Source *ZooKeeperInstance `json:"source,omitempty"`
	// +kubebuilder:required
	Target *StroageLocation `json:"target,omitempty"`
}

// ZooKeeperBackupStatus defines the observed state of ZooKeeperBackup
type ZooKeeperBackupStatus struct {
	Status  ClusterBackupStatus `json:"status,omitempty"`
	Message string              `json:"message,omitempty"`
	Records []BackupRecord      `json:"record,omitempty"`
}

type BackupRecord struct {
	Key string `json:"key,omitempty"`
	// +optional
	StartedTime metav1.Time `json:"startTime,omitempty"`
	// +optional
	FinshedTime metav1.Time `json:"finishTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ZooKeeperBackup is the Schema for the zookeeperbackups API
// +kubebuilder:resource:shortName="zkbackup",scope="Namespaced"
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.mode"
// +kubebuilder:printcolumn:name="Schedule",type="string",priority=1,JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Suspend",type="boolean",priority=1,JSONPath=".spec.suspend"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ZooKeeperBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZooKeeperBackupSpec   `json:"spec,omitempty"`
	Status ZooKeeperBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// ZooKeeperBackupList contains a list of ZooKeeperBackup
type ZooKeeperBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZooKeeperBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZooKeeperBackup{}, &ZooKeeperBackupList{})
}
