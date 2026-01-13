package backends

import (
	"context"
	"fmt"
	"testing"
)

func TestNewBackend(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		expectError bool
	}{
		{"versitygw", "versitygw", false},
		{"minio", "minio", false},
		{"garage", "garage", false},
		{"unsupported", "unsupported", true},
	}

	config := Config{
		EndpointURL: "http://localhost:9000",
		AccessKey:   "test-access",
		SecretKey:   "test-secret",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewBackend(tt.backendName, config)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for backend %s, got nil", tt.backendName)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for backend %s: %v", tt.backendName, err)
				}
				if backend == nil {
					t.Errorf("expected non-nil backend for %s", tt.backendName)
				}
			}
		})
	}
}

func TestVersityGWGetEndpointURL(t *testing.T) {
	config := Config{
		EndpointURL: "http://test.example.com:9000",
		AccessKey:   "test-access",
		SecretKey:   "test-secret",
	}

	backend := NewVersityGW(config)

	if backend.GetEndpointURL() != config.EndpointURL {
		t.Errorf("expected endpoint %s, got %s", config.EndpointURL, backend.GetEndpointURL())
	}
}

func TestMinIOGetEndpointURL(t *testing.T) {
	config := Config{
		EndpointURL: "http://minio.example.com:9000",
		AccessKey:   "test-access",
		SecretKey:   "test-secret",
	}

	backend := NewMinIO(config)

	if backend.GetEndpointURL() != config.EndpointURL {
		t.Errorf("expected endpoint %s, got %s", config.EndpointURL, backend.GetEndpointURL())
	}
}

func TestGarageGetEndpointURL(t *testing.T) {
	config := Config{
		EndpointURL: "http://garage.example.com:3900",
		AccessKey:   "test-access",
		SecretKey:   "test-secret",
	}

	backend := NewGarage(config)

	if backend.GetEndpointURL() != config.EndpointURL {
		t.Errorf("expected endpoint %s, got %s", config.EndpointURL, backend.GetEndpointURL())
	}
}

func TestMinIONotImplemented(t *testing.T) {
	ctx := context.Background()
	config := Config{
		EndpointURL: "http://localhost:9000",
		AccessKey:   "test",
		SecretKey:   "test",
	}

	backend := NewMinIO(config)

	// Test operations that should return "not implemented"
	err := backend.CreateUser(ctx, "user", "pass", nil, nil, nil)
	if err == nil {
		t.Error("expected error for CreateUser, got nil")
	}

	err = backend.ChangeBucketOwner(ctx, "bucket", "owner")
	if err == nil {
		t.Error("expected error for ChangeBucketOwner, got nil")
	}
}

func TestGarageNotImplemented(t *testing.T) {
	ctx := context.Background()
	config := Config{
		EndpointURL: "http://localhost:3900",
		AccessKey:   "test",
		SecretKey:   "test",
	}

	backend := NewGarage(config)

	// Test operations that should return "not implemented"
	err := backend.CreateUser(ctx, "user", "pass", nil, nil, nil)
	if err == nil {
		t.Error("expected error for CreateUser, got nil")
	}

	err = backend.ChangeBucketOwner(ctx, "bucket", "owner")
	if err == nil {
		t.Error("expected error for ChangeBucketOwner, got nil")
	}
}

func TestMockBackend_UserOperations(t *testing.T) {
	ctx := context.Background()
	mock := NewMockBackend("http://localhost:9000")

	// Test CreateUser
	err := mock.CreateUser(ctx, "user1", "secret1", nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Test user exists
	exists, err := mock.UserExists(ctx, "user1")
	if err != nil {
		t.Fatalf("UserExists failed: %v", err)
	}
	if !exists {
		t.Error("expected user to exist")
	}

	// Test duplicate user creation
	err = mock.CreateUser(ctx, "user1", "secret2", nil, nil, nil)
	if err == nil {
		t.Error("expected error when creating duplicate user")
	}

	// Test UpdateUser
	newSecret := "new-secret"
	err = mock.UpdateUser(ctx, "user1", &newSecret, nil, nil)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	if mock.Users["user1"].SecretKey != "new-secret" {
		t.Error("secret key was not updated")
	}

	// Test DeleteUser
	err = mock.DeleteUser(ctx, "user1")
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	exists, err = mock.UserExists(ctx, "user1")
	if err != nil {
		t.Fatalf("UserExists failed: %v", err)
	}
	if exists {
		t.Error("expected user to not exist after deletion")
	}
}

func TestMockBackend_BucketOperations(t *testing.T) {
	ctx := context.Background()
	mock := NewMockBackend("http://localhost:9000")

	owner := "test-owner"

	// Test CreateBucket
	err := mock.CreateBucket(ctx, "bucket1", &owner)
	if err != nil {
		t.Fatalf("CreateBucket failed: %v", err)
	}

	// Test bucket exists
	exists, err := mock.BucketExists(ctx, "bucket1")
	if err != nil {
		t.Fatalf("BucketExists failed: %v", err)
	}
	if !exists {
		t.Error("expected bucket to exist")
	}

	// Test duplicate bucket creation
	err = mock.CreateBucket(ctx, "bucket1", &owner)
	if err == nil {
		t.Error("expected error when creating duplicate bucket")
	}

	// Test GetBucketOwner
	bucketOwner, err := mock.GetBucketOwner(ctx, "bucket1")
	if err != nil {
		t.Fatalf("GetBucketOwner failed: %v", err)
	}
	if bucketOwner != "test-owner" {
		t.Errorf("expected owner 'test-owner', got '%s'", bucketOwner)
	}

	// Test ChangeBucketOwner
	err = mock.ChangeBucketOwner(ctx, "bucket1", "new-owner")
	if err != nil {
		t.Fatalf("ChangeBucketOwner failed: %v", err)
	}

	bucketOwner, err = mock.GetBucketOwner(ctx, "bucket1")
	if err != nil {
		t.Fatalf("GetBucketOwner failed: %v", err)
	}
	if bucketOwner != "new-owner" {
		t.Errorf("expected owner 'new-owner', got '%s'", bucketOwner)
	}

	// Test DeleteBucket
	err = mock.DeleteBucket(ctx, "bucket1")
	if err != nil {
		t.Fatalf("DeleteBucket failed: %v", err)
	}

	exists, err = mock.BucketExists(ctx, "bucket1")
	if err != nil {
		t.Fatalf("BucketExists failed: %v", err)
	}
	if exists {
		t.Error("expected bucket to not exist after deletion")
	}
}

func TestMockBackend_ErrorInjection(t *testing.T) {
	ctx := context.Background()
	mock := NewMockBackend("http://localhost:9000")

	testErr := fmt.Errorf("injected error")

	// Test UserExists error
	mock.UserExistsError = testErr
	_, err := mock.UserExists(ctx, "user")
	if err != testErr {
		t.Errorf("expected injected error, got %v", err)
	}

	// Test CreateUser error
	mock.UserExistsError = nil
	mock.CreateUserError = testErr
	err = mock.CreateUser(ctx, "user", "secret", nil, nil, nil)
	if err != testErr {
		t.Errorf("expected injected error, got %v", err)
	}

	// Test BucketExists error
	mock.BucketExistsError = testErr
	_, err = mock.BucketExists(ctx, "bucket")
	if err != testErr {
		t.Errorf("expected injected error, got %v", err)
	}

	// Test CreateBucket error
	mock.BucketExistsError = nil
	mock.CreateBucketError = testErr
	err = mock.CreateBucket(ctx, "bucket", nil)
	if err != testErr {
		t.Errorf("expected injected error, got %v", err)
	}
}

func TestMockBackend_CallTracking(t *testing.T) {
	ctx := context.Background()
	mock := NewMockBackend("http://localhost:9000")

	// Make various calls
	mock.TestConnection(ctx)
	mock.UserExists(ctx, "user")
	mock.CreateUser(ctx, "user", "secret", nil, nil, nil)
	mock.BucketExists(ctx, "bucket")
	owner := "owner"
	mock.CreateBucket(ctx, "bucket", &owner)

	// Verify call counts
	if mock.TestConnectionCalls != 1 {
		t.Errorf("expected 1 TestConnection call, got %d", mock.TestConnectionCalls)
	}
	if mock.UserExistsCalls != 1 {
		t.Errorf("expected 1 UserExists call, got %d", mock.UserExistsCalls)
	}
	if mock.CreateUserCalls != 1 {
		t.Errorf("expected 1 CreateUser call, got %d", mock.CreateUserCalls)
	}
	if mock.BucketExistsCalls != 1 {
		t.Errorf("expected 1 BucketExists call, got %d", mock.BucketExistsCalls)
	}
	if mock.CreateBucketCalls != 1 {
		t.Errorf("expected 1 CreateBucket call, got %d", mock.CreateBucketCalls)
	}

	// Test Reset
	mock.Reset()
	if mock.TestConnectionCalls != 0 {
		t.Errorf("expected 0 calls after reset, got %d", mock.TestConnectionCalls)
	}
	if len(mock.Users) != 0 {
		t.Errorf("expected 0 users after reset, got %d", len(mock.Users))
	}
	if len(mock.Buckets) != 0 {
		t.Errorf("expected 0 buckets after reset, got %d", len(mock.Buckets))
	}
}

func TestMockBackend_WithOptionalFields(t *testing.T) {
	ctx := context.Background()
	mock := NewMockBackend("http://localhost:9000")

	role := "admin"
	userID := 1000
	groupID := 2000

	err := mock.CreateUser(ctx, "user", "secret", &role, &userID, &groupID)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user := mock.Users["user"]
	if user.Role == nil || *user.Role != "admin" {
		t.Errorf("expected role 'admin', got %v", user.Role)
	}
	if user.UserID == nil || *user.UserID != 1000 {
		t.Errorf("expected userID 1000, got %v", user.UserID)
	}
	if user.GroupID == nil || *user.GroupID != 2000 {
		t.Errorf("expected groupID 2000, got %v", user.GroupID)
	}

	// Test UpdateUser with optional fields
	newUserID := 1001
	newGroupID := 2001
	err = mock.UpdateUser(ctx, "user", nil, &newUserID, &newGroupID)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}

	if *user.UserID != 1001 {
		t.Errorf("expected updated userID 1001, got %d", *user.UserID)
	}
	if *user.GroupID != 2001 {
		t.Errorf("expected updated groupID 2001, got %d", *user.GroupID)
	}
}
