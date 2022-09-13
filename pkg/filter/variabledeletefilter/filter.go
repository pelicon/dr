package variabledeletefilter

import (
	"strings"
	"sync"

	udsdrv1alpha1 "github.com/DaoCloud/udsdr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var FilterName udsdrv1alpha1.FilterName = "VariableDeleteFilter"
var (
	logger = log.WithField("module", "variabledeletefilter")
)

type VariableDeleteFilter struct {
	*sync.Mutex
	udsdrv1alpha1.DRFilterConfig
}

func New() udsdrv1alpha1.Filter {
	return &VariableDeleteFilter{
		Mutex: &sync.Mutex{},
	}
}

func (vdf *VariableDeleteFilter) Out(objr *udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	logger.Infof("kinddeletelist: %+v", vdf.DRFilterConfig.VariableDeleteFilter.KindDeleteList)
	for _, kindDelete := range vdf.DRFilterConfig.VariableDeleteFilter.KindDeleteList {
		logger.Infof("gvk of resource: %+v, gvk of config: %+v", udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()), kindDelete)
		if udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()) == *kindDelete.Kind {
			doDeleteVariables(objr, kindDelete.KeyValueDelete)
		}
	}
	for _, objectDelete := range vdf.DRFilterConfig.VariableDeleteFilter.ObjectDeleteList {
		if (udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()) == *objectDelete.Kind) &&
			(objr.Unstructured.GetNamespace() == objectDelete.Object.ObjectKey.Namespace) &&
			(objr.Unstructured.GetName() == objectDelete.Object.ObjectKey.Name) {
			doDeleteVariables(objr, objectDelete.KeyValueDelete)
		}
	}
	return objr, nil
}

func doDeleteVariables(objr *udsdrv1alpha1.ObjResource, rules []udsdrv1alpha1.VariableKey) {
	for _, path := range rules {
		unstructured.RemoveNestedField(objr.Unstructured.Object, strings.Split(string(path), ",")...)
	}
}

func (vdf *VariableDeleteFilter) In(*udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	return nil, nil
}

func (vdf *VariableDeleteFilter) SetConfig(drconf *udsdrv1alpha1.DRFilterConfig) error {
	vdf.Lock()
	defer vdf.Unlock()

	vdf.DRFilterConfig = *drconf
	logger.Infof("variabledeletefilter's config:%v", vdf.DRFilterConfig.VariableDeleteFilter)
	return nil
}
