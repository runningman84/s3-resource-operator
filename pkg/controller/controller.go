package controller

import (
	"context"
	"fmt"

	"github.com/runningman84/s3-resource-operator/pkg/backends"
	"github.com/runningman84/s3-resource-operator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SecretReconciler reconciles Secrets with S3 backend
type SecretReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Backend         backends.Backend
	AnnotationKey   string
	EnforceEndpoint bool
}

// NewSecretReconciler creates a new reconciler instance
func NewSecretReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	backend backends.Backend,
	annotationKey string,
	enforceEndpoint bool,
) *SecretReconciler {
	return &SecretReconciler{
		Client:          client,
		Scheme:          scheme,
		Backend:         backend,
		AnnotationKey:   annotationKey,
		EnforceEndpoint: enforceEndpoint,
	}
}

// Reconcile handles Secret events
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Secret
	var secret corev1.Secret
	if err := r.Get(ctx, req.NamespacedName, &secret); err != nil {
		// Secret was deleted or doesn't exist
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Skip if not annotated
	if !r.isAnnotated(&secret) {
		return ctrl.Result{}, nil
	}

	logger.Info("Reconciling secret", "namespace", secret.Namespace, "name", secret.Name)

	if err := r.handleSecret(ctx, &secret); err != nil {
		logger.Error(err, "Failed to handle secret")
		metrics.IncrementErrors()
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Only watch secrets with our annotation
	pred := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return false
		}
		return r.isAnnotated(secret)
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(pred).
		Complete(r)
}

func (r *SecretReconciler) handleSecret(ctx context.Context, secret *corev1.Secret) error {
	logger := log.FromContext(ctx)
	timer := metrics.StartTimer()
	defer metrics.RecordHandleSecretDuration(timer)
	metrics.IncrementSecretsProcessed()

	data, err := r.decodeSecretData(secret)
	if err != nil {
		return err
	}

	// Extract and validate required fields
	bucketName := r.getField(data, "bucket-name", "BUCKET_NAME")
	accessKey := r.getField(data, "access-key", "ACCESS_KEY", "ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
	secretKey := r.getField(data, "secret-key", "SECRET_KEY", "SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")
	endpointURL := r.getField(data, "endpoint-url", "ENDPOINT_URL", "AWS_ENDPOINT_URL", "AWS_ENDPOINTS")

	if bucketName == "" || accessKey == "" || secretKey == "" {
		return fmt.Errorf("secret %s/%s is missing required fields (bucket-name, access-key, secret-key)",
			secret.Namespace, secret.Name)
	}

	// Check endpoint URL if enforcement is enabled
	if r.EnforceEndpoint && endpointURL != "" && endpointURL != r.Backend.GetEndpointURL() {
		logger.Info("Skipping secret: endpoint URL mismatch",
			"secret", fmt.Sprintf("%s/%s", secret.Namespace, secret.Name),
			"secretEndpoint", endpointURL,
			"operatorEndpoint", r.Backend.GetEndpointURL())
		return nil
	}

	// Get optional fields
	userID := r.parseIntField(data, "user-id", "USER_ID")
	groupID := r.parseIntField(data, "group-id", "GROUP_ID")
	role := r.getFieldPtr(data, "role", "ROLE")

	// Create or update user
	userExists, err := r.Backend.UserExists(ctx, accessKey)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !userExists {
		if err := r.Backend.CreateUser(ctx, accessKey, secretKey, role, userID, groupID); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		metrics.IncrementUsersCreated()
	} else {
		if err := r.Backend.UpdateUser(ctx, accessKey, &secretKey, userID, groupID); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		metrics.IncrementUsersUpdated()
	}

	// Create bucket if it doesn't exist
	bucketExists, err := r.Backend.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !bucketExists {
		if err := r.Backend.CreateBucket(ctx, bucketName, &accessKey); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		metrics.IncrementBucketsCreated()
	} else {
		// Check if owner needs to be changed
		currentOwner, err := r.Backend.GetBucketOwner(ctx, bucketName)
		if err == nil && currentOwner != accessKey {
			if err := r.Backend.ChangeBucketOwner(ctx, bucketName, accessKey); err != nil {
				return fmt.Errorf("failed to change bucket owner: %w", err)
			}
			metrics.IncrementBucketOwnersChanged()
		}
	}

	return nil
}

func (r *SecretReconciler) isAnnotated(secret *corev1.Secret) bool {
	if secret.Annotations == nil {
		return false
	}
	_, exists := secret.Annotations[r.AnnotationKey]
	return exists
}

func (r *SecretReconciler) decodeSecretData(secret *corev1.Secret) (map[string]string, error) {
	decoded := make(map[string]string)
	for key, value := range secret.Data {
		decoded[key] = string(value)
	}

	// Also handle StringData (already decoded)
	for key, value := range secret.StringData {
		decoded[key] = value
	}

	return decoded, nil
}

func (r *SecretReconciler) getField(data map[string]string, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok && val != "" {
			return val
		}
	}
	return ""
}

func (r *SecretReconciler) getFieldPtr(data map[string]string, keys ...string) *string {
	val := r.getField(data, keys...)
	if val == "" {
		return nil
	}
	return &val
}

func (r *SecretReconciler) parseIntField(data map[string]string, keys ...string) *int {
	val := r.getField(data, keys...)
	if val == "" {
		return nil
	}
	var i int
	if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
		return &i
	}
	return nil
}
