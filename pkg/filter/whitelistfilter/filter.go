package whitelistfilter

import (
	"sync"

	udsdrv1alpha1 "github.com/DaoCloud/udsdr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
)

var FilterName udsdrv1alpha1.FilterName = "WhiteListFilter"
var (
	logger = log.WithField("module", "whitelistfilter")
)

type WhiteListFilter struct {
	*sync.Mutex
	udsdrv1alpha1.DRFilterConfig
}

func New() udsdrv1alpha1.Filter {
	return &WhiteListFilter{
		Mutex: &sync.Mutex{},
	}
}

func (wlf *WhiteListFilter) Out(objr *udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	if wlf.DRFilterConfig.WhiteListFilter == nil {
		logger.Debugf("whitelist config is nil, resource: %+v", objr.Unstructured)
		return nil, udsdrv1alpha1.ErrNoPassFilter
	}

	for _, gvk := range wlf.DRFilterConfig.WhiteListFilter.KindWhiteList {
		logger.Infof("gvk of whitelist: %v, gvk of resource: %+v", gvk, udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()))
		if udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()) == gvk {
			logger.Debugf("resource in whitelist, resource: %+v", objr.Unstructured)
			return objr, nil
		}
	}
	for _, gvkObj := range wlf.WhiteListFilter.ObjectWhiteList {
		if objr.Unstructured.GetName() == gvkObj.ObjectKey.Name {
			logger.Debugf("resource in whitelist, resource: %+v", objr.Unstructured)
			return objr, nil
		}
	}
	logger.Debugf("resource not in whitelist, resource: %+v", objr.Unstructured)
	return nil, udsdrv1alpha1.ErrNoPassFilter
}

func (wlf *WhiteListFilter) In(*udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	return nil, nil
}

func (wlf *WhiteListFilter) SetConfig(drconf *udsdrv1alpha1.DRFilterConfig) error {
	wlf.Lock()
	defer wlf.Unlock()

	wlf.DRFilterConfig = *drconf
	return nil
}
