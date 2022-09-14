package listwatch

import (
	"context"

	udsdrv1alpha1 "github.com/pelicon/dr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	logger = log.WithField("module", "resourcemanager-listwatch")
)

var (
	ListWatchExcludeGroups = []string{
		"coordination.k8s.io",
		"events.k8s.io",
		"udsdr.dce.daocloud.io",
	}
	ListWatchExcludeResources = []string{
		"endpoints",
		"nodes",
		"events",
		"pods",
		"replicasets",
		"namespaces",
		"endpointslices",
		"controllerrevisions",
		"secrets",
		"limitranges",
		"apiservices",
		"csinodes",

		//temp ignore
		"roles",
		"rolebindings",
		"clusterrolebindings",
		//		"configmaps",
		"clusterroles",
		"customresourcedefinitions",
		"serviceaccounts",
	}
)

type collector struct {
	ResourceChan chan *udsdrv1alpha1.ObjResource
}

func New(resourceChan chan *udsdrv1alpha1.ObjResource) *collector {
	return &collector{
		ResourceChan: resourceChan,
	}
}

func (c *collector) Start(ctx context.Context) {
	logger.Debugf("start listwatch")
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	gvrToWatch := make([]*schema.GroupVersionResource, 0)
	dClient, _ := discovery.NewDiscoveryClientForConfig(cfg)
	resources, _ := discovery.ServerPreferredResources(dClient)

	for _, k := range resources {
		for _, r := range k.APIResources {
			if gv, err := schema.ParseGroupVersion(k.GroupVersion); err == nil {
				group := gv.Group
				version := gv.Version
				verbs := []string(r.Verbs)
				if len(verbs) == 0 {
					continue
				}
				if !canListWatch(verbs) || !shouldListWatchGroup(gv) || !shouldListWatchResource(r.Name) {
					continue
				}
				if len(r.Group) != 0 {
					group = r.Group
				}
				if len(r.Version) != 0 {
					version = r.Version
				}
				gvrToWatch = append(gvrToWatch, &schema.GroupVersionResource{
					Group:    group,
					Version:  version,
					Resource: r.Name,
				})
			}
		}
	}

	handlerFuncGenerate := func(gvr *schema.GroupVersionResource) cache.ResourceEventHandlerFuncs {
		logger.Debugf("start %s informer", gvr.String())
		return cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				unobjMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				unobj := unstructured.Unstructured{
					Object: unobjMap,
				}
				objr := udsdrv1alpha1.ObjResource{
					Unstructured: &unobj,
					Action:       udsdrv1alpha1.ObjectActionCreate,
					GVR:          gvr,
				}
				logger.Debugf("informer adding resource %s(namespace:%s name:%s)",
					gvr.Resource,
					objr.Unstructured.GetNamespace(),
					objr.Unstructured.GetName())
				c.ResourceChan <- &objr
			},
			UpdateFunc: func(_, obj interface{}) {
				unobjMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				unobj := unstructured.Unstructured{
					Object: unobjMap,
				}
				objr := udsdrv1alpha1.ObjResource{
					Unstructured: &unobj,
					Action:       udsdrv1alpha1.ObjectActionUpdate,
					GVR:          gvr,
				}
				logger.Debugf("informer updating resource %s(namespace:%s name:%s)",
					gvr.Resource,
					objr.Unstructured.GetNamespace(),
					objr.Unstructured.GetName())
				c.ResourceChan <- &objr
			},
			DeleteFunc: func(obj interface{}) {
				unobjMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				unobj := unstructured.Unstructured{
					Object: unobjMap,
				}
				objr := udsdrv1alpha1.ObjResource{
					Unstructured: &unobj,
					Action:       udsdrv1alpha1.ObjectActionDelete,
					GVR:          gvr,
				}
				logger.Debugf("informer deleting %s(namespace:%s name:%s)",
					gvr.Resource,
					objr.Unstructured.GetNamespace(),
					objr.Unstructured.GetName())
				c.ResourceChan <- &objr
			},
		}
	}

	di := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, 0, "", nil)
	for _, gvr := range gvrToWatch {
		informer := di.ForResource(*gvr)
		informer.Informer().AddEventHandler(handlerFuncGenerate(gvr))
	}

	di.WaitForCacheSync(ctx.Done())
	di.Start(ctx.Done())
}

func shouldListWatchGroup(gv schema.GroupVersion) bool {
	for _, list := range ListWatchExcludeGroups {
		if list == gv.Group {
			return false
		}
	}
	return true
}

func shouldListWatchResource(resource string) bool {
	for _, list := range ListWatchExcludeResources {
		if list == resource {
			return false
		}
	}
	return true
}

func canListWatch(verbs []string) bool {
	var canList, canWatch bool
	for _, verb := range verbs {
		if verb == "watch" {
			canWatch = true
		}
		if verb == "list" {
			canList = true
		}
	}

	return canList && canWatch
}
