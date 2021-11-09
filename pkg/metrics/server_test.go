package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	v1alpha1 "github.com/openshift/api/sharedresource/v1alpha1"
	sharev1alpha1 "github.com/openshift/client-go/sharedresource/listers/sharedresource/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var (
	portOffset uint32 = 0
)

func runMetricsServer(t *testing.T) (int, chan<- struct{}) {
	port := MetricsPort + int(atomic.AddUint32(&portOffset, 1))

	ch := make(chan struct{})
	server := BuildServer(port)
	go RunServer(server, ch)

	return port, ch
}

func TestRunServer(t *testing.T) {
	port, ch := runMetricsServer(t)
	defer close(ch)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("error while querying metrics server: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Server response status is %q instead of 200", resp.Status)
	}
}

func testQueryGaugeMetric(t *testing.T, testName string, port, value int, query string) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("error requesting metrics server: %v in test %q", err, testName)
	}
	metrics := findMetricsByCounter(resp.Body, query)
	if len(metrics) == 0 {
		t.Fatalf("unable to locate metric %s in test %q", query, testName)
	}
	if metrics[0].Gauge.Value == nil {
		t.Fatalf("metric did not have value %s in test %q", query, testName)
	}
	if *metrics[0].Gauge.Value != float64(value) {
		t.Fatalf("incorrect metric value %v for query %s in test %q", *metrics[0].Gauge.Value, query, testName)
	}
}

func findMetricsByCounter(buf io.ReadCloser, name string) []*io_prometheus_client.Metric {
	defer buf.Close()
	mf := io_prometheus_client.MetricFamily{}
	decoder := expfmt.NewDecoder(buf, "text/plain")
	for err := decoder.Decode(&mf); err == nil; err = decoder.Decode(&mf) {
		if *mf.Name == name {
			return mf.Metric
		}
	}
	return nil
}

type queryResult struct {
	queryName string
	total     int
}

func TestMetricQueries(t *testing.T) {
	for _, test := range []struct {
		name         string
		expected     []queryResult
		secretLister sharev1alpha1.SharedSecretLister
		cmLister     sharev1alpha1.SharedConfigMapLister
	}{
		{
			name: "One secret, one config map",
			expected: []queryResult{
				{"openshift_csi_share_configmap_total", 1},
				{"openshift_csi_share_secret_total", 1},
			},
			secretLister: &fakeSecretShareLister{
				secretShares: []*v1alpha1.SharedSecret{
					{
						Spec: v1alpha1.SharedSecretSpec{
							SecretRef: v1alpha1.SharedSecretReference{
								Name:      "secret-name",
								Namespace: "namespace-1",
							},
						},
					},
				},
			},
			cmLister: &fakeConfigMapShareLister{
				cmShares: []*v1alpha1.SharedConfigMap{
					{
						Spec: v1alpha1.SharedConfigMapSpec{
							ConfigMapRef: v1alpha1.SharedConfigMapReference{
								Name:      "config-map-name",
								Namespace: "namespace-2",
							},
						},
					},
				},
			},
		},
		{
			name: "Two secrets, no config maps",
			expected: []queryResult{
				{"openshift_csi_share_secret_total", 2},
				{"openshift_csi_share_configmap_total", 0},
			},
			secretLister: &fakeSecretShareLister{
				secretShares: []*v1alpha1.SharedSecret{
					{
						Spec: v1alpha1.SharedSecretSpec{
							SecretRef: v1alpha1.SharedSecretReference{
								Name:      "secret-name",
								Namespace: "namespace-1",
							},
						},
					},
					{
						Spec: v1alpha1.SharedSecretSpec{
							SecretRef: v1alpha1.SharedSecretReference{
								Name:      "secret-name-2",
								Namespace: "namespace-1",
							},
						},
					},
				},
			},
			cmLister: &fakeConfigMapShareLister{
				cmShares: []*v1alpha1.SharedConfigMap{},
			},
		},
		{
			name: "No secrets, two config maps",
			expected: []queryResult{
				{"openshift_csi_share_configmap_total", 2},
				{"openshift_csi_share_secret_total", 0},
			},
			secretLister: &fakeSecretShareLister{
				secretShares: []*v1alpha1.SharedSecret{},
			},
			cmLister: &fakeConfigMapShareLister{
				cmShares: []*v1alpha1.SharedConfigMap{
					{
						Spec: v1alpha1.SharedConfigMapSpec{
							ConfigMapRef: v1alpha1.SharedConfigMapReference{
								Name:      "config-map-name",
								Namespace: "namespace-2",
							},
						},
					},
					{
						Spec: v1alpha1.SharedConfigMapSpec{
							ConfigMapRef: v1alpha1.SharedConfigMapReference{
								Name:      "config-map-name-2",
								Namespace: "namespace-2",
							},
						},
					},
				},
			},
		},
	} {
		sc = sharesCollector{
			sharedSecretLister:    test.secretLister,
			sharedConfigMapLister: test.cmLister,
			isCreated:             true,
		}
		prometheus.MustRegister(&sc)

		port, ch := runMetricsServer(t)
		for _, e := range test.expected {
			testQueryGaugeMetric(t, test.name, port, e.total, e.queryName)
		}
		close(ch)

		prometheus.Unregister(&sc)
	}
}
