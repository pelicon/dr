package basenamespaceworker

import (
	"context"

	udsdrv1alpha1 "github.com/pelicon/dr/pkg/apis/udsdr/v1alpha1"
	"github.com/pelicon/dr/pkg/filter"
	"github.com/pelicon/dr/pkg/resourcemanager"
	"github.com/pelicon/dr/pkg/transportmanager"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = log.WithField("module", "base_namespace_worker")
)

type NamesapceWorker interface {
	Active()
	Deactive()
}

type BaseNamesapceWorker struct {
	ctx                        context.Context
	activeCtx                  context.Context
	activeCtxCancel            context.CancelFunc
	Namespace                  udsdrv1alpha1.Namespace
	Filters                    *filter.FilterAggregation
	ResourceManager            resourcemanager.ResourceManager
	TransportManager           transportmanager.TransportManager
	ObjQueue                   workqueue.Interface
	k8sControllerClient        k8sclient.Client
	namespacedStatusUpdateFunc udsdrv1alpha1.NamespacedStatusUpdateFunc
	transportAdapter           udsdrv1alpha1.TransportAdapter
	collectorType              udsdrv1alpha1.ResourceCollectorType
}

func (bnw *BaseNamesapceWorker) filter(obj *udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	var (
		err       error
		objPassed *udsdrv1alpha1.ObjResource
	)

	for _, filter := range bnw.Filters.Filters {
		if objPassed, err = filter.Out(obj); err != nil {
			logger.WithError(err).Errorf("resource %s(namespace:%s name:%s) failed to pass filter",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			return nil, err
		}
	}
	return objPassed, nil
}

func NewBaseNamesapceWorker(
	ctx context.Context,
	k8sControllerClient k8sclient.Client,
	namespacedStatusUpdateFunc udsdrv1alpha1.NamespacedStatusUpdateFunc,
	transportAdapter udsdrv1alpha1.TransportAdapter,
	collectorType udsdrv1alpha1.ResourceCollectorType,
	namespace udsdrv1alpha1.Namespace,
) NamesapceWorker {
	bnw := &BaseNamesapceWorker{
		ctx:                        ctx,
		Namespace:                  namespace,
		Filters:                    filter.GetFilterAggregation(namespace),
		ObjQueue:                   workqueue.New(),
		k8sControllerClient:        k8sControllerClient,
		namespacedStatusUpdateFunc: namespacedStatusUpdateFunc,
		transportAdapter:           transportAdapter,
		collectorType:              collectorType,
	}

	return bnw
}

func (bnw *BaseNamesapceWorker) Active() {
	logger.Info("activing namespaceWorker")
	activeCtx, activeCtxCancel := context.WithCancel(bnw.ctx)
	bnw.activeCtx = activeCtx
	bnw.activeCtxCancel = activeCtxCancel

	resourceManager := resourcemanager.NewBaseResourceManager(activeCtx, bnw.collectorType)
	transportManager := transportmanager.NewBasicTransportManager(
		activeCtx,
		bnw.Namespace,
		bnw.k8sControllerClient,
		bnw.namespacedStatusUpdateFunc,
		bnw.transportAdapter)

	bnw.ResourceManager = resourceManager
	bnw.TransportManager = transportManager

	go bnw.TransportManager.Run()
	go bnw.ResourceManager.Run()

	go wait.UntilWithContext(activeCtx, bnw.objEnqueue, 0)
	go wait.UntilWithContext(activeCtx, bnw.run, 0)
}

func (bnw *BaseNamesapceWorker) Deactive() {
	bnw.activeCtxCancel()
}

func (bnw *BaseNamesapceWorker) objEnqueue(ctx context.Context) {
	objChan := bnw.ResourceManager.GetResourceChan()
	for obj := range objChan {
		bnw.ObjQueue.Add(obj)
		logger.Infof("obj %s(namespace:%s name:%s) enqueue namespaceWorker",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
	}
}

func (bnw *BaseNamesapceWorker) run(ctx context.Context) {
	logger.Info("running namespaceWorker")
	untypedObj, qClosed := bnw.ObjQueue.Get()
	defer bnw.ObjQueue.Done(untypedObj)
	if qClosed {
		logger.Errorf("queue closed")
		return
	}

	obj, canConvert := untypedObj.(*udsdrv1alpha1.ObjResource)
	if !canConvert {
		logger.Errorf("resource %s(namespace:%s name:%s) can not be converted to *udsdrv1alpha1.ObjResource",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
		return
	}

	namespace := bnw.getObjectNamesapce(obj)
	if namespace == bnw.Namespace {
		logger.Infof("start processing resource %s(namespace:%s name:%s)",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
		if objPassed, err := bnw.filter(obj); err != nil {
			logger.WithError(err).Errorf("resource %s(namespace:%s name:%s) failed to pass filter",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
		} else if objPassed != nil {
			bnw.TransportManager.AddResourceObj(objPassed)
		}
	}
}

func (bnw *BaseNamesapceWorker) getObjectNamesapce(obj *udsdrv1alpha1.ObjResource) udsdrv1alpha1.Namespace {
	namespace := udsdrv1alpha1.Namespace(obj.Unstructured.GetNamespace())
	if len(namespace) == 0 {
		return udsdrv1alpha1.ClusterResourceDelegator
	}
	return namespace
}
