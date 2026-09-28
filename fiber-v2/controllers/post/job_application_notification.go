package post

import (
	"errors"
	"log"
	"net/url"
	"strings"

	"lib"
	"models"
	"models/notify"
	"post/notificationevent"
)

var errInvalidSavedJobApplication = errors.New("saved job application unavailable")

type savedJobApplicationNotification struct {
	Jaid, FirstName, LastName string
}

// Only the record read by the server after insertion may supply a notification.
// Persisting first keeps a failed notification insert from producing a live event.
func prepareSavedJobApplicationNotification(jaid string, read func(string) (savedJobApplicationNotification, error), persist func(notificationevent.Message) error) (notificationevent.Message, error) {
	if jaid == "" || read == nil || persist == nil {
		return notificationevent.Message{}, errInvalidSavedJobApplication
	}
	record, err := read(jaid)
	if err != nil {
		return notificationevent.Message{}, err
	}
	if record.Jaid != jaid || strings.TrimSpace(record.FirstName) == "" || strings.TrimSpace(record.LastName) == "" {
		return notificationevent.Message{}, errInvalidSavedJobApplication
	}
	message := notificationevent.Message{
		Message:     record.FirstName + " " + record.LastName + " tarafından bir iş başvurusu gönderildi.",
		RequestLink: "/panel/is-basvurulari/" + url.PathEscape(record.Jaid) + "?notification=true",
	}
	if err := persist(message); err != nil {
		return notificationevent.Message{}, err
	}
	return message, nil
}

func emitSavedJobApplicationNotification(jaid string, read func(string) (savedJobApplicationNotification, error), persist func(notificationevent.Message) error, publish func(notificationevent.Message) error) string {
	message, err := prepareSavedJobApplicationNotification(jaid, read, persist)
	if err != nil || publish == nil {
		return "notification_prepare"
	}
	if publish(message) != nil {
		return "notification_publish"
	}
	return ""
}

func scheduleSavedJobApplicationNotification(utilities *models.Utilities, jaid string) {
	go notifySavedJobApplication(utilities, jaid)
}

func notifySavedJobApplication(utilities *models.Utilities, jaid string) {
	orm := utilities.Orm
	stage := emitSavedJobApplicationNotification(jaid, func(id string) (savedJobApplicationNotification, error) {
		getApplication := orm.Select([]string{"jaid", "first_name", "last_name"})
		getApplication.Table("job_applications")
		getApplication.Where("jaid", "=", id)
		getApplication.Finish()
		if err := getApplication.Execute(); err != nil {
			return savedJobApplicationNotification{}, err
		}
		rows, err := getApplication.Rows()
		if err != nil {
			return savedJobApplicationNotification{}, err
		}
		if len(rows) != 1 {
			return savedJobApplicationNotification{}, errInvalidSavedJobApplication
		}
		return savedJobApplicationNotification{
			Jaid: lib.String(rows[0]["jaid"]), FirstName: lib.String(rows[0]["first_name"]), LastName: lib.String(rows[0]["last_name"]),
		}, nil
	}, func(message notificationevent.Message) error {
		insertNotification := orm.Insert(
			[]string{"message", "notification_type", "notification_level", "link"},
			[]interface{}{message.Message, "info", "ik", message.RequestLink},
		)
		insertNotification.Table("notifications")
		insertNotification.Finish()
		return insertNotification.Execute()
	}, func(message notificationevent.Message) error {
		return notificationevent.Publish(utilities.NotificationHub, message, notificationevent.CurrentRecipients(utilities.UserStatusReader, notificationevent.ApplicationRecipients(func(uid notify.UserID) (notificationevent.User, bool) {
			getRole := orm.Select([]string{"role"})
			getRole.Table("users")
			getRole.Where("uid", "=", string(uid))
			getRole.Finish()
			if getRole.Execute() != nil {
				return notificationevent.User{}, false
			}
			rows, err := getRole.Rows()
			if err != nil || len(rows) != 1 {
				return notificationevent.User{}, false
			}
			return notificationevent.User{Role: notify.Role(lib.String(rows[0]["role"]))}, true
		})))
	})
	if stage != "" {
		log.Printf("operation=AddJobApplication stage=%s", stage)
	}
}
