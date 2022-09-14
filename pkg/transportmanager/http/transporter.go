package http

import (
	"context"
	"sync"

	udsdrv1alpha1 "github.com/pelicon/dr/pkg/apis/udsdr/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type httpTransporter struct {
	*sync.Mutex
	ctx                        context.Context
	K8sControllerClient        k8sclient.Client
	NamespacedStatusUpdateFunc udsdrv1alpha1.NamespacedStatusUpdateFunc
	PairClusterSettings        *udsdrv1alpha1.PairClusterSettings
	ResourceVersionCache       map[types.UID]string
}

func New(ctx context.Context, k8sControllerClient k8sclient.Client, namespacedStatusUpdateFunc udsdrv1alpha1.NamespacedStatusUpdateFunc) *httpTransporter {
	return &httpTransporter{
		Mutex:                      &sync.Mutex{},
		ctx:                        ctx,
		K8sControllerClient:        k8sControllerClient,
		NamespacedStatusUpdateFunc: namespacedStatusUpdateFunc,
		ResourceVersionCache:       make(map[types.UID]string),
	}
}

func (ht *httpTransporter) SetConfig(cConfigs *udsdrv1alpha1.PairClusterSettings) {
	ht.Lock()
	defer ht.Unlock()

	ht.PairClusterSettings = cConfigs
}

func (ht *httpTransporter) Transport(obj *udsdrv1alpha1.ObjResource) error {
	return nil
}
