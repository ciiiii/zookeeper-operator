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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ZooKeeperConditionType string

const (
	ClusterConditionReady     ZooKeeperConditionType = "Ready"
	ClusterConditionScaleUp   ZooKeeperConditionType = "ScaleUp"
	ClusterConditionScaleDown ZooKeeperConditionType = "ScaleDown"
	ClusterConditionError     ZooKeeperConditionType = "Error"
)

// ZooKeeperClusterSpec defines the desired state of ZooKeeperCluster
type ZooKeeperClusterSpec struct {
	Image string `json:"image,omitempty"`
	// +optional
	HelperImage string            `json:"helperImage,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas,omitempty"`
	// +optional
	// +kubebuilder:default=cluster.local
	ClusterDomain string `json:"clusterDomain"`
	// +optional
	// +kubebuilder:default=false
	ClearPersistence bool `json:"clearPersistence,omitempty"`
	// +kubebuilder:default={}
	Config *ZooKeeperServerConfig `json:"config"`
}

type ZooKeeperServerConfig struct {
	// should be immutable
	// +optional
	// +kubebuilder:default=2181
	ClientPort int32 `json:"clientPort"`
	// +optional
	// +kubebuilder:default=2888
	FollowerPort int32 `json:"followerPort"`
	// +optional
	// +kubebuilder:default=3888
	LeaderElectionPort int32 `json:"leaderElectionPort"`
	// +optional
	// +kubebuilder:default=/data
	DataDir string `json:"dataDir"`
	// +optional
	// +kubebuilder:default=/conf
	RawConfigDir string `json:"rawConfigDir"`
	// +optional
	// +kubebuilder:default=/data/conf
	ConfigDir string `json:"configDir"`
	// +optional
	// +kubebuilder:default=zoo.cfg
	StaticConfig string `json:"staticConfig"`
	// +optional
	// +kubebuilder:default=zoo.cfg.dynamic
	DynamicConfig string `json:"dynamicConfig"`
}

// ZooKeeperClusterStatus defines the observed state of ZooKeeperCluster
type ZooKeeperClusterStatus struct {
	Replicas      int32                `json:"replicas,omitempty"`
	ReadyReplicas int32                `json:"readyReplicas,omitempty"`
	Service       string               `json:"service,omitempty"`
	Servers       []*ZooKeeperServer   `json:"servers,omitempty"`
	Conditions    []ZooKeeperCondition `json:"conditions,omitempty"`
}

type ZooKeeperServer struct {
	Name    string `json:"name,omitempty"`
	MyId    string `json:"myId,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Ready   string `json:"ready,omitempty"`
	Message string `json:"message,omitempty"`
}

type ZooKeeperCondition struct {
	Type               ZooKeeperConditionType `json:"type,omitempty"`
	Status             v1.ConditionStatus     `json:"status,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	Message            string                 `json:"message,omitempty"`
	LastUpdateTime     metav1.Time            `json:"lastUpdateTime,omitempty"`
	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:shortName="zkcluster",scope="Namespaced"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// ZooKeeperCluster is the Schema for the zookeeperclusters API
type ZooKeeperCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ZooKeeperClusterSpec   `json:"spec,omitempty"`
	Status ZooKeeperClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ZooKeeperClusterList contains a list of ZooKeeperCluster
type ZooKeeperClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ZooKeeperCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZooKeeperCluster{}, &ZooKeeperClusterList{})
}
