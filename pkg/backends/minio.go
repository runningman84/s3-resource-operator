package backends

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// MinIO implements the Backend interface for MinIO
type MinIO struct {
	endpointURL string
	accessKey   string
	secretKey   string
	s3Client    *s3.S3
}

// NewMinIO creates a new MinIO backend
func NewMinIO(config Config) *MinIO {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         aws.String(config.EndpointURL),
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	}))

	return &MinIO{
		endpointURL: config.EndpointURL,
		accessKey:   config.AccessKey,
		secretKey:   config.SecretKey,
		s3Client:    s3.New(sess),
	}
}

func (m *MinIO) GetEndpointURL() string {
	return m.endpointURL
}

func (m *MinIO) TestConnection(ctx context.Context) error {
	_, err := m.s3Client.ListBucketsWithContext(ctx, &s3.ListBucketsInput{})
	return err
}

func (m *MinIO) CreateBucket(ctx context.Context, bucketName string, owner *string) error {
	_, err := m.s3Client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func (m *MinIO) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := m.s3Client.DeleteBucketWithContext(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func (m *MinIO) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	_, err := m.s3Client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (m *MinIO) GetBucketOwner(ctx context.Context, bucketName string) (string, error) {
	return "", fmt.Errorf("not implemented for MinIO")
}

func (m *MinIO) ChangeBucketOwner(ctx context.Context, bucketName, newOwner string) error {
	return fmt.Errorf("not implemented for MinIO")
}

func (m *MinIO) CreateUser(ctx context.Context, accessKey, secretKey string, role *string, userID, groupID *int) error {
	return fmt.Errorf("not implemented for MinIO")
}

func (m *MinIO) DeleteUser(ctx context.Context, accessKey string) error {
	return fmt.Errorf("not implemented for MinIO")
}

func (m *MinIO) UpdateUser(ctx context.Context, accessKey string, secretKey *string, userID, groupID *int) error {
	return fmt.Errorf("not implemented for MinIO")
}

func (m *MinIO) UserExists(ctx context.Context, accessKey string) (bool, error) {
	return false, fmt.Errorf("not implemented for MinIO")
}
