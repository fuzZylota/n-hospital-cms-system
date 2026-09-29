package postgres

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"database/postgres/internal/dbtest"
	"models/data"
)

func personalFixture() data.PersonalNotification {
	branch := int64(2)
	return data.PersonalNotification{
		Kind: data.RequestCreated, SubjectID: 73, Message: "synthetic event",
		NotificationType: "info", NotificationLevel: "santral",
		Link: "/panel/randevu-talepleri/73", BranchID: &branch,
		RecipientUIDs: []int64{11, 22},
	}
}

func TestPersonalNotificationCreateSeparatesRecipients(t *testing.T) {
	connector := dbtest.NewConnector(dbtest.Begin(),
		dbtest.Query(dbtest.NewRows([]string{"nid"}, []any{int64(41)})),
		dbtest.Exec(1), dbtest.Exec(1), dbtest.Commit())
	db := sql.OpenDB(connector)
	defer db.Close()
	nid, err := NewPersonalNotificationRepository(db).CreatePersonalNotification(context.Background(), personalFixture())
	if err != nil || nid != 41 || connector.Remaining() != 0 {
		t.Fatalf("create failed: nid=%d err=%v remaining=%d", nid, err, connector.Remaining())
	}
	events := connector.Events()
	if got := []dbtest.Kind{events[0].Kind, events[1].Kind, events[2].Kind, events[3].Kind, events[4].Kind}; !reflect.DeepEqual(got, []dbtest.Kind{dbtest.BeginKind, dbtest.QueryKind, dbtest.ExecKind, dbtest.ExecKind, dbtest.CommitKind}) {
		t.Fatalf("transaction order: %v", got)
	}
	if !strings.Contains(events[1].SQL, "delivery_model") || !strings.Contains(events[1].SQL, "RETURNING nid") ||
		!strings.Contains(events[2].SQL, "notification_receipts") ||
		events[2].Args[0].Value != int64(41) || events[2].Args[1].Value != int64(11) ||
		events[3].Args[0].Value != int64(41) || events[3].Args[1].Value != int64(22) {
		t.Fatal("event and two distinct recipient records were not bound")
	}
}

func TestPersonalNotificationInvalidInputNeverBegins(t *testing.T) {
	for _, mutate := range []func(*data.PersonalNotification){
		func(x *data.PersonalNotification) { x.Kind = "unknown" },
		func(x *data.PersonalNotification) { x.SubjectID = 0 },
		func(x *data.PersonalNotification) { x.SubjectID = -1 },
		func(x *data.PersonalNotification) { x.SubjectID = 1 << 32 },
		func(x *data.PersonalNotification) { x.RecipientUIDs = nil },
		func(x *data.PersonalNotification) { x.RecipientUIDs = []int64{} },
		func(x *data.PersonalNotification) { x.RecipientUIDs = []int64{11, 11} },
		func(x *data.PersonalNotification) { x.RecipientUIDs = []int64{0} },
		func(x *data.PersonalNotification) { x.RecipientUIDs = []int64{-1} },
		func(x *data.PersonalNotification) { x.RecipientUIDs = []int64{1 << 32} },
		func(x *data.PersonalNotification) { x.NotificationType = "other" },
		func(x *data.PersonalNotification) { x.NotificationLevel = "other" },
		func(x *data.PersonalNotification) { branch := int64(0); x.BranchID = &branch },
		func(x *data.PersonalNotification) { x.Link = "https://outside.invalid/" },
	} {
		input := personalFixture()
		mutate(&input)
		connector := dbtest.NewConnector()
		db := sql.OpenDB(connector)
		_, err := NewPersonalNotificationRepository(db).CreatePersonalNotification(context.Background(), input)
		db.Close()
		if !errors.Is(err, ErrInvalidPersonalNotification) || len(connector.Events()) != 0 {
			t.Fatalf("invalid input reached database: %v", connector.Events())
		}
	}
}

func TestPersonalNotificationFailuresDoNotCommitOrExposeBackendCause(t *testing.T) {
	backend := errors.New("private synthetic database detail")
	for _, tc := range []struct {
		name  string
		steps []dbtest.Step
	}{
		{"begin", []dbtest.Step{dbtest.BeginError(backend)}},
		{"duplicate event", []dbtest.Step{dbtest.Begin(), dbtest.QueryError(backend), dbtest.Rollback()}},
		{"missing returned nid", []dbtest.Step{dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"nid"})), dbtest.Rollback()}},
		{"receipt error", []dbtest.Step{dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"nid"}, []any{41})), dbtest.ExecError(backend), dbtest.Rollback()}},
		{"zero receipt", []dbtest.Step{dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"nid"}, []any{41})), dbtest.Exec(0), dbtest.Rollback()}},
		{"second receipt error", []dbtest.Step{dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"nid"}, []any{41})), dbtest.Exec(1), dbtest.ExecError(backend), dbtest.Rollback()}},
		{"rollback error", []dbtest.Step{dbtest.Begin(), dbtest.QueryError(backend), dbtest.RollbackError(backend)}},
		{"commit", []dbtest.Step{dbtest.Begin(), dbtest.Query(dbtest.NewRows([]string{"nid"}, []any{41})), dbtest.Exec(1), dbtest.Exec(1), dbtest.CommitError(backend)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			connector := dbtest.NewConnector(tc.steps...)
			db := sql.OpenDB(connector)
			defer db.Close()
			nid, err := NewPersonalNotificationRepository(db).CreatePersonalNotification(context.Background(), personalFixture())
			if nid != 0 || !errors.Is(err, ErrPersonalNotificationStorage) || strings.Contains(err.Error(), "private") || connector.Remaining() != 0 {
				t.Fatalf("unsafe failure: nid=%d err=%v remaining=%d", nid, err, connector.Remaining())
			}
			events := connector.Events()
			if tc.name != "commit" && tc.name != "begin" && events[len(events)-1].Kind != dbtest.RollbackKind {
				t.Fatalf("missing rollback: %v", events)
			}
			if tc.name != "commit" {
				for _, event := range events {
					if event.Kind == dbtest.CommitKind {
						t.Fatal("failure committed")
					}
				}
			}
		})
	}
}
