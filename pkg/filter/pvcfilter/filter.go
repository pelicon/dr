package pvcfilter

import (
	udsdrv1alpha1 "github.com/pelicon/dr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var FilterName udsdrv1alpha1.FilterName = "PVCFilter"
var (
	logger = log.WithField("module", "pvcfilter")
)

type PVCFilter struct{}

func New() udsdrv1alpha1.Filter {
	return &PVCFilter{}
}

func (pf *PVCFilter) Out(objr *udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	if *objr.GVR != udsdrv1alpha1.PersistentVolumeClaimGVR {
		return objr, nil
	}
	volumeName, _, _ := unstructured.NestedString(objr.Unstructured.Object, "spec", "volumeName")
	if volumeName == "" {
		logger.Debugf("resource pvc has no volume, resource: %+v", objr.Unstructured)
		return nil, udsdrv1alpha1.ErrNoPassFilter
	}
	return objr, nil
}

func (pf *PVCFilter) In(*udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	return nil, nil
}

func (pf *PVCFilter) SetConfig(drconf *udsdrv1alpha1.DRFilterConfig) error {
	return nil
}
