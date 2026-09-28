package post

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"lib"
	"lib/userstatus"
	"models"
	"models/data"
	"models/notify"
	"post/notificationevent"
)

var errInvalidSavedContactRequest = errors.New("saved contact request unavailable")

// A contact notification contains no form fields. The link uses only the
// canonical ID returned by a server-side read after the successful insert.
func prepareSavedContactRequestNotification(crid string, read func(string) (string, error), persist func(notificationevent.Message) error) (notificationevent.Message, error) {
	if crid == "" || read == nil || persist == nil {
		return notificationevent.Message{}, errInvalidSavedContactRequest
	}
	id, err := strconv.ParseUint(crid, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != crid {
		return notificationevent.Message{}, errInvalidSavedContactRequest
	}
	savedID, err := read(crid)
	if err != nil {
		return notificationevent.Message{}, err
	}
	if savedID != crid {
		return notificationevent.Message{}, errInvalidSavedContactRequest
	}
	message := notificationevent.Message{
		Message:     "Yeni bir iletişim talebi alındı.",
		RequestLink: "/panel/iletisim-istekleri/" + savedID + "?notification=true",
	}
	if err := persist(message); err != nil {
		return notificationevent.Message{}, err
	}
	return message, nil
}

func emitSavedContactRequestNotification(crid string, read func(string) (string, error), persist func(notificationevent.Message) error, publish func(notificationevent.Message) error) string {
	message, err := prepareSavedContactRequestNotification(crid, read, persist)
	if err != nil || publish == nil {
		return "notification_prepare"
	}
	if publish(message) != nil {
		return "notification_publish"
	}
	return ""
}

func scheduleSavedContactRequestNotification(utilities *models.Utilities, crid string) {
	go notifySavedContactRequest(utilities, crid)
}

func notifySavedContactRequest(utilities *models.Utilities, crid string) {
	orm := utilities.Orm
	stage := emitSavedContactRequestNotification(crid, func(id string) (string, error) {
		getRequest := orm.Select([]string{"crid"})
		getRequest.Table("contact_requests")
		getRequest.Where("crid", "=", id)
		getRequest.Finish()
		if err := getRequest.Execute(); err != nil {
			return "", err
		}
		rows, err := getRequest.Rows()
		if err != nil {
			return "", err
		}
		if len(rows) != 1 {
			return "", errInvalidSavedContactRequest
		}
		return lib.String(rows[0]["crid"]), nil
	}, func(message notificationevent.Message) error {
		insertNotification := orm.Insert(
			[]string{"message", "notification_type", "notification_level", "link"},
			[]interface{}{message.Message, "info", "admin", message.RequestLink},
		)
		insertNotification.Table("notifications")
		insertNotification.Finish()
		return insertNotification.Execute()
	}, func(message notificationevent.Message) error {
		return notificationevent.Publish(utilities.NotificationHub, message, contactAdminRecipients(utilities.UserStatusReader))
	})
	if stage != "" {
		log.Printf("operation=AddContactRequest stage=%s", stage)
	}
}

func contactAdminRecipients(reader data.UserStatusReader) notify.Predicate {
	return func(client notify.Client) bool {
		if !notificationevent.Recipient(client) {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		expire, role, err := userstatus.ShouldExpireAuthCookies(ctx, reader, string(client.Metadata.UserID))
		return err == nil && !expire && role == "admin"
	}
}
