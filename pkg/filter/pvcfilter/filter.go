package pvcfilter

import (
	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var FilterName drv1alpha1.FilterName = "PVCFilter"
var (
	logger = log.WithField("module", "pvcfilter")
)

type PVCFilter struct{}

func New() drv1alpha1.Filter {
	return &PVCFilter{}
}

func (pf *PVCFilter) Out(objr *drv1alpha1.ObjResource) (*drv1alpha1.ObjResource, error) {
	if *objr.GVR != drv1alpha1.PersistentVolumeClaimGVR {
		return objr, nil
	}
	volumeName, _, _ := unstructured.NestedString(objr.Unstructured.Object, "spec", "volumeName")
	if volumeName == "" {
		logger.Debugf("resource pvc has no volume, resource: %+v", objr.Unstructured)
		return nil, drv1alpha1.ErrNoPassFilter
	}
	return objr, nil
}

func (pf *PVCFilter) In(*drv1alpha1.ObjResource) (*drv1alpha1.ObjResource, error) {
	return nil, nil
}

func (pf *PVCFilter) SetConfig(drconf *drv1alpha1.DRFilterConfig) error {
	return nil
}
