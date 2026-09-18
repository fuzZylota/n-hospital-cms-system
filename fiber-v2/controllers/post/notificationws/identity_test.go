package notificationws

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"models/notify"
	"reflect"
	"strconv"
	"testing"
)

func uidCases() []struct {
	name  string
	value any
	want  notify.UserID
} {
	return []struct {
		name  string
		value any
		want  notify.UserID
	}{
		{"one", notify.UserID("1"), "1"},
		{"forty-two", notify.UserID("42"), "42"},
		{"max", notify.UserID(strconv.FormatInt(math.MaxInt64, 10)), "9223372036854775807"},
		{"empty", notify.UserID(""), ""},
		{"zero", notify.UserID("0"), ""},
		{"negative", notify.UserID("-1"), ""},
		{"plus", notify.UserID("+1"), ""},
		{"leading space", notify.UserID(" 1"), ""},
		{"trailing space", notify.UserID("1 "), ""},
		{"leading zero", notify.UserID("01"), ""},
		{"two zeroes", notify.UserID("00"), ""},
		{"decimal", notify.UserID("1.0"), ""},
		{"scientific", notify.UserID("1e2"), ""},
		{"alphabetic", notify.UserID("abc"), ""},
		{"sensitive input", notify.UserID("private-invalid-uid"), ""},
		{"overflow", notify.UserID("9223372036854775808"), ""},
		{"long overflow", notify.UserID("10000000000000000000"), ""},
		{"whitespace", notify.UserID(" \t\r\n"), ""},
		{"embedded whitespace", notify.UserID("1\t2"), ""},
		{"non-ASCII digit", notify.UserID("١"), ""},
		{"NUL", notify.UserID("1\x00"), ""},
		{"plain string", "42", ""},
		{"integer", int64(42), ""},
		{"connection ID", notify.ConnectionID("42"), ""},
		{"nil", nil, ""},
	}
}

func assertSafeIdentityError(t *testing.T, err error) {
	t.Helper()
	if err != errIdentity || err.Error() != "notification websocket: identity unavailable" {
		t.Fatal("identity rejection did not use the fixed error")
	}
	for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
		if fmt.Sprintf(format, err) != fmt.Sprintf(format, errIdentity) {
			t.Fatal("identity error retained input")
		}
	}
	var parseErr *strconv.NumError
	var code failure
	if errors.Unwrap(err) != nil || errors.As(err, &parseErr) || !errors.As(err, &code) || code != errIdentity || !errors.Is(err, errIdentity) {
		t.Fatal("unsafe identity error chain")
	}
	if _, ok := err.(interface{ Unwrap() []error }); ok {
		t.Fatal("identity error retained multiple causes")
	}
}

func TestAuthenticatedUserIDCanonicalTable(t *testing.T) {
	for _, tc := range uidCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			uid, err := authenticatedUserID(tc.value)
			if tc.want == "" {
				assertSafeIdentityError(t, err)
				if uid != "" {
					t.Fatal("invalid identity returned")
				}
			} else if err != nil || uid != tc.want {
				t.Fatal("canonical positive identity rejected/changed")
			}
		})
	}
}

type countedRandom struct {
	reader io.Reader
	calls  int
	bytes  int
}

func (r *countedRandom) Read(p []byte) (int, error) {
	r.calls++
	n, err := r.reader.Read(p)
	r.bytes += n
	return n, err
}

// Runs the exact gate used by Handler. The upgrade endpoint is a test callback,
// not a real Fiber/WebSocket handshake; it invokes the real session algorithm.
func TestAuthenticatedUpgradeOrderAndRefusal(t *testing.T) {
	for _, tc := range uidCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := observe(hub(t))
			s := socket()
			s.reads <- frame{textMessage, []byte(`{"uid":"0"}`)}
			random := &countedRandom{reader: rand.Reader}
			var order []string
			var local any
			upgrades, writes := 0, 0
			err := authenticatedUpgrade(func() (any, error) {
				order = append(order, "auth")
				return tc.value, nil // Auth success must not bypass UID validation.
			}, func(uid notify.UserID) {
				order = append(order, "local")
				writes++
				local = uid
			}, func() error {
				order = append(order, "upgrade")
				upgrades++
				return serve(h, s, local, Subscriber, nil, random)
			})
			if tc.want == "" {
				assertSafeIdentityError(t, err)
				if local != nil || writes != 0 || upgrades != 0 || h.calls.Load() != 0 || random.calls != 0 || s.closeCalls.Load() != 0 || !reflect.DeepEqual(order, []string{"auth"}) {
					t.Fatal("invalid HTTP identity reached local/upgrade/session")
				}
				return
			}
			if err != nil || local != tc.want || writes != 1 || upgrades != 1 || h.calls.Load() != 1 || random.calls != 1 || random.bytes != 32 || !reflect.DeepEqual(order, []string{"auth", "local", "upgrade"}) {
				t.Fatal("authenticated upgrade count/order changed")
			}
			client := <-h.registered
			if client.Metadata.UserID != tc.want {
				t.Fatal("HTTP canonical identity not transferred to session")
			}
		})
	}
}

func TestAuthenticatedUpgradeAuthErrorAndUpgradeError(t *testing.T) {
	raw := errors.New("private-invalid-auth-uid")
	err := authenticatedUpgrade(func() (any, error) { return notify.UserID("42"), raw },
		func(notify.UserID) { t.Fatal("local written after auth failure") },
		func() error { t.Fatal("upgrade after auth failure"); return nil })
	assertSafeIdentityError(t, err)
	if errors.Is(err, raw) {
		t.Fatal("raw auth error retained")
	}
	upgradeFailure := errors.New("test upgrade failure")
	err = authenticatedUpgrade(func() (any, error) { return notify.UserID("42"), nil },
		func(notify.UserID) {}, func() error { return upgradeFailure })
	if err != upgradeFailure {
		t.Fatal("upgrade error changed")
	}
}

func TestSessionCanonicalIdentityBeforeRegistration(t *testing.T) {
	for _, tc := range uidCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := observe(hub(t))
			s := socket()
			s.reads <- frame{textMessage, []byte(`{"uid":"-1","role":"admin","protocol":"other"}`)}
			random := &countedRandom{reader: rand.Reader}
			err := serve(h, s, tc.value, Subscriber, nil, random)
			if s.closeCalls.Load() != 1 {
				t.Fatal("session socket was not closed exactly once")
			}
			if tc.want == "" {
				assertSafeIdentityError(t, err)
				metadata, identityErr := identity(tc.value)
				if identityErr != errIdentity || metadata != (notify.Metadata{}) || random.calls != 0 || h.calls.Load() != 0 || len(h.registered) != 0 {
					t.Fatal("invalid session identity produced metadata/entropy/registration")
				}
				// No Registration exists: the sole watcher launch, after successful
				// Register in serve, is unreachable (also covered by wiring checks).
				return
			}
			if err != nil || h.calls.Load() != 1 || random.calls != 1 || random.bytes != 32 {
				t.Fatal("positive session registration/entropy count changed")
			}
			client := <-h.registered
			id, decodeErr := hex.DecodeString(string(client.ID))
			if decodeErr != nil || len(id) != 32 || string(client.ID) == string(tc.want) || client.Metadata != (notify.Metadata{UserID: tc.want, Protocol: recipientProtocol}) {
				t.Fatal("session identity or connection identity changed")
			}
			registration := <-h.registration
			wait(t, registration.Done())
			registration.Unregister()
			registration.Unregister()
			if s.closeCalls.Load() != 1 {
				t.Fatal("late cleanup closed socket again")
			}
		})
	}
}
