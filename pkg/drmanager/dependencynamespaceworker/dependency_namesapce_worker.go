package dependencynamespaceworker

import (
	"context"
	"fmt"
	"time"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	dcpvc "github.com/pelicon/dr/pkg/drmanager/dependencynamespaceworker/dependencychecker/persistentvolumeclaim"
	dcsts "github.com/pelicon/dr/pkg/drmanager/dependencynamespaceworker/dependencychecker/statefulset"
	"github.com/pelicon/dr/pkg/filter"
	"github.com/pelicon/dr/pkg/namespacecrstatusupdater"
	"github.com/pelicon/dr/pkg/resourcemanager"
	"github.com/pelicon/dr/pkg/transportmanager"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	logger = log.WithField("module", "dependency_namespace_worker")
)

type DependencyNamesapceWorker struct {
	ctx                 context.Context
	nsCRName            string
	nsCRNamespace       string
	Namespace           drv1alpha1.Namespace
	Filters             *filter.FilterAggregation
	ResourceManager     resourcemanager.ResourceManager
	TransportManager    transportmanager.TransportManager
	ObjQueue            workqueue.RateLimitingInterface
	k8sControllerClient k8sclient.Client
	transportAdapter    drv1alpha1.TransportAdapter
	collectorType       drv1alpha1.ResourceCollectorType
	dependencies        map[types.UID]struct{}
	dependencyChecks    []drv1alpha1.DependencyChecker
	statusUpdater       *namespacecrstatusupdater.StatusUpdater
}

func (dnw *DependencyNamesapceWorker) filter(obj *drv1alpha1.ObjResource) (*drv1alpha1.ObjResource, error) {
	var (
		err       error
		objPassed *drv1alpha1.ObjResource
	)

	for _, filter := range dnw.Filters.Filters {
		if objPassed, err = filter.Out(obj); err != nil {
			if err == drv1alpha1.ErrNoPassFilter {
				return nil, nil
			}
			logger.WithError(err).Errorf("resource %s(namespace:%s name:%s) failed to pass filter",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			return nil, err
		}
	}
	return objPassed, nil
}

func NewDependencyNamesapceWorker(
	ctx context.Context,
	nsCRName string,
	nsCRNamespace string,
	k8sControllerClient k8sclient.Client,
	transportAdapter drv1alpha1.TransportAdapter,
	collectorType drv1alpha1.ResourceCollectorType,
	namespace drv1alpha1.Namespace,
	statusUpdater *namespacecrstatusupdater.StatusUpdater,
) drv1alpha1.NamesapceWorker {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(time.Second, time.Second*5)
	dnw := &DependencyNamesapceWorker{
		ctx:                 ctx,
		nsCRName:            nsCRName,
		nsCRNamespace:       nsCRNamespace,
		Namespace:           namespace,
		Filters:             filter.GetFilterAggregation(namespace),
		ObjQueue:            workqueue.NewRateLimitingQueue(rateLimiter),
		k8sControllerClient: k8sControllerClient,
		transportAdapter:    transportAdapter,
		collectorType:       collectorType,
		dependencies:        make(map[types.UID]struct{}),
		statusUpdater:       statusUpdater,
	}

	config := k8sconfig.GetConfigOrDie()
	dClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Panic("failed to get client")
	}

	dnw.dependencyChecks = []drv1alpha1.DependencyChecker{
		dcpvc.NewDependencyChecker(dClient),
		dcsts.NewDependencyChecker(dClient),
	}

	return dnw
}

func (dnw *DependencyNamesapceWorker) namespacedStatusUpdateFunc(condition drv1alpha1.SyncedCondition) {
	dnw.statusUpdater.UpdateCondition(dnw.nsCRNamespace, dnw.nsCRName, condition)
}

func (dnw *DependencyNamesapceWorker) Run() {
	logger.Infof("running namespaceWorker namespace: %s(NSDR: %s)", dnw.Namespace, dnw.nsCRName)

	resourceManager := resourcemanager.NewBaseResourceManager(dnw.ctx, dnw.collectorType)
	transportManager := transportmanager.NewBasicTransportManager(
		dnw.ctx,
		dnw.Namespace,
		dnw.k8sControllerClient,
		dnw.namespacedStatusUpdateFunc,
		dnw.transportAdapter)

	dnw.ResourceManager = resourceManager
	dnw.TransportManager = transportManager

	logger.Infof("running namespaceWorker TransportManager namespace: %s(NSDR: %s)", dnw.Namespace, dnw.nsCRName)
	go dnw.TransportManager.Run()
	logger.Infof("running namespaceWorker ResourceManager namespace: %s(NSDR: %s)", dnw.Namespace, dnw.nsCRName)
	go dnw.ResourceManager.Run()

	go wait.UntilWithContext(dnw.ctx, dnw.objEnqueue, 0)
	go wait.UntilWithContext(dnw.ctx, dnw.run, 0)

	<-dnw.ctx.Done()
	dnw.ObjQueue.ShutDown()
}

func (dnw *DependencyNamesapceWorker) objEnqueue(ctx context.Context) {
	objChan := dnw.ResourceManager.GetResourceChan()
	for obj := range objChan {
		dnw.ObjQueue.AddRateLimited(obj)
		logger.Infof("obj %s(namespace:%s name:%s) enqueue",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
	}
}

func (dnw *DependencyNamesapceWorker) run(ctx context.Context) {
	logger.Info("[DNW] running dependency namespaceWorker")
	untypedObj, qClosed := dnw.ObjQueue.Get()
	defer dnw.ObjQueue.Done(untypedObj)
	if qClosed {
		logger.Errorf("[DNW] queue closed")
		return
	}

	obj, canConvert := untypedObj.(*drv1alpha1.ObjResource)
	if !canConvert {
		logger.Errorf("[DNW] resource %s(namespace:%s name:%s) can not be converted to *drv1alpha1.ObjResource",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
		return
	}

	namespace := dnw.getObjectNamesapce(obj)
	// TODO move dependencies map into yaml
	if _, exists := dnw.dependencies[obj.Unstructured.GetUID()]; exists || namespace == dnw.Namespace {
		logger.Debugf("[DNW] start processing resource %s(namespace:%s name:%s)",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())

		objPassed, err := dnw.filter(obj)
		if err != nil {
			logger.WithError(err).Errorf("[DNW] resource %s(namespace:%s name:%s) failed to pass filter",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			return
		}
		if objPassed == nil {
			logger.Debugf("[DNW] skiped resource %s(namespace:%s name:%s), filter return nil",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			return
		}
		if obj.Action == drv1alpha1.ObjectActionDelete {
			logger.Debugf("[DNW] delete dependencies for resource %s(namespace:%s name:%s), because it will be deleted",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			delete(dnw.dependencies, obj.Unstructured.GetUID())
		}

		// deal with dependency
		if obj.Action == drv1alpha1.ObjectActionCreate || obj.Action == drv1alpha1.ObjectActionUpdate {
			logger.Debugf("[DNW] checking dependencies for resource %s(namespace:%s name:%s)",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())

			dependencies, err := dnw.getDependencies(obj)
			if err != nil {
				logger.Errorf("[DNW] resource %s(namespace:%s name:%s) check dependency failed",
					obj.GVR.Resource,
					obj.Unstructured.GetNamespace(),
					obj.Unstructured.GetName())
				return
			}

			if ntd := dnw.getNotTransportedDependencies(dependencies); len(ntd) != 0 {
				logger.Debugf("[DNW] dependencies fetched %d", len(ntd))
				dnw.ObjQueue.AddRateLimited(obj)
				dnw.requeueDependencies(ntd)
				return
			}
		}

		logger.Debugf("[DNW] start transport resource %s(namespace:%s name:%s)",
			obj.GVR.Resource,
			obj.Unstructured.GetNamespace(),
			obj.Unstructured.GetName())
		dnw.TransportManager.AddResourceObj(objPassed)
	}
}

func (dnw *DependencyNamesapceWorker) getNotTransportedDependencies(objs []*drv1alpha1.ObjResource) []*drv1alpha1.ObjResource {
	npd := make([]*drv1alpha1.ObjResource, 0)
	for _, obj := range objs {
		if !dnw.TransportManager.IsResourceTransported(obj) {
			logger.Debugf("resource %s(namespace:%s name:%s) was not transported as a dependency",
				obj.GVR.Resource,
				obj.Unstructured.GetNamespace(),
				obj.Unstructured.GetName())
			npd = append(npd, obj)
		}
	}

	return npd
}

func (dnw *DependencyNamesapceWorker) requeueDependencies(objs []*drv1alpha1.ObjResource) {
	for _, obj := range objs {
		dnw.ObjQueue.AddRateLimited(obj)
		dnw.dependencies[obj.Unstructured.GetUID()] = struct{}{}
	}
}

func (dnw *DependencyNamesapceWorker) getDependencies(obj *drv1alpha1.ObjResource) ([]*drv1alpha1.ObjResource, error) {
	logger.Debugf("going checking dependencies for resource: %+v", obj.Unstructured)
	deps := make([]*drv1alpha1.ObjResource, 0)
	for _, dpc := range dnw.dependencyChecks {
		if dpc.ShouldCheck(obj) {
			if dps, err := dpc.DependencyCheck(obj); err != nil {
				return nil, err
			} else if len(dps) > 0 {
				deps = append(deps, dps...)
			}
		}
	}
	logger.Debugf("dependencies of resource checked, resource: %+v, dependencies: %+v", obj.Unstructured, deps)
	return deps, nil
}

func (dnw *DependencyNamesapceWorker) getObjectNamesapce(obj *drv1alpha1.ObjResource) drv1alpha1.Namespace {
	namespace := drv1alpha1.Namespace(obj.Unstructured.GetNamespace())
	if len(namespace) == 0 {
		return drv1alpha1.ClusterResourceDelegator
	}
	return namespace
}

func GetObjResourceKey(obj *drv1alpha1.ObjResource) string {
	return fmt.Sprintf("%s-%s-%s", obj.GVR.String(), obj.Unstructured.GetNamespace(), obj.Unstructured.GetName())
}
