package randevular

import (
	"errors"
	"strings"
	"testing"
	"time"

	"post/notificationevent"
)

func TestSavedRequestNotificationUsesOnlyReadRecord(t *testing.T) {
	var order []string
	var saved notificationevent.NewRequest
	read := func(id string) (savedRequestNotification, error) {
		order = append(order, "read")
		if id != "saved-1" {
			t.Fatal("unexpected lookup ID")
		}
		return savedRequestNotification{
			Rrid: "saved-1", Sid: "branch-b", PatientFirstName: "Yapay", PatientLastName: "Kisi",
			PatientPhone: "000", PatientEmail: "synthetic@example.invalid", Message: "Yapay sikayet",
			PreferredDate: time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC),
		}, nil
	}
	event, err := prepareSavedRequestNotification("saved-1", read, func(event notificationevent.NewRequest) error {
		order = append(order, "persist")
		saved = event
		return nil
	})
	if err != nil || strings.Join(order, ",") != "read,persist" {
		t.Fatalf("unexpected saved request delivery: %v, %v", err, order)
	}
	if saved != event || saved.Sid != "branch-b" || saved.RequestLink != "/panel/randevu-talepleri/saved-1?notification=true" ||
		saved.NotificationType != "info" || !strings.Contains(saved.NotificationMessage, "Yapay Kisi") ||
		strings.Contains(saved.NotificationMessage, "Yapay sikayet") {
		t.Fatal("notification not derived consistently from saved request")
	}
}

func TestSavedRequestNotificationFailsWithoutVerifiedRecord(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
		read func(string) (savedRequestNotification, error)
	}{
		{"fake ID", "missing", func(string) (savedRequestNotification, error) {
			return savedRequestNotification{}, errInvalidSavedRequest
		}},
		{"lookup error", "saved-1", func(string) (savedRequestNotification, error) {
			return savedRequestNotification{}, errors.New("query failed")
		}},
		{"different record or branch claim", "saved-1", func(string) (savedRequestNotification, error) {
			return savedRequestNotification{Rrid: "other-branch-id", Sid: "branch-b"}, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			persisted := 0
			event, err := prepareSavedRequestNotification(tc.id, tc.read, func(notificationevent.NewRequest) error { persisted++; return nil })
			if err == nil || persisted != 0 || event != (notificationevent.NewRequest{}) {
				t.Fatalf("unverified record produced notification: err=%v persist=%d", err, persisted)
			}
		})
	}
}

func TestSavedRequestNotificationDoesNotBroadcastFailedPersistence(t *testing.T) {
	read := func(string) (savedRequestNotification, error) { return savedRequestNotification{Rrid: "saved-1"}, nil }
	event, err := prepareSavedRequestNotification("saved-1", read, func(notificationevent.NewRequest) error { return errors.New("insert failed") })
	if err == nil || event != (notificationevent.NewRequest{}) {
		t.Fatal("failed persistence was broadcast")
	}
}
