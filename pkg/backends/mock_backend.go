package backends

import (
	"context"
	"fmt"
	"sync"
)

// MockBackend is a mock implementation of the Backend interface for testing
type MockBackend struct {
	mu sync.Mutex

	EndpointURL string
	Buckets     map[string]string // bucketName -> owner
	Users       map[string]*MockUser

	// Error injection
	TestConnectionError    error
	CreateBucketError      error
	DeleteBucketError      error
	BucketExistsError      error
	GetBucketOwnerError    error
	ChangeBucketOwnerError error
	CreateUserError        error
	DeleteUserError        error
	UpdateUserError        error
	UserExistsError        error

	// Call tracking
	TestConnectionCalls    int
	CreateBucketCalls      int
	DeleteBucketCalls      int
	BucketExistsCalls      int
	GetBucketOwnerCalls    int
	ChangeBucketOwnerCalls int
	CreateUserCalls        int
	DeleteUserCalls        int
	UpdateUserCalls        int
	UserExistsCalls        int
}

type MockUser struct {
	AccessKey string
	SecretKey string
	Role      *string
	UserID    *int
	GroupID   *int
}

func NewMockBackend(endpointURL string) *MockBackend {
	return &MockBackend{
		EndpointURL: endpointURL,
		Buckets:     make(map[string]string),
		Users:       make(map[string]*MockUser),
	}
}

func (m *MockBackend) TestConnection(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TestConnectionCalls++
	return m.TestConnectionError
}

func (m *MockBackend) CreateBucket(ctx context.Context, bucketName string, owner *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateBucketCalls++

	if m.CreateBucketError != nil {
		return m.CreateBucketError
	}

	if _, exists := m.Buckets[bucketName]; exists {
		return fmt.Errorf("bucket %s already exists", bucketName)
	}

	ownerName := ""
	if owner != nil {
		ownerName = *owner
	}
	m.Buckets[bucketName] = ownerName
	return nil
}

func (m *MockBackend) DeleteBucket(ctx context.Context, bucketName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteBucketCalls++

	if m.DeleteBucketError != nil {
		return m.DeleteBucketError
	}

	if _, exists := m.Buckets[bucketName]; !exists {
		return fmt.Errorf("bucket %s does not exist", bucketName)
	}

	delete(m.Buckets, bucketName)
	return nil
}

func (m *MockBackend) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BucketExistsCalls++

	if m.BucketExistsError != nil {
		return false, m.BucketExistsError
	}

	_, exists := m.Buckets[bucketName]
	return exists, nil
}

func (m *MockBackend) GetBucketOwner(ctx context.Context, bucketName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GetBucketOwnerCalls++

	if m.GetBucketOwnerError != nil {
		return "", m.GetBucketOwnerError
	}

	owner, exists := m.Buckets[bucketName]
	if !exists {
		return "", fmt.Errorf("bucket %s does not exist", bucketName)
	}

	return owner, nil
}

func (m *MockBackend) ChangeBucketOwner(ctx context.Context, bucketName, newOwner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ChangeBucketOwnerCalls++

	if m.ChangeBucketOwnerError != nil {
		return m.ChangeBucketOwnerError
	}

	if _, exists := m.Buckets[bucketName]; !exists {
		return fmt.Errorf("bucket %s does not exist", bucketName)
	}

	m.Buckets[bucketName] = newOwner
	return nil
}

func (m *MockBackend) CreateUser(ctx context.Context, accessKey, secretKey string, role *string, userID, groupID *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreateUserCalls++

	if m.CreateUserError != nil {
		return m.CreateUserError
	}

	if _, exists := m.Users[accessKey]; exists {
		return fmt.Errorf("user %s already exists", accessKey)
	}

	m.Users[accessKey] = &MockUser{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Role:      role,
		UserID:    userID,
		GroupID:   groupID,
	}
	return nil
}

func (m *MockBackend) DeleteUser(ctx context.Context, accessKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeleteUserCalls++

	if m.DeleteUserError != nil {
		return m.DeleteUserError
	}

	if _, exists := m.Users[accessKey]; !exists {
		return fmt.Errorf("user %s does not exist", accessKey)
	}

	delete(m.Users, accessKey)
	return nil
}

func (m *MockBackend) UpdateUser(ctx context.Context, accessKey string, secretKey *string, userID, groupID *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateUserCalls++

	if m.UpdateUserError != nil {
		return m.UpdateUserError
	}

	user, exists := m.Users[accessKey]
	if !exists {
		return fmt.Errorf("user %s does not exist", accessKey)
	}

	if secretKey != nil {
		user.SecretKey = *secretKey
	}
	if userID != nil {
		user.UserID = userID
	}
	if groupID != nil {
		user.GroupID = groupID
	}

	return nil
}

func (m *MockBackend) UserExists(ctx context.Context, accessKey string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UserExistsCalls++

	if m.UserExistsError != nil {
		return false, m.UserExistsError
	}

	_, exists := m.Users[accessKey]
	return exists, nil
}

func (m *MockBackend) GetEndpointURL() string {
	return m.EndpointURL
}

// Reset clears all state for testing
func (m *MockBackend) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Buckets = make(map[string]string)
	m.Users = make(map[string]*MockUser)

	m.TestConnectionError = nil
	m.CreateBucketError = nil
	m.DeleteBucketError = nil
	m.BucketExistsError = nil
	m.GetBucketOwnerError = nil
	m.ChangeBucketOwnerError = nil
	m.CreateUserError = nil
	m.DeleteUserError = nil
	m.UpdateUserError = nil
	m.UserExistsError = nil

	m.TestConnectionCalls = 0
	m.CreateBucketCalls = 0
	m.DeleteBucketCalls = 0
	m.BucketExistsCalls = 0
	m.GetBucketOwnerCalls = 0
	m.ChangeBucketOwnerCalls = 0
	m.CreateUserCalls = 0
	m.DeleteUserCalls = 0
	m.UpdateUserCalls = 0
	m.UserExistsCalls = 0
}
