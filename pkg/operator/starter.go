package operator

import (
	"context"
	"fmt"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/ghodss/yaml"

	opv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	shareclientv1alpha1 "github.com/openshift/client-go/sharedresource/clientset/versioned"
	shareinformer "github.com/openshift/client-go/sharedresource/informers/externalversions"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/csi/csicontrollerset"
	"github.com/openshift/library-go/pkg/operator/csi/csidrivernodeservicecontroller"
	goc "github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/v1helpers"

	"github.com/openshift/shared-resources-operator/pkg/generated"
	"github.com/openshift/shared-resources-operator/pkg/metrics"
)

const (
	// Operand and operator run in the same namespace
	defaultNamespace = "openshift-cluster-csi-drivers"
	operatorName     = "csi-driver-shared-resource-operator"
	operandName      = "csi-driver-shared-resource"

	defaultResyncDuration = 20 * time.Minute
)

func RunOperator(ctx context.Context, controllerConfig *controllercmd.ControllerContext) error {
	// Create core clientset and informers
	kubeClient := kubeclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, operatorName))
	kubeInformersForNamespaces := v1helpers.NewKubeInformersForNamespaces(kubeClient, defaultNamespace, "")

	// Create config clientset and informer. This is used to get the cluster ID
	configClient := configclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, operatorName))
	configInformers := configinformers.NewSharedInformerFactory(configClient, defaultResyncDuration)

	shareClient := shareclientv1alpha1.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, operatorName))
	shareInformersFactory := shareinformer.NewSharedInformerFactory(shareClient, defaultResyncDuration)

	sharedSecretsLister := shareInformersFactory.Sharedresource().V1alpha1().SharedSecrets().Lister()
	sharedConfigMapsLister := shareInformersFactory.Sharedresource().V1alpha1().SharedConfigMaps().Lister()
	if err := metrics.InitializeShareCollector(sharedSecretsLister, sharedConfigMapsLister); err != nil {
		return err
	}

	// Create apiextensions clientset for creating the CRDs
	apiextensionsClient := apiextensionsclient.NewForConfigOrDie(controllerConfig.KubeConfig)

	// Create GenericOperatorclient. This is used by the library-go controllers created down below
	gvr := opv1.SchemeGroupVersion.WithResource("clustercsidrivers")
	operatorClient, dynamicInformers, err := goc.NewClusterScopedOperatorClientWithConfigName(controllerConfig.KubeConfig, gvr, string(opv1.SharedResourcesCSIDriver))
	if err != nil {
		return err
	}

	klog.V(5).Info("Generating dynamicClient")
	dynamicClient, err := dynamic.NewForConfig(controllerConfig.KubeConfig)
	if err != nil {
		return err
	}

	crds := []string{"0000_10_sharedsecret.crd.yaml", "0000_10_sharedconfigmap.crd.yaml"}
	for _, crd := range crds {
		data, err := generated.Asset(crd)
		if err != nil {
			return fmt.Errorf("error occurred reading file %q: %s", crd, err)
		}

		var customResourceDefinition apiextensionsv1.CustomResourceDefinition
		if err := yaml.Unmarshal(data, &customResourceDefinition); err != nil {
			return fmt.Errorf("error occurred unmarshalling file %q into object: %s", crd, err)
		}

		foundItems, err := apiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", customResourceDefinition.Name),
		})
		if err != nil {
			return fmt.Errorf("unexpected error occurred listing CustomResourceDefinitions: %s", err)
		}

		if len(foundItems.Items) > 0 {
			klog.Infof("CustomResourceDefinition %q already exists, skipping creation.", customResourceDefinition.Name)
		} else {

			if _, err := apiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, &customResourceDefinition, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("error occurred creating CustomResourceDefinition: %s", err)
			}
			klog.Infof("Successfully created CustomResourceDefinition %q.", customResourceDefinition.Name)
		}
	}

	csiControllerSet := csicontrollerset.NewCSIControllerSet(
		operatorClient,
		controllerConfig.EventRecorder,
	).WithLogLevelController().WithManagementStateController(
		operandName,
		false,
	).WithStaticResourcesController(
		"SharedResourcesDriverStaticResourcesController",
		kubeClient,
		dynamicClient,
		kubeInformersForNamespaces,
		generated.Asset,
		[]string{
			"csidriver.yaml",
			"node_sa.yaml",
			"service.yaml",
			"metrics_service.yaml",
			"servicemonitor.yaml",
			"rbac/privileged_role.yaml",
			"rbac/node_role.yaml",
			"rbac/node_privileged_binding.yaml",
			"rbac/node_binding.yaml",
			"rbac/prometheus_role.yaml",
			"rbac/prometheus_rolebinding.yaml",
		},
	).WithCSIConfigObserverController(
		"SharedResourcesDriverCSIConfigObserverController",
		configInformers,
	).WithCSIDriverNodeService(
		"SharedResourcesDriverNodeServiceController",
		generated.Asset,
		"node.yaml",
		kubeClient,
		kubeInformersForNamespaces.InformersFor(defaultNamespace),
		nil, // Node doesn't need to react to any changes
		csidrivernodeservicecontroller.WithObservedProxyDaemonSetHook(),
	)

	if err != nil {
		return err
	}

	klog.Info("Starting the informers")
	go kubeInformersForNamespaces.Start(ctx.Done())
	go dynamicInformers.Start(ctx.Done())
	go configInformers.Start(ctx.Done())
	go shareInformersFactory.Start(ctx.Done())

	klog.Info("Starting controllerset")
	go csiControllerSet.Run(ctx, 1)

	klog.Info("Starting metrics collection")

	klog.Info("Starting metrics endpoint")
	server := metrics.BuildServer(metrics.MetricsPort)
	go metrics.RunServer(server, ctx.Done())

	<-ctx.Done()

	return fmt.Errorf("stopped")
}
