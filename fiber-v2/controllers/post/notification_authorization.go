package post

import (
	"context"
	"time"

	"lib/userstatus"
	"models"
	"models/data"
	"models/notify"
	"post/notificationws"
)

// authorizeNotificationText is called after upgrade for each inbound text.
// It does not trust the upgrade's role or a UID supplied in the frame.
func authorizeNotificationText(utilities *models.Utilities, uid notify.UserID, event notificationws.Event, payload []byte) bool {
	if utilities == nil {
		return false
	}
	return authorizeNotificationTextWithState(utilities.UserStatusReader, uid, event, payload)
}

func authorizeNotificationTextWithState(reader data.UserStatusReader, uid notify.UserID, event notificationws.Event, payload []byte) bool {
	if uid == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	expire, _, err := userstatus.ShouldExpireAuthCookies(ctx, reader, string(uid))
	if err != nil || expire {
		return false
	}
	switch event {
	case notificationws.Subscriber:
		return true
	case notificationws.Contact:
		return false // Only AddContactRequest may produce contact notifications.
	case notificationws.Application:
		return false // Only AddJobApplication may produce application notifications.
	case notificationws.Appointment:
		return false // Only AddRandevuRequest may produce request notifications.
	default:
		return false
	}
}
