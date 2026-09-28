package post

import (
	"context"
	"time"

	"lib/userstatus"
	"models"
	"models/data"
	"models/notify"
	"post/notificationevent"
	"post/notificationws"
)

// authorizeNotificationText checks current account state after WebSocket upgrade.
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
	expire, currentRole, err := userstatus.ShouldExpireAuthCookies(ctx, reader, string(uid))
	if err != nil || expire {
		return false
	}
	role := notify.Role(currentRole)
	switch event {
	case notificationws.Subscriber, notificationws.Contact:
		return true
	case notificationws.Application:
		return notificationevent.Application(role)
	case notificationws.Appointment:
		return false // Only AddRandevuRequest may produce request notifications.
	default:
		return false
	}
}
