package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/runningman84/s3-resource-operator/pkg/backends"
	"github.com/runningman84/s3-resource-operator/pkg/controller"
	"github.com/runningman84/s3-resource-operator/pkg/metrics"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

var (
	metricsPort     = flag.Int("metrics-port", 8080, "Port for metrics and health endpoints")
	annotationKey   = flag.String("annotation-key", "s3-resource-operator.io/enabled", "Annotation key to filter secrets")
	s3EndpointURL   = flag.String("s3-endpoint-url", "", "S3 endpoint URL")
	rootAccessKey   = flag.String("root-access-key", "", "Root access key for S3 backend")
	rootSecretKey   = flag.String("root-secret-key", "", "Root secret key for S3 backend")
	backendName     = flag.String("backend-name", "versitygw", "Backend type (versitygw, minio, garage)")
	enforceEndpoint = flag.Bool("enforce-endpoint-check", true, "Skip secrets with mismatched endpoint URLs")
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	// Setup logger
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

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
		setupLog.Error(fmt.Errorf("missing required configuration"), "Missing S3_ENDPOINT_URL, ROOT_ACCESS_KEY, or ROOT_SECRET_KEY")
		os.Exit(1)
	}

	setupLog.Info("Starting S3 Resource Operator",
		"backend", *backendName,
		"annotationKey", *annotationKey,
		"endpoint", *s3EndpointURL)

	// Initialize metrics
	metrics.Register()

	// Initialize backend
	backend, err := backends.NewBackend(*backendName, backends.Config{
		EndpointURL: *s3EndpointURL,
		AccessKey:   *rootAccessKey,
		SecretKey:   *rootSecretKey,
	})
	if err != nil {
		setupLog.Error(err, "Failed to initialize backend")
		os.Exit(1)
	}

	// Test backend connection
	setupLog.Info("Testing backend connection...")
	ctx := ctrl.SetupSignalHandler()
	if err := backend.TestConnection(ctx); err != nil {
		setupLog.Error(err, "Backend connection test failed")
		os.Exit(1)
	}
	setupLog.Info("Backend connection test passed")

	// Get Kubernetes config (controller-runtime handles this automatically via flags)
	config := ctrl.GetConfigOrDie()

	// Create manager
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: fmt.Sprintf(":%d", *metricsPort),
		},
		HealthProbeBindAddress: fmt.Sprintf(":%d", *metricsPort+1),
	})
	if err != nil {
		setupLog.Error(err, "Unable to create manager")
		os.Exit(1)
	}

	// Add health endpoints
	if err := mgr.AddHealthzCheck("healthz", func(_ *http.Request) error { return nil }); err != nil {
		setupLog.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", func(_ *http.Request) error { return nil }); err != nil {
		setupLog.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	// Create and register reconciler
	if err = controller.NewSecretReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		backend,
		*annotationKey,
		*enforceEndpoint,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create controller")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}
