package observability

// Metric naming conventions — format: <domain>.<resource>.<operation>.<type>
// Use these constants to ensure consistent metric names across the codebase.
const (
	MetricRequestsTotal    = "app.requests.total"
	MetricRequestsDuration = "app.requests.duration_ms"

	MetricS3UploadsTotal   = "infra.s3.uploads.total"
	MetricS3DownloadsTotal = "infra.s3.downloads.total"

	MetricEmailSentTotal   = "infra.email.sent.total"
	MetricEmailFailedTotal = "infra.email.failed.total"

	MetricCacheHitsTotal   = "infra.cache.hits.total"
	MetricCacheMissesTotal = "infra.cache.misses.total"

	MetricOutboxProcessedTotal = "outbox.processed.total"
	MetricOutboxFailedTotal    = "outbox.failed.total"
	MetricOutboxLatencyMS      = "outbox.latency_ms"
)

// Attribute key conventions — format: <namespace>.<attribute>
// Use these constants for span attributes and structured log fields.
const (
	AttrRequestID     = "request.id"
	AttrUserID        = "user.id"
	AttrFileKey       = "file.key"
	AttrFileBucket    = "file.bucket"
	AttrEmailTo       = "email.to"
	AttrQueueTopic    = "queue.topic"
	AttrErrorCode     = "error.code"
	AttrErrorCategory = "error.category"
)
