package metrics

import (
	"testing"
	"time"
)

func TestStartTimer(t *testing.T) {
	timer := StartTimer()
	time.Sleep(10 * time.Millisecond)

	duration := timer.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", duration)
	}
}

func TestMetricsIncrement(t *testing.T) {
	// These should not panic
	IncrementSecretsProcessed()
	IncrementErrors()
	IncrementUsersCreated()
	IncrementUsersDeleted()
	IncrementUsersUpdated()
	IncrementBucketsCreated()
	IncrementBucketOwnersChanged()
}

func TestMetricsDuration(t *testing.T) {
	timer := StartTimer()
	time.Sleep(5 * time.Millisecond)

	// These should not panic
	RecordSyncDuration(timer)
	RecordHandleSecretDuration(timer)
}
