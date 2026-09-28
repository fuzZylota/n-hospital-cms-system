package postgres

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"database/postgres/internal/dbtest"
	"headerbuttons/parentread"
)

// This joins the SQL script, synthetic rows, and consumer decision without a DB.
func TestHeaderParentReadSeam(t *testing.T) {
	parentID := "9"
	for _, test := range []struct {
		name          string
		rows          *dbtest.Rows
		queryErr      error
		authenticated bool
		wantStatus    int
		wantButtons   []parentread.Button
		wantQuery     bool
		wantClose     bool
	}{
		{
			name:          "multiple rows and null parent",
			rows:          dbtest.NewRows([]string{"hbid", "title", "parent_id"}, []any{int64(7), "Ana", nil}, []any{int64(12), "Alt", int64(9)}),
			authenticated: true,
			wantStatus:    200,
			wantButtons:   []parentread.Button{{Hbid: "7", Title: "Ana"}, {Hbid: "12", Title: "Alt", ParentId: parentID}},
			wantQuery:     true,
			wantClose:     true,
		},
		{
			name:          "empty rows",
			rows:          dbtest.NewRows([]string{"hbid", "title", "parent_id"}),
			authenticated: true,
			wantStatus:    200,
			wantButtons:   []parentread.Button{},
			wantQuery:     true,
			wantClose:     true,
		},
		{
			name:          "query error",
			queryErr:      errors.New("private backend detail"),
			authenticated: true,
			wantStatus:    500,
			wantQuery:     true,
		},
		{
			name:          "scan error",
			rows:          dbtest.NewRows([]string{"hbid", "title", "parent_id"}, []any{"invalid-id", "Ana", nil}),
			authenticated: true,
			wantStatus:    500,
			wantQuery:     true,
			wantClose:     true,
		},
		{
			name:       "denied without query",
			rows:       dbtest.NewRows([]string{"hbid", "title", "parent_id"}),
			wantStatus: 403,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			step := dbtest.Query(test.rows)
			if test.queryErr != nil {
				step = dbtest.QueryError(test.queryErr)
			}
			connector, repository := openHeaderButtonRepository(t, step)
			result := parentread.Response(context.Background(), test.authenticated, repository)
			if result["status"] != test.wantStatus {
				t.Fatalf("response status = %v, want %d", result["status"], test.wantStatus)
			}
			if test.wantStatus == 500 && result["message"] != "Internal server error" {
				t.Fatal("database failure leaked into the consumer response")
			}
			if test.wantStatus == 403 && result["message"] != "Forbidden" {
				t.Fatal("denied response changed")
			}
			if test.wantStatus == 200 {
				buttons, ok := result["header_buttons"].([]parentread.Button)
				if !ok || !reflect.DeepEqual(buttons, test.wantButtons) {
					t.Fatalf("response buttons = %#v, want %#v", result["header_buttons"], test.wantButtons)
				}
			} else if _, leaked := result["header_buttons"]; leaked {
				t.Fatal("failed read exposed a partial result")
			}
			if test.wantQuery {
				assertHeaderButtonQuery(t, connector)
			} else if len(connector.Events()) != 0 || connector.Remaining() != 1 {
				t.Fatal("denied request reached the repository")
			}
			if test.rows != nil {
				wantCloseCount := 0
				if test.wantClose {
					wantCloseCount = 1
				}
				if test.rows.CloseCount() != wantCloseCount {
					t.Fatal("row iterator close count changed")
				}
			}
		})
	}
}
