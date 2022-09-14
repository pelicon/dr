package drnamespace

import (
	"context"
	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	"github.com/pelicon/dr/pkg/configs"
	"github.com/pelicon/dr/pkg/drmanager"
	"github.com/pelicon/dr/pkg/namespacecrstatusupdater"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	k8sworkqueue "k8s.io/client-go/util/workqueue"
	//"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	previousNamespaceCR map[string]*drv1alpha1.DRNamespace
	logger              = log.WithField("module", "DRNamespaceController")
)

// Add creates a new DRNamespace Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	ctx := context.TODO()
	// cmWatcher, err := configs.NewBaseConfigWatcher(ctx, 0)
	// if err != nil {
	// 	log.Error(err, "failed to start cm watcher")
	// 	return nil
	// }
	// cmWatcher.Run()
	previousNamespaceCR = make(map[string]*drv1alpha1.DRNamespace)
	cu := namespacecrstatusupdater.NewStatusUpdater(ctx, mgr.GetClient())
	cu.Run()

	return &ReconcileDRNamespace{
		ctx:                      ctx,
		client:                   mgr.GetClient(),
		scheme:                   mgr.GetScheme(),
		drNamespaces:             make(map[drv1alpha1.Namespace]drmanager.DRManager),
		drNamespaceCtxCancelFunc: make(map[drv1alpha1.Namespace]context.CancelFunc),
		statusUpdater:            cu,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("drnamespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DRNamespace
	err = c.Watch(&source.Kind{Type: &drv1alpha1.DRNamespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDRNamespace implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDRNamespace{}

// ReconcileDRNamespace reconciles a DRNamespace object
type ReconcileDRNamespace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	ctx                      context.Context
	client                   client.Client
	scheme                   *runtime.Scheme
	drNamespaces             map[drv1alpha1.Namespace]drmanager.DRManager
	drNamespaceCtxCancelFunc map[drv1alpha1.Namespace]context.CancelFunc
	queue                    k8sworkqueue.Interface
	statusUpdater            *namespacecrstatusupdater.StatusUpdater
}

// Reconcile reads that state of the cluster for a DRNamespace object and makes changes based on the state read
func (r *ReconcileDRNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger.WithField("Request.Namespace", request.Namespace).WithField("Request.Name", request.Name).Info("Reconcile DRNamespace")

	// Fetch the DRNamespace instance
	drNamespaceInstance := &drv1alpha1.DRNamespace{}
	err := r.client.Get(r.ctx, request.NamespacedName, drNamespaceInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue

			if ctxCancel, exists := r.drNamespaceCtxCancelFunc[drv1alpha1.Namespace(request.Namespace)]; exists {
				ctxCancel()
			}
			delete(r.drNamespaces, drv1alpha1.Namespace(request.Namespace))
			delete(r.drNamespaceCtxCancelFunc, drv1alpha1.Namespace(request.Namespace))
			logger.Debugf("drNamespaceInstance not found, done clearing, namespace: %v", request.Namespace)

			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//oldDRNamespaceInstance := drNamespaceInstance.DeepCopy()

	// `Role` is immutable
	var restartRequire bool
	if len(drNamespaceInstance.Status.Role) == 0 && len(drNamespaceInstance.Spec.Role) != 0 {
		drNamespaceInstance.Status.Role = drNamespaceInstance.Spec.Role
	}

	// sync CollectorType
	if drNamespaceInstance.Status.CollectorType != drNamespaceInstance.Spec.CollectorType {
		drNamespaceInstance.Status.CollectorType = drNamespaceInstance.Spec.CollectorType
		restartRequire = true
	}

	// sync TransportAdapter
	if drNamespaceInstance.Status.TransportAdapter != drNamespaceInstance.Spec.TransportAdapter {
		drNamespaceInstance.Status.TransportAdapter = drNamespaceInstance.Spec.TransportAdapter
		restartRequire = true
	}

	// sync DRPairClusterSettingsObjs
	if _, exists := previousNamespaceCR[drNamespaceInstance.Namespace]; !exists { //TODO || !reflect.DeepEqual(previousNamespaceCR.Spec.DRFilterConfig, drNamespaceInstance.Spec.DRFilterConfig) {
		if _, exists := previousNamespaceCR[drNamespaceInstance.Namespace]; !exists {
			logger.Debugf("previousNamespaceCR: nil")
		} else {
			logger.Debugf("previousNamespaceCR.Spec.DRFilterConfig: %+v", previousNamespaceCR[drNamespaceInstance.Namespace].Spec.DRFilterConfig)
		}
		logger.Debugf("drNamespaceInstance.Spec.DRFilterConfig: %+v", drNamespaceInstance.Spec.DRFilterConfig)
		logger.Debugf("going to update filter config and restart")
		configs.GetConfigContainer().UpdateFilterConfigToContainer(drv1alpha1.Namespace(drNamespaceInstance.Namespace), &drNamespaceInstance.Spec.DRFilterConfig)
		restartRequire = true
	}

	namespace := drv1alpha1.Namespace(drNamespaceInstance.Namespace)
	if _, exists := previousNamespaceCR[drNamespaceInstance.Namespace]; !exists || drNamespaceInstance.Status.DRPairClusterName != drNamespaceInstance.Spec.DRPairClusterName {
		drNamespaceInstance.Status.DRPairClusterName = drNamespaceInstance.Spec.DRPairClusterName
		if err := configs.GetConfigContainer().UpdateNamespaceConfigedCluster(
			namespace,
			drNamespaceInstance.Status.DRPairClusterName,
		); err != nil {
			return reconcile.Result{}, err
		}
		restartRequire = true
	}

	// sync Active
	if _, exists := r.drNamespaceCtxCancelFunc[namespace]; !exists || drNamespaceInstance.Spec.Active != drNamespaceInstance.Status.Active {
		drNamespaceInstance.Status.Active = drNamespaceInstance.Spec.Active
		if restartRequire || !drNamespaceInstance.Status.Active {
			if ctxCancel, exists := r.drNamespaceCtxCancelFunc[namespace]; exists {
				ctxCancel()
				delete(r.drNamespaces, namespace)
				delete(r.drNamespaceCtxCancelFunc, namespace)
				logger.Debugf("namespaceWorker deactivted")
			}
		}

		if drNamespaceInstance.Status.Active {
			// init new drmgr
			drCtx, drCancel := context.WithCancel(r.ctx)
			drmgr := drmanager.NewDRManager(
				drCtx,
				drNamespaceInstance.GetName(),
				drNamespaceInstance.GetNamespace(),
				r.client,
				drNamespaceInstance.Status.TransportAdapter,
				drNamespaceInstance.Status.CollectorType,
				r.statusUpdater,
			)

			drmgr.AddNamespaceWorker(namespace)
			r.drNamespaces[namespace] = drmgr
			r.drNamespaceCtxCancelFunc[namespace] = drCancel
			logger.Debugf("namespaceWorker activted")
		}
	}

	previousNamespaceCR[drNamespaceInstance.Namespace] = drNamespaceInstance
	// update status
	r.statusUpdater.UpdateStatus(drNamespaceInstance.Namespace, drNamespaceInstance.Name, drNamespaceInstance.Status.DeepCopy())
	return reconcile.Result{}, nil
}
