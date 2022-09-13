package kubeapiserver

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	udsdrv1alpha1 "github.com/DaoCloud/udsdr/pkg/apis/udsdr/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	// "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logger = log.WithField("module", "transportmanager-kubeapiserver")
)

type kubeApiserverTransporter struct {
	*sync.Mutex
	ctx                        context.Context
	K8sControllerClient        k8sclient.Client
	NamespacedStatusUpdateFunc udsdrv1alpha1.NamespacedStatusUpdateFunc
	PairClusterSettings        *udsdrv1alpha1.PairClusterSettings
	ResourceVersionCache       map[types.UID]string
}

func New(ctx context.Context, k8sControllerClient k8sclient.Client, namespacedStatusUpdateFunc udsdrv1alpha1.NamespacedStatusUpdateFunc) *kubeApiserverTransporter {
	return &kubeApiserverTransporter{
		Mutex:                      &sync.Mutex{},
		ctx:                        ctx,
		K8sControllerClient:        k8sControllerClient,
		NamespacedStatusUpdateFunc: namespacedStatusUpdateFunc,
		ResourceVersionCache:       make(map[types.UID]string),
	}
}

func (kat *kubeApiserverTransporter) SetConfig(cConfigs *udsdrv1alpha1.PairClusterSettings) {
	kat.Lock()
	defer kat.Unlock()

	kat.PairClusterSettings = cConfigs
}

func (kat *kubeApiserverTransporter) Transport(obj *udsdrv1alpha1.ObjResource) error {
	if kat.PairClusterSettings == nil {
		return fmt.Errorf("PairClusterSettings not set")
	}
	kubeApiserverTransportorSetting := kat.PairClusterSettings.KubeApiserverTransportorSetting
	// certData, keyData, caData, err := readCertsFromPath()
	// if err != nil {
	// 	return err
	// }
	// logger.Infof("certData: %v, keyData: %v, caData: %v\n", certData, keyData, caData)
	// clientCfg := rest.Config{
	// 	// Host: kubeApiserverTransportorSetting.KubeApiServerHost,
	// 	Host: "10.6.234.6:11081",
	// 	TLSClientConfig: rest.TLSClientConfig{
	// 		CertData: certData,
	// 		KeyData:  keyData,
	// 		CAData:   caData,
	// 		// CertFile: "/etc/daocloud/dce/certs/kube-admin.crt",
	// 		// KeyFile:  "/etc/daocloud/dce/certs/kube-admin.key",
	// 		// CAFile:   "/etc/daocloud/dce/certs/ca.crt",
	// 	},
	// 	QPS:   20,
	// 	Burst: 30,
	// }
	clientCfg := rest.Config{
		Host: kubeApiserverTransportorSetting.KubeApiServerHost,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
			// CAData:   []byte(kubeApiserverTransportorSetting.CertData),
			KeyData:  []byte(kubeApiserverTransportorSetting.KeyData),
			CertData: []byte(kubeApiserverTransportorSetting.CertData),
		},
		QPS:   float32(kubeApiserverTransportorSetting.QPS),
		Burst: kubeApiserverTransportorSetting.Burst,
	}
	client, err := dynamic.NewForConfig(&clientCfg)
	if err != nil {
		return err
	}
	// unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "resourceVersion")
	// unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "annotations", "pv.kubernetes.io/bind-completed")
	// fmt.Printf("Going transporting %v+\n", obj.Unstructured)
	switch obj.Action {
	case udsdrv1alpha1.ObjectActionCreate:
		obj = prepareTransportWhenCreating(obj)
		logger.Debugf("Going transporting %+v", obj.Unstructured)
		_, err = client.Resource(*obj.GVR).
			Namespace(obj.Unstructured.GetNamespace()).
			Create(obj.Unstructured, v1.CreateOptions{})
		if err != nil {
			logger.Errorf("Transport err: %v, obj: %+v", err, obj.Unstructured)
			return err
		}
		logger.Debugf("Successfully transported %+v", obj.Unstructured)
	case udsdrv1alpha1.ObjectActionUpdate:
		logger.Debugf("Going updating %+v", obj.Unstructured)
		originalUnstructured, getErr := client.Resource(*obj.GVR).Namespace(obj.Unstructured.GetNamespace()).Get(obj.Unstructured.GetName(), v1.GetOptions{})
		if errors.IsNotFound(getErr) {
			obj = prepareTransportWhenCreating(obj)
			logger.Debugf("Resource not exist, create resource instead of updating %+v", obj.Unstructured)
			// unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "resourceVersion")
			_, err = client.Resource(*obj.GVR).
				Namespace(obj.Unstructured.GetNamespace()).
				Create(obj.Unstructured, v1.CreateOptions{})
			if err != nil {
				logger.Errorf("Create instead of updating err: %v, obj: %+v", err, obj.Unstructured)
				return err
			}
			logger.Debugf("Successfully created instead of updating %+v", obj.Unstructured)
			return nil
		}
		// _, err = client.Resource(*obj.GVR).Namespace(obj.Unstructured.GetNamespace()).Update(obj.Unstructured, v1.UpdateOptions{})
		obj = prepareTransportWhenUpdating(obj, originalUnstructured)
		patch := k8sclient.MergeFrom(originalUnstructured)
		// unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "resourceVersion")
		patchData, errPatchData := patch.Data(obj.Unstructured)
		if errPatchData != nil {
			logger.Errorf("patch.Data() err: %v", errPatchData)
			return err
		}
		logger.Debugf("patchData: %+v", patchData)
		logger.Debugf("patchData: %s", patchData)
		_, err := client.Resource(*obj.GVR).Namespace(obj.Unstructured.GetNamespace()).Patch(obj.Unstructured.GetName(), patch.Type(), patchData, v1.PatchOptions{})
		if err != nil {
			logger.Errorf("Update err: %v, obj: %+v", err, obj.Unstructured)
			return err
		}
		logger.Debugf("Successfully updated %+v", obj.Unstructured)
	case udsdrv1alpha1.ObjectActionDelete:
		logger.Debugf("Going deleting %+v", obj.Unstructured)
		err = client.Resource(*obj.GVR).Namespace(obj.Unstructured.GetNamespace()).Delete(obj.Unstructured.GetName(), &v1.DeleteOptions{})
		if err != nil {
			logger.Errorf("Delete err: %v, obj: %+v", err, obj.Unstructured)
			return err
		}
		logger.Debugf("Successfully deleted %+v", obj.Unstructured)
	}
	return nil
}

func prepareTransportWhenCreating(obj *udsdrv1alpha1.ObjResource) *udsdrv1alpha1.ObjResource {
	unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "uid")

	if *obj.GVR == udsdrv1alpha1.PersistentVolumeGVR {
		unstructured.RemoveNestedField(obj.Unstructured.Object, "spec", "claimRef", "resourceVersion")
		unstructured.RemoveNestedField(obj.Unstructured.Object, "spec", "claimRef", "uid")
		unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "annotations", "pv.kubernetes.io/bind-completed")
	}

	if *obj.GVR == udsdrv1alpha1.PersistentVolumeClaimGVR {
		unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "annotations", "pv.kubernetes.io/bind-completed")
		unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "annotations", "pv.kubernetes.io/bound-by-controller")
	}
	return obj
}

func prepareTransportWhenUpdating(obj *udsdrv1alpha1.ObjResource, originalUnstructured *unstructured.Unstructured) *udsdrv1alpha1.ObjResource {
	originalResourceVersion := originalUnstructured.GetResourceVersion()
	obj.Unstructured.SetResourceVersion(originalResourceVersion)
	originalUID := originalUnstructured.GetUID()
	obj.Unstructured.SetUID(originalUID)

	if *obj.GVR == udsdrv1alpha1.PersistentVolumeGVR {
		originalClaimRefResourceVersion, _, _ := unstructured.NestedString(originalUnstructured.Object, "spec", "claimRef", "resourceVersion")
		unstructured.SetNestedField(obj.Unstructured.Object, originalClaimRefResourceVersion, "spec", "claimRef", "resourceVersion")
		originalClaimRefUID, _, _ := unstructured.NestedString(originalUnstructured.Object, "spec", "claimRef", "uid")
		unstructured.SetNestedField(obj.Unstructured.Object, originalClaimRefUID, "spec", "claimRef", "uid")
	}
	return obj
}

func prepareTransport(obj *udsdrv1alpha1.ObjResource, originalUnstructured *unstructured.Unstructured) {
	switch obj.Action {
	case udsdrv1alpha1.ObjectActionCreate:
		unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "resourceVersion")
		unstructured.RemoveNestedField(obj.Unstructured.Object, "metadata", "uid")
	case udsdrv1alpha1.ObjectActionUpdate:
		originalResourceVersion := originalUnstructured.GetResourceVersion()
		obj.Unstructured.SetResourceVersion(originalResourceVersion)
		originalUID := originalUnstructured.GetUID()
		obj.Unstructured.SetUID(originalUID)
	}

	if *obj.GVR == udsdrv1alpha1.PersistentVolumeGVR {
		switch obj.Action {
		case udsdrv1alpha1.ObjectActionCreate:
			unstructured.RemoveNestedField(obj.Unstructured.Object, "spec", "claimRef", "resourceVersion")
			unstructured.RemoveNestedField(obj.Unstructured.Object, "spec", "claimRef", "uid")
		case udsdrv1alpha1.ObjectActionUpdate:
			originalClaimRefResourceVersion, _, _ := unstructured.NestedString(originalUnstructured.Object, "spec", "claimRef", "resourceVersion")
			unstructured.SetNestedField(obj.Unstructured.Object, originalClaimRefResourceVersion, "spec", "claimRef", "resourceVersion")
			originalClaimRefUID, _, _ := unstructured.NestedString(originalUnstructured.Object, "spec", "claimRef", "uid")
			unstructured.SetNestedField(obj.Unstructured.Object, originalClaimRefUID, "spec", "claimRef", "uid")
		}
	}
}

func readCertsFromPath() (certData, keyData, caData []byte, err error) {
	certFile, err := os.Open("/etc/daocloud/dce/certs/kube-admin.crt")
	if err != nil {
		return certData, keyData, caData, err
	}
	defer certFile.Close()
	certData, err = ioutil.ReadAll(certFile)
	if err != nil {
		return certData, keyData, caData, err
	}

	keyFile, err := os.Open("/etc/daocloud/dce/certs/kube-admin.key")
	if err != nil {
		return certData, keyData, caData, err
	}
	defer keyFile.Close()
	keyData, err = ioutil.ReadAll(keyFile)
	if err != nil {
		return certData, keyData, caData, err
	}

	caFile, err := os.Open("/etc/daocloud/dce/certs/ca.crt")
	if err != nil {
		return certData, keyData, caData, err
	}
	defer caFile.Close()
	caData, err = ioutil.ReadAll(caFile)
	if err != nil {
		return certData, keyData, caData, err
	}

	return certData, keyData, caData, nil
}
