package statefulset

import (
	"reflect"

	udsdrv1alpha1 "github.com/DaoCloud/udsdr/pkg/apis/udsdr/v1alpha1"
	"github.com/DaoCloud/udsdr/pkg/utils"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type DependencyChecker struct {
	client dynamic.Interface
}

var (
	logger = log.WithField("module", "statefulSet_dependency_checker")
)

func (dc *DependencyChecker) DependencyCheck(obj *udsdrv1alpha1.ObjResource) ([]*udsdrv1alpha1.ObjResource, error) {
	dependenciesUnstructured := make([]unstructured.Unstructured, 0)
	//todo
	podList, err := dc.client.Resource(udsdrv1alpha1.PodGVR).List(v1.ListOptions{})
	if err != nil {
		logger.Errorf("list pod err: %v, resource: %+v", err.Error(), obj.Unstructured)
		return nil, err
	}
	ownedPods := siftPodsForStatefulSet(podList.Items, obj.Unstructured.GetUID())
	for _, pod := range ownedPods {
		dpvcs, err := checkPodVolumesDependency(dc.client, pod)
		if err != nil {
			logger.Errorf("checkPodVolumesDependency err: %v, pod: %+v", err.Error(), pod)
			return nil, err
		}
		logger.Debugf("dpvcs: %+v", dpvcs)
		dependenciesUnstructured = append(dependenciesUnstructured, dpvcs...)
	}

	logger.Debugf("dependenciesUnstructured are: %+v", dependenciesUnstructured)

	dependenciesObjResource := make([]*udsdrv1alpha1.ObjResource, 0)
	for _, u := range dependenciesUnstructured {
		objr := udsdrv1alpha1.ObjResource{
			GVR:          utils.GVK2GVR(u.GroupVersionKind()),
			Action:       obj.Action,
			Unstructured: &u,
			Namespaced:   true,
		}
		dependenciesObjResource = append(dependenciesObjResource, &objr)
	}

	return dependenciesObjResource, nil
}

func siftPodsForStatefulSet(unsiftedPods []unstructured.Unstructured, uidOfStatefulSet types.UID) []unstructured.Unstructured {
	siftedPods := make([]unstructured.Unstructured, 0)
	for _, pod := range unsiftedPods {
		ownerReferences := pod.GetOwnerReferences()
		// logger.Debugf("check pod ownerReferences: pod: %+v, ownerReferences: %+v", pod, ownerReferences)
		if ownerReferences == nil {
			continue
		}
		if ownerReferences[0].UID == uidOfStatefulSet {
			siftedPods = append(siftedPods, pod)
		}
	}
	return siftedPods
}

func checkPodVolumesDependency(client dynamic.Interface, pod unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	dependentPVCList := make([]unstructured.Unstructured, 0)
	volumes, exist, err := unstructured.NestedSlice(pod.Object, "spec", "volumes")
	if err != nil {
		logger.Errorf("get volumes of pod err: %v, pod: %+v", err.Error(), pod)
		return nil, err
	}
	if !exist {
		logger.Debugf("pod has no volumes, pod: %+v", pod)
		return nil, nil
	}
	if len(volumes) == 0 {
		logger.Debugf("pod's volumes list is empty, pod: %+v", pod)
		return nil, nil
	}
	for _, volume := range volumes {
		pvcMappedValue := reflect.ValueOf(volume).MapIndex(reflect.ValueOf("persistentVolumeClaim"))
		logger.Debugf("pvc of pod is: %+v, type is %T", pvcMappedValue, pvcMappedValue)
		if !pvcMappedValue.IsValid() {
			logger.Debugf("pod has no pvc, pod: %+v", pod)
			// return nil, nil
			continue
		}
		// pvcValue := reflect.ValueOf(pvc)
		// mapKeys := pvc.MapKeys()
		// logger.Debugf("mapkeys are %+v", mapKeys)
		pvcValue := reflect.ValueOf(pvcMappedValue.Interface())
		// claimName := pvc.MapIndex(reflect.ValueOf("claimName"))

		//todo 是否靠谱
		claimNameMappedValue := pvcValue.MapIndex(reflect.ValueOf("claimName"))
		claimNameStr := reflect.ValueOf(claimNameMappedValue.Interface()).String()
		logger.Debugf("claimName is %+v", claimNameStr)
		dependentPVC, err := client.Resource(udsdrv1alpha1.PersistentVolumeClaimGVR).Namespace(pod.GetNamespace()).Get(claimNameStr, v1.GetOptions{})
		if err != nil {
			logger.Errorf("get dependentPVC for pod err: %v, pod: %+v", err.Error(), pod)
			return nil, err
			// continue
		}
		logger.Debugf("dependentPVC is %+v", dependentPVC)
		dependentPVCList = append(dependentPVCList, *dependentPVC.DeepCopy())
	}
	logger.Debugf("dependentPVCList is %+v", dependentPVCList)
	return dependentPVCList, nil
}

func (dc *DependencyChecker) ShouldCheck(obj *udsdrv1alpha1.ObjResource) bool {
	return *obj.GVR == udsdrv1alpha1.StatefulSetGVR
}

func NewDependencyChecker(client dynamic.Interface) udsdrv1alpha1.DependencyChecker {
	return &DependencyChecker{
		client: client,
	}
}
