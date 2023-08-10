package deploymentcontroller

import (
	"bytes"
	"os"

	operatorv1 "github.com/openshift/api/operator/v1"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	"github.com/openshift/csi-driver-shared-resource-operator/assets"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/csi/csidrivercontrollerservicecontroller"
	"github.com/openshift/library-go/pkg/operator/deploymentcontroller"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultNamespace                    = "openshift-cluster-csi-drivers"
	envSharedResourceDriverWebhookImage = "WEBHOOK_IMAGE"
	infraConfigName                     = "cluster"
	webhookSecretName                   = "shared-resource-csi-driver-webhook-serving-cert"
)

func NewWebHookDeploymentController(kubeClient kubernetes.Interface,
	operatorClient v1helpers.OperatorClientWithFinalizers,
	kubeInformersForNamespaces v1helpers.KubeInformersForNamespaces,
	configInformer configinformers.SharedInformerFactory,
	recorder events.Recorder) factory.Controller {

	nodeLister := kubeInformersForNamespaces.InformersFor("").Core().V1().Nodes().Lister()
	secretInformer := kubeInformersForNamespaces.InformersFor(defaultNamespace).Core().V1().Secrets()

	return deploymentcontroller.NewDeploymentController(
		"SharedResourceCSIDriverWebhookController",
		assets.MustAsset("webhook/deployment.yaml"),
		recorder,
		operatorClient,
		kubeClient,
		kubeInformersForNamespaces.InformersFor(defaultNamespace).Apps().V1().Deployments(),
		[]factory.Informer{
			secretInformer.Informer(),
			configInformer.Config().V1().Infrastructures().Informer(),
		},
		[]deploymentcontroller.ManifestHookFunc{
			replaceAll("${WEBHOOK_IMAGE}", os.Getenv(envSharedResourceDriverWebhookImage)),
		},
		csidrivercontrollerservicecontroller.WithControlPlaneTopologyHook(configInformer),
		csidrivercontrollerservicecontroller.WithReplicasHook(nodeLister),
		csidrivercontrollerservicecontroller.WithSecretHashAnnotationHook(
			defaultNamespace,
			webhookSecretName,
			secretInformer,
		),
	)
}

func replaceAll(old, new string) deploymentcontroller.ManifestHookFunc {
	return func(spec *operatorv1.OperatorSpec, manifest []byte) ([]byte, error) {
		return bytes.ReplaceAll(manifest, []byte(old), []byte(new)), nil
	}
}
