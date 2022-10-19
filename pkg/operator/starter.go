package operator

import (
	"context"
	"fmt"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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

	"github.com/openshift/csi-driver-shared-resource-operator/assets"
	"github.com/openshift/csi-driver-shared-resource-operator/pkg/deploymentcontroller"
	"github.com/openshift/csi-driver-shared-resource-operator/pkg/metrics"
)

const (
	// Operand and operator run in the same namespace
	defaultNamespace    = "openshift-cluster-csi-drivers"
	operatorName        = "csi-driver-shared-resource-operator"
	operandName         = "csi-driver-shared-resource"
	skipValidationLabel = "csi.sharedresource.openshift.io/skip-validation"

	defaultResyncDuration = 20 * time.Minute
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(admissionv1.AddToScheme(scheme))
	utilruntime.Must(admissionregistrationv1.AddToScheme(scheme))
}

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

	err = ensureCRDSExist(ctx, apiextensionsClient)
	if err != nil {
		return err
	}
	crdTicker := time.NewTicker(10 * time.Minute)
	crdDone := make(chan bool)
	go func() {
		for {
			select {
			case <-crdDone:
				return
			case <-crdTicker.C:
				ensureCRDSExist(ctx, apiextensionsClient)
			}
		}
	}()

	err = ensureConfigurationConfigMapExists(ctx, kubeClient)
	if err != nil {
		return err
	}
	cmTicker := time.NewTicker(10 * time.Minute)
	cmDone := make(chan bool)
	go func() {
		for {
			select {
			case <-cmDone:
				return
			case <-cmTicker.C:
				ensureConfigurationConfigMapExists(ctx, kubeClient)
			}
		}
	}()

	setSkipValidationLabelForNamespace(ctx, kubeClient)

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
		assets.ReadFile,
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
			"webhook/sa.yaml",
			"webhook/configmap.yaml",
			"webhook/pdb.yaml",
			"webhook/service.yaml",
			"webhook/validating_webhook_configuration.yaml",
		},
	).WithCSIConfigObserverController(
		"SharedResourcesDriverCSIConfigObserverController",
		configInformers,
	).WithCSIDriverNodeService(
		"SharedResourcesDriverNodeServiceController",
		assets.ReadFile,
		"node.yaml",
		kubeClient,
		kubeInformersForNamespaces.InformersFor(defaultNamespace),
		nil, // Node doesn't need to react to any changes
		csidrivernodeservicecontroller.WithObservedProxyDaemonSetHook(),
	)

	webhookDeploymentController := deploymentcontroller.NewWebHookDeploymentController(
		kubeClient,
		operatorClient,
		kubeInformersForNamespaces,
		configInformers,
		controllerConfig.EventRecorder,
	)

	klog.Info("Starting the informers")
	go kubeInformersForNamespaces.Start(ctx.Done())
	go dynamicInformers.Start(ctx.Done())
	go configInformers.Start(ctx.Done())
	go shareInformersFactory.Start(ctx.Done())

	klog.Info("Starting controllerset")
	go csiControllerSet.Run(ctx, 1)

	klog.Info("Starting webhookDeploymentController")
	go webhookDeploymentController.Run(ctx, 1)

	klog.Info("Starting metrics collection")

	klog.Info("Starting metrics endpoint")
	server := metrics.BuildServer(metrics.MetricsPort)
	go metrics.RunServer(server, ctx.Done())

	<-ctx.Done()
	crdDone <- true
	cmDone <- true

	return fmt.Errorf("stopped")
}

// when we promote out of tech preview and into OCP in general, the shared resource CRDs will be vendored into
// openshift apiserver and CRD existence will be managed just as it is managed for all the other openshift CRDS;
// in the interim, this method and the associated ticker created is a "cheap / meets min / don't go down the path
// of shared informers" means for dealing with inadvertent deletes of the CRD
func ensureCRDSExist(ctx context.Context, apiextensionsClient apiextensionsclient.Interface) error {
	crds := []string{"0000_10_sharedsecret.crd.yaml", "0000_10_sharedconfigmap.crd.yaml"}
	for _, crd := range crds {
		data, err := assets.ReadFile(crd)
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
	return nil
}

// the driver will run without our configuration configmap present, but we still prefer to have explicit configuration
// present, so we employ some cheap / meets min / don't go down the path
// of shared informers" means for dealing with inadvertent deletes of the configuration configmap
func ensureConfigurationConfigMapExists(ctx context.Context, kubeClient kubeclient.Interface) error {
	cmData, err := assets.ReadFile("config_configmap.yaml")
	if err != nil {
		return fmt.Errorf("error occurred reading file 'config_configmap.yaml': %s", err)
	}
	configMap := &corev1.ConfigMap{}
	if err := yaml.Unmarshal(cmData, configMap); err != nil {
		return fmt.Errorf("error occurred unmarshalling file 'config_configmap.yaml': %s", err)
	}
	_, err = kubeClient.CoreV1().ConfigMaps(defaultNamespace).Get(ctx, configMap.Name, metav1.GetOptions{})
	if err != nil && kerrors.IsNotFound(err) {
		if _, err = kubeClient.CoreV1().ConfigMaps(defaultNamespace).Create(ctx, configMap, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("error occurred creating ConfigMap %q: %s", configMap.Name, err)
		}
		klog.Infof("Successfully created ConfigMap %q", configMap.Name)
	} else if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("unexpected error determining if %q exists: %s", configMap.Name, err)
	} else {
		klog.Infof("ConfigMap %q already exists, skipping creation", configMap.Name)
	}
	return nil
}

// setSkipValidationLabelForNamespace sets the label skipValidationLabel for a the defaultNamespace.
func setSkipValidationLabelForNamespace(ctx context.Context, kubeClient kubeclient.Interface) error {
	ns, err := kubeClient.CoreV1().Namespaces().Get(ctx, defaultNamespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unexpected error determining if %q exists: %s", defaultNamespace, err)
	}

	ns.ObjectMeta.Labels[skipValidationLabel] = "true"
	_, err = kubeClient.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update namespace %q with label %q: %s", defaultNamespace, skipValidationLabel, err)
	}
	return nil
}
