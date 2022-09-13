package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DRNamespaceSpec defines the desired state of DRNamespace
type DRNamespaceSpec struct {
	DRComponents      `json:",inline"`
	DRPairClusterName ClusterName    `json:"drPairCluster"`
	DRFilterConfig    DRFilterConfig `json:"drFilterConfig,omitempty"`
}

// DRNamespaceStatus defines the observed state of DRNamespace
type DRNamespaceStatus struct {
	DRComponents      `json:",inline"`
	DRPairClusterName ClusterName    `json:"drPairCluster"`
	DRFilterConfig    DRFilterConfig `json:"drFilterConfig,omitempty"`
	//TODO default variabledeletefilterconfigstatus
	SyncedConditions map[string]SyncedCondition `json:"syncConditions,omitempty"`
}

type SyncedCondition struct {
	GroupVersionKindObject    GroupVersionKindObject `json:"groupVersionKindObject,omitempty"`
	LastSyncedResourceVersion string                 `json:"lastResourceVersion,omitempty"`
	LastSyncedStatus          SyncedStatus           `json:"lastSyncedStatus,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="namespace",type=string,JSONPath=`.metadata.namespace`,description="DR namespace"
// +kubebuilder:resource:path=drnamespaces,scope=Namespaced,shortName=drns
// +kubebuilder:printcolumn:name="role",type=string,JSONPath=`.status.role`,description="DR role"
// +kubebuilder:printcolumn:name="active",type=string,JSONPath=`.status.active`,description="DR role"
// +kubebuilder:printcolumn:name="transportAdapter",type=string,JSONPath=`.status.transportAdapter`,description="DR role"
// +kubebuilder:printcolumn:name="collectorType",type=string,JSONPath=`.status.collectorType`,description="DR role"
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="age",type=date,JSONPath=`.metadata.creationTimestamp`
type DRNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DRNamespaceSpec   `json:"spec,omitempty"`
	Status DRNamespaceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DRNamespaceList contains a list of DRNamespace
type DRNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DRNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DRNamespace{}, &DRNamespaceList{})
}
