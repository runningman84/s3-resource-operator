package backends

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/s3"
	"k8s.io/klog/v2"
)

// VersityGW implements the Backend interface for VersityGW
type VersityGW struct {
	endpointURL string
	accessKey   string
	secretKey   string
	s3Client    *s3.S3
	httpClient  *http.Client
	signer      *v4.Signer
}

// NewVersityGW creates a new VersityGW backend
func NewVersityGW(config Config) *VersityGW {
	sess := session.Must(session.NewSession(&aws.Config{
		Endpoint:         aws.String(config.EndpointURL),
		Region:           aws.String("us-east-1"),
		Credentials:      credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	}))

	creds := credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, "")

	return &VersityGW{
		endpointURL: config.EndpointURL,
		accessKey:   config.AccessKey,
		secretKey:   config.SecretKey,
		s3Client:    s3.New(sess),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		signer:      v4.NewSigner(creds),
	}
}

func (v *VersityGW) GetEndpointURL() string {
	return v.endpointURL
}

func (v *VersityGW) TestConnection(ctx context.Context) error {
	klog.Infof("Testing connection to VersityGW: %s", v.endpointURL)

	// Test listing buckets
	buckets, err := v.listBucketsRaw(ctx)
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}
	klog.Infof("Successfully listed %d bucket(s)", len(buckets))

	// Test listing users
	users, err := v.listUsersRaw(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}
	klog.Infof("Successfully listed %d user(s)", len(users))

	return nil
}

func (v *VersityGW) CreateBucket(ctx context.Context, bucketName string, owner *string) error {
	exists, err := v.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}
	if exists {
		klog.Infof("Bucket %s already exists", bucketName)
		return nil
	}

	_, err = v.s3Client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	klog.Infof("Created bucket: %s", bucketName)

	if owner != nil {
		return v.ChangeBucketOwner(ctx, bucketName, *owner)
	}

	return nil
}

func (v *VersityGW) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := v.s3Client.DeleteBucketWithContext(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	klog.Infof("Deleted bucket: %s", bucketName)
	return nil
}

func (v *VersityGW) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	_, err := v.s3Client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}

func (v *VersityGW) GetBucketOwner(ctx context.Context, bucketName string) (string, error) {
	url := fmt.Sprintf("%s/%s?acl", v.endpointURL, bucketName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	if err := v.signRequest(req, nil); err != nil {
		return "", err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get bucket ACL: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var acl struct {
		Owner struct {
			ID string `xml:"ID"`
		} `xml:"Owner"`
	}

	if err := xml.Unmarshal(body, &acl); err != nil {
		return "", err
	}

	return acl.Owner.ID, nil
}

func (v *VersityGW) ChangeBucketOwner(ctx context.Context, bucketName, newOwner string) error {
	url := fmt.Sprintf("%s/change-bucket-owner/?bucket=%s&owner=%s", v.endpointURL, bucketName, newOwner)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, nil)
	if err != nil {
		return err
	}

	if err := v.signRequest(req, nil); err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to change bucket owner: status %d, body: %s", resp.StatusCode, string(body))
	}

	klog.Infof("Changed owner of bucket %s to %s", bucketName, newOwner)
	return nil
}

func (v *VersityGW) CreateUser(ctx context.Context, accessKey, secretKey string, role *string, userID, groupID *int) error {
	exists, err := v.UserExists(ctx, accessKey)
	if err != nil {
		return err
	}
	if exists {
		klog.Infof("User %s already exists", accessKey)
		return nil
	}

	url := fmt.Sprintf("%s/create-user", v.endpointURL)

	userPayload := fmt.Sprintf(`<Account><Access>%s</Access><Secret>%s</Secret>`, accessKey, secretKey)
	if role != nil {
		userPayload += fmt.Sprintf(`<Role>%s</Role>`, *role)
	} else {
		userPayload += `<Role>user</Role>`
	}
	if userID != nil {
		userPayload += fmt.Sprintf(`<UserID>%d</UserID>`, *userID)
	}
	if groupID != nil {
		userPayload += fmt.Sprintf(`<GroupID>%d</GroupID>`, *groupID)
	}
	userPayload += `</Account>`

	payloadBytes := []byte(userPayload)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/xml")

	if err := v.signRequest(req, payloadBytes); err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create user: status %d, body: %s", resp.StatusCode, string(body))
	}

	klog.Infof("Created user: %s", accessKey)
	return nil
}

func (v *VersityGW) DeleteUser(ctx context.Context, accessKey string) error {
	url := fmt.Sprintf("%s/delete-user?access=%s", v.endpointURL, accessKey)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, nil)
	if err != nil {
		return err
	}

	if err := v.signRequest(req, nil); err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user: status %d, body: %s", resp.StatusCode, string(body))
	}

	klog.Infof("Deleted user: %s", accessKey)
	return nil
}

func (v *VersityGW) UpdateUser(ctx context.Context, accessKey string, secretKey *string, userID, groupID *int) error {
	url := fmt.Sprintf("%s/update-user?access=%s", v.endpointURL, accessKey)

	userPayload := `<MutableProps>`
	if secretKey != nil {
		userPayload += fmt.Sprintf(`<Secret>%s</Secret>`, *secretKey)
	}
	if userID != nil {
		userPayload += fmt.Sprintf(`<UserID>%d</UserID>`, *userID)
	}
	if groupID != nil {
		userPayload += fmt.Sprintf(`<GroupID>%d</GroupID>`, *groupID)
	}
	userPayload += `</MutableProps>`

	payloadBytes := []byte(userPayload)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/xml")

	if err := v.signRequest(req, payloadBytes); err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user: status %d, body: %s", resp.StatusCode, string(body))
	}

	klog.Infof("Updated user: %s", accessKey)
	return nil
}

func (v *VersityGW) UserExists(ctx context.Context, accessKey string) (bool, error) {
	users, err := v.listUsersRaw(ctx)
	if err != nil {
		return false, err
	}

	for _, user := range users {
		if user == accessKey {
			return true, nil
		}
	}

	return false, nil
}

func (v *VersityGW) listBucketsRaw(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/list-buckets", v.endpointURL)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, nil)
	if err != nil {
		return nil, err
	}

	if err := v.signRequest(req, nil); err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list buckets: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Buckets []struct {
			Name  string `xml:"Name"`
			Owner string `xml:"Owner"`
		} `xml:"Buckets"`
	}

	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	buckets := make([]string, len(result.Buckets))
	for i, b := range result.Buckets {
		buckets[i] = b.Name
	}

	return buckets, nil
}

func (v *VersityGW) listUsersRaw(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/list-users", v.endpointURL)
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, nil)
	if err != nil {
		return nil, err
	}

	if err := v.signRequest(req, nil); err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list users: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Accounts []struct {
			Access string `xml:"Access"`
		} `xml:"Accounts"`
	}

	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	users := make([]string, len(result.Accounts))
	for i, u := range result.Accounts {
		users[i] = u.Access
	}

	return users, nil
}

func (v *VersityGW) signRequest(req *http.Request, body []byte) error {
	var bodyReader io.ReadSeeker
	if body != nil {
		bodyReader = bytes.NewReader(body)
		hash := sha256.Sum256(body)
		req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(hash[:]))
	} else {
		bodyReader = bytes.NewReader([]byte{})
		req.Header.Set("X-Amz-Content-Sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
	}

	_, err := v.signer.Sign(req, bodyReader, "s3", "us-east-1", time.Now())
	return err
}
