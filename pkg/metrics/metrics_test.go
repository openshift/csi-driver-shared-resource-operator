package metrics

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	v1alpha1 "github.com/openshift/api/sharedresource/v1alpha1"
	sharev1alpha1 "github.com/openshift/client-go/sharedresource/listers/sharedresource/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/labels"
)

type fakeConfigMapShareLister struct {
	cmShares []*v1alpha1.SharedConfigMap
}

func (l *fakeConfigMapShareLister) List(selector labels.Selector) (ret []*v1alpha1.SharedConfigMap, err error) {
	return l.cmShares, nil
}

func (l *fakeConfigMapShareLister) Get(name string) (*v1alpha1.SharedConfigMap, error) {
	for _, cm := range l.cmShares {
		if cm.Spec.ConfigMapRef.Name == name {
			return cm, nil
		}
	}

	return nil, nil
}

type fakeSecretShareLister struct {
	secretShares []*v1alpha1.SharedSecret
}

func (l *fakeSecretShareLister) List(selector labels.Selector) (ret []*v1alpha1.SharedSecret, err error) {
	return l.secretShares, nil
}

func (l *fakeSecretShareLister) Get(name string) (*v1alpha1.SharedSecret, error) {
	for _, secret := range l.secretShares {
		if secret.Spec.SecretRef.Name == name {
			return secret, nil
		}
	}

	return nil, nil
}

type fakeResponseWriter struct {
	bytes.Buffer
	statusCode int
	header     http.Header
}

func (f *fakeResponseWriter) Header() http.Header {
	return f.header
}

func (f *fakeResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

func TestMetrics(t *testing.T) {
	for _, test := range []struct {
		name         string
		expected     []string
		secretLister sharev1alpha1.SharedSecretLister
		cmLister     sharev1alpha1.SharedConfigMapLister
	}{
		{
			name: "One secret, one config map",
			expected: []string{
				"# HELP openshift_csi_share_configmap_total Counts ConfigMap objects shared by the CSI shared resource driver",
				"# TYPE openshift_csi_share_configmap_total gauge",
				"openshift_csi_share_configmap_total 1",
				"# HELP openshift_csi_share_secret_total Counts Secret objects shared by the CSI shared resource driver",
				"# TYPE openshift_csi_share_secret_total gauge",
				"openshift_csi_share_secret_total 1",
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
			expected: []string{
				"# HELP openshift_csi_share_secret_total Counts Secret objects shared by the CSI shared resource driver",
				"# TYPE openshift_csi_share_secret_total gauge",
				"openshift_csi_share_secret_total 2",
				"# HELP openshift_csi_share_configmap_total Counts ConfigMap objects shared by the CSI shared resource driver",
				"# TYPE openshift_csi_share_configmap_total gauge",
				"openshift_csi_share_configmap_total 0",
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
			expected: []string{
				"# HELP openshift_csi_share_configmap_total Counts ConfigMap objects shared by the CSI shared resource driver",
				"# TYPE openshift_csi_share_configmap_total gauge",
				"openshift_csi_share_configmap_total 2",
				"# HELP openshift_csi_share_secret_total Counts Secret objects shared by the CSI shared resource driver",
				"# TYPE openshift_csi_share_secret_total gauge",
				"openshift_csi_share_secret_total 0",
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

		registry := prometheus.NewRegistry()
		sc = sharesCollector{
			sharedSecretLister:    test.secretLister,
			sharedConfigMapLister: test.cmLister,
			isCreated:             true,
		}

		registry.MustRegister(&sc)

		h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.PanicOnError})
		rw := &fakeResponseWriter{header: http.Header{}}
		h.ServeHTTP(rw, &http.Request{})

		respStr := rw.String()

		for _, s := range test.expected {
			if !strings.Contains(respStr, s) {
				t.Errorf("testcase %s: expected string %s did not appear in %s", test.name, s, respStr)
			}
		}
	}
}
