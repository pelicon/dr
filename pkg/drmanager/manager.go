package drmanager

import (
	"context"
	udsdrv1alpha1 "github.com/pelicon/dr/pkg/apis/udsdr/v1alpha1"
	//	"github.com/pelicon/dr/pkg/drmanager/basenamespaceworker"
	"github.com/pelicon/dr/pkg/drmanager/dependencynamespaceworker"
	"github.com/pelicon/dr/pkg/namespacecrstatusupdater"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type DRManager interface {
	AddNamespaceWorker(udsdrv1alpha1.Namespace)
}

type BaseDrManager struct {
	ctx                 context.Context
	nsCRName            string
	nsCRNamespace       string
	conditionsCh        chan udsdrv1alpha1.SyncedCondition
	k8sControllerClient k8sclient.Client
	transportAdapter    udsdrv1alpha1.TransportAdapter
	collectorType       udsdrv1alpha1.ResourceCollectorType
	NamespaceWorkers    map[udsdrv1alpha1.Namespace]udsdrv1alpha1.NamesapceWorker
	statusUpdater       *namespacecrstatusupdater.StatusUpdater
}

func NewDRManager(
	ctx context.Context,
	nsCRName string,
	nsCRNamespace string,
	k8sControllerClient k8sclient.Client,
	transportAdapter udsdrv1alpha1.TransportAdapter,
	collectorType udsdrv1alpha1.ResourceCollectorType,
	statusUpdater *namespacecrstatusupdater.StatusUpdater,
) DRManager {
	return &BaseDrManager{
		ctx:                 ctx,
		nsCRName:            nsCRName,
		nsCRNamespace:       nsCRNamespace,
		k8sControllerClient: k8sControllerClient,
		conditionsCh:        make(chan udsdrv1alpha1.SyncedCondition, 1),
		transportAdapter:    transportAdapter,
		collectorType:       collectorType,
		NamespaceWorkers:    make(map[udsdrv1alpha1.Namespace]udsdrv1alpha1.NamesapceWorker),
		statusUpdater:       statusUpdater,
	}
}

func (bdm *BaseDrManager) AddNamespaceWorker(namespace udsdrv1alpha1.Namespace) {
	namespaceWorker := dependencynamespaceworker.NewDependencyNamesapceWorker(
		bdm.ctx,
		bdm.nsCRName,
		bdm.nsCRNamespace,
		bdm.k8sControllerClient,
		bdm.transportAdapter,
		bdm.collectorType,
		namespace,
		bdm.statusUpdater)
	bdm.NamespaceWorkers[namespace] = namespaceWorker
	go namespaceWorker.Run()
}
