package notificationevent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"models/notify"
	"reflect"
	"strings"
	"testing"
)

type recordingHub struct {
	payload []byte
	room    notify.RoomID
	calls   int
	err     error
}

func (h *recordingHub) Register(context.Context, notify.RoomID, notify.Client, notify.Send) (notify.Registration, error) {
	panic("unused")
}
func (h *recordingHub) Shutdown(context.Context) error { return nil }
func (h *recordingHub) Broadcast(room notify.RoomID, p []byte, _ notify.Predicate) (notify.BroadcastResult, error) {
	h.calls++
	h.room = room
	h.payload = append([]byte(nil), p...)
	return notify.BroadcastResult{}, h.err
}

func TestWireShapesSingleEncodingAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name    string
		message any
		want    string
	}{
		{"appointment", Message{Message: "hello", RequestLink: "/panel/randevu-talepleri/id?notification=true"}, `{"uid":"","message":"hello","request_link":"/panel/randevu-talepleri/id?notification=true"}`},
		{"application", Message{Message: "hello", RequestLink: "/panel/is-basvurulari/id?notification=true"}, `{"uid":"","message":"hello","request_link":"/panel/is-basvurulari/id?notification=true"}`},
		{"contact", Message{Message: "hello", RequestLink: "/panel/iletisim-istekleri/id?notification=true"}, `{"uid":"","message":"hello","request_link":"/panel/iletisim-istekleri/id?notification=true"}`},
		{"new request", NewRequest{Type: "new_randevu_talebi", Rrid: "id", Sid: "branch"}, `{"type":"new_randevu_talebi","rrid":"id","patient_first_name":"","patient_last_name":"","patient_phone":"","message":"","created_at":"","status":"","sube_name":"","sid":"branch"}`},
		{"delete", Deleted{Type: "randevu_talebi_silindi", Rrid: "id"}, `{"type":"randevu_talebi_silindi","rrid":"id"}`},
		{"status", Status{Type: "randevu_talebi_status", Rrid: "id", NewStatus: "yeni"}, `{"type":"randevu_talebi_status","rrid":"id","new_status":"yeni"}`},
		{"empty", Message{}, `{"uid":"","message":"","request_link":""}`},
		{"nil", (*Message)(nil), `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &recordingHub{}
			if Publish(h, tc.message, nil) != nil {
				t.Fatal("publish failed")
			}
			if h.calls != 1 || h.room != "notifications" {
				t.Fatal("wrong broadcast")
			}
			var got, want any
			if json.Unmarshal(h.payload, &got) != nil || json.Unmarshal([]byte(tc.want), &want) != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("wire shape changed/double encoded")
			}
		})
	}
}

func TestPublicationFailureDoesNotLeakOrPublishInvalidJSON(t *testing.T) {
	h := &recordingHub{}
	err := Publish(h, make(chan int), nil)
	if err == nil || h.calls != 0 {
		t.Fatal("marshal failure published")
	}
	h.err = errors.New("private payload metadata")
	err = Publish(h, Message{}, nil)
	if err == nil || h.calls != 1 || errors.Is(err, h.err) {
		t.Fatal("transport error retained")
	}
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		if strings.Contains(fmt.Sprintf(format, err), "private") {
			t.Fatal("error leak")
		}
	}
	if Publish(nil, Message{}, nil) == nil {
		t.Fatal("nil hub accepted")
	}
}

func TestEveryEventRecipientMatrix(t *testing.T) {
	for _, role := range []notify.Role{"admin", "moderator", "santral", "ik", "other"} {
		for _, sameBranch := range []bool{false, true} {
			for _, permission := range []bool{false, true} {
				for _, protocol := range []notify.Protocol{"kullanici", "randevu"} {
					for _, uid := range []notify.UserID{"server_uid_with_underscore", "other-user", ""} {
						name := fmt.Sprintf("%s/same=%t/permission=%t/%s/uid=%s", role, sameBranch, permission, protocol, uid)
						t.Run(name, func(t *testing.T) {
							branch := notify.BranchID("different")
							if sameBranch {
								branch = "target"
							}
							lookups, permissions := 0, 0
							lookup := func(got notify.UserID) (User, bool) {
								lookups++
								if got != uid {
									t.Fatal("ConnectionID used for UID")
								}
								return User{role, branch}, true
							}
							canView := func(got notify.UserID, target notify.BranchID) bool {
								permissions++
								if got != uid || target != "target" {
									t.Fatal("permission target changed")
								}
								return permission && got == "server_uid_with_underscore"
							}
							c := notify.Client{ID: "opaque-connection", Metadata: notify.Metadata{UserID: uid, Role: "admin", BranchID: "target", Protocol: protocol}}
							eligible := uid != "" && protocol == "kullanici"
							for _, test := range []struct {
								name      string
								predicate notify.Predicate
								want      bool
							}{
								{"appointment", AppointmentRecipients("target", lookup), eligible && (role == "admin" || role == "moderator" || (role == "santral" && sameBranch))},
								{"application", ApplicationRecipients(lookup), eligible && (role == "admin" || role == "moderator" || role == "ik")},
								{"new request", RequestRecipients("target", lookup, canView), eligible && (role == "admin" || role == "moderator" || (permission && uid == "server_uid_with_underscore"))},
								{"contact", Recipient, eligible}, {"delete", Recipient, eligible}, {"status", Recipient, eligible},
							} {
								if test.predicate(c) != test.want {
									t.Fatalf("wrong recipient rule: %s", test.name)
								}
							}
							if !eligible && (lookups != 0 || permissions != 0) {
								t.Fatal("ineligible client queried")
							}
						})
					}
				}
			}
		}
	}
}

func TestLookupFailuresDenyAndIgnoreStaleMetadata(t *testing.T) {
	c := notify.Client{ID: "not-user", Metadata: notify.Metadata{UserID: "server", Role: "admin", BranchID: "target", Protocol: "kullanici"}}
	for _, lookup := range []Lookup{func(notify.UserID) (User, bool) { return User{}, false }, func(notify.UserID) (User, bool) { return User{Role: "other", Branch: "different"}, true }} {
		for _, p := range []notify.Predicate{AppointmentRecipients("target", lookup), ApplicationRecipients(lookup), RequestRecipients("target", lookup, func(notify.UserID, notify.BranchID) bool { return false })} {
			if p(c) {
				t.Fatal("stale metadata granted access")
			}
		}
	}
}
