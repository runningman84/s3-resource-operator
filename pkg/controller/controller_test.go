package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/runningman84/s3-resource-operator/pkg/backends"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}

func TestIsAnnotated_Runtime(t *testing.T) {
	r := &SecretReconciler{
		AnnotationKey: "s3-resource-operator.io/enabled",
	}

	tests := []struct {
		name     string
		secret   *corev1.Secret
		expected bool
	}{
		{
			name: "annotated secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"s3-resource-operator.io/enabled": "true",
					},
				},
			},
			expected: true,
		},
		{
			name: "not annotated",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"other-annotation": "value",
					},
				},
			},
			expected: false,
		},
		{
			name: "nil annotations",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.isAnnotated(tt.secret)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHandleSecret_CreateNewUserAndBucket_Runtime(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	scheme := newTestScheme()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket-name": []byte("test-bucket"),
			"access-key":  []byte("test-key"),
			"secret-key":  []byte("test-secret"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	r := &SecretReconciler{
		Client:          client,
		Scheme:          scheme,
		Backend:         mockBackend,
		AnnotationKey:   "test-annotation",
		EnforceEndpoint: false,
	}

	err := r.handleSecret(context.Background(), secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockBackend.CreateUserCalls != 1 {
		t.Errorf("expected 1 CreateUser call, got %d", mockBackend.CreateUserCalls)
	}

	if mockBackend.CreateBucketCalls != 1 {
		t.Errorf("expected 1 CreateBucket call, got %d", mockBackend.CreateBucketCalls)
	}
}

func TestHandleSecret_MissingRequiredFields_Runtime(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	scheme := newTestScheme()

	tests := []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "missing bucket name",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"access-key": []byte("key"),
					"secret-key": []byte("secret"),
				},
			},
		},
		{
			name: "missing access key",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"bucket-name": []byte("bucket"),
					"secret-key":  []byte("secret"),
				},
			},
		},
		{
			name: "missing secret key",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					"bucket-name": []byte("bucket"),
					"access-key":  []byte("key"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.secret.ObjectMeta.Name = "test-secret"
			tt.secret.ObjectMeta.Namespace = "default"

			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.secret).Build()

			r := &SecretReconciler{
				Client:        client,
				Scheme:        scheme,
				Backend:       mockBackend,
				AnnotationKey: "test-annotation",
			}

			err := r.handleSecret(context.Background(), tt.secret)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestHandleSecret_EndpointEnforcement_Runtime(t *testing.T) {
	scheme := newTestScheme()

	tests := []struct {
		name            string
		endpointURL     string
		enforceEndpoint bool
		expectErr       bool
		expectCalls     bool
	}{
		{
			name:            "matching endpoint",
			endpointURL:     "http://localhost:9000",
			enforceEndpoint: true,
			expectErr:       false,
			expectCalls:     true,
		},
		{
			name:            "mismatched endpoint",
			endpointURL:     "http://other-server:9000",
			enforceEndpoint: true,
			expectErr:       false,
			expectCalls:     false,
		},
		{
			name:            "no endpoint specified",
			endpointURL:     "",
			enforceEndpoint: true,
			expectErr:       false,
			expectCalls:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := backends.NewMockBackend("http://localhost:9000")

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"bucket-name": []byte("test-bucket"),
					"access-key":  []byte("test-key"),
					"secret-key":  []byte("test-secret"),
				},
			}

			if tt.endpointURL != "" {
				secret.Data["endpoint-url"] = []byte(tt.endpointURL)
			}

			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

			r := &SecretReconciler{
				Client:          client,
				Scheme:          scheme,
				Backend:         mockBackend,
				AnnotationKey:   "test-annotation",
				EnforceEndpoint: tt.enforceEndpoint,
			}

			err := r.handleSecret(context.Background(), secret)
			if tt.expectErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectCalls && mockBackend.CreateUserCalls == 0 {
				t.Error("expected CreateUser to be called")
			}
			if !tt.expectCalls && mockBackend.CreateUserCalls > 0 {
				t.Error("did not expect CreateUser to be called")
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestReconcile_SecretNotFound(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	scheme := newTestScheme()

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := &SecretReconciler{
		Client:        client,
		Scheme:        scheme,
		Backend:       mockBackend,
		AnnotationKey: "s3-resource-operator.io/enabled",
	}

	// Request for non-existent secret
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "non-existent",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error for non-existent secret, got: %v", err)
	}
	if result.Requeue {
		t.Error("did not expect requeue")
	}
}

func TestReconcile_SecretNotAnnotated(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	scheme := newTestScheme()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			// No annotation
		},
		Data: map[string][]byte{
			"bucket-name": []byte("test-bucket"),
			"access-key":  []byte("test-key"),
			"secret-key":  []byte("test-secret"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	r := &SecretReconciler{
		Client:        client,
		Scheme:        scheme,
		Backend:       mockBackend,
		AnnotationKey: "s3-resource-operator.io/enabled",
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "test-secret",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Requeue {
		t.Error("did not expect requeue")
	}

	// Backend should not be called
	if mockBackend.CreateUserCalls != 0 {
		t.Errorf("expected 0 CreateUser calls, got %d", mockBackend.CreateUserCalls)
	}
}

func TestReconcile_AnnotatedSecret_Success(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	scheme := newTestScheme()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				"s3-resource-operator.io/enabled": "true",
			},
		},
		Data: map[string][]byte{
			"bucket-name": []byte("test-bucket"),
			"access-key":  []byte("test-key"),
			"secret-key":  []byte("test-secret"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	r := &SecretReconciler{
		Client:          client,
		Scheme:          scheme,
		Backend:         mockBackend,
		AnnotationKey:   "s3-resource-operator.io/enabled",
		EnforceEndpoint: false,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "test-secret",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Requeue {
		t.Error("did not expect requeue")
	}

	// Verify backend was called
	if mockBackend.CreateUserCalls != 1 {
		t.Errorf("expected 1 CreateUser call, got %d", mockBackend.CreateUserCalls)
	}
	if mockBackend.CreateBucketCalls != 1 {
		t.Errorf("expected 1 CreateBucket call, got %d", mockBackend.CreateBucketCalls)
	}
}

func TestReconcile_AnnotatedSecret_HandleError(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	mockBackend.CreateUserError = fmt.Errorf("test error")
	scheme := newTestScheme()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				"s3-resource-operator.io/enabled": "true",
			},
		},
		Data: map[string][]byte{
			"bucket-name": []byte("test-bucket"),
			"access-key":  []byte("test-key"),
			"secret-key":  []byte("test-secret"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	r := &SecretReconciler{
		Client:          client,
		Scheme:          scheme,
		Backend:         mockBackend,
		AnnotationKey:   "s3-resource-operator.io/enabled",
		EnforceEndpoint: false,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "test-secret",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result.Requeue {
		t.Error("did not expect requeue")
	}
}

func TestReconcile_MissingRequiredFields(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	scheme := newTestScheme()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Annotations: map[string]string{
				"s3-resource-operator.io/enabled": "true",
			},
		},
		Data: map[string][]byte{
			// Missing bucket-name
			"access-key": []byte("test-key"),
			"secret-key": []byte("test-secret"),
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

	r := &SecretReconciler{
		Client:        client,
		Scheme:        scheme,
		Backend:       mockBackend,
		AnnotationKey: "s3-resource-operator.io/enabled",
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "test-secret",
		},
	}

	result, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
	if result.Requeue {
		t.Error("did not expect requeue")
	}
}
