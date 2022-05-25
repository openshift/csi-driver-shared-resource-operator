package deploymentcontroller

import (
	"bytes"
	"os"

	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/csi-driver-shared-resource-operator/assets"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/deploymentcontroller"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultNamespace                    = "openshift-cluster-csi-drivers"
	envSharedResourceDriverWebhookImage = "WEBHOOK_IMAGE"
)

func NewWebHookDeploymentController(kubeClient kubernetes.Interface,
	operatorClient v1helpers.OperatorClientWithFinalizers,
	kubeInformersForNamespaces v1helpers.KubeInformersForNamespaces,
	recorder events.Recorder) factory.Controller {

	return deploymentcontroller.NewDeploymentController(
		"SharedResourceCSIDriverWebhookController",
		assets.MustAsset("webhook/deployment.yaml"),
		recorder,
		operatorClient,
		kubeClient,
		kubeInformersForNamespaces.InformersFor(defaultNamespace).Apps().V1().Deployments(),
		nil, // optionalInformers
		[]deploymentcontroller.ManifestHookFunc{
			replaceAll("${WEBHOOK_IMAGE}", os.Getenv(envSharedResourceDriverWebhookImage)),
		},
	)
}

func replaceAll(old, new string) deploymentcontroller.ManifestHookFunc {
	return func(spec *operatorv1.OperatorSpec, manifest []byte) ([]byte, error) {
		return bytes.ReplaceAll(manifest, []byte(old), []byte(new)), nil
	}
}
