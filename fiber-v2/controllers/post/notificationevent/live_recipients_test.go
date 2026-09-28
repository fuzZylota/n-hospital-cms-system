package notificationevent

import (
	"context"
	"errors"
	"models/data"
	"models/notify"
	"testing"
)

type sec003cReader struct {
	status data.UserStatus
	err    error
}

func (r *sec003cReader) LookupUserStatus(context.Context, string) (data.UserStatus, error) {
	return r.status, r.err
}

func TestSEC003COutboundCurrentStateAndEventRules(t *testing.T) {
	client := notify.Client{Metadata: notify.Metadata{UserID: "42", Protocol: "kullanici", Role: "admin", BranchID: "branch-a"}}
	for _, tc := range []struct {
		name   string
		status data.UserStatus
		err    error
		branch notify.BranchID
		rule   string
		want   bool
	}{
		{"active appointment", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "branch-a", "appointment", true},
		{"inactive", data.UserStatus{Found: true, Active: false, Role: "admin"}, nil, "branch-a", "appointment", false},
		{"deleted", data.UserStatus{}, nil, "branch-a", "contact", false},
		{"role downgrade", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "branch-a", "application", false},
		{"branch changed", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "branch-b", "appointment", false},
		{"lookup error", data.UserStatus{}, errors.New("private query"), "branch-a", "contact", false},
		{"active contact", data.UserStatus{Found: true, Active: true, Role: "ik"}, nil, "", "contact", true},
		{"active request with branch permission", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "branch-b", "request-allowed", true},
		{"request permission removed", data.UserStatus{Found: true, Active: true, Role: "santral"}, nil, "branch-b", "request-denied", false},
		{"active delete status", data.UserStatus{Found: true, Active: true, Role: "ik"}, nil, "", "global", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &sec003cReader{status: tc.status, err: tc.err}
			lookup := func(notify.UserID) (User, bool) {
				return User{Role: notify.Role(reader.status.Role), Branch: tc.branch}, true
			}
			var rule notify.Predicate
			switch tc.rule {
			case "appointment":
				rule = AppointmentRecipients("branch-a", lookup)
			case "application":
				rule = ApplicationRecipients(lookup)
			case "request-allowed", "request-denied":
				rule = RequestRecipients("branch-a", lookup, func(notify.UserID, notify.BranchID) bool { return tc.rule == "request-allowed" })
			default:
				rule = Recipient
			}
			if CurrentRecipients(reader, rule)(client) != tc.want {
				t.Fatal("stale recipient remained authorized or active user rejected")
			}
		})
	}
}
