package parentread

import (
	"context"
	"encoding/json"
	"errors"
	"models/data"
	"reflect"
	"strings"
	"testing"
)

type fakeReader struct {
	calls   int
	parents []data.HeaderParent
	err     error
	ctx     context.Context
}

func (r *fakeReader) ListHeaderParents(ctx context.Context) ([]data.HeaderParent, error) {
	r.calls++
	r.ctx = ctx
	return r.parents, r.err
}

func TestResponseContract(t *testing.T) {
	parent := "9"
	for _, tc := range []struct {
		name          string
		auth          bool
		reader        *fakeReader
		status, calls int
		message       string
	}{
		{"denied", false, &fakeReader{}, 403, 0, "Forbidden"},
		{"success", true, &fakeReader{parents: []data.HeaderParent{{ID: "7", Title: "Ana", ParentID: nil}, {ID: "12", Title: "Alt", ParentID: &parent}}}, 200, 1, "Header buttons fetched successfully"},
		{"empty", true, &fakeReader{}, 200, 1, "Header buttons fetched successfully"},
		{"nil reader", true, nil, 500, 0, "Internal server error"},
		{"error", true, &fakeReader{err: errors.New("postgres://user:password@host private-backend")}, 500, 1, "Internal server error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reader data.HeaderButtonReader
			if tc.reader != nil {
				reader = tc.reader
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			got := Response(ctx, tc.auth, reader)
			if got["status"] != tc.status || got["message"] != tc.message {
				t.Fatalf("response=%v", got)
			}
			if tc.reader != nil && tc.reader.calls != tc.calls {
				t.Fatalf("calls=%d", tc.reader.calls)
			}
			if tc.calls == 1 && tc.reader.ctx != ctx {
				t.Fatal("context replaced")
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), "password") || strings.Contains(string(encoded), "private-backend") {
				t.Fatal("error leak")
			}
			if tc.status != 200 && len(got) != 2 {
				t.Fatal("error response shape changed")
			}
			if tc.name == "empty" && !strings.Contains(string(encoded), `"header_buttons":[]`) {
				t.Fatal("empty list is not []")
			}
			if tc.name == "success" {
				want := []Button{{Hbid: "7", Title: "Ana"}, {Hbid: "12", Title: "Alt", ParentId: "9"}}
				if !reflect.DeepEqual(got["header_buttons"], want) {
					t.Fatal("mapping changed")
				}
				if !strings.Contains(string(encoded), `"parent_id":""`) || !strings.Contains(string(encoded), `"hbid":"7"`) {
					t.Fatal("ID/null JSON changed")
				}
			}
		})
	}
}
