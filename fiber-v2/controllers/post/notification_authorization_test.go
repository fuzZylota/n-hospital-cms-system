package post

import (
	"context"
	"errors"
	"testing"

	"models/data"
	"models/notify"
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
		name     string
		status   data.UserStatus
		err      error
		event    notificationws.Event
		branch   notify.BranchID
		branchOK bool
		payload  []byte
		want     bool
	}{
		{"admin appointment", data.UserStatus{Found: true, Active: true, Role: "admin"}, nil, notificationws.Appointment, "", false, frame, true},
		{"moderator appointment", data.UserStatus{Found: true, Active: true, Role: "moderator"}, nil, notificationws.Appointment, "", false, frame, true},
		{"santral own branch", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Appointment, "branch-a", true, frame, true},
		{"santral other branch", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Appointment, "branch-b", true, frame, false},
		{"santral branch lookup failed", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Appointment, "", false, frame, false},
		{"inactive", data.UserStatus{Found: true, Active: false, Role: "admin"}, nil, notificationws.Appointment, "", false, frame, false},
		{"missing", data.UserStatus{}, nil, notificationws.Appointment, "", false, frame, false},
		{"blank role", data.UserStatus{Found: true, Active: true}, nil, notificationws.Appointment, "", false, frame, false},
		{"status lookup failed", data.UserStatus{}, errors.New("private query"), notificationws.Appointment, "", false, frame, false},
		{"malformed appointment", data.UserStatus{Found: true, Active: true, Role: "admin"}, nil, notificationws.Appointment, "", false, []byte(`{"message":"not-json"}`), false},
		{"application ik", data.UserStatus{Found: true, Active: true, Role: "ik"}, nil, notificationws.Application, "", false, frame, true},
		{"application santral", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Application, "", false, frame, false},
		{"contact active", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, notificationws.Contact, "", false, frame, true},
		{"unknown", data.UserStatus{Found: true, Active: true, Role: "admin"}, nil, notificationws.Unknown, "", false, frame, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &sec003cStatus{status: tc.status, err: tc.err}
			got := authorizeNotificationTextWithState(reader, func(notify.UserID) (notify.BranchID, bool) { return tc.branch, tc.branchOK }, "42", tc.event, tc.payload)
			if got != tc.want {
				t.Fatal("current account or event permission ignored")
			}
		})
	}
}
