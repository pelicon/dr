package v1alpha1

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	PersistentVolumeGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "persistentvolumes",
	}
	PersistentVolumeClaimGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "persistentvolumeclaims",
	}
	PodGVR = schema.GroupVersionResource{
		Version:  "v1",
		Resource: "pods",
	}
	StatefulSetGVR = schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	}
)
