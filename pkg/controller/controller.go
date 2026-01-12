package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/runningman84/s3-resource-operator/pkg/backends"
	"github.com/runningman84/s3-resource-operator/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	defaultResyncPeriod = 5 * time.Minute
)

// Controller watches Kubernetes secrets and manages S3 resources
type Controller struct {
	clientset        *kubernetes.Clientset
	backend          backends.Backend
	annotationKey    string
	enforceEndpoint  bool
	processedSecrets map[string]string // namespace/name -> resourceVersion
}

// NewController creates a new controller instance
func NewController(
	clientset *kubernetes.Clientset,
	backend backends.Backend,
	annotationKey string,
	enforceEndpoint bool,
) *Controller {
	return &Controller{
		clientset:        clientset,
		backend:          backend,
		annotationKey:    annotationKey,
		enforceEndpoint:  enforceEndpoint,
		processedSecrets: make(map[string]string),
	}
}

// Run starts the controller
func (c *Controller) Run(ctx context.Context) error {
	klog.Info("Performing initial sync...")
	if err := c.sync(ctx); err != nil {
		return fmt.Errorf("initial sync failed: %w", err)
	}

	klog.Info("Starting watch loop...")
	return c.watchSecrets(ctx)
}

func (c *Controller) sync(ctx context.Context) error {
	timer := metrics.StartTimer()
	defer metrics.RecordSyncDuration(timer)

	secrets, err := c.findAnnotatedSecrets(ctx)
	if err != nil {
		return err
	}

	klog.Infof("Found %d annotated secret(s)", len(secrets))

	for _, secret := range secrets {
		if err := c.handleSecret(ctx, &secret); err != nil {
			klog.Errorf("Error handling secret %s/%s: %v", secret.Namespace, secret.Name, err)
			metrics.IncrementErrors()
		}
	}

	return nil
}

func (c *Controller) watchSecrets(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			klog.Info("Watch loop stopped")
			return nil
		default:
		}

		watcher, err := c.clientset.CoreV1().Secrets(corev1.NamespaceAll).Watch(ctx, metav1.ListOptions{
			TimeoutSeconds: func() *int64 { i := int64(300); return &i }(),
		})
		if err != nil {
			klog.Errorf("Failed to start watch: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		c.processWatchEvents(ctx, watcher)
		watcher.Stop()

		// Periodic resync
		time.Sleep(defaultResyncPeriod)
		if err := c.sync(ctx); err != nil {
			klog.Errorf("Resync failed: %v", err)
		}
	}
}

func (c *Controller) processWatchEvents(ctx context.Context, watcher watch.Interface) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				klog.Info("Watch channel closed, will restart")
				return
			}

			secret, ok := event.Object.(*corev1.Secret)
			if !ok {
				continue
			}

			if !c.isAnnotated(secret) {
				continue
			}

			key := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)

			switch event.Type {
			case watch.Added, watch.Modified:
				// Skip if we've already processed this version
				if c.processedSecrets[key] == secret.ResourceVersion {
					continue
				}

				klog.Infof("Processing secret %s (event: %s)", key, event.Type)
				if err := c.handleSecret(ctx, secret); err != nil {
					klog.Errorf("Error handling secret %s: %v", key, err)
					metrics.IncrementErrors()
				} else {
					c.processedSecrets[key] = secret.ResourceVersion
				}

			case watch.Deleted:
				klog.Infof("Secret %s deleted", key)
				delete(c.processedSecrets, key)
			}
		}
	}
}

func (c *Controller) handleSecret(ctx context.Context, secret *corev1.Secret) error {
	timer := metrics.StartTimer()
	defer metrics.RecordHandleSecretDuration(timer)
	metrics.IncrementSecretsProcessed()

	data, err := c.decodeSecretData(secret)
	if err != nil {
		return err
	}

	// Extract and validate required fields
	bucketName := c.getField(data, "bucket-name", "BUCKET_NAME")
	accessKey := c.getField(data, "access-key", "ACCESS_KEY", "ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
	secretKey := c.getField(data, "secret-key", "SECRET_KEY", "SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")
	endpointURL := c.getField(data, "endpoint-url", "ENDPOINT_URL", "AWS_ENDPOINT_URL")

	if bucketName == "" || accessKey == "" || secretKey == "" {
		return fmt.Errorf("secret %s/%s is missing required fields (bucket-name, access-key, secret-key)",
			secret.Namespace, secret.Name)
	}

	// Check endpoint URL if enforcement is enabled
	if c.enforceEndpoint && endpointURL != "" && endpointURL != c.backend.GetEndpointURL() {
		klog.Warningf("Skipping secret %s/%s: endpoint URL %s does not match operator configuration %s",
			secret.Namespace, secret.Name, endpointURL, c.backend.GetEndpointURL())
		return nil
	}

	// Get optional fields
	userID := c.parseIntField(data, "user-id", "USER_ID")
	groupID := c.parseIntField(data, "group-id", "GROUP_ID")
	role := c.getFieldPtr(data, "role", "ROLE")

	// Create or update user
	userExists, err := c.backend.UserExists(ctx, accessKey)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !userExists {
		if err := c.backend.CreateUser(ctx, accessKey, secretKey, role, userID, groupID); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		metrics.IncrementUsersCreated()
	} else {
		if err := c.backend.UpdateUser(ctx, accessKey, &secretKey, userID, groupID); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		metrics.IncrementUsersUpdated()
	}

	// Create bucket if it doesn't exist
	bucketExists, err := c.backend.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !bucketExists {
		if err := c.backend.CreateBucket(ctx, bucketName, &accessKey); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		metrics.IncrementBucketsCreated()
	} else {
		// Check if owner needs to be changed
		currentOwner, err := c.backend.GetBucketOwner(ctx, bucketName)
		if err == nil && currentOwner != accessKey {
			if err := c.backend.ChangeBucketOwner(ctx, bucketName, accessKey); err != nil {
				return fmt.Errorf("failed to change bucket owner: %w", err)
			}
			metrics.IncrementBucketOwnersChanged()
		}
	}

	return nil
}

func (c *Controller) findAnnotatedSecrets(ctx context.Context) ([]corev1.Secret, error) {
	secretList, err := c.clientset.CoreV1().Secrets(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var annotated []corev1.Secret
	for _, secret := range secretList.Items {
		if c.isAnnotated(&secret) {
			annotated = append(annotated, secret)
		}
	}

	return annotated, nil
}

func (c *Controller) isAnnotated(secret *corev1.Secret) bool {
	if secret.Annotations == nil {
		return false
	}
	_, exists := secret.Annotations[c.annotationKey]
	return exists
}

func (c *Controller) decodeSecretData(secret *corev1.Secret) (map[string]string, error) {
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

func (c *Controller) getField(data map[string]string, keys ...string) string {
	for _, key := range keys {
		if val, ok := data[key]; ok && val != "" {
			return val
		}
	}
	return ""
}

func (c *Controller) getFieldPtr(data map[string]string, keys ...string) *string {
	val := c.getField(data, keys...)
	if val == "" {
		return nil
	}
	return &val
}

func (c *Controller) parseIntField(data map[string]string, keys ...string) *int {
	val := c.getField(data, keys...)
	if val == "" {
		return nil
	}
	var i int
	if _, err := fmt.Sscanf(val, "%d", &i); err == nil {
		return &i
	}
	return nil
}
