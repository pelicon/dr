package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PairClusterSettings struct {
	KubeApiserverTransportorSetting *KubeApiServerSettings  `json:"kubeApiServerSettings,omitempty"`
	HTTPTransportorSetting          *HTTPTransportorSetting `json:"httpTransportorSetting,omitempty"`
}

// TODO on stage 3
type HTTPTransportorSetting struct {
}

// KubeApiServerSettings kubernetes Apiserver settings
type KubeApiServerSettings struct {
	KubeApiServerHost string `json:"kubeApiServerHost,omitempty"`
	CertData          string `json:"certData,omitempty"`
	KeyData           string `json:"keyData,omitempty"`
	CAData            string `json:"caData,omitempty"`
	QPS               int    `json:"qps,omitempty"`
	Burst             int    `json:"burst,omitempty"`
}

// DRClusterSpec defines the desired state of DRCluster
type DRClusterSpec struct {
	DRComponents        `json:",inline"`
	PairClusterSettings PairClusterSettings `json:"pairClusterSettings,omitempty"`
}

// DRClusterStatus defines the observed state of DRCluster
type DRClusterStatus struct {
	DRComponents      `json:",inline"`
	LastSuccessSynced string              `json:"lastSuccessSynced,omitempty"`
	ClusterConditions []*ClusterCondition `json:"clusterConditions,omitempty"`
}

type ClusterCondition struct {
	Message           `json:",inline"`
	ClusterName       ClusterName `json:"clusterName,omitempty"`
	LastHeartbeatTime string      `json:"lastHeartbeatTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=drclusters,scope=Cluster,shortName=drc
// +kubebuilder:printcolumn:name="role",type=string,JSONPath=`.status.role`,description="DR role"
// +kubebuilder:printcolumn:name="active",type=string,JSONPath=`.status.active`,description="DR role"
// +kubebuilder:printcolumn:name="transportAdapter",type=string,JSONPath=`.status.transportAdapter`,description="DR role"
// +kubebuilder:printcolumn:name="collectorType",type=string,JSONPath=`.status.collectorType`,description="DR role"
// +kubebuilder:printcolumn:name="age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:subresource:status
type DRCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DRClusterSpec   `json:"spec,omitempty"`
	Status DRClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DRClusterList contains a list of DRCluster
type DRClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DRCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DRCluster{}, &DRClusterList{})
}
