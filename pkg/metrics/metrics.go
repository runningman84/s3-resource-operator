package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Secrets processing metrics
	secretsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_secrets_processed_total",
		Help: "Total number of secrets processed",
	})

	errorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_errors_total",
		Help: "Total number of errors encountered",
	})

	syncDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "s3_operator_sync_duration_seconds",
		Help:    "Duration of a sync cycle",
		Buckets: prometheus.DefBuckets,
	})

	handleSecretDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "s3_operator_handle_secret_duration_seconds",
		Help:    "Duration of handling a secret",
		Buckets: prometheus.DefBuckets,
	})

	// Resource operation metrics
	usersCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_users_created_total",
		Help: "Total number of users created",
	})

	usersDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_users_deleted_total",
		Help: "Total number of users deleted",
	})

	usersUpdated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_users_updated_total",
		Help: "Total number of users updated",
	})

	bucketsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_buckets_created_total",
		Help: "Total number of buckets created",
	})

	bucketOwnersChanged = promauto.NewCounter(prometheus.CounterOpts{
		Name: "s3_operator_bucket_owners_changed_total",
		Help: "Total number of bucket owners changed",
	})
)

// Register initializes all metrics (called automatically by promauto)
func Register() {
	// Metrics are auto-registered by promauto package
}

// Timer is a helper for timing operations
type Timer struct {
	start time.Time
}

// StartTimer creates a new timer
func StartTimer() Timer {
	return Timer{start: time.Now()}
}

// Duration returns the elapsed time since the timer was started
func (t Timer) Duration() time.Duration {
	return time.Since(t.start)
}

// IncrementSecretsProcessed increments the secrets processed counter
func IncrementSecretsProcessed() {
	secretsProcessed.Inc()
}

// IncrementErrors increments the errors counter
func IncrementErrors() {
	errorsTotal.Inc()
}

// RecordSyncDuration records the duration of a sync cycle
func RecordSyncDuration(timer Timer) {
	syncDuration.Observe(timer.Duration().Seconds())
}

// RecordHandleSecretDuration records the duration of handling a secret
func RecordHandleSecretDuration(timer Timer) {
	handleSecretDuration.Observe(timer.Duration().Seconds())
}

// IncrementUsersCreated increments the users created counter
func IncrementUsersCreated() {
	usersCreated.Inc()
}

// IncrementUsersDeleted increments the users deleted counter
func IncrementUsersDeleted() {
	usersDeleted.Inc()
}

// IncrementUsersUpdated increments the users updated counter
func IncrementUsersUpdated() {
	usersUpdated.Inc()
}

// IncrementBucketsCreated increments the buckets created counter
func IncrementBucketsCreated() {
	bucketsCreated.Inc()
}

// IncrementBucketOwnersChanged increments the bucket owners changed counter
func IncrementBucketOwnersChanged() {
	bucketOwnersChanged.Inc()
}
