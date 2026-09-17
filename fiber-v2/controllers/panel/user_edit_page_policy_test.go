package panel

import (
	"errors"
	"testing"
)

func TestAuthorizeUserEditPage(t *testing.T) {
	tests := []struct {
		name             string
		authenticatedUID string
		targetUID        string
		actor            userEditPageActor
		lookupErr        error
		wantAllowed      bool
	}{
		{
			name:             "admin opens another user page",
			authenticatedUID: "1",
			targetUID:        "2",
			actor:            userEditPageActor{UID: "1", Role: "admin", IsActive: true},
			wantAllowed:      true,
		},
		{
			name:             "non-admin opens own page",
			authenticatedUID: "2",
			targetUID:        "2",
			actor:            userEditPageActor{UID: "2", Role: "moderator", IsActive: true},
			wantAllowed:      true,
		},
		{
			name:             "non-admin cannot open another user page",
			authenticatedUID: "2",
			targetUID:        "3",
			actor:            userEditPageActor{UID: "2", Role: "santral", IsActive: true},
			wantAllowed:      false,
		},
		{
			name:             "inactive actor fails closed",
			authenticatedUID: "2",
			targetUID:        "2",
			actor:            userEditPageActor{UID: "2", Role: "ik", IsActive: false},
			wantAllowed:      false,
		},
		{
			name:             "missing actor fails closed",
			authenticatedUID: "2",
			targetUID:        "2",
			actor:            userEditPageActor{},
			wantAllowed:      false,
		},
		{
			name:             "actor lookup error fails closed",
			authenticatedUID: "2",
			targetUID:        "2",
			lookupErr:        errors.New("lookup failed"),
			wantAllowed:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookup := func(string) (userEditPageActor, error) {
				return test.actor, test.lookupErr
			}
			_, err := authorizeUserEditPage(test.authenticatedUID, test.targetUID, lookup)
			if (err == nil) != test.wantAllowed {
				t.Fatalf("authorizeUserEditPage() error = %v, wantAllowed = %v", err, test.wantAllowed)
			}
		})
	}
}
