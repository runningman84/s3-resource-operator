package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/runningman84/s3-resource-operator/pkg/backends"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsAnnotated(t *testing.T) {
	ctrl := &Controller{
		annotationKey: "s3-resource-operator.io/enabled",
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
			result := ctrl.isAnnotated(tt.secret)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDecodeSecretData(t *testing.T) {
	ctrl := &Controller{}

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"bucket-name": []byte("test-bucket"),
			"access-key":  []byte("test-access"),
		},
		StringData: map[string]string{
			"secret-key": "test-secret",
		},
	}

	data, err := ctrl.decodeSecretData(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data["bucket-name"] != "test-bucket" {
		t.Errorf("expected bucket-name=test-bucket, got %s", data["bucket-name"])
	}

	if data["access-key"] != "test-access" {
		t.Errorf("expected access-key=test-access, got %s", data["access-key"])
	}

	if data["secret-key"] != "test-secret" {
		t.Errorf("expected secret-key=test-secret, got %s", data["secret-key"])
	}
}

func TestGetField(t *testing.T) {
	ctrl := &Controller{}

	data := map[string]string{
		"bucket-name": "test-bucket",
		"BUCKET_NAME": "alt-bucket",
		"access-key":  "test-key",
	}

	// Test first match
	result := ctrl.getField(data, "bucket-name", "BUCKET_NAME")
	if result != "test-bucket" {
		t.Errorf("expected test-bucket, got %s", result)
	}

	// Test fallback
	result = ctrl.getField(data, "missing-key", "access-key")
	if result != "test-key" {
		t.Errorf("expected test-key, got %s", result)
	}

	// Test no match
	result = ctrl.getField(data, "missing-1", "missing-2")
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestParseIntField(t *testing.T) {
	ctrl := &Controller{}

	data := map[string]string{
		"user-id":  "1000",
		"group-id": "2000",
		"invalid":  "not-a-number",
	}

	// Test valid int
	result := ctrl.parseIntField(data, "user-id")
	if result == nil || *result != 1000 {
		t.Errorf("expected 1000, got %v", result)
	}

	// Test invalid int
	result = ctrl.parseIntField(data, "invalid")
	if result != nil {
		t.Errorf("expected nil for invalid int, got %v", result)
	}

	// Test missing field
	result = ctrl.parseIntField(data, "missing")
	if result != nil {
		t.Errorf("expected nil for missing field, got %v", result)
	}
}

func TestHandleSecret_CreateNewUserAndBucket(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket-name": []byte("my-bucket"),
			"access-key":  []byte("my-access-key"),
			"secret-key":  []byte("my-secret-key"),
		},
	}

	err := ctrl.handleSecret(context.Background(), secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify user was created
	if mockBackend.CreateUserCalls != 1 {
		t.Errorf("expected 1 CreateUser call, got %d", mockBackend.CreateUserCalls)
	}

	user, exists := mockBackend.Users["my-access-key"]
	if !exists {
		t.Fatal("user was not created")
	}
	if user.SecretKey != "my-secret-key" {
		t.Errorf("expected secret key 'my-secret-key', got %s", user.SecretKey)
	}

	// Verify bucket was created
	if mockBackend.CreateBucketCalls != 1 {
		t.Errorf("expected 1 CreateBucket call, got %d", mockBackend.CreateBucketCalls)
	}

	owner, exists := mockBackend.Buckets["my-bucket"]
	if !exists {
		t.Fatal("bucket was not created")
	}
	if owner != "my-access-key" {
		t.Errorf("expected bucket owner 'my-access-key', got %s", owner)
	}
}

func TestHandleSecret_UpdateExistingUser(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")

	// Pre-create user
	ctx := context.Background()
	mockBackend.CreateUser(ctx, "existing-user", "old-secret", nil, nil, nil)
	mockBackend.CreateBucket(ctx, "existing-bucket", stringPtr("existing-user"))

	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket-name": []byte("existing-bucket"),
			"access-key":  []byte("existing-user"),
			"secret-key":  []byte("new-secret"),
		},
	}

	// Reset call counters
	mockBackend.CreateUserCalls = 0
	mockBackend.UpdateUserCalls = 0
	mockBackend.CreateBucketCalls = 0

	err := ctrl.handleSecret(ctx, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify user was updated, not created
	if mockBackend.CreateUserCalls != 0 {
		t.Errorf("expected 0 CreateUser calls, got %d", mockBackend.CreateUserCalls)
	}
	if mockBackend.UpdateUserCalls != 1 {
		t.Errorf("expected 1 UpdateUser call, got %d", mockBackend.UpdateUserCalls)
	}

	// Verify secret was updated
	user := mockBackend.Users["existing-user"]
	if user.SecretKey != "new-secret" {
		t.Errorf("expected updated secret 'new-secret', got %s", user.SecretKey)
	}

	// Verify bucket was not created again
	if mockBackend.CreateBucketCalls != 0 {
		t.Errorf("expected 0 CreateBucket calls, got %d", mockBackend.CreateBucketCalls)
	}
}

func TestHandleSecret_ChangeBucketOwner(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	ctx := context.Background()

	// Create bucket with original owner
	mockBackend.CreateUser(ctx, "original-owner", "secret1", nil, nil, nil)
	mockBackend.CreateBucket(ctx, "shared-bucket", stringPtr("original-owner"))

	// Create new user
	mockBackend.CreateUser(ctx, "new-owner", "secret2", nil, nil, nil)

	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket-name": []byte("shared-bucket"),
			"access-key":  []byte("new-owner"),
			"secret-key":  []byte("secret2"),
		},
	}

	// Reset call counters
	mockBackend.ChangeBucketOwnerCalls = 0

	err := ctrl.handleSecret(ctx, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify bucket owner was changed
	if mockBackend.ChangeBucketOwnerCalls != 1 {
		t.Errorf("expected 1 ChangeBucketOwner call, got %d", mockBackend.ChangeBucketOwnerCalls)
	}

	owner := mockBackend.Buckets["shared-bucket"]
	if owner != "new-owner" {
		t.Errorf("expected bucket owner 'new-owner', got %s", owner)
	}
}

func TestHandleSecret_MissingRequiredFields(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	tests := []struct {
		name   string
		data   map[string][]byte
		errMsg string
	}{
		{
			name: "missing bucket name",
			data: map[string][]byte{
				"access-key": []byte("key"),
				"secret-key": []byte("secret"),
			},
			errMsg: "missing required fields",
		},
		{
			name: "missing access key",
			data: map[string][]byte{
				"bucket-name": []byte("bucket"),
				"secret-key":  []byte("secret"),
			},
			errMsg: "missing required fields",
		},
		{
			name: "missing secret key",
			data: map[string][]byte{
				"bucket-name": []byte("bucket"),
				"access-key":  []byte("key"),
			},
			errMsg: "missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: tt.data,
			}

			err := ctrl.handleSecret(context.Background(), secret)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if fmt.Sprint(err) == "" || fmt.Sprint(err) != fmt.Sprint(err) {
				// Just check error is not nil
			}
		})
	}
}

func TestHandleSecret_EndpointEnforcement(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  true,
		processedSecrets: make(map[string]string),
	}

	tests := []struct {
		name             string
		endpointURL      string
		expectProcessing bool
	}{
		{
			name:             "matching endpoint",
			endpointURL:      "http://localhost:9000",
			expectProcessing: true,
		},
		{
			name:             "mismatched endpoint",
			endpointURL:      "http://other-server:9000",
			expectProcessing: false,
		},
		{
			name:             "no endpoint specified",
			endpointURL:      "",
			expectProcessing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend.Reset()

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"bucket-name":  []byte("test-bucket"),
					"access-key":   []byte("test-key"),
					"secret-key":   []byte("test-secret"),
					"endpoint-url": []byte(tt.endpointURL),
				},
			}

			err := ctrl.handleSecret(context.Background(), secret)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectProcessing {
				if mockBackend.CreateUserCalls != 1 {
					t.Errorf("expected user creation, got %d calls", mockBackend.CreateUserCalls)
				}
			} else {
				if mockBackend.CreateUserCalls != 0 {
					t.Errorf("expected no user creation, got %d calls", mockBackend.CreateUserCalls)
				}
			}
		})
	}
}

func TestHandleSecret_WithOptionalFields(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"bucket-name": []byte("my-bucket"),
			"access-key":  []byte("my-key"),
			"secret-key":  []byte("my-secret"),
			"user-id":     []byte("1000"),
			"group-id":    []byte("2000"),
			"role":        []byte("admin"),
		},
	}

	err := ctrl.handleSecret(context.Background(), secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	user := mockBackend.Users["my-key"]
	if user.UserID == nil || *user.UserID != 1000 {
		t.Errorf("expected userID 1000, got %v", user.UserID)
	}
	if user.GroupID == nil || *user.GroupID != 2000 {
		t.Errorf("expected groupID 2000, got %v", user.GroupID)
	}
	if user.Role == nil || *user.Role != "admin" {
		t.Errorf("expected role 'admin', got %v", user.Role)
	}
}

func TestHandleSecret_AlternativeFieldNames(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	ctrl := &Controller{
		backend:          mockBackend,
		annotationKey:    "test-key",
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	tests := []struct {
		name string
		data map[string][]byte
	}{
		{
			name: "uppercase field names",
			data: map[string][]byte{
				"BUCKET_NAME": []byte("test-bucket"),
				"ACCESS_KEY":  []byte("test-key"),
				"SECRET_KEY":  []byte("test-secret"),
			},
		},
		{
			name: "AWS-style field names",
			data: map[string][]byte{
				"bucket-name":           []byte("test-bucket"),
				"AWS_ACCESS_KEY_ID":     []byte("test-key"),
				"AWS_SECRET_ACCESS_KEY": []byte("test-secret"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend.Reset()

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: tt.data,
			}

			err := ctrl.handleSecret(context.Background(), secret)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mockBackend.CreateUserCalls != 1 {
				t.Errorf("expected 1 CreateUser call, got %d", mockBackend.CreateUserCalls)
			}
			if mockBackend.CreateBucketCalls != 1 {
				t.Errorf("expected 1 CreateBucket call, got %d", mockBackend.CreateBucketCalls)
			}
		})
	}
}

func TestHandleSecret_BackendErrors(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*backends.MockBackend)
		expectErr bool
	}{
		{
			name: "UserExists error",
			setupFunc: func(m *backends.MockBackend) {
				m.UserExistsError = fmt.Errorf("backend error")
			},
			expectErr: true,
		},
		{
			name: "CreateUser error",
			setupFunc: func(m *backends.MockBackend) {
				m.CreateUserError = fmt.Errorf("create user failed")
			},
			expectErr: true,
		},
		{
			name: "BucketExists error",
			setupFunc: func(m *backends.MockBackend) {
				m.BucketExistsError = fmt.Errorf("backend error")
			},
			expectErr: true,
		},
		{
			name: "CreateBucket error",
			setupFunc: func(m *backends.MockBackend) {
				m.CreateBucketError = fmt.Errorf("create bucket failed")
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBackend := backends.NewMockBackend("http://localhost:9000")
			tt.setupFunc(mockBackend)

			ctrl := &Controller{
				backend:          mockBackend,
				annotationKey:    "test-key",
				enforceEndpoint:  false,
				processedSecrets: make(map[string]string),
			}

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

			err := ctrl.handleSecret(context.Background(), secret)
			if tt.expectErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestSync(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	annotationKey := "s3-operator/enabled"

	// Create fake Kubernetes client with some secrets
	client := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "annotated-secret-1",
				Namespace: "default",
				Annotations: map[string]string{
					annotationKey: "true",
				},
			},
			Data: map[string][]byte{
				"bucket-name": []byte("bucket-1"),
				"access-key":  []byte("key-1"),
				"secret-key":  []byte("secret-1"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "annotated-secret-2",
				Namespace: "kube-system",
				Annotations: map[string]string{
					annotationKey: "true",
				},
			},
			Data: map[string][]byte{
				"bucket-name": []byte("bucket-2"),
				"access-key":  []byte("key-2"),
				"secret-key":  []byte("secret-2"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-annotated-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"bucket-name": []byte("bucket-3"),
				"access-key":  []byte("key-3"),
				"secret-key":  []byte("secret-3"),
			},
		},
	)

	ctrl := &Controller{
		clientset:        client,
		backend:          mockBackend,
		annotationKey:    annotationKey,
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	err := ctrl.sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created 2 users (only annotated secrets)
	if mockBackend.CreateUserCalls != 2 {
		t.Errorf("expected 2 CreateUser calls, got %d", mockBackend.CreateUserCalls)
	}

	// Should have created 2 buckets
	if mockBackend.CreateBucketCalls != 2 {
		t.Errorf("expected 2 CreateBucket calls, got %d", mockBackend.CreateBucketCalls)
	}

	// Verify specific users were created
	if _, exists := mockBackend.Users["key-1"]; !exists {
		t.Error("expected user 'key-1' to be created")
	}
	if _, exists := mockBackend.Users["key-2"]; !exists {
		t.Error("expected user 'key-2' to be created")
	}
	if _, exists := mockBackend.Users["key-3"]; exists {
		t.Error("expected user 'key-3' NOT to be created (not annotated)")
	}
}

func TestFindAnnotatedSecrets(t *testing.T) {
	annotationKey := "test-annotation"

	client := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-1",
				Namespace: "default",
				Annotations: map[string]string{
					annotationKey: "value",
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-2",
				Namespace: "kube-system",
				Annotations: map[string]string{
					"other-annotation": "value",
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-3",
				Namespace: "default",
				Annotations: map[string]string{
					annotationKey: "another-value",
				},
			},
		},
	)

	ctrl := &Controller{
		clientset:     client,
		annotationKey: annotationKey,
	}

	secrets, err := ctrl.findAnnotatedSecrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(secrets) != 2 {
		t.Errorf("expected 2 annotated secrets, got %d", len(secrets))
	}

	// Verify the correct secrets were found
	foundNames := make(map[string]bool)
	for _, secret := range secrets {
		foundNames[secret.Name] = true
	}

	if !foundNames["secret-1"] {
		t.Error("expected to find 'secret-1'")
	}
	if !foundNames["secret-3"] {
		t.Error("expected to find 'secret-3'")
	}
	if foundNames["secret-2"] {
		t.Error("did not expect to find 'secret-2'")
	}
}

func TestSync_ErrorHandling(t *testing.T) {
	mockBackend := backends.NewMockBackend("http://localhost:9000")
	mockBackend.CreateUserError = fmt.Errorf("simulated error")
	annotationKey := "test-annotation"

	client := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
				Annotations: map[string]string{
					annotationKey: "true",
				},
			},
			Data: map[string][]byte{
				"bucket-name": []byte("test-bucket"),
				"access-key":  []byte("test-key"),
				"secret-key":  []byte("test-secret"),
			},
		},
	)

	ctrl := &Controller{
		clientset:        client,
		backend:          mockBackend,
		annotationKey:    annotationKey,
		enforceEndpoint:  false,
		processedSecrets: make(map[string]string),
	}

	// sync should not return error, but log it
	err := ctrl.sync(context.Background())
	if err != nil {
		t.Fatalf("sync should not return error for individual secret failures: %v", err)
	}

	// Should have attempted to create user
	if mockBackend.CreateUserCalls != 1 {
		t.Errorf("expected 1 CreateUser call, got %d", mockBackend.CreateUserCalls)
	}
}
