package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/runningman84/s3-resource-operator/pkg/backends"
	"github.com/runningman84/s3-resource-operator/pkg/controller"
	"github.com/runningman84/s3-resource-operator/pkg/metrics"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	kubeconfig      = flag.String("kubeconfig", "", "Path to kubeconfig file (for out-of-cluster usage)")
	metricsPort     = flag.Int("metrics-port", 8000, "Port for metrics and health endpoints")
	annotationKey   = flag.String("annotation-key", "s3-resource-operator.io/enabled", "Annotation key to filter secrets")
	s3EndpointURL   = flag.String("s3-endpoint-url", "", "S3 endpoint URL")
	rootAccessKey   = flag.String("root-access-key", "", "Root access key for S3 backend")
	rootSecretKey   = flag.String("root-secret-key", "", "Root secret key for S3 backend")
	backendName     = flag.String("backend-name", "versitygw", "Backend type (versitygw, minio, garage)")
	enforceEndpoint = flag.Bool("enforce-endpoint-check", true, "Skip secrets with mismatched endpoint URLs")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// Load from environment if not provided as flags
	if *s3EndpointURL == "" {
		*s3EndpointURL = os.Getenv("S3_ENDPOINT_URL")
	}
	if *rootAccessKey == "" {
		*rootAccessKey = os.Getenv("ROOT_ACCESS_KEY")
	}
	if *rootSecretKey == "" {
		*rootSecretKey = os.Getenv("ROOT_SECRET_KEY")
	}
	if os.Getenv("BACKEND_NAME") != "" {
		*backendName = os.Getenv("BACKEND_NAME")
	}
	if os.Getenv("ANNOTATION_KEY") != "" {
		*annotationKey = os.Getenv("ANNOTATION_KEY")
	}

	// Validate required configuration
	if *s3EndpointURL == "" || *rootAccessKey == "" || *rootSecretKey == "" {
		klog.Fatal("Missing required configuration: S3_ENDPOINT_URL, ROOT_ACCESS_KEY, ROOT_SECRET_KEY")
	}

	klog.Infof("Starting S3 Resource Operator")
	klog.Infof("Backend: %s", *backendName)
	klog.Infof("Annotation key: %s", *annotationKey)
	klog.Infof("Endpoint: %s", *s3EndpointURL)

	// Initialize metrics
	metrics.Register()

	// Start metrics server
	go startMetricsServer(*metricsPort)

	// Initialize backend
	backend, err := backends.NewBackend(*backendName, backends.Config{
		EndpointURL: *s3EndpointURL,
		AccessKey:   *rootAccessKey,
		SecretKey:   *rootSecretKey,
	})
	if err != nil {
		klog.Fatalf("Failed to initialize backend: %v", err)
	}

	// Test backend connection
	klog.Info("Testing backend connection...")
	if err := backend.TestConnection(context.Background()); err != nil {
		klog.Fatalf("Backend connection test failed: %v", err)
	}
	klog.Info("Backend connection test passed")

	// Create Kubernetes client
	config, err := getKubeConfig(*kubeconfig)
	if err != nil {
		klog.Fatalf("Failed to get kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Create and start controller
	ctrl := controller.NewController(
		clientset,
		backend,
		*annotationKey,
		*enforceEndpoint,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		klog.Infof("Received signal %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	// Run controller
	klog.Info("Starting controller...")
	if err := ctrl.Run(ctx); err != nil {
		klog.Fatalf("Controller error: %v", err)
	}

	klog.Info("Operator shutdown complete")
}

func getKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		klog.Info("Using in-cluster Kubernetes configuration")
		return config, nil
	}

	// Fall back to local kubeconfig
	klog.Info("Could not load in-cluster config, falling back to local kubeconfig")
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}

func startMetricsServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	addr := fmt.Sprintf(":%d", port)
	klog.Infof("Metrics server listening on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		klog.Fatalf("Metrics server failed: %v", err)
	}
}
