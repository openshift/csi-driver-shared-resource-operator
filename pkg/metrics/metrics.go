package metrics

import (
	"sync"

	"github.com/blang/semver"
	sharev1alpha1 "github.com/openshift/client-go/sharedresource/listers/sharedresource/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

const (
	separator = "_"

	sharesSubsystem = "openshift_csi_share"

	cm     = "configmap"
	secret = "secret"

	cmCountName     = sharesSubsystem + separator + cm
	secretCountName = sharesSubsystem + separator + secret

	MetricsPort = 6000
)

var (
	secretCountDesc = prometheus.NewDesc(
		secretCountName,
		"Counts Secret objects shared by the CSI shared resource driver",
		[]string{},
		nil,
	)

	cmCountDesc = prometheus.NewDesc(
		cmCountName,
		"Counts ConfigMap objects shared by the CSI shared resource driver",
		[]string{},
		nil,
	)

	sc = sharesCollector{}
)

type sharesCollector struct {
	sharedSecretLister    sharev1alpha1.SharedSecretLister
	sharedConfigMapLister sharev1alpha1.SharedConfigMapLister
	isCreated             bool
	createLock            sync.Mutex

	mountCountLock sync.Mutex
}

func InitializeShareCollector(sl sharev1alpha1.SharedSecretLister, cml sharev1alpha1.SharedConfigMapLister) error {
	if !sc.isCreated {
		sc.sharedSecretLister = sl
		sc.sharedConfigMapLister = cml
		err := prometheus.Register(&sc)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create will mark deprecated state for the collector
func (sc *sharesCollector) Create(version *semver.Version) bool {
	sc.createLock.Lock()
	defer sc.createLock.Unlock()
	sc.isCreated = true

	return sc.IsCreated()
}

func (sc *sharesCollector) IsCreated() bool {
	return sc.isCreated
}

// ClearState will clear all the states marked by Create.
func (sc *sharesCollector) ClearState() {
	sc.createLock.Lock()
	defer sc.createLock.Unlock()
	sc.isCreated = false
}

// FQName returns the fully-qualified metric name of the collector.
func (sc *sharesCollector) FQName() string {
	return sharesSubsystem
}

func (sc *sharesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- secretCountDesc
	ch <- cmCountDesc
}

func (sc *sharesCollector) Collect(ch chan<- prometheus.Metric) {
	sharedSecrets, err := sc.sharedSecretLister.List(labels.Everything())
	if err != nil {
		klog.V(4).Error(err, "Error while collecting shared Secrets for metrics")
		return
	}
	sharedConfigMaps, err := sc.sharedConfigMapLister.List(labels.Everything())
	if err != nil {
		klog.V(4).Error(err, "Error while collecting shared ConfigMaps for metrics")
		return
	}

	secretCounts := len(sharedSecrets)
	cmCounts := len(sharedConfigMaps)

	ch <- prometheus.MustNewConstMetric(
		secretCountDesc,
		prometheus.GaugeValue,
		float64(secretCounts),
	)

	ch <- prometheus.MustNewConstMetric(
		cmCountDesc,
		prometheus.GaugeValue,
		float64(cmCounts),
	)
}
