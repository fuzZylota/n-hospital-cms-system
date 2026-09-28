package post

import (
	"context"
	"encoding/json"
	"time"

	"lib"
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
	return authorizeNotificationTextWithState(utilities.UserStatusReader, func(uid notify.UserID) (notify.BranchID, bool) {
		if utilities.Orm == nil {
			return "", false
		}
		query := utilities.Orm.Select([]string{"sid"})
		query.Table("users")
		query.Where("uid", "=", string(uid))
		query.Finish()
		if query.Execute() != nil {
			return "", false
		}
		rows, err := query.Rows()
		if err != nil || len(rows) != 1 {
			return "", false
		}
		return notify.BranchID(lib.String(rows[0]["sid"])), true
	}, uid, event, payload)
}

func authorizeNotificationTextWithState(reader data.UserStatusReader, branchOf func(notify.UserID) (notify.BranchID, bool), uid notify.UserID, event notificationws.Event, payload []byte) bool {
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
		var frame struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(payload, &frame) != nil || frame.Message == "" {
			return false
		}
		var request struct {
			Sid string `json:"sid"`
		}
		if json.Unmarshal([]byte(frame.Message), &request) != nil || request.Sid == "" {
			return false
		}
		if role == "santral" {
			if branchOf == nil {
				return false
			}
			branch, ok := branchOf(uid)
			return ok && notificationevent.Appointment(role, branch, notify.BranchID(request.Sid))
		}
		return notificationevent.Appointment(role, "", notify.BranchID(request.Sid))
	default:
		return false
	}
}
