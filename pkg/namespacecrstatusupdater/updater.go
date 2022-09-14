package namespacecrstatusupdater

import (
	"context"
	"strings"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sworkqueue "k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = log.WithField("module", "StatusUpdater")
)

type StatusUpdater struct {
	ctx    context.Context
	client client.Client
	queue  k8sworkqueue.Interface
}

type statusToUpdate struct {
	drNamespaceCRNamespace, drNamespaceCRName string
	status                                    *drv1alpha1.DRNamespaceStatus
}

func NewStatusUpdater(ctx context.Context, client client.Client) *StatusUpdater {
	return &StatusUpdater{
		ctx:    ctx,
		client: client,
		queue:  k8sworkqueue.New(),
	}
}

func (cu *StatusUpdater) UpdateCondition(drNamespaceCRNamespace, drNamespaceCRName string, conditionNeedUpdate drv1alpha1.SyncedCondition) {
	name, namespace := drNamespaceCRName, drNamespaceCRNamespace
	objectKey := k8sruntimeclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	nsInstance := &drv1alpha1.DRNamespace{}
	if err := cu.client.Get(cu.ctx, objectKey, nsInstance); err != nil {
		logger.WithError(err).Error("Failed to get DRNS instance")
		return
	}

	status := &nsInstance.Status
	mapKey := GetConditionName(conditionNeedUpdate.GroupVersionKindObject.Kind, conditionNeedUpdate.GroupVersionKindObject.Namespace, conditionNeedUpdate.GroupVersionKindObject.Name)

	logger.Debugf("condition to update %s(namespace: %s name: %s) to drnamespace namespace: %s name: %s, version: %s, cu.lastSyncedStatus: %s",
		conditionNeedUpdate.GroupVersionKindObject.Kind,
		conditionNeedUpdate.GroupVersionKindObject.Namespace,
		conditionNeedUpdate.GroupVersionKindObject.Name,
		namespace,
		name,
		conditionNeedUpdate.LastSyncedResourceVersion,
		conditionNeedUpdate.LastSyncedStatus)

	if status.SyncedConditions == nil {
		status.SyncedConditions = make(map[string]drv1alpha1.SyncedCondition)
	}
	status.SyncedConditions[mapKey] = conditionNeedUpdate
	cu.UpdateStatus(drNamespaceCRNamespace, drNamespaceCRName, status)
}

func (cu *StatusUpdater) UpdateStatus(drNamespaceCRNamespace, drNamespaceCRName string, status *drv1alpha1.DRNamespaceStatus) {
	statusToUpdateInstance := &statusToUpdate{
		drNamespaceCRNamespace: drNamespaceCRNamespace,
		drNamespaceCRName:      drNamespaceCRName,
		status:                 status,
	}

	cu.queue.Add(statusToUpdateInstance)
}

func (cu *StatusUpdater) Run() {
	go wait.UntilWithContext(cu.ctx, cu.run, 0)
}

// CR conditions update only here to avoid race conditions
func (cu *StatusUpdater) run(ctx context.Context) {
	// STEP1: Dequeue condition
	statusNeedUpdateUntyped, shutdown := cu.queue.Get()
	if shutdown {
		logger.Fatal("Queue closed")
	}
	defer cu.queue.Done(statusNeedUpdateUntyped)

	statusWithCRInfo := statusNeedUpdateUntyped.(*statusToUpdate)

	// STEP2: Get dr instance
	name, namespace, statusNeedUpdate := statusWithCRInfo.drNamespaceCRName, statusWithCRInfo.drNamespaceCRNamespace, statusWithCRInfo.status
	logger.Debugf("start update drnamespace (namespace: %s name: %s)",
		namespace,
		name,
	)
	objectKey := k8sruntimeclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	nsInstance := &drv1alpha1.DRNamespace{}
	if err := cu.client.Get(cu.ctx, objectKey, nsInstance); err != nil {
		logger.WithError(err).Error("Failed to get DRNS instance")
		return
	}

	nsInstance.Status = *statusNeedUpdate

	// STEP3: update dr namespace CR status
	if err := cu.client.Status().Update(cu.ctx, nsInstance); err != nil {
		logger.WithError(err).Errorf("failed to update resource condition %+v, on %s-%s", statusNeedUpdate, namespace, name)
		return
	}
}

func GetConditionName(pieces ...string) string {
	return strings.Join(pieces, "-")
}
