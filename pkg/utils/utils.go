package utils

import "k8s.io/apimachinery/pkg/runtime/schema"

func GVK2GVR(in schema.GroupVersionKind) *schema.GroupVersionResource {
	out := schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Kind,
	}
	return &out
}
