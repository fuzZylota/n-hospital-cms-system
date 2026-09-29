package data

import "context"

// PersonalNotificationKind identifies the saved object event, not the visual
// notification_type (info, warning, and so on).
type PersonalNotificationKind string

const (
	RequestCreated     PersonalNotificationKind = "request_created"
	ApplicationCreated PersonalNotificationKind = "application_created"
	ContactCreated     PersonalNotificationKind = "contact_created"
)

// PersonalNotification describes one event and its complete recipient set.
// The producer must resolve eligible, active UIDs from the database; a live
// WebSocket connection list is not a durable recipient set.
type PersonalNotification struct {
	Kind              PersonalNotificationKind
	SubjectID         int64
	Message           string
	NotificationType  string
	NotificationLevel string
	Link              string
	BranchID          *int64
	RecipientUIDs     []int64
}

// PersonalNotificationWriter is deliberately not wired to current producers.
type PersonalNotificationWriter interface {
	CreatePersonalNotification(context.Context, PersonalNotification) (int64, error)
}
