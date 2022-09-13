package variablemappingfilter

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	udsdrv1alpha1 "github.com/DaoCloud/udsdr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var FilterName udsdrv1alpha1.FilterName = "VariableMappingFilter"
var (
	logger = log.WithField("module", "variablemappingfilter")
)

type VariableMappingFilter struct {
	*sync.Mutex
	udsdrv1alpha1.DRFilterConfig
}

func New() udsdrv1alpha1.Filter {
	return &VariableMappingFilter{
		Mutex: &sync.Mutex{},
	}
}

// func (vmf *VariableMappingFilter) Out(objr *udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
// 	logger.Infof("variable mapping configs: %+v", vmf.DRFilterConfig.VariableMappingFilter)
// 	for _, kindVariableMappings := range vmf.DRFilterConfig.VariableMappingFilter.KindVariableMappings {
// 		logger.Infof("kindvariablemapping.kind: %+v, objr.gvk: %+v", kindVariableMappings.Kind, udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()))
// 		if udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()) == *kindVariableMappings.Kind {
// 			logger.Infof("keyvaluemappings: %+v", kindVariableMappings.KeyValueMappings)
// 			for k, v := range kindVariableMappings.KeyValueMappings {
// 				logger.Infof("keyvaluemappings, key: %v, value: %v", k, v)
// 				path := strings.Split(string(k), ",")
// 				if _, exist, _ := unstructured.NestedFieldCopy(objr.Unstructured.Object, path...); exist {
// 					// if _, exist := objr.Unstructured.Object[string(k)]; exist {
// 						if err := setMappingValue(v, path, objr); err != nil {
// 							return nil, err
// 						}
// 					// switch v.VariableType {
// 					// case udsdrv1alpha1.VariableTypeStr:
// 					// 	oldStrValue, ok, err := unstructured.NestedString(objr.Unstructured.Object, path...)
// 					// 	if err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// 	if ok {
// 					// 		newStrValue := strings.Replace(oldStrValue, v.FromSubStr, v.ToSubStr, 1)
// 					// 		if err := unstructured.SetNestedField(objr.Unstructured.Object, newStrValue, path...); err != nil {
// 					// 			return nil, err
// 					// 		}
// 					// 	}
// 					// case udsdrv1alpha1.VariableTypeBool:
// 					// 	newBoolValue, err := strconv.ParseBool(v.ToSubStr)
// 					// 	if err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// 	if err := unstructured.SetNestedField(objr.Unstructured.Object, newBoolValue, path...); err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// case udsdrv1alpha1.VariableTypeInt:
// 					// 	newIntValue, err := strconv.Atoi(v.ToSubStr)
// 					// 	if err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// 	if err := unstructured.SetNestedField(objr.Unstructured.Object, int64(newIntValue), path...); err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// case udsdrv1alpha1.VariableTypeFloat:
// 					// 	newFloatValue, err := strconv.ParseFloat(v.ToSubStr, 64)
// 					// 	if err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// 	if err := unstructured.SetNestedField(objr.Unstructured.Object, newFloatValue, path...); err != nil {
// 					// 		return nil, err
// 					// 	}
// 					// default:
// 					// 	return nil, fmt.Errorf("unsupported type")
// 					// }
// 				}
// 			}
// 		}
// 	}
// 	for _, objectVariableMappings := range vmf.DRFilterConfig.VariableMappingFilter.ObjectVariableMappings {
// 		if (udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()) == *objectVariableMappings.Kind) &&
// 			(objr.Unstructured.GetNamespace() == objectVariableMappings.Object.ObjectKey.Namespace) &&
// 			(objr.Unstructured.GetName() == objectVariableMappings.Object.ObjectKey.Name) {
// 			for k, v := range objectVariableMappings.KeyValueMappings {
// 				if _, exist := objr.Unstructured.Object[string(k)]; exist {
// 					path := strings.Split(string(k), ",")
// 					if err := setMappingValue(v, path, objr); err != nil {
// 						return nil, err
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return objr, nil
// }

func setMappingValue(v udsdrv1alpha1.VariableMapping, path []string, m map[string]interface{}) error {
	switch v.VariableType {
	case udsdrv1alpha1.VariableTypeStr:
		oldStrValue, ok, err := unstructured.NestedString(m, path...)
		if err != nil {
			return err
		}
		if ok {
			newStrValue := strings.Replace(oldStrValue, v.FromSubStr, v.ToSubStr, 1)
			if err := unstructured.SetNestedField(m, newStrValue, path...); err != nil {
				return err
			}
		}
	case udsdrv1alpha1.VariableTypeBool:
		newBoolValue, err := strconv.ParseBool(v.ToSubStr)
		if err != nil {
			return err
		}
		if err := unstructured.SetNestedField(m, newBoolValue, path...); err != nil {
			return err
		}
	case udsdrv1alpha1.VariableTypeInt:
		newIntValue, err := strconv.Atoi(v.ToSubStr)
		if err != nil {
			return err
		}
		if err := unstructured.SetNestedField(m, int64(newIntValue), path...); err != nil {
			return err
		}
	case udsdrv1alpha1.VariableTypeFloat:
		newFloatValue, err := strconv.ParseFloat(v.ToSubStr, 64)
		if err != nil {
			return err
		}
		if err := unstructured.SetNestedField(m, newFloatValue, path...); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type")
	}
	return nil
}

func (vmf *VariableMappingFilter) Out(objr *udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	for _, kindVariableMappings := range vmf.DRFilterConfig.VariableMappingFilter.KindVariableMappings {
		if udsdrv1alpha1.GroupVersionKind(objr.Unstructured.GroupVersionKind()) == *kindVariableMappings.Kind {
			for k, v := range kindVariableMappings.KeyValueMappings {
				path := strings.Split(string(k), ",")
				// if _, exist, _ := unstructured.NestedFieldCopy(objr.Unstructured.Object, path...); exist {

				// }
				if _, exist, _ := unstructured.NestedFieldCopy(objr.Unstructured.Object, path[0]); exist {
					if err := iteratePath(path[1:], &v, objr.Unstructured.Object[path[0]]); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return objr, nil
}

func iteratePath(path []string, v *udsdrv1alpha1.VariableMapping, inf interface{}) error {

	infType := reflect.TypeOf(inf)
	logger.Infof("type of interface:%v", infType)
	logger.Infof("value of interface:%v", reflect.ValueOf(inf))
	logger.Infof("kind of interface:%v", infType.Kind())
	logger.Infof("path:%v", path)

	// if len(path) == 1 {
	// 	//the last layer is expected to be a map
	// 	// infType := reflect.TypeOf(inf)
	// 	if infType.Kind() != reflect.Map {
	// 		return fmt.Errorf("interface not a map")
	// 	}
	// 	m := inf.(map[string]interface{})
	// 	if err := setMappingValue(*v, path, m); err != nil {
	// 		return err
	// 	}
	// 	return nil
	// }

	infValue := reflect.ValueOf(inf)

	switch infType.Kind() {
	case reflect.Map:

		if len(path) == 1 {
			//the last layer is expected to be a map
			// infType := reflect.TypeOf(inf)
			if infType.Kind() != reflect.Map {
				logger.Errorf("interface not a map")
				return fmt.Errorf("interface not a map")
			}
			m := inf.(map[string]interface{})
			if err := setMappingValue(*v, path, m); err != nil {
				return err
			}
			return nil
		}

		mappedValue := infValue.MapIndex(reflect.ValueOf(path[0]))
		logger.Infof("mapped value:%+v", mappedValue)
		if !mappedValue.IsValid() {
			logger.Errorf("invalid value")
			return nil
		}
		// mappedType := mappedValue.Type()
		mappedInf := mappedValue.Interface()

		// infMap := inf.(map[string]interface{})

		err := iteratePath(path[1:], v, mappedInf)
		return err
	case reflect.Slice:
		for i := 0; i < infValue.Len(); i++ {

			ele := infValue.Index(i)
			eleInf := ele.Interface()
			logger.Infof("type of eleInf:%v", reflect.TypeOf(eleInf))
			logger.Infof("value of eleInf:%v", reflect.ValueOf(eleInf))
			logger.Infof("kind of eleInf:%v", reflect.TypeOf(eleInf).Kind())

			if len(path) == 1 {
				//the last layer is expected to be a map
				// infType := reflect.TypeOf(inf)
				if reflect.TypeOf(eleInf).Kind() != reflect.Map {
					logger.Errorf("interface not a map")
					return fmt.Errorf("interface not a map")
				}
				m := eleInf.(map[string]interface{})
				if err := setMappingValue(*v, path, m); err != nil {
					return err
				}
				// return nil
				continue
			}

			if err := iteratePath(path[1:], v, eleInf); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("wrong type")
	}
}

func (vmf *VariableMappingFilter) In(*udsdrv1alpha1.ObjResource) (*udsdrv1alpha1.ObjResource, error) {
	return nil, nil
}

func (vmf *VariableMappingFilter) SetConfig(drconf *udsdrv1alpha1.DRFilterConfig) error {
	vmf.Lock()
	defer vmf.Unlock()

	vmf.DRFilterConfig = *drconf
	return nil
}
