package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	SyncedStatusStart   SyncedStatus = "dr_sync_start"
	SyncedStatusRunning SyncedStatus = "dr_sync_running"
	SyncedStatusFinish  SyncedStatus = "dr_sync_fin"
	SyncedStatusFailed  SyncedStatus = "dr_sync_fail"

	ObjectActionCreate ObjectAction = "dr_obj_create"
	ObjectActionUpdate ObjectAction = "dr_obj_update"
	ObjectActionDelete ObjectAction = "dr_obj_delete"

	ResourceCollectorTypeListWatch ResourceCollectorType = "dr_resource_collector_list_watch"
	ResourceCollectorTypePeriodic  ResourceCollectorType = "dr_resource_collector_periodic"

	TransportAdapterKubeApiserver TransportAdapter = "dr_transport_adapter_kube_apiserver"
	TransportAdapterHTTP          TransportAdapter = "dr_transport_adapter_http"

	ClusterRoleProduction ClusterRole = "dr_role_prod"
	ClusterRoleBackup     ClusterRole = "dr_role_bkup"

	DRFilterKeyName string = "drFilters"

	VariableTypeInt   VariableType = "var_t_int"
	VariableTypeBool  VariableType = "var_t_bool"
	VariableTypeFloat VariableType = "var_t_float"
	VariableTypeStr   VariableType = "var_t_str"

	ClusterResourceDelegator = "kube-system"
)

var (
	ErrNoPassFilter = errors.New("no pass filter err")
)

// FilterName plugin unique name
type FilterName string
type VariableKey string
type VariableValue string
type UUID string
type SyncedStatus string
type ObjectAction string
type ResourceCollectorType string
type TransportAdapter string
type Namespace string
type ClusterRole string
type VariableType string
type ClusterName string

type Message struct {
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Type    string `json:"type,omitempty"`
}

type VariableMappings struct {
	Object           *GroupVersionKindObject         `json:"object,omitempty"`
	Kind             *GroupVersionKind               `json:"kind,omitempty"`
	KeyValueMappings map[VariableKey]VariableMapping `json:"keyValueMappings,omitempty"`
}

type VariableDelete struct {
	Object         *GroupVersionKindObject `json:"object,omitempty"`
	Kind           *GroupVersionKind       `json:"kind,omitempty"`
	KeyValueDelete []VariableKey           `json:"keyValueDelete,omitempty"`
}

type VariableMapping struct {
	FromSubStr   string       `json:"fromSubStr,omitempty"`
	ToSubStr     string       `json:"toSubStr,omitempty"`
	VariableType VariableType `json:"variableType,omitempty"`
}

type GroupVersionKind struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

type GroupVersionKindObject struct {
	GroupVersionKind `json:",inline"`
	ObjectKey        `json:",inline"`
}

type GroupVersionResource struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version,omitempty"`
	Resource string `json:"resource,omitempty"`
}

type ObjectKey struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// DRFilterConfig represents filter manager config of DR module.
// it's used as a configmap data in Kubernetes, namespace kube-system
// means admin user, who had able to manage cluter resources.
type DRFilterConfig struct {
	VariableDeleteFilter  *VariableDeleteFilterConfig  `json:"variableDeleteFilter,omitempty"`
	VariableMappingFilter *VariableMappingFilterConfig `json:"variableMappingFilter,omitempty"`
	WhiteListFilter       *WhiteListFilterConfig       `json:"whiteListFilter,omitempty"`
}

type VariableDeleteFilterConfig struct {
	ObjectDeleteList []VariableDelete `json:"objectVariableDelete,omitempty"`
	KindDeleteList   []VariableDelete `json:"kindVariableDelete,omitempty"`
}

type WhiteListFilterConfig struct {
	ObjectWhiteList []GroupVersionKindObject `json:"objectWhiteList,omitempty"`
	KindWhiteList   []GroupVersionKind       `json:"kindWhiteList,omitempty"`
}

type VariableMappingFilterConfig struct {
	ObjectVariableMappings []VariableMappings `json:"objectVariableMappings,omitempty"`
	KindVariableMappings   []VariableMappings `json:"kindListVariableMappings,omitempty"`
}

type ObjResource struct {
	Unstructured *unstructured.Unstructured
	GVR          *schema.GroupVersionResource
	Namespaced   bool
	Action       ObjectAction
}

type NamespacedStatusUpdateFunc func(SyncedCondition)

type DRComponents struct {
	Role             string                `json:"role,omitempty"`
	Active           bool                  `json:"active"`
	TransportAdapter TransportAdapter      `json:"transportAdapter,omitempty"`
	CollectorType    ResourceCollectorType `json:"collectorType,omitempty"`
}

type Filter interface {
	SetConfig(drconf *DRFilterConfig) error
	Out(*ObjResource) (*ObjResource, error)
	In(*ObjResource) (*ObjResource, error)
}

type DependencyChecker interface {
	DependencyCheck(*ObjResource) ([]*ObjResource, error)
	ShouldCheck(*ObjResource) bool
}

type NamesapceWorker interface {
	Run()
}

type NamesapceCRStatus struct {
	ObjectKey
	Status           DRNamespaceStatus
	Active           bool
	TransportAdapter TransportAdapter
	CollectorType    ResourceCollectorType
}
