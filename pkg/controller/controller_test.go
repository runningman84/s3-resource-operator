package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
