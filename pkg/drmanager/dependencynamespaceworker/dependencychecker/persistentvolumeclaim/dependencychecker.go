package persistentvolumeclaim

import (
	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type DependencyChecker struct {
	client dynamic.Interface
}

var (
	logger = log.WithField("module", "pvc_dependency_checker")
)

func (dc *DependencyChecker) DependencyCheck(obj *drv1alpha1.ObjResource) ([]*drv1alpha1.ObjResource, error) {
	logger.Debugf("pvc dependencies checking")
	volumeName, _, _ := unstructured.NestedString(obj.Unstructured.Object, "spec", "volumeName")
	if volumeName == "" {
		logger.Debugf("pvc has no volume, resource: %+v", obj.Unstructured)
		return nil, nil
	}
	dependentPV, err := dc.client.Resource(drv1alpha1.PersistentVolumeGVR).Get(volumeName, v1.GetOptions{})
	if err != nil {
		logger.Errorf("get dependentPV for pvc err, resource: %+v", obj.Unstructured)
		return nil, err
	}
	logger.Debugf("got dependentPV for pvc, pvc: %+v, pv: %+v", obj.Unstructured, dependentPV)
	objrDependentPV := drv1alpha1.ObjResource{
		Unstructured: dependentPV,
		GVR:          &drv1alpha1.PersistentVolumeGVR,
		Namespaced:   false,
		Action:       obj.Action,
	}
	return []*drv1alpha1.ObjResource{&objrDependentPV}, nil
}

func (dc *DependencyChecker) ShouldCheck(obj *drv1alpha1.ObjResource) bool {
	logger.Debugf("PVCGVR: %+v", drv1alpha1.PersistentVolumeClaimGVR)
	logger.Debugf("GVR of resource: %+v", *obj.GVR)
	// shouldCheck := (*obj.GVR).Resource == drv1alpha1.PersistentVolumeClaimGVR.Resource
	shouldCheck := *obj.GVR == drv1alpha1.PersistentVolumeClaimGVR
	// shouldCheck := reflect.DeepEqual(*obj.GVR, drv1alpha1.PersistentVolumeClaimGVR)
	logger.Debugf("shouldCheck: %v", shouldCheck)
	// return *obj.GVR == drv1alpha1.PersistentVolumeClaimGVR
	return shouldCheck
}

func NewDependencyChecker(client dynamic.Interface) drv1alpha1.DependencyChecker {
	return &DependencyChecker{
		client: client,
	}
}
