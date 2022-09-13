package filter

import (
	"sync"

	udsdrv1alpha1 "github.com/DaoCloud/udsdr/pkg/apis/udsdr/v1alpha1"
	configs "github.com/DaoCloud/udsdr/pkg/configs"
	"github.com/DaoCloud/udsdr/pkg/filter/pvcfilter"
	"github.com/DaoCloud/udsdr/pkg/filter/variabledeletefilter"
	"github.com/DaoCloud/udsdr/pkg/filter/variablemappingfilter"
	"github.com/DaoCloud/udsdr/pkg/filter/whitelistfilter"
	log "github.com/sirupsen/logrus"
)

var (
	logger = log.WithField("module", "filter")
)

type FilterAggregation struct {
	*sync.Mutex
	Namespace udsdrv1alpha1.Namespace
	Filters   []udsdrv1alpha1.Filter
}

func (fa *FilterAggregation) UpdateFilterHook(fConfig *udsdrv1alpha1.DRFilterConfig) {
	fa.Lock()
	defer fa.Unlock()

	fa.Filters = getFilters(fConfig)
}

func GetFilterAggregation(namespace udsdrv1alpha1.Namespace) *FilterAggregation {
	fa := &FilterAggregation{
		Mutex:     &sync.Mutex{},
		Namespace: namespace,
		Filters:   make([]udsdrv1alpha1.Filter, 0),
	}
	defer configs.GetConfigContainer().RegFilterConfigListener(namespace, fa.UpdateFilterHook)

	return fa
}

func getFilters(fConfig *udsdrv1alpha1.DRFilterConfig) []udsdrv1alpha1.Filter {
	addDefaultConfig(fConfig)
	filters := make([]udsdrv1alpha1.Filter, 0)

	whitelistFilter := whitelistfilter.New()
	if fConfig != nil && fConfig.WhiteListFilter != nil {
		logger.Infof("getting WhiteListFilter: %+v", fConfig.WhiteListFilter)
		whitelistFilter.SetConfig(fConfig)
		// filter := whitelistfilter.New()
		// filter.SetConfig(fConfig)
		// filters = append(filters, filter)
	}
	filters = append(filters, whitelistFilter)

	if fConfig != nil && fConfig.VariableMappingFilter != nil {
		logger.Infof("getting VariableMappingFilter: %+v", fConfig.VariableMappingFilter)
		filter := variablemappingfilter.New()
		filter.SetConfig(fConfig)
		filters = append(filters, filter)
	}
	if fConfig != nil && fConfig.VariableDeleteFilter != nil {
		logger.Infof("getting VariableDeleteFilter: %+v", fConfig.VariableDeleteFilter)
		filter := variabledeletefilter.New()
		filter.SetConfig(fConfig)
		filters = append(filters, filter)
	}
	pvcFilter := pvcfilter.New()
	filters = append(filters, pvcFilter)

	logger.Debugf("filters generated: %v filters: %+v", len(filters), filters)
	return filters
}

func addDefaultConfig(fConfig *udsdrv1alpha1.DRFilterConfig) {
	// enrich this if more default configs needed by filter
	// if fConfig == nil || fConfig.WhiteListFilter == nil {
	// 	return
	// }
	if fConfig == nil {
		fConfig = &udsdrv1alpha1.DRFilterConfig{}
	}

	if fConfig.WhiteListFilter == nil {
		fConfig.WhiteListFilter = &udsdrv1alpha1.WhiteListFilterConfig{}
	}

	/*
		deploymentGVK := udsdrv1alpha1.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "Deployment",
		}
		daemonSetGVK := udsdrv1alpha1.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
		}
		statefulSet := udsdrv1alpha1.GroupVersionKind{
			Group:   "apps",
			Version: "v1",
			Kind:    "StatefulSet",
		}
		pvcGVK := udsdrv1alpha1.GroupVersionKind{
			Version: "v1",
			Kind:    "PersistentVolumeClaim",
		}
		pvGVK := udsdrv1alpha1.GroupVersionKind{
			Version: "v1",
			Kind:    "PersistentVolume",
		}
		serviceGVK := udsdrv1alpha1.GroupVersionKind{
			Version: "v1",
			Kind:    "Service",
		}

		defaultWhiteList := []udsdrv1alpha1.GroupVersionKind{deploymentGVK, serviceGVK, daemonSetGVK, statefulSet, pvcGVK, pvGVK}

		fConfig.WhiteListFilter.KindWhiteList = append(fConfig.WhiteListFilter.KindWhiteList, defaultWhiteList...)

	*/
	for _, gvk := range fConfig.WhiteListFilter.KindWhiteList {
		vd := udsdrv1alpha1.VariableDelete{
			Kind: &gvk,
			// KeyValueDelete: []udsdrv1alpha1.VariableKey{"metadata,resourceVersion", "metadata,uid"},
			KeyValueDelete: []udsdrv1alpha1.VariableKey{"metadata,uid"},
		}
		if fConfig.VariableDeleteFilter == nil {
			vdFilterConfig := udsdrv1alpha1.VariableDeleteFilterConfig{}
			fConfig.VariableDeleteFilter = &vdFilterConfig
		}
		fConfig.VariableDeleteFilter.KindDeleteList = append(fConfig.VariableDeleteFilter.KindDeleteList, vd)
	}
	for _, gvkObj := range fConfig.WhiteListFilter.ObjectWhiteList {
		vd := udsdrv1alpha1.VariableDelete{
			Object:         &gvkObj,
			KeyValueDelete: []udsdrv1alpha1.VariableKey{"metadata,uid"},
		}
		if fConfig.VariableDeleteFilter == nil {
			vdFilterConfig := udsdrv1alpha1.VariableDeleteFilterConfig{}
			fConfig.VariableDeleteFilter = &vdFilterConfig
		}
		fConfig.VariableDeleteFilter.ObjectDeleteList = append(fConfig.VariableDeleteFilter.ObjectDeleteList, vd)
	}
}
