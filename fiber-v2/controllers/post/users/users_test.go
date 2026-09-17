package users

import (
	"errors"
	"strings"
	"testing"
)

func lookupUserEditActor(actor userEditActor, lookupErr error) userEditActorLookup {
	return func(string) (userEditActor, error) {
		return actor, lookupErr
	}
}

func validSelfProfileFields() map[string][]string {
	return map[string][]string{
		"name":     {"Ayşe"},
		"surname":  {"Yılmaz"},
		"email":    {"ayse@example.com"},
		"phone":    {"+90 555 111 22 33"},
		"timezone": {"Europe/Istanbul"},
	}
}

func withSelfUID(fields map[string][]string) map[string][]string {
	withUID := make(map[string][]string, len(fields)+1)
	for field, values := range fields {
		withUID[field] = values
	}
	withUID["uid"] = []string{"2"}
	return withUID
}

func TestDecideUserEditRequestStrictSelfAllowlist(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "moderator", IsActive: true}

	tests := []struct {
		name        string
		fields      map[string][]string
		wantAllowed bool
	}{
		{
			name:        "five profile fields are accepted",
			fields:      validSelfProfileFields(),
			wantAllowed: true,
		},
		{
			name:        "matching protocol uid is accepted",
			fields:      map[string][]string{"uid": {"2"}, "name": {"Ayşe"}},
			wantAllowed: true,
		},
		{
			name:        "old field is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "old_role": {"admin"}},
			wantAllowed: false,
		},
		{
			name:        "password is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "password": {"secret"}},
			wantAllowed: false,
		},
		{
			name:        "password-like field is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "password_confirm": {"secret"}},
			wantAllowed: false,
		},
		{
			name:        "last login is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "last_login": {"2026-09-17"}},
			wantAllowed: false,
		},
		{
			name:        "unknown field is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "unexpected": {"value"}},
			wantAllowed: false,
		},
		{
			name:        "permission is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "perm_view_7": {"true"}},
			wantAllowed: false,
		},
		{
			name:        "role is rejected",
			fields:      map[string][]string{"name": {"Ayşe"}, "role": {"admin"}},
			wantAllowed: false,
		},
		{
			name:        "mixed profile and protected payload is rejected",
			fields:      map[string][]string{"email": {"ayse@example.com"}, "sid": {"7"}},
			wantAllowed: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decideUserEditRequest("2", "2", withSelfUID(test.fields), lookupUserEditActor(nonAdmin, nil))
			if (err == nil) != test.wantAllowed {
				t.Fatalf("decideUserEditRequest() error = %v, wantAllowed = %v", err, test.wantAllowed)
			}
		})
	}
}

func TestDecideUserEditRequestValidatesTargetUID(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "moderator", IsActive: true}

	tests := []struct {
		name      string
		targetUID string
		fields    map[string][]string
	}{
		{name: "missing route uid", targetUID: "", fields: map[string][]string{"name": {"Ayşe"}}},
		{name: "zero route uid", targetUID: "0", fields: map[string][]string{"name": {"Ayşe"}}},
		{name: "negative route uid", targetUID: "-2", fields: map[string][]string{"name": {"Ayşe"}}},
		{name: "missing body uid", targetUID: "2", fields: map[string][]string{"name": {"Ayşe"}}},
		{name: "zero body uid", targetUID: "2", fields: map[string][]string{"uid": {"0"}, "name": {"Ayşe"}}},
		{name: "negative body uid", targetUID: "2", fields: map[string][]string{"uid": {"-2"}, "name": {"Ayşe"}}},
		{name: "body uid mismatch", targetUID: "2", fields: map[string][]string{"uid": {"3"}, "name": {"Ayşe"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decideUserEditRequest("2", test.targetUID, test.fields, lookupUserEditActor(nonAdmin, nil))
			if err == nil {
				t.Fatal("decideUserEditRequest() expected an error")
			}
		})
	}
}

func TestDecideUserEditRequestUsesCurrentActor(t *testing.T) {
	tests := []struct {
		name      string
		actor     userEditActor
		lookupErr error
		wantError bool
	}{
		{name: "active actor", actor: userEditActor{UID: "2", Role: "ik", IsActive: true}},
		{name: "inactive actor", actor: userEditActor{UID: "2", Role: "ik", IsActive: false}, wantError: true},
		{name: "missing actor", actor: userEditActor{}, wantError: true},
		{name: "lookup error", lookupErr: errors.New("lookup failed"), wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decideUserEditRequest("2", "2", map[string][]string{"uid": {"2"}, "name": {"Ayşe"}}, lookupUserEditActor(test.actor, test.lookupErr))
			if (err != nil) != test.wantError {
				t.Fatalf("decideUserEditRequest() error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestDecideUserEditRequestAdminPermissionIntent(t *testing.T) {
	admin := userEditActor{UID: "1", Role: "admin", IsActive: true}

	withPermission, err := decideUserEditRequest(
		"1",
		"2",
		map[string][]string{"name": {"Updated"}, "role": {"moderator"}, "perm_view_7": {"true"}},
		lookupUserEditActor(admin, nil),
	)
	if err != nil {
		t.Fatalf("admin edit with permission error = %v", err)
	}
	if !withPermission.Policy.IsAdmin || !withPermission.Policy.UpdatePermissions {
		t.Fatalf("admin permission policy = %+v", withPermission.Policy)
	}

	withoutPermission, err := decideUserEditRequest(
		"1",
		"2",
		map[string][]string{"name": {"Updated"}, "role": {"moderator"}},
		lookupUserEditActor(admin, nil),
	)
	if err != nil {
		t.Fatalf("admin edit without permission error = %v", err)
	}
	if withoutPermission.Policy.UpdatePermissions {
		t.Fatal("admin request without permission fields must preserve permissions")
	}
}

func TestDecideUserEditRequestValidatesExplicitProfileValues(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "santral", IsActive: true}

	tests := []struct {
		name   string
		fields map[string][]string
	}{
		{name: "blank name", fields: map[string][]string{"name": {" "}}},
		{name: "blank timezone", fields: map[string][]string{"timezone": {""}}},
		{name: "invalid email", fields: map[string][]string{"email": {"not-an-email"}}},
		{name: "invalid phone", fields: map[string][]string{"phone": {"123"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decideUserEditRequest("2", "2", withSelfUID(test.fields), lookupUserEditActor(nonAdmin, nil))
			if !errors.Is(err, errInvalidUserProfile) {
				t.Fatalf("decideUserEditRequest() error = %v, want invalid profile", err)
			}
		})
	}
}

func TestDecideUserEditRequestNormalizesFieldNames(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "moderator", IsActive: true}
	decision, err := decideUserEditRequest(
		"2",
		"2",
		map[string][]string{" Name ": {"Ayşe"}, "UID": {"2"}},
		lookupUserEditActor(nonAdmin, nil),
	)
	if err != nil {
		t.Fatalf("normalized request error = %v", err)
	}
	if _, ok := decision.Fields["name"]; !ok {
		t.Fatal("normalized name field is missing")
	}
	if _, ok := decision.Fields["uid"]; !ok {
		t.Fatal("normalized uid field is missing")
	}
}

func TestDecideUserEditRequestRejectsAmbiguousAndBracketFields(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "moderator", IsActive: true}

	tests := []struct {
		name   string
		fields map[string][]string
	}{
		{
			name: "normalized case collision",
			fields: map[string][]string{
				"uid":  {"2"},
				"name": {"Ayşe"},
				"Name": {"Ayşe"},
			},
		},
		{
			name:   "same field repeated with the same value",
			fields: map[string][]string{"uid": {"2"}, "name": {"Ayşe", "Ayşe"}},
		},
		{
			name:   "same field repeated with different values",
			fields: map[string][]string{"uid": {"2"}, "name": {"Ayşe", "Fatma"}},
		},
		{name: "name bracket field", fields: map[string][]string{"uid": {"2"}, "name[]": {"Ayşe"}}},
		{name: "role bracket field", fields: map[string][]string{"uid": {"2"}, "role[]": {"admin"}}},
		{name: "permissions bracket field", fields: map[string][]string{"uid": {"2"}, "permissions[]": {"admin"}}},
		{name: "unknown bracket field", fields: map[string][]string{"uid": {"2"}, "unknown[]": {"value"}}},
		{name: "old role field", fields: map[string][]string{"uid": {"2"}, "old_role": {"admin"}}},
		{
			name:   "allowed and unknown fields mixed",
			fields: map[string][]string{"uid": {"2"}, "name": {"Ayşe"}, "unknown": {"value"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decideUserEditRequest("2", "2", test.fields, lookupUserEditActor(nonAdmin, nil))
			if err == nil {
				t.Fatal("decideUserEditRequest() expected an error")
			}
		})
	}
}

func TestDecideUserEditRequestPhoneValidation(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "moderator", IsActive: true}

	accepted := []string{
		"0555 123 45 67",
		"05551234567",
		"+90 555 123 45 67",
		"2125551234",
		"123456789012345",
	}
	for _, phone := range accepted {
		t.Run("accept "+phone, func(t *testing.T) {
			_, err := decideUserEditRequest(
				"2",
				"2",
				withSelfUID(map[string][]string{"phone": {phone}}),
				lookupUserEditActor(nonAdmin, nil),
			)
			if err != nil {
				t.Fatalf("decideUserEditRequest() error = %v", err)
			}
		})
	}

	rejected := []string{
		"----------",
		"()  --",
		"123456789",
		"1234567890123456",
		"0555 123 45 AB",
		"0555+1234567",
		"++905551234567",
		"0555.123.45.67",
	}
	for _, phone := range rejected {
		t.Run("reject "+phone, func(t *testing.T) {
			_, err := decideUserEditRequest(
				"2",
				"2",
				withSelfUID(map[string][]string{"phone": {phone}}),
				lookupUserEditActor(nonAdmin, nil),
			)
			if !errors.Is(err, errInvalidUserProfile) {
				t.Fatalf("decideUserEditRequest() error = %v, want invalid profile", err)
			}
		})
	}
}

func TestDecideUserEditRequestValidatesAdminProfileValues(t *testing.T) {
	admin := userEditActor{UID: "1", Role: "admin", IsActive: true}

	_, err := decideUserEditRequest(
		"1",
		"2",
		map[string][]string{"phone": {"----------"}},
		lookupUserEditActor(admin, nil),
	)
	if !errors.Is(err, errInvalidUserProfile) {
		t.Fatalf("decideUserEditRequest() error = %v, want invalid profile", err)
	}
}

func TestDecideUserEditRequestUnicodeNameValidation(t *testing.T) {
	nonAdmin := userEditActor{UID: "2", Role: "moderator", IsActive: true}

	accepted := []string{
		"Şen",
		"Çağrı",
		"Öztürk",
		"Nur Sena",
		"O'Connor",
		"D’Arcy",
		"Anne-Marie",
		"Élodie",
		"李雷",
		" Şen ",
		strings.Repeat("Ş", 255),
	}
	for _, value := range accepted {
		for _, field := range []string{"name", "surname"} {
			t.Run("accept "+field+" "+value, func(t *testing.T) {
				_, err := decideUserEditRequest(
					"2",
					"2",
					withSelfUID(map[string][]string{field: {value}}),
					lookupUserEditActor(nonAdmin, nil),
				)
				if err != nil {
					t.Fatalf("decideUserEditRequest() error = %v", err)
				}
			})
		}
	}

	rejected := []string{
		"Ş",
		"--",
		"''",
		"12",
		"A1",
		"Ali😀",
		"-Ali",
		"Ali-",
		"'Ali",
		"Ali'",
		"   ",
		"Ali--Veli",
		"Ali  Veli",
		strings.Repeat("Ş", 256),
	}
	for _, value := range rejected {
		t.Run("reject "+value, func(t *testing.T) {
			_, err := decideUserEditRequest(
				"2",
				"2",
				withSelfUID(map[string][]string{"name": {value}}),
				lookupUserEditActor(nonAdmin, nil),
			)
			if !errors.Is(err, errInvalidUserProfile) {
				t.Fatalf("decideUserEditRequest() error = %v, want invalid profile", err)
			}
		})
	}
}

func TestParseBranchPermissionChangesBindsTargetAndBranch(t *testing.T) {
	fields := map[string][]string{
		"perm_view_7":   {"0", "1"},
		"perm_delete_7": {"0"},
		"perm_delete_9": {"true"},
	}

	changes, err := parseBranchPermissionChanges(42, fields)
	if err != nil {
		t.Fatalf("parseBranchPermissionChanges() error = %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("len(changes) = %d, want 2", len(changes))
	}

	first := changes[0]
	if first.TargetUID != 42 || first.BranchID != 7 {
		t.Fatalf("first change target = (%d, %d), want (42, 7)", first.TargetUID, first.BranchID)
	}
	if first.CanView == nil || !*first.CanView {
		t.Error("first change CanView should be explicitly true")
	}
	if first.CanDelete == nil || *first.CanDelete {
		t.Error("first change CanDelete should be explicitly false")
	}

	second := changes[1]
	if second.TargetUID != 42 || second.BranchID != 9 {
		t.Fatalf("second change target = (%d, %d), want (42, 9)", second.TargetUID, second.BranchID)
	}
	if second.CanView != nil {
		t.Error("missing view permission must remain unset")
	}
	if second.CanDelete == nil || !*second.CanDelete {
		t.Error("second change CanDelete should be explicitly true")
	}
}

func TestParseBranchPermissionChangesRejectsUnscopedBranch(t *testing.T) {
	_, err := parseBranchPermissionChanges(42, map[string][]string{"perm_view_not-a-branch": {"true"}})
	if err == nil {
		t.Fatal("parseBranchPermissionChanges() expected an invalid branch error")
	}
}
