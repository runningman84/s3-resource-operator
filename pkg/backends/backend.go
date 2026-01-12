package backends

import "context"

// Backend defines the interface for S3-compatible storage backends
type Backend interface {
	// TestConnection tests the connection to the backend
	TestConnection(ctx context.Context) error

	// Bucket operations
	CreateBucket(ctx context.Context, bucketName string, owner *string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	GetBucketOwner(ctx context.Context, bucketName string) (string, error)
	ChangeBucketOwner(ctx context.Context, bucketName, newOwner string) error

	// User operations
	CreateUser(ctx context.Context, accessKey, secretKey string, role *string, userID, groupID *int) error
	DeleteUser(ctx context.Context, accessKey string) error
	UpdateUser(ctx context.Context, accessKey string, secretKey *string, userID, groupID *int) error
	UserExists(ctx context.Context, accessKey string) (bool, error)

	// GetEndpointURL returns the configured endpoint URL
	GetEndpointURL() string
}

// Config holds common backend configuration
type Config struct {
	EndpointURL string
	AccessKey   string
	SecretKey   string
}

// NewBackend creates a new backend instance based on the backend name
func NewBackend(name string, config Config) (Backend, error) {
	switch name {
	case "versitygw":
		return NewVersityGW(config), nil
	case "minio":
		return NewMinIO(config), nil
	case "garage":
		return NewGarage(config), nil
	default:
		return nil, ErrUnsupportedBackend{Backend: name}
	}
}

// ErrUnsupportedBackend is returned when an unknown backend is requested
type ErrUnsupportedBackend struct {
	Backend string
}

func (e ErrUnsupportedBackend) Error() string {
	return "unsupported backend: " + e.Backend
}
