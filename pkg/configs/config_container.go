package configs

import (
	"fmt"
	"sync"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
)

type ConfigContainer interface {
	GetFilterConfigs() map[drv1alpha1.Namespace]*drv1alpha1.DRFilterConfig

	GetClusterConfigs() map[drv1alpha1.ClusterName]*drv1alpha1.PairClusterSettings

	UpdateClusterConfigToContainer(
		clusterName drv1alpha1.ClusterName,
		clusterConfigs *drv1alpha1.PairClusterSettings,
	) error

	UpdateNamespaceConfigedCluster(
		namespace drv1alpha1.Namespace,
		clusterName drv1alpha1.ClusterName,
	) error

	UpdateFilterConfigToContainer(
		namespace drv1alpha1.Namespace,
		filterConfigs *drv1alpha1.DRFilterConfig,
	)

	RegClusterConfigListener(
		namespace drv1alpha1.Namespace,
		fn func(*drv1alpha1.PairClusterSettings),
	) error

	RegFilterConfigListener(
		namespace drv1alpha1.Namespace,
		fn func(*drv1alpha1.DRFilterConfig),
	) error

	NotifyFilterListeners(
		namespace drv1alpha1.Namespace,
		filterConfig *drv1alpha1.DRFilterConfig,
	) error

	NotifyClusterConfigChange(
		clusterName drv1alpha1.ClusterName,
		clusterConfigs *drv1alpha1.PairClusterSettings,
	) error

	NotifyNamespaceConfigedClusterChange(
		namespace drv1alpha1.Namespace,
		clusterName drv1alpha1.ClusterName,
	) error
}

type drConfig struct {
	*sync.RWMutex
	// ClusterConfigs update by cluster CRDs
	ClusterConfigs map[drv1alpha1.ClusterName]*drv1alpha1.PairClusterSettings
	// NamespaceConfigedCluster mapping between DR namespace and cluster config
	NamespaceConfigedCluster map[drv1alpha1.Namespace]drv1alpha1.ClusterName
	// FilterConfigs filter configs
	FilterConfigs map[drv1alpha1.Namespace]*drv1alpha1.DRFilterConfig
	// ClusterConfigListeners
	ClusterConfigListeners map[drv1alpha1.Namespace]func(*drv1alpha1.PairClusterSettings)
	// FilterConfigListeners
	FilterConfigListeners map[drv1alpha1.Namespace]func(*drv1alpha1.DRFilterConfig)
}

var (
	drConfigContainer ConfigContainer
)

func GetConfigContainer() ConfigContainer {
	return drConfigContainer
}

func initConfigContainer() {
	logger.Info("init config container")

	clusterConfigs := make(map[drv1alpha1.ClusterName]*drv1alpha1.PairClusterSettings)
	namespaceConfigedCluster := make(map[drv1alpha1.Namespace]drv1alpha1.ClusterName)
	filterConfigs := make(map[drv1alpha1.Namespace]*drv1alpha1.DRFilterConfig)
	clusterConfigListeners := make(map[drv1alpha1.Namespace]func(*drv1alpha1.PairClusterSettings))
	filterConfigListeners := make(map[drv1alpha1.Namespace]func(*drv1alpha1.DRFilterConfig))

	drConfigContainer = &drConfig{
		RWMutex:                  &sync.RWMutex{},
		ClusterConfigs:           clusterConfigs,
		NamespaceConfigedCluster: namespaceConfigedCluster,
		FilterConfigs:            filterConfigs,
		ClusterConfigListeners:   clusterConfigListeners,
		FilterConfigListeners:    filterConfigListeners,
	}
}

func init() {
	initConfigContainer()
}

func (drc *drConfig) UpdateClusterConfigToContainer(
	clusterName drv1alpha1.ClusterName,
	clusterConfigs *drv1alpha1.PairClusterSettings,
) error {
	logger.Debugf("updating cluster configs into container. cluster name: %s, config: %+v", clusterName, clusterConfigs)

	if clusterConfigs == nil {
		err := fmt.Errorf("cluster config cannot be nil")
		logger.WithError(err)
		return fmt.Errorf("cluster config cannot be nil")
	}

	drc.Lock()
	drc.ClusterConfigs[clusterName] = clusterConfigs
	drc.Unlock()

	drc.NotifyClusterConfigChange(clusterName, clusterConfigs)

	return nil
}

func (drc *drConfig) UpdateNamespaceConfigedCluster(namespace drv1alpha1.Namespace, clusterName drv1alpha1.ClusterName) error {
	logger.Debugf("updating namespace configs into container. ns: %s, clustername: %+v", namespace, clusterName)

	if len(namespace) == 0 {
		return fmt.Errorf("namespace cannot be empty")
	}
	if len(clusterName) == 0 {
		return fmt.Errorf("cluster name cannot be empty")
	}

	drc.Lock()
	drc.NamespaceConfigedCluster[namespace] = clusterName
	drc.Unlock()

	drc.NotifyNamespaceConfigedClusterChange(namespace, clusterName)

	return nil
}

func (drc *drConfig) UpdateFilterConfigToContainer(
	namespace drv1alpha1.Namespace,
	filterConfigs *drv1alpha1.DRFilterConfig,
) {
	logger.Debugf("updating filter configs into container. ns: %s, config: %+v", namespace, filterConfigs)

	drc.Lock()
	drc.FilterConfigs[namespace] = filterConfigs
	drc.Unlock()

	drc.NotifyFilterListeners(namespace, filterConfigs)
}

func (drc *drConfig) RegClusterConfigListener(namespace drv1alpha1.Namespace, fn func(*drv1alpha1.PairClusterSettings)) error {
	logger.Debugf("reg cluster config listener into container. ns: %s", namespace)

	clusterName, exists := drc.NamespaceConfigedCluster[namespace]
	if !exists {
		return fmt.Errorf("namesapce has not config cluster")
	}

	drc.Lock()
	drc.ClusterConfigListeners[namespace] = fn
	drc.Unlock()

	drc.NotifyNamespaceConfigedClusterChange(namespace, clusterName)

	return nil
}

func (drc *drConfig) RegFilterConfigListener(namespace drv1alpha1.Namespace, fn func(*drv1alpha1.DRFilterConfig)) error {
	logger.Debugf("reg filter config listener into container. ns: %s", namespace)

	drc.Lock()
	drc.FilterConfigListeners[namespace] = fn
	drc.Unlock()

	drc.NotifyFilterListeners(namespace, drc.FilterConfigs[namespace])
	return nil
}

func (drc *drConfig) NotifyFilterListeners(namespace drv1alpha1.Namespace, filterConfig *drv1alpha1.DRFilterConfig) error {
	logger.Debugf("notify filter listener. ns: %s, config: %+v", namespace, filterConfig)

	drc.RLock()
	defer drc.RUnlock()

	if listener, exists := drc.FilterConfigListeners[namespace]; exists {
		listener(filterConfig)
	}
	return nil
}

func (drc *drConfig) NotifyClusterConfigChange(clusterName drv1alpha1.ClusterName, clusterConfigs *drv1alpha1.PairClusterSettings) error {
	logger.Debugf("notify cluster listener. ns: %s, config: %+v", clusterName, clusterConfigs)

	drc.RLock()
	defer drc.RUnlock()

	relatedNamesapces := make([]drv1alpha1.Namespace, 0)
	for namespace, cluster := range drc.NamespaceConfigedCluster {
		if cluster == clusterName {
			relatedNamesapces = append(relatedNamesapces, namespace)
		}
	}

	if len(relatedNamesapces) == 0 {
		// TODO log
		return nil
	}

	for _, relatedNamesapce := range relatedNamesapces {
		if listener, exists := drc.ClusterConfigListeners[relatedNamesapce]; exists {
			listener(clusterConfigs)
		} else {
			// TODO err
		}
	}

	return nil
}

func (drc *drConfig) NotifyNamespaceConfigedClusterChange(namespace drv1alpha1.Namespace, clusterName drv1alpha1.ClusterName) error {
	logger.Debugf("notify namesapce listener. ns: %s, config: %+v", namespace, clusterName)

	drc.RLock()
	defer drc.RUnlock()

	if clusterConfig, exists := drc.ClusterConfigs[clusterName]; exists {
		if listener, exists := drc.ClusterConfigListeners[namespace]; exists {
			listener(clusterConfig)
		} else {
			err := fmt.Errorf("failed to find ClusterConfigListener in namespace %s", namespace)
			logger.WithError(err)
			return err
		}
	} else {
		err := fmt.Errorf("failed to find ClusterConfig %s", clusterName)
		logger.WithError(err)
		return err
	}

	return nil
}

func (drc *drConfig) GetClusterConfigs() map[drv1alpha1.ClusterName]*drv1alpha1.PairClusterSettings {
	drc.RLock()
	defer drc.RUnlock()

	return drc.ClusterConfigs
}

func (drc *drConfig) GetFilterConfigs() map[drv1alpha1.Namespace]*drv1alpha1.DRFilterConfig {
	drc.RLock()
	defer drc.RUnlock()

	return drc.FilterConfigs
}
