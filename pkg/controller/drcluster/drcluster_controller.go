package drcluster

import (
	"context"
	//	"reflect"

	drv1alpha1 "github.com/pelicon/dr/pkg/apis/dr/v1alpha1"
	configs "github.com/pelicon/dr/pkg/configs"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	logger = log.WithField("module", "DRClusterController")
)

// Add creates a new DRCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	ctx := context.TODO()
	return &ReconcileDRCluster{
		ctx:    ctx,
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("drcluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource DRCluster
	err = c.Watch(&source.Kind{Type: &drv1alpha1.DRCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDRCluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDRCluster{}

// ReconcileDRCluster reconciles a DRCluster object
type ReconcileDRCluster struct {
	ctx context.Context
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a DRCluster object and makes changes based on the state read
func (r *ReconcileDRCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger.WithField("Request.Namespace", request.Namespace).WithField("Request.Name", request.Name).Info("Reconcile DRCluster")

	drClusterInstance := &drv1alpha1.DRCluster{}
	err := r.client.Get(r.ctx, request.NamespacedName, drClusterInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// errClear := clearRelativeDRNamespaces(r.client, r.ctx, drClusterInstance.GetClusterName())
			// if errClear != nil {
			// 	logger.Errorf("Err clearing relative drNamespaces, clusterName: %v", drClusterInstance.GetClusterName())
			// 	return reconcile.Result{Requeue: true}, errClear
			// }
			// logger.Debugf("Cleared relative drNamespaces, clusterName: %v", drClusterInstance.GetClusterName())

			errDeactive := deActiveRelativeDRNamespaces(r.client, r.ctx, drClusterInstance.GetClusterName())
			if errDeactive != nil {
				logger.Errorf("Err deactiving relative drNamespaces, clusterName: %v", drClusterInstance.GetClusterName())
				return reconcile.Result{Requeue: true}, errDeactive
			}
			logger.Debugf("Deactived relative drNamespaces, clusterName: %v", drClusterInstance.GetClusterName())

			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// `Role` is immutable
	if len(drClusterInstance.Status.Role) == 0 && len(drClusterInstance.Spec.Role) != 0 {
		drClusterInstance.Status.Role = drClusterInstance.Spec.Role
	}

	// sync Active to namesapce CRDs
	if drClusterInstance.Spec.Active != drClusterInstance.Status.Active {
		drNamespaceListInstance := &drv1alpha1.DRNamespaceList{}
		err := r.client.List(r.ctx, drNamespaceListInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, err
		}

		for _, drNamespaceInstance := range drNamespaceListInstance.Items {
			if string(drNamespaceInstance.Spec.DRPairClusterName) != drClusterInstance.Name {
				continue
			}
			newInstance := drNamespaceInstance.DeepCopy()
			newInstance.Spec.Active = drClusterInstance.Spec.Active

			if err := r.client.Update(r.ctx, newInstance); err != nil {
				return reconcile.Result{}, err
			}
		}
		drClusterInstance.Status.Active = drClusterInstance.Spec.Active
	}

	// sync TransportAdapter to namesapce CRDs
	if drClusterInstance.Spec.TransportAdapter != drClusterInstance.Status.TransportAdapter {
		drNamespaceListInstance := &drv1alpha1.DRNamespaceList{}
		err := r.client.List(r.ctx, drNamespaceListInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			return reconcile.Result{Requeue: true}, err
		}
		for _, drNamespaceInstance := range drNamespaceListInstance.Items {
			newInstance := drNamespaceInstance.DeepCopy()
			newInstance.Spec.TransportAdapter = drClusterInstance.Spec.TransportAdapter
			if err := r.client.Update(r.ctx, newInstance); err != nil {
				return reconcile.Result{}, err
			}
		}
		drClusterInstance.Status.TransportAdapter = drClusterInstance.Spec.TransportAdapter
	}

	// sync CollectorType to namesapce CRDs
	if drClusterInstance.Spec.CollectorType != drClusterInstance.Status.CollectorType {
		drNamespaceListInstance := &drv1alpha1.DRNamespaceList{}
		err := r.client.List(r.ctx, drNamespaceListInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, err
		}
		for _, drNamespaceInstance := range drNamespaceListInstance.Items {
			newInstance := drNamespaceInstance.DeepCopy()
			newInstance.Spec.CollectorType = drClusterInstance.Spec.CollectorType
			if err := r.client.Update(r.ctx, newInstance); err != nil {
				return reconcile.Result{}, err
			}
		}
		drClusterInstance.Status.CollectorType = drClusterInstance.Spec.CollectorType
	}

	// update cluster config
	savedClusterConfig := configs.GetConfigContainer().GetClusterConfigs()
	pairClusterSettings := &drClusterInstance.Spec.PairClusterSettings
	clusterName := drv1alpha1.ClusterName(drClusterInstance.Name)
	//	if !reflect.DeepEqual(savedClusterConfig[clusterName], pairClusterSettings) {
	savedClusterConfig[clusterName] = pairClusterSettings.DeepCopy()
	configs.GetConfigContainer().UpdateClusterConfigToContainer(clusterName, pairClusterSettings)
	//	}

	// update status
	if err := r.client.Status().Update(r.ctx, drClusterInstance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func clearRelativeDRNamespaces(cli client.Client, ctx context.Context, clusterName string) error {
	drNamespaceListInstance := &drv1alpha1.DRNamespaceList{}
	errList := cli.List(ctx, drNamespaceListInstance)
	if errList != nil {
		if errors.IsNotFound(errList) {
			logger.Debugf("No relative drNamespaces, clusterName: %v", clusterName)
			return nil
		}
		return errList
	}
	for _, drNamespaceInstance := range drNamespaceListInstance.Items {
		if drNamespaceInstance.GetClusterName() == clusterName {
			errDelete := cli.Delete(ctx, &drNamespaceInstance)
			if errDelete != nil {
				logger.Errorf("Err deleting relative drNamespace, clusterName: %v, drNamespace: %+v", drNamespaceInstance)
				return errDelete
			}
			logger.Debugf("Deleted relative drNamespace, clusterName: %v, drNamespace: %+v", drNamespaceInstance)
		}
	}
	return nil
}

func deActiveRelativeDRNamespaces(cli client.Client, ctx context.Context, clusterName string) error {
	drNamespaceListInstance := &drv1alpha1.DRNamespaceList{}
	errList := cli.List(ctx, drNamespaceListInstance)
	if errList != nil {
		if errors.IsNotFound(errList) {
			logger.Debugf("No relative drNamespaces, clusterName: %v", clusterName)
			return nil
		}
		return errList
	}
	for _, drNamespaceInstance := range drNamespaceListInstance.Items {
		if drNamespaceInstance.GetClusterName() == clusterName {
			// errDelete := cli.Delete(ctx, &drNamespaceInstance)
			// if errDelete != nil {
			// 	logger.Errorf("Err deleting relative drNamespace, clusterName: %v, drNamespace: %+v", drNamespaceInstance)
			// 	return errDelete
			// }
			// logger.Debugf("Deleted relative drNamespace, clusterName: %v, drNamespace: %+v", drNamespaceInstance)
			newInstance := drNamespaceInstance.DeepCopy()
			newInstance.Spec.Active = false

			if errUpdate := cli.Update(ctx, newInstance); errUpdate != nil {
				logger.Errorf("Err deactiving relative drNamespaces, clusterName: %v, drNamespace: %+v", clusterName, newInstance)
				return errUpdate
			}
			logger.Debugf("Deactived relative drNamespace, clusterName: %v, drNamespace: %+v", clusterName, newInstance)
		}
	}
	return nil
}
