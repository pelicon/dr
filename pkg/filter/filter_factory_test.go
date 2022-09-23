package filter

import (
	"testing"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func Test_UpdateFilterHook(t *testing.T) {
	ns := drv1alpha1.Namespace("test_ns")
	f := GetFilterAggregation(ns)

	conf := &drv1alpha1.DRFilterConfig{
		VariableDeleteFilter: &drv1alpha1.VariableDeleteFilterConfig{
			KindDeleteList: []drv1alpha1.VariableDelete{
				{
					Kind: &drv1alpha1.GroupVersionKind{
						Group:   "test.io",
						Version: "v1",
						Kind:    "kind1",
					},
					KeyValueDelete: []drv1alpha1.VariableKey{
						drv1alpha1.VariableKey("spec.delThis"),
					},
				},
			},
		},
		VariableMappingFilter: &drv1alpha1.VariableMappingFilterConfig{
			KindVariableMappings: []drv1alpha1.VariableMappings{
				{
					Kind: &drv1alpha1.GroupVersionKind{
						Group:   "test.io",
						Version: "v1",
						Kind:    "kind1",
					},
					KeyValueMappings: map[drv1alpha1.VariableKey]drv1alpha1.VariableMapping{
						drv1alpha1.VariableKey("spec.col"): {
							FromSubStr:   "substr1",
							ToSubStr:     "substr2",
							VariableType: drv1alpha1.VariableTypeStr,
						},
					},
				},
			},
		},
		WhiteListFilter: &drv1alpha1.WhiteListFilterConfig{
			KindWhiteList: []drv1alpha1.GroupVersionKind{
				{
					Group:   "test.io",
					Version: "v1",
					Kind:    "kind1",
				},
			},
		},
	}

	f.UpdateFilterHook(conf)
	assert.Equal(t, 4, len(f.Filters))
}
