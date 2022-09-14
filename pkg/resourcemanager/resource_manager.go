package resourcemanager

import (
	"context"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	"github.com/pelicon/dr/pkg/resourcemanager/listwatch"
	"github.com/pelicon/dr/pkg/resourcemanager/periodic"
	log "github.com/sirupsen/logrus"
)

var (
	logger = log.WithField("module", "transportmanager-kubeapiserver")
)

type ResourceManager interface {
	GetResourceChan() <-chan *drv1alpha1.ObjResource
	Run()
}

type Collector interface {
	// liner start do not use goroutine
	Start(context.Context)
}

type BaseResourceManager struct {
	ctx           context.Context
	CollectorType drv1alpha1.ResourceCollectorType
	Collector     Collector
	ResourceChan  chan *drv1alpha1.ObjResource
}

func NewBaseResourceManager(ctx context.Context, collectorType drv1alpha1.ResourceCollectorType) ResourceManager {
	resourceChan := make(chan *drv1alpha1.ObjResource)
	return &BaseResourceManager{
		ctx:           ctx,
		CollectorType: collectorType,
		Collector:     newCollector(collectorType, resourceChan),
		ResourceChan:  resourceChan,
	}
}

func (brm *BaseResourceManager) Run() {
	logger.Debugf("base resource manager to run")
	brm.run()

	<-brm.ctx.Done()
	close(brm.ResourceChan)
}

func (brm *BaseResourceManager) run() {
	go brm.Collector.Start(brm.ctx)
}

func (brm *BaseResourceManager) GetResourceChan() <-chan *drv1alpha1.ObjResource {
	return brm.ResourceChan
}

func newCollector(collectorType drv1alpha1.ResourceCollectorType, resourceChan chan *drv1alpha1.ObjResource) Collector {
	switch collectorType {
	case drv1alpha1.ResourceCollectorTypeListWatch:
		return listwatch.New(resourceChan)
	case drv1alpha1.ResourceCollectorTypePeriodic:
		return periodic.New()
	}
	//todo do not return nil
	return nil
}
