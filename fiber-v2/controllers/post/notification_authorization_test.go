package post

import (
	"context"
	"errors"
	"testing"

	"models/data"
	"post/notificationws"
)

type sec003cStatus struct {
	status data.UserStatus
	err    error
}

func (s *sec003cStatus) LookupUserStatus(context.Context, string) (data.UserStatus, error) {
	return s.status, s.err
}

func TestSEC003CInboundCurrentStateAndEvent(t *testing.T) {
	frame := []byte(`{"message":"{\"sid\":\"branch-a\"}"}`)
	for _, tc := range []struct {
		name   string
		status data.UserStatus
		err    error
		event  notificationws.Event
		want   bool
	}{
		{"active appointment frame denied", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Appointment, false},
		{"admin repeat appointment frame denied", data.UserStatus{Found: true, Active: true, Role: "admin"}, nil, notificationws.Appointment, false},
		{"inactive", data.UserStatus{Found: true, Active: false, Role: "santral"}, nil, notificationws.Appointment, false},
		{"deleted", data.UserStatus{}, nil, notificationws.Appointment, false},
		{"role downgrade", data.UserStatus{Found: true, Active: true, Role: "ik"}, nil, notificationws.Appointment, false},
		{"lookup error", data.UserStatus{}, errors.New("private query"), notificationws.Appointment, false},
		{"application ik", data.UserStatus{Found: true, Active: true, Role: "ik"}, nil, notificationws.Application, true},
		{"application santral", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Application, false},
		{"contact active", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Contact, true},
		{"unknown denied", data.UserStatus{Found: true, Active: true, Role: "admin"}, nil, notificationws.Unknown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &sec003cStatus{status: tc.status, err: tc.err}
			got := authorizeNotificationTextWithState(reader, "42", tc.event, frame)
			if got != tc.want {
				t.Fatal("current account or event permission ignored")
			}
		})
	}
}

func TestF05AppointmentProducerFramesNeverAuthorize(t *testing.T) {
	reader := &sec003cStatus{status: data.UserStatus{Found: true, Active: true, Role: "admin"}}
	frames := [][]byte{
		[]byte(`{"message":"{\"rrid\":\"missing\",\"sid\":\"branch-a\"}"}`),
		[]byte(`{"message":"{\"rrid\":\"other-branch\",\"sid\":\"branch-b\"}"}`),
		[]byte(`{"message":"{\"rrid\":\"missing\",\"sid\":\"branch-a\"}"}`),
	}
	for _, frame := range frames {
		if authorizeNotificationTextWithState(reader, "42", notificationws.Appointment, frame) {
			t.Fatal("a forged or repeated appointment frame reached the producer")
		}
	}
}
