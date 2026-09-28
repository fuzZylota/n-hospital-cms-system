// Package notificationevent holds the existing wire shapes and recipient rules.
// It does not authenticate users or treat metadata snapshots as current roles.
package notificationevent

import (
	"context"
	"encoding/json"
	"time"

	"lib/userstatus"
	"models/data"
	"models/notify"
)

type Message struct {
	Uid         string `json:"uid"`
	Message     string `json:"message"`
	RequestLink string `json:"request_link"`
}

type NewRequest struct {
	Type                string `json:"type"`
	Rrid                string `json:"rrid"`
	PatientFirstName    string `json:"patient_first_name"`
	PatientLastName     string `json:"patient_last_name"`
	PatientPhone        string `json:"patient_phone"`
	Message             string `json:"message"`
	CreatedAt           string `json:"created_at"`
	Status              string `json:"status"`
	SubeName            string `json:"sube_name"`
	Sid                 string `json:"sid"`
	NotificationMessage string `json:"notification_message,omitempty"`
	NotificationType    string `json:"notification_type,omitempty"`
	RequestLink         string `json:"request_link,omitempty"`
}

type Deleted struct {
	Type string `json:"type"`
	Rrid string `json:"rrid"`
}

type Status struct {
	Type      string `json:"type"`
	Rrid      string `json:"rrid"`
	NewStatus string `json:"new_status"`
}

type failure uint8

const errPublish failure = 1

func (failure) Error() string { return "notification: publication failed" }

// Publish serializes once. Best-effort callers log a fixed stage and preserve
// their primary database result. No raw marshal/transport cause is retained.
func Publish(hub notify.Hub, message any, predicate notify.Predicate) error {
	payload, err := json.Marshal(message)
	if err != nil || hub == nil {
		return errPublish
	}
	_, err = hub.Broadcast("notifications", payload, predicate)
	if err != nil {
		return errPublish
	}
	return nil
}

func Recipient(client notify.Client) bool {
	return client.Metadata.UserID != "" && client.Metadata.Protocol == "kullanici"
}

// CurrentRecipients adds the current account decision to each existing event rule.
// Admission and queued delivery call it outside the hub registry lock.
func CurrentRecipients(reader data.UserStatusReader, rule notify.Predicate) notify.Predicate {
	return func(client notify.Client) bool {
		if !Recipient(client) || rule == nil {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		expire, _, err := userstatus.ShouldExpireAuthCookies(ctx, reader, string(client.Metadata.UserID))
		return err == nil && !expire && rule(client)
	}
}

func Appointment(role notify.Role, branch, target notify.BranchID) bool {
	return role == "admin" || role == "moderator" || (role == "santral" && branch == target)
}

func Application(role notify.Role) bool {
	return role == "admin" || role == "moderator" || role == "ik"
}

func GlobalRequestRole(role notify.Role) bool { return role == "admin" || role == "moderator" }

// Lookups are fresh server-side queries. Metadata.Role/BranchID are never used
// as authority. A false lookup/permission result includes database failures.
type User struct {
	Role   notify.Role
	Branch notify.BranchID
}
type Lookup func(notify.UserID) (User, bool)
type Permission func(notify.UserID, notify.BranchID) bool

func AppointmentRecipients(target notify.BranchID, lookup Lookup) notify.Predicate {
	return func(c notify.Client) bool {
		if !Recipient(c) {
			return false
		}
		user, ok := lookup(c.Metadata.UserID)
		return ok && Appointment(user.Role, user.Branch, target)
	}
}

func ApplicationRecipients(lookup Lookup) notify.Predicate {
	return func(c notify.Client) bool {
		if !Recipient(c) {
			return false
		}
		user, ok := lookup(c.Metadata.UserID)
		return ok && Application(user.Role)
	}
}

func RequestRecipients(target notify.BranchID, lookup Lookup, permission Permission) notify.Predicate {
	return func(c notify.Client) bool {
		if !Recipient(c) {
			return false
		}
		user, ok := lookup(c.Metadata.UserID)
		return ok && (GlobalRequestRole(user.Role) || permission(c.Metadata.UserID, target))
	}
}
