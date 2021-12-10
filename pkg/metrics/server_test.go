package metrics

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	mr "math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	v1alpha1 "github.com/openshift/api/sharedresource/v1alpha1"
	sharev1alpha1 "github.com/openshift/client-go/sharedresource/listers/sharedresource/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	portOffset uint32 = 0
)

func TestMain(m *testing.M) {
	var err error

	mr.Seed(time.Now().UnixNano())

	tlsKey, tlsCRT, err = generateTempCertificates()
	if err != nil {
		panic(err)
	}

	// sets the default http client to skip certificate check.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	code := m.Run()
	os.Remove(tlsKey)
	os.Remove(tlsCRT)
	os.Exit(code)
}

func generateTempCertificates() (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, key.Public(), key)
	if err != nil {
		return "", "", err
	}

	cert, err := ioutil.TempFile("", "testcert-")
	if err != nil {
		return "", "", err
	}
	defer cert.Close()
	pem.Encode(cert, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	keyPath, err := ioutil.TempFile("", "testkey-")
	if err != nil {
		return "", "", err
	}
	defer keyPath.Close()
	pem.Encode(keyPath, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return keyPath.Name(), cert.Name(), nil
}

func blockUntilServerStarted(port int) error {
	return wait.PollImmediate(100*time.Millisecond, 5*time.Second, func() (bool, error) {
		if _, err := http.Get(fmt.Sprintf("https://localhost:%d/metrics", port)); err != nil {
			// in case error is "connection refused", server is not up (yet)
			// it is possible that it is still being started
			// in that case we need to try more
			if utilnet.IsConnectionRefused(err) {
				return false, nil
			}

			// in case of a different error, return immediately
			return true, err
		}

		// no error, stop polling the server, continue with the test logic
		return true, nil
	})
}

func runMetricsServer(t *testing.T) (int, chan<- struct{}) {
	port := MetricsPort + int(atomic.AddUint32(&portOffset, 1))

	ch := make(chan struct{})
	server := BuildServer(port)
	go RunServer(server, ch)

	if err := blockUntilServerStarted(port); err != nil {
		t.Fatalf("error while waiting for metrics server: %v", err)
	}

	return port, ch
}

func TestRunServer(t *testing.T) {
	port, ch := runMetricsServer(t)
	defer close(ch)

	resp, err := http.Get(fmt.Sprintf("https://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("error while querying metrics server: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Server response status is %q instead of 200", resp.Status)
	}
}

func testQueryGaugeMetric(t *testing.T, testName string, port, value int, query string) {
	resp, err := http.Get(fmt.Sprintf("https://localhost:%d/metrics", port))
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
				{"openshift_csi_share_configmap", 1},
				{"openshift_csi_share_secret", 1},
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
				{"openshift_csi_share_secret", 2},
				{"openshift_csi_share_configmap", 0},
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
				{"openshift_csi_share_configmap", 2},
				{"openshift_csi_share_secret", 0},
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
