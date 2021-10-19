package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

// BuildServer creates the http.Server struct
func BuildServer(port int) *http.Server {
	if port <= 0 {
		klog.Error("invalid port for metric server")
		return nil
	}

	bindAddr := fmt.Sprintf(":%d", port)
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:    bindAddr,
		Handler: router,
	}

	return srv
}

// StopServer stops the server; for tls secret rotation
func StopServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		klog.Warningf("Problem shutting down HTTP server: %v", err)
	}
}

// RunServer starts the metrics server.
func RunServer(srv *http.Server, stopCh <-chan struct{}) {
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			klog.Errorf("error starting metrics server: %v", err)
		}
	}()
	<-stopCh
	if err := srv.Close(); err != nil {
		klog.Errorf("error closing metrics server: %v", err)
	}
}
