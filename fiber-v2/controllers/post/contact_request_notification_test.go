package post

import (
	"errors"
	"os"
	"strings"
	"testing"

	"lib"
	"models/data"
	"models/notify"
	"post/notificationevent"
)

func TestF05SavedContactRequestOneNotificationAndEvent(t *testing.T) {
	readCount, persistCount, publishCount := 0, 0, 0
	var stored notificationevent.Message
	stage := emitSavedContactRequestNotification("42", func(id string) (string, error) {
		readCount++
		if id != "42" {
			t.Fatal("read did not use the inserted ID")
		}
		return id, nil
	}, func(message notificationevent.Message) error {
		persistCount++
		stored = message
		return nil
	}, func(message notificationevent.Message) error {
		publishCount++
		if message != stored {
			t.Fatal("published event differs from the saved notification")
		}
		return nil
	})
	if stage != "" || readCount != 1 || persistCount != 1 || publishCount != 1 ||
		stored.Message != "Yeni bir iletişim talebi alındı." ||
		stored.RequestLink != "/panel/iletisim-istekleri/42?notification=true" || stored.Uid != "" {
		t.Fatal("saved contact request did not produce one general notification and event")
	}
}

func TestF05SavedContactRequestFailureStopsSideEffects(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id         string
		read       func(string) (string, error)
		persistErr error
		wantRead   int
		wantInsert int
	}{
		{"missing ID", "", func(id string) (string, error) { return id, nil }, nil, 0, 0},
		{"untrusted path ID", "42/other", func(id string) (string, error) { return id, nil }, nil, 0, 0},
		{"read failure", "42", func(string) (string, error) { return "", errors.New("private read error") }, nil, 1, 0},
		{"missing record", "42", func(string) (string, error) { return "", nil }, nil, 1, 0},
		{"different record", "42", func(string) (string, error) { return "43", nil }, nil, 1, 0},
		{"insert failure", "42", func(id string) (string, error) { return id, nil }, errors.New("private insert error"), 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readCount, persistCount, publishCount := 0, 0, 0
			stage := emitSavedContactRequestNotification(tc.id, func(id string) (string, error) {
				readCount++
				return tc.read(id)
			}, func(notificationevent.Message) error {
				persistCount++
				return tc.persistErr
			}, func(notificationevent.Message) error {
				publishCount++
				return nil
			})
			if stage != "notification_prepare" || readCount != tc.wantRead || persistCount != tc.wantInsert || publishCount != 0 {
				t.Fatal("unverified contact request produced a notification or live event")
			}
		})
	}
}

func TestF05SavedContactRequestPublishFailureHasSafeStage(t *testing.T) {
	stage := emitSavedContactRequestNotification("42", func(id string) (string, error) { return id, nil },
		func(notificationevent.Message) error { return nil },
		func(notificationevent.Message) error { return errors.New("private transport error") })
	if stage != "notification_publish" {
		t.Fatal("publish failure did not produce a fixed stage")
	}
}

func TestF05ContactAdminRecipients(t *testing.T) {
	for _, tc := range []struct {
		role   string
		active bool
		want   bool
	}{
		{"admin", true, true}, {"moderator", true, false}, {"santral", true, false},
		{"ik", true, false}, {"admin", false, false},
	} {
		t.Run(tc.role+map[bool]string{true: " active", false: " inactive"}[tc.active], func(t *testing.T) {
			status := &sec003cStatus{status: data.UserStatus{Found: true, Active: tc.active, Role: tc.role}}
			predicate := contactAdminRecipients(status)
			if got := predicate(notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "kullanici", Role: "admin"}}); got != tc.want {
				t.Fatal("contact recipient rule admitted the wrong current role")
			}
		})
	}
	if !contactAdminRecipients(&sec003cStatus{status: data.UserStatus{Found: true, Active: true, Role: "admin"}})(notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "kullanici", Role: "moderator"}}) {
		t.Fatal("stale metadata role overrode the current admin role")
	}
	if contactAdminRecipients(&sec003cStatus{status: data.UserStatus{Found: true, Active: true, Role: "admin"}})(notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "iletisim", Role: "admin"}}) {
		t.Fatal("non-subscriber protocol received a contact notification")
	}
	if contactAdminRecipients(nil)(notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "kullanici"}}) {
		t.Fatal("missing current-account reader admitted a contact recipient")
	}
	if contactAdminRecipients(&sec003cStatus{err: errors.New("private lookup error")})(notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "kullanici"}}) {
		t.Fatal("current-account lookup failure admitted a contact recipient")
	}
}

func TestF05ContactMailBeforeNotificationWiring(t *testing.T) {
	source, err := os.ReadFile("post.go")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "func AddContactRequest(")
	end := strings.Index(string(source), "func DeleteContactRequest(")
	if start < 0 || end <= start {
		t.Fatal("contact handler unavailable")
	}
	handler := string(source[start:end])
	insert := strings.Index(handler, "InsertContactRequest.LastInsertId()")
	scheduled := strings.Index(handler, "defer scheduleSavedContactRequestNotification(utilities, crid)")
	mail := strings.Index(handler, "lib.DeliverEmailAfterPersistence(")
	success := strings.LastIndex(handler, `"status":  201`)
	if insert < 0 || scheduled <= insert || mail <= scheduled || success <= mail {
		t.Fatal("contact insert, deferred notification, email and success order changed")
	}
	if !strings.Contains(string(source), "}, nil, websocket.Config{") {
		t.Fatal("WebSocket notification producer callback was restored")
	}
	helper, err := os.ReadFile("contact_request_notification.go")
	if err != nil || !strings.Contains(string(helper), "emitSavedContactRequestNotification(crid") ||
		!strings.Contains(string(helper), `[]interface{}{message.Message, "info", "admin", message.RequestLink}`) ||
		!strings.Contains(string(helper), "contactAdminRecipients(utilities.UserStatusReader)") {
		t.Fatal("saved contact request is not bound to an admin notification")
	}
}

func TestF05ContactEmailErrorBeforeNotification(t *testing.T) {
	steps := []string{"record_insert"}
	mailErr := lib.DeliverEmailAfterPersistence(func() error {
		steps = append(steps, "email_error")
		return errors.New("synthetic mail failure")
	}, nil)
	if mailErr == nil {
		t.Fatal("fake email failure was lost")
	}
	steps = append(steps, "http_success")
	stage := emitSavedContactRequestNotification("42", func(id string) (string, error) {
		steps = append(steps, "record_read")
		return id, nil
	}, func(notificationevent.Message) error {
		steps = append(steps, "notification_insert")
		return nil
	}, func(notificationevent.Message) error {
		steps = append(steps, "live_event")
		return nil
	})
	if stage != "" || strings.Join(steps, ",") != "record_insert,email_error,http_success,record_read,notification_insert,live_event" {
		t.Fatal("email failure changed the saved contact notification or success sequence")
	}
}
