package query

// Metric names for ProjectionEngine observability.
// These are defined as constants so a future sprint can wire them into
// OTel/prometheus without renaming.
const (
	MetricProjectionDuration     = "projection_duration_ms"
	MetricProjectionSuccessTotal = "projection_success_total"
	MetricProjectionFailureTotal = "projection_failure_total"
	MetricProjectionLag          = "projection_lag_events"
	MetricActiveSubscriptions    = "projection_active_subscriptions"
)
