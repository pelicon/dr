package drmanager

import (
	"context"
	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	//	"github.com/pelicon/dr/pkg/drmanager/basenamespaceworker"
	"github.com/pelicon/dr/pkg/drmanager/dependencynamespaceworker"
	"github.com/pelicon/dr/pkg/namespacecrstatusupdater"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type DRManager interface {
	AddNamespaceWorker(drv1alpha1.Namespace)
}

type BaseDrManager struct {
	ctx                 context.Context
	nsCRName            string
	nsCRNamespace       string
	conditionsCh        chan drv1alpha1.SyncedCondition
	k8sControllerClient k8sclient.Client
	transportAdapter    drv1alpha1.TransportAdapter
	collectorType       drv1alpha1.ResourceCollectorType
	NamespaceWorkers    map[drv1alpha1.Namespace]drv1alpha1.NamesapceWorker
	statusUpdater       *namespacecrstatusupdater.StatusUpdater
}

func NewDRManager(
	ctx context.Context,
	nsCRName string,
	nsCRNamespace string,
	k8sControllerClient k8sclient.Client,
	transportAdapter drv1alpha1.TransportAdapter,
	collectorType drv1alpha1.ResourceCollectorType,
	statusUpdater *namespacecrstatusupdater.StatusUpdater,
) DRManager {
	return &BaseDrManager{
		ctx:                 ctx,
		nsCRName:            nsCRName,
		nsCRNamespace:       nsCRNamespace,
		k8sControllerClient: k8sControllerClient,
		conditionsCh:        make(chan drv1alpha1.SyncedCondition, 1),
		transportAdapter:    transportAdapter,
		collectorType:       collectorType,
		NamespaceWorkers:    make(map[drv1alpha1.Namespace]drv1alpha1.NamesapceWorker),
		statusUpdater:       statusUpdater,
	}
}

func (bdm *BaseDrManager) AddNamespaceWorker(namespace drv1alpha1.Namespace) {
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
