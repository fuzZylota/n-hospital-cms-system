package randevular

import (
	"errors"
	"net/url"
	"time"

	"post/notificationevent"
)

var errInvalidSavedRequest = errors.New("saved appointment request unavailable")

type savedRequestNotification struct {
	Rrid, Sid, PatientFirstName, PatientLastName, PatientPhone, PatientEmail string
	Message, Status, SubeName                                                string
	PreferredDate, PreferredTime, CreatedAt                                  time.Time
}

// The saved request, never a WebSocket frame, supplies notification content and branch.
func prepareSavedRequestNotification(rrid string, read func(string) (savedRequestNotification, error), persist func(notificationevent.NewRequest) error) (notificationevent.NewRequest, error) {
	if rrid == "" || read == nil || persist == nil {
		return notificationevent.NewRequest{}, errInvalidSavedRequest
	}
	record, err := read(rrid)
	if err != nil {
		return notificationevent.NewRequest{}, err
	}
	if record.Rrid != rrid {
		return notificationevent.NewRequest{}, errInvalidSavedRequest
	}

	text := record.PatientFirstName + " " + record.PatientLastName + " tarafından"
	if record.PatientEmail != "" {
		if !record.PreferredDate.IsZero() {
			text += ", " + record.PreferredDate.Format("02.01.2006") + " tarihinde "
		}
		if !record.PreferredTime.IsZero() {
			text += ", " + record.PreferredTime.Format("15:04") + " saatinde "
		}
		if record.PreferredDate.IsZero() && record.PreferredTime.IsZero() {
			text += " randevu talebi gönderildi."
		} else {
			text += "gerçekleşmek üzere randevu talebi gönderildi."
		}
	} else {
		text += " hızlı randevu formuyla randevu talebi gönderildi."
	}
	createdAt := ""
	if !record.CreatedAt.IsZero() {
		createdAt = record.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	event := notificationevent.NewRequest{
		Type: "new_randevu_talebi", Rrid: record.Rrid, Sid: record.Sid,
		PatientFirstName: record.PatientFirstName, PatientLastName: record.PatientLastName,
		PatientPhone: record.PatientPhone, Message: record.Message,
		CreatedAt: createdAt, Status: record.Status, SubeName: record.SubeName,
		NotificationMessage: text, NotificationType: "info",
		RequestLink: "/panel/randevu-talepleri/" + url.PathEscape(record.Rrid) + "?notification=true",
	}
	if err := persist(event); err != nil {
		return notificationevent.NewRequest{}, err
	}
	return event, nil
}
