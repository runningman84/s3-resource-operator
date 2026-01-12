package backends

import (
	"context"
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
