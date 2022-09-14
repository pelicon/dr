package transportmanager

import (
	"context"
	"strconv"
	"strings"
	"time"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	configs "github.com/pelicon/dr/pkg/configs"
	"github.com/pelicon/dr/pkg/transportmanager/http"
	"github.com/pelicon/dr/pkg/transportmanager/kubeapiserver"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = log.WithField("module", "transportmanager")
)

type TransportManager interface {
	IsResourceTransported(*drv1alpha1.ObjResource) bool
	AddResourceObj(*drv1alpha1.ObjResource)
	Run()
}

type Transporter interface {
	SetConfig(*drv1alpha1.PairClusterSettings)
	Transport(*drv1alpha1.ObjResource) error
}

type BasicTransportManager struct {
	ctx                        context.Context
	namespace                  drv1alpha1.Namespace
	K8sControllerClient        k8sclient.Client
	NamespacedStatusUpdateFunc drv1alpha1.NamespacedStatusUpdateFunc
	Versions                   map[types.UID]string
	// ObjQueue                   workqueue.Interface
	ObjQueue         workqueue.RateLimitingInterface
	TransportAdapter drv1alpha1.TransportAdapter
	Transporter      Transporter
}

func (btm *BasicTransportManager) IsResourceTransported(obj *drv1alpha1.ObjResource) bool {
	uid := obj.Unstructured.GetUID()
	if _, exists := btm.Versions[uid]; exists {
		return true
	}

	return false
}

func (btm *BasicTransportManager) AddResourceObj(obj *drv1alpha1.ObjResource) {
	btm.ObjQueue.Add(obj)
	logger.Debugf("obj %s(namespace:%s name:%s)enqueue transportManager",
		obj.GVR.Resource,
		obj.Unstructured.GetNamespace(),
		obj.Unstructured.GetName())
}

func (btm *BasicTransportManager) Run() {
	go wait.UntilWithContext(btm.ctx, btm.transport, 0)

	<-btm.ctx.Done()
	btm.ObjQueue.ShutDown()
}

func (btm *BasicTransportManager) transport(ctx context.Context) {
	untypedObj, qClosed := btm.ObjQueue.Get()
	defer btm.ObjQueue.Done(untypedObj)
	if qClosed {
		logger.Errorf("queue closed")
		return
	}

	obj, canConvert := untypedObj.(*drv1alpha1.ObjResource)
	if !canConvert {
		logger.Errorf("obj can not be converted")
		return
	}

	logger.Debugf("start transporting resource %s(namespace:%s name:%s)",
		obj.GVR.Resource,
		obj.Unstructured.GetNamespace(),
		obj.Unstructured.GetName())

	syncedCondition := generateSyncedCondition(obj)

	if !btm.isNewVersion(obj) {
		logger.Debugf("skiping older version resource %s(namespace:%s name:%s)",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
		return
	}

	objToTransport := obj.DeepCopy()
	// btm.prepareTransport(objToTransport)
	if err := btm.Transporter.Transport(objToTransport); err != nil {
		syncedCondition.LastSyncedStatus = drv1alpha1.SyncedStatus("fail")
		// TODO @keichen.yi should encapsulate below into a func. e.g. shouldSkipErr(err error) bool
		// TODO @keichen.yi error should warped and match it by type
		if strings.Contains(err.Error(), "already exists") {
			logger.Debugf("skiping already exists resource %s(namespace:%s name:%s)",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			btm.markAsSuccessed(obj)
		} else {
			logger.WithError(err).Errorf("resource %s(namespace:%s name:%s) transporting failed",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			// btm.ObjQueue.Add(obj)
			btm.ObjQueue.AddRateLimited(obj)
		}
	} else {
		btm.markAsSuccessed(obj)
	}

	logger.Debugf("going updating drns status")
	btm.NamespacedStatusUpdateFunc(syncedCondition)
	logger.Debugf("updating drns status done")
}

// func ignoreErr(err error) bool {

// }

func (btm *BasicTransportManager) isNewVersion(obj *drv1alpha1.ObjResource) bool {
	recievedObjVerison := obj.Unstructured.GetResourceVersion()
	recievedObjUID := obj.Unstructured.GetUID()
	markedResourceVersion, exists := btm.Versions[recievedObjUID]
	if !exists {
		return true
	}

	// always can convert version(STRING) into INT
	recievedObjVerisonInt, _ := strconv.Atoi(recievedObjVerison)
	markedResourceVersionInt, _ := strconv.Atoi(markedResourceVersion)
	if markedResourceVersionInt < recievedObjVerisonInt {
		return true
	}

	return false
}

func (btm *BasicTransportManager) markAsSuccessed(obj *drv1alpha1.ObjResource) {
	btm.Versions[obj.Unstructured.GetUID()] = obj.Unstructured.GetResourceVersion()
}

// func (btm *BasicTransportManager) prepareTransport(obj *drv1alpha1.ObjResource) {
// 	if obj.Action != drv1alpha1.ObjectActionUpdate {
// 		unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "resourceVersion")
// 	}
// 	unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "uid")
// }

func generateSyncedCondition(obj *drv1alpha1.ObjResource) drv1alpha1.SyncedCondition {
	gvk := drv1alpha1.GroupVersionKind{
		Group:   obj.GVR.Group,
		Version: obj.GVR.Version,
		Kind:    obj.GVR.Resource,
	}
	objectKey := drv1alpha1.ObjectKey{
		Namespace: obj.Unstructured.GetNamespace(),
		Name:      obj.Unstructured.GetName(),
	}
	gvko := drv1alpha1.GroupVersionKindObject{
		GroupVersionKind: gvk,
		ObjectKey:        objectKey,
	}

	syncedCondition := drv1alpha1.SyncedCondition{
		GroupVersionKindObject:    gvko,
		LastSyncedResourceVersion: obj.Unstructured.GetResourceVersion(),
		LastSyncedStatus:          "success",
	}
	return syncedCondition
}

func NewBasicTransportManager(
	ctx context.Context,
	namespace drv1alpha1.Namespace,
	k8sControllerClient k8sclient.Client,
	namespacedStatusUpdateFunc drv1alpha1.NamespacedStatusUpdateFunc,
	transportAdapter drv1alpha1.TransportAdapter,
) TransportManager {
	transporter := newTransporter(ctx, k8sControllerClient, namespacedStatusUpdateFunc, transportAdapter)
	configs.GetConfigContainer().RegClusterConfigListener(namespace, transporter.SetConfig)
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(time.Second, time.Second*5)

	return &BasicTransportManager{
		ctx:                        ctx,
		namespace:                  namespace,
		K8sControllerClient:        k8sControllerClient,
		Versions:                   make(map[types.UID]string),
		NamespacedStatusUpdateFunc: namespacedStatusUpdateFunc,
		// ObjQueue:                   workqueue.New(), // TODO rate limit
		ObjQueue:         workqueue.NewRateLimitingQueue(rateLimiter),
		TransportAdapter: transportAdapter,
		Transporter:      transporter,
	}
}

func newTransporter(
	ctx context.Context,
	k8sControllerClient k8sclient.Client,
	namespacedStatusUpdateFunc drv1alpha1.NamespacedStatusUpdateFunc,
	transportAdapter drv1alpha1.TransportAdapter,
) Transporter {
	switch transportAdapter {
	case drv1alpha1.TransportAdapterKubeApiserver:
		return kubeapiserver.New(ctx, k8sControllerClient, namespacedStatusUpdateFunc)
	case drv1alpha1.TransportAdapterHTTP:
		return http.New(ctx, k8sControllerClient, namespacedStatusUpdateFunc)
	}
	//todo do not return nil

	return nil
}
