package backends

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Garage implements the Backend interface for Garage
type Garage struct {
	endpointURL string
	accessKey   string
	secretKey   string
	s3Client    *s3.S3
}

// NewGarage creates a new Garage backend
func NewGarage(config Config) *Garage {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         aws.String(config.EndpointURL),
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	}))

	return &Garage{
		endpointURL: config.EndpointURL,
		accessKey:   config.AccessKey,
		secretKey:   config.SecretKey,
		s3Client:    s3.New(sess),
	}
}

func (g *Garage) GetEndpointURL() string {
	return g.endpointURL
}

func (g *Garage) TestConnection(ctx context.Context) error {
	_, err := g.s3Client.ListBucketsWithContext(ctx, &s3.ListBucketsInput{})
	return err
}

func (g *Garage) CreateBucket(ctx context.Context, bucketName string, owner *string) error {
	_, err := g.s3Client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func (g *Garage) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := g.s3Client.DeleteBucketWithContext(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err
}

func (g *Garage) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	_, err := g.s3Client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (g *Garage) GetBucketOwner(ctx context.Context, bucketName string) (string, error) {
	return "", fmt.Errorf("not implemented for Garage")
}

func (g *Garage) ChangeBucketOwner(ctx context.Context, bucketName, newOwner string) error {
	return fmt.Errorf("not implemented for Garage")
}

func (g *Garage) CreateUser(ctx context.Context, accessKey, secretKey string, role *string, userID, groupID *int) error {
	return fmt.Errorf("not implemented for Garage")
}

func (g *Garage) DeleteUser(ctx context.Context, accessKey string) error {
	return fmt.Errorf("not implemented for Garage")
}

func (g *Garage) UpdateUser(ctx context.Context, accessKey string, secretKey *string, userID, groupID *int) error {
	return fmt.Errorf("not implemented for Garage")
}

func (g *Garage) UserExists(ctx context.Context, accessKey string) (bool, error) {
	return false, fmt.Errorf("not implemented for Garage")
}
