package filter

import (
	"sync"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	configs "github.com/pelicon/dr/pkg/configs"
	"github.com/pelicon/dr/pkg/filter/pvcfilter"
	"github.com/pelicon/dr/pkg/filter/variabledeletefilter"
	"github.com/pelicon/dr/pkg/filter/variablemappingfilter"
	"github.com/pelicon/dr/pkg/filter/whitelistfilter"
	log "github.com/sirupsen/logrus"
)

var (
	logger = log.WithField("module", "filter")
)

type FilterAggregation struct {
	*sync.Mutex
	Namespace drv1alpha1.Namespace
	Filters   []drv1alpha1.Filter
}

func (fa *FilterAggregation) UpdateFilterHook(fConfig *drv1alpha1.DRFilterConfig) {
	fa.Lock()
	defer fa.Unlock()

	fa.Filters = getFilters(fConfig)
}

func GetFilterAggregation(namespace drv1alpha1.Namespace) *FilterAggregation {
	fa := &FilterAggregation{
		Mutex:     &sync.Mutex{},
		Namespace: namespace,
		Filters:   make([]drv1alpha1.Filter, 0),
	}
	defer configs.GetConfigContainer().RegFilterConfigListener(namespace, fa.UpdateFilterHook)

	return fa
}

func getFilters(fConfig *drv1alpha1.DRFilterConfig) []drv1alpha1.Filter {
	addDefaultConfig(fConfig)
	filters := make([]drv1alpha1.Filter, 0)

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

func addDefaultConfig(fConfig *drv1alpha1.DRFilterConfig) {
	// enrich this if more default configs needed by filter
	if fConfig == nil {
		fConfig = &drv1alpha1.DRFilterConfig{}
	}

	if fConfig.WhiteListFilter == nil {
		fConfig.WhiteListFilter = &drv1alpha1.WhiteListFilterConfig{}
	}

	for _, gvk := range fConfig.WhiteListFilter.KindWhiteList {
		vd := drv1alpha1.VariableDelete{
			Kind: &gvk,
			// KeyValueDelete: []drv1alpha1.VariableKey{"metadata,resourceVersion", "metadata,uid"},
			KeyValueDelete: []drv1alpha1.VariableKey{"metadata,uid"},
		}
		if fConfig.VariableDeleteFilter == nil {
			vdFilterConfig := drv1alpha1.VariableDeleteFilterConfig{}
			fConfig.VariableDeleteFilter = &vdFilterConfig
		}
		fConfig.VariableDeleteFilter.KindDeleteList = append(fConfig.VariableDeleteFilter.KindDeleteList, vd)
	}
	for _, gvkObj := range fConfig.WhiteListFilter.ObjectWhiteList {
		vd := drv1alpha1.VariableDelete{
			Object:         &gvkObj,
			KeyValueDelete: []drv1alpha1.VariableKey{"metadata,uid"},
		}
		if fConfig.VariableDeleteFilter == nil {
			vdFilterConfig := drv1alpha1.VariableDeleteFilterConfig{}
			fConfig.VariableDeleteFilter = &vdFilterConfig
		}
		fConfig.VariableDeleteFilter.ObjectDeleteList = append(fConfig.VariableDeleteFilter.ObjectDeleteList, vd)
	}
}
