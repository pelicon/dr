package configs

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	udsdrv1alpha1 "github.com/pelicon/dr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	k8scache "k8s.io/client-go/tools/cache"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	logger = log.WithField("module", "configs")
)

type ConfigWatcher interface {
	Run()
}

type baseConfigWatcher struct {
	ctx                      context.Context
	configMapInformerFactory k8sinformers.SharedInformerFactory
	objectKeys               []udsdrv1alpha1.ObjectKey
	client                   kubernetes.Interface
}

func NewBaseConfigWatcher(ctx context.Context, resync time.Duration) (ConfigWatcher, error) {
	config, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &baseConfigWatcher{
		ctx:                      ctx,
		client:                   client,
		configMapInformerFactory: k8sinformers.NewSharedInformerFactory(client, resync),
		objectKeys:               make([]udsdrv1alpha1.ObjectKey, 0),
	}, nil
}

func (bcw *baseConfigWatcher) Run() {
	cmInformer := bcw.configMapInformerFactory.Core().V1().ConfigMaps()
	cmInformer.Informer().AddEventHandler(k8scache.ResourceEventHandlerFuncs{
		AddFunc:    bcw.onAdd,
		UpdateFunc: bcw.onUpdate,
		DeleteFunc: bcw.onDelete,
	})
	go bcw.configMapInformerFactory.Start(bcw.ctx.Done())
	bcw.configMapInformerFactory.WaitForCacheSync(bcw.ctx.Done())
	logger.Info("configmap informer started")
}

func (bcw *baseConfigWatcher) onAdd(untyped interface{}) {
	if cm, canConver := untyped.(*corev1.ConfigMap); canConver {
		if !isDRRelatedCM(cm) {
			return
		}
		logger.Debugf("add dr related configmap, namespace: %s, name: %s", cm.Namespace, cm.Name)
		rawData := cm.Data[udsdrv1alpha1.DRFilterKeyName]
		filterConfig := &udsdrv1alpha1.DRFilterConfig{}
		if err := json.Unmarshal([]byte(rawData), filterConfig); err == nil {
			// currently cm should deploy at the same namespace with the dr namesapce,
			// so we use cm.Namesapce directly
			GetConfigContainer().UpdateFilterConfigToContainer(udsdrv1alpha1.Namespace(cm.Namespace), filterConfig)
		} else {
			logger.WithError(err).Errorf("failed to recognize dr configmap data body %s, name: %s namespace: %s", rawData, cm.Name, cm.Namespace)
		}
	}
	logger.Debug("configmap added")
}

func (bcw *baseConfigWatcher) onUpdate(_, untyped interface{}) {
	if cm, canConver := untyped.(*corev1.ConfigMap); canConver {
		if !isDRRelatedCM(cm) {
			return
		}
		logger.Debugf("update dr related configmap %+v", cm)
		rawData := cm.Data[udsdrv1alpha1.DRFilterKeyName]
		filterConfig := &udsdrv1alpha1.DRFilterConfig{}
		if err := json.Unmarshal([]byte(rawData), filterConfig); err == nil {
			// currently cm should deploy at the same namespace with the dr namesapce,
			// so we use cm.Namesapce directly
			GetConfigContainer().UpdateFilterConfigToContainer(udsdrv1alpha1.Namespace(cm.Namespace), filterConfig)
		} else {
			logger.WithError(err).Errorf("failed to recognize dr configmap data body %s, name: %s namespace: %s", rawData, cm.Name, cm.Namespace)
		}
	}
	logger.Debug("configmap updated")
}

// TODO
func (bcw *baseConfigWatcher) onDelete(obj interface{}) {}

func (bcw *baseConfigWatcher) RegisterCM(obj udsdrv1alpha1.ObjectKey) {
	logger.Infof("reg dr related cm %+v", obj)
	bcw.objectKeys = append(bcw.objectKeys, obj)
}

func (bcw *baseConfigWatcher) UnregisterCM(objToFind udsdrv1alpha1.ObjectKey) {
	logger.Infof("reg dr related cm %+v", objToFind)
	cmToKeep := make([]udsdrv1alpha1.ObjectKey, len(bcw.objectKeys))
	for _, key := range bcw.objectKeys {
		if reflect.DeepEqual(objToFind, key) {
			continue
		}
		cmToKeep = append(cmToKeep, key)
	}

	bcw.objectKeys = cmToKeep
}

func isDRRelatedCM(cm *corev1.ConfigMap) bool {
	if _, exists := cm.Data[udsdrv1alpha1.DRFilterKeyName]; exists {
		return true
	}
	return false
}
