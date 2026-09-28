package post

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lib"
	"models/data"
	"models/notify"
	"post/notificationevent"
)

func TestF05SavedJobApplicationNotification(t *testing.T) {
	readCount, persistCount, publishCount := 0, 0, 0
	var stored notificationevent.Message
	stage := emitSavedJobApplicationNotification("saved-7", func(id string) (savedJobApplicationNotification, error) {
		readCount++
		if id != "saved-7" {
			t.Fatal("lookup used an untrusted ID")
		}
		return savedJobApplicationNotification{Jaid: id, FirstName: "Yapay", LastName: "Kisi"}, nil
	}, func(event notificationevent.Message) error {
		persistCount++
		stored = event
		return nil
	}, func(event notificationevent.Message) error {
		publishCount++
		if event != stored {
			t.Fatal("live event differs from persisted notification")
		}
		return nil
	})
	if stage != "" || readCount != 1 || persistCount != 1 || publishCount != 1 ||
		stored.Message != "Yapay Kisi tarafından bir iş başvurusu gönderildi." ||
		stored.RequestLink != "/panel/is-basvurulari/saved-7?notification=true" {
		t.Fatal("saved application did not produce one consistent notification")
	}
}

func TestF05SavedJobApplicationRejectsUnverifiedOrFailedPersistence(t *testing.T) {
	for _, tc := range []struct {
		name       string
		read       func(string) (savedJobApplicationNotification, error)
		persistErr error
	}{
		{"database read error", func(string) (savedJobApplicationNotification, error) {
			return savedJobApplicationNotification{}, errors.New("private read error")
		}, nil},
		{"missing record", func(string) (savedJobApplicationNotification, error) { return savedJobApplicationNotification{}, nil }, nil},
		{"foreign record", func(string) (savedJobApplicationNotification, error) {
			return savedJobApplicationNotification{Jaid: "other", FirstName: "Wrong", LastName: "Person"}, nil
		}, nil},
		{"insert error", func(id string) (savedJobApplicationNotification, error) {
			return savedJobApplicationNotification{Jaid: id, FirstName: "Yapay", LastName: "Kisi"}, nil
		}, errors.New("private insert error")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			persistCount, publishCount := 0, 0
			stage := emitSavedJobApplicationNotification("saved-7", tc.read, func(notificationevent.Message) error {
				persistCount++
				return tc.persistErr
			}, func(notificationevent.Message) error {
				publishCount++
				return nil
			})
			if stage != "notification_prepare" || publishCount != 0 || (tc.persistErr == nil && persistCount != 0) || (tc.persistErr != nil && persistCount != 1) {
				t.Fatal("notification side effects occurred in the wrong stage")
			}
		})
	}
}

func TestF05SavedJobApplicationPublishFailureHasSafeStage(t *testing.T) {
	persisted, published := 0, 0
	stage := emitSavedJobApplicationNotification("saved/7", func(id string) (savedJobApplicationNotification, error) {
		return savedJobApplicationNotification{Jaid: id, FirstName: "Yapay", LastName: "Kisi"}, nil
	}, func(message notificationevent.Message) error {
		persisted++
		if message.RequestLink != "/panel/is-basvurulari/saved%2F7?notification=true" {
			t.Fatal("application ID was not escaped in panel link")
		}
		return nil
	}, func(notificationevent.Message) error {
		published++
		return errors.New("private transport error")
	})
	if stage != "notification_publish" || persisted != 1 || published != 1 {
		t.Fatal("publish failure did not preserve the persisted notification boundary")
	}
}

func TestF05EmailErrorPrecedesSavedNotificationWithoutChangingSuccess(t *testing.T) {
	steps := []string{"record_insert"}
	emailErr := lib.DeliverEmailAfterPersistence(func() error {
		steps = append(steps, "email_error")
		return errors.New("synthetic mail failure")
	}, nil)
	if emailErr == nil {
		t.Fatal("fake email failure was lost")
	}
	stage := emitSavedJobApplicationNotification("saved-7", func(id string) (savedJobApplicationNotification, error) {
		steps = append(steps, "record_read")
		return savedJobApplicationNotification{Jaid: id, FirstName: "Yapay", LastName: "Kisi"}, nil
	}, func(notificationevent.Message) error {
		steps = append(steps, "notification_insert")
		return nil
	}, func(notificationevent.Message) error {
		steps = append(steps, "live_event")
		return nil
	})
	if stage != "" || strings.Join(steps, ",") != "record_insert,email_error,record_read,notification_insert,live_event" {
		t.Fatal("email failure changed the post-persistence notification sequence")
	}
}

func TestF05ApplicationRecipientsKeepCurrentRoles(t *testing.T) {
	for _, tc := range []struct {
		role   string
		active bool
		want   bool
	}{
		{"admin", true, true}, {"moderator", true, true}, {"ik", true, true},
		{"santral", true, false}, {"ik", false, false},
	} {
		t.Run(tc.role+map[bool]string{true: " active", false: " inactive"}[tc.active], func(t *testing.T) {
			status := &sec003cStatus{status: data.UserStatus{Found: true, Active: tc.active, Role: tc.role}}
			predicate := notificationevent.CurrentRecipients(status, notificationevent.ApplicationRecipients(func(uid notify.UserID) (notificationevent.User, bool) {
				if uid != "42" {
					return notificationevent.User{}, false
				}
				return notificationevent.User{Role: notify.Role(tc.role)}, true
			}))
			if got := predicate(notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "kullanici"}}); got != tc.want {
				t.Fatal("application recipient policy changed")
			}
		})
	}
}

func TestF05JobApplicationMailBeforeNotificationWiring(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("post.go"))
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "func AddJobApplication(")
	end := strings.Index(string(source), "func DeleteJobApplication(")
	if start < 0 || end <= start {
		t.Fatal("job application handler unavailable")
	}
	handler := string(source[start:end])
	mail := strings.Index(handler, "lib.DeliverEmailAfterPersistence(")
	notification := strings.Index(handler, "defer scheduleSavedJobApplicationNotification(utilities, lid)")
	success := strings.LastIndex(handler, `"status":  201`)
	if mail < 0 || notification < 0 || notification >= mail || success <= mail ||
		!strings.Contains(handler, "insertReq.LastInsertId()") {
		t.Fatal("mail, saved-record notification and HTTP partial-success sequence changed")
	}
	helper, err := os.ReadFile("job_application_notification.go")
	if err != nil || !strings.Contains(string(helper), "emitSavedJobApplicationNotification(jaid") ||
		!strings.Contains(string(helper), "notificationevent.ApplicationRecipients(") {
		t.Fatal("server notification helper is not wired to the saved application and recipients")
	}
}
