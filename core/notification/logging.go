package notification

import (
	"context"
	"log"
	"os"
)

// Compile-time check.
var _ NotificationPort = (*LoggingAdapter)(nil)

// LoggingAdapter is a NotificationPort implementation that logs
// notifications to stdout. It is the Sprint 0 default adapter;
// real channel adapters (Discord, Slack, etc.) replace it in
// later sprints.
type LoggingAdapter struct {
	logger *log.Logger
}

// NewLoggingAdapter creates a LoggingAdapter that writes to stdout.
func NewLoggingAdapter() *LoggingAdapter {
	return &LoggingAdapter{
		logger: log.New(os.Stdout, "[notification] ", log.LstdFlags),
	}
}

// Send logs the notification payload.
func (l *LoggingAdapter) Send(_ context.Context, notification NotificationPayload) error {
	if notification.Error != "" {
		l.logger.Printf("FAILED intent=%s type=%s error=%s trace=%s",
			notification.IntentID, notification.Type, notification.Error, notification.TraceID)
		return nil
	}
	l.logger.Printf("SENT intent=%s type=%s summary=%s trace=%s",
		notification.IntentID, notification.Type, notification.Summary, notification.TraceID)
	return nil
}
