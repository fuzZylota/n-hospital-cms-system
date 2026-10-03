package randevular

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"lib"
	"models"
	"models/data"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

type createFixture struct {
	mu                                                                        sync.Mutex
	role                                                                      string
	active                                                                    bool
	requestSID                                                                int64
	requestPresent, requestSIDNull, linked, branchExists, doctorExists        bool
	conflicts                                                                 int64
	failure                                                                   string
	userReads, requestReads, linkedReads, branchReads, doctorReads, slotReads int
	insertCount, postReads, beginCount, commitCount, rollbackCount            int
	insertArgs                                                                []driver.NamedValue
	events                                                                    []string
	userSID                                                                   sql.NullInt64
	perms                                                                     map[int64]bool
	sidReads, permReads                                                       int
	requestLock                                                               chan struct{}
	requestLockAttempt                                                        chan struct{}
	commitGate                                                                chan struct{}
	commitEntered                                                             chan struct{}
}

func readyCreateFixture(sid int64) *createFixture {
	f := &createFixture{role: "admin", active: true, userSID: sql.NullInt64{Int64: sid, Valid: true}, perms: branchPerms(sid), requestSID: sid, requestPresent: true, branchExists: true, doctorExists: true, requestLock: make(chan struct{}, 1)}
	f.requestLock <- struct{}{}
	return f
}

// branchPerms: sid -> can_view=true satırları (listede olmayan şubede satır yoktur).
func branchPerms(sids ...int64) map[int64]bool {
	m := map[int64]bool{}
	for _, sid := range sids {
		m[sid] = true
	}
	return m
}

func (f *createFixture) record(event string) {
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
}

func (f *createFixture) Connect(context.Context) (driver.Conn, error) { return &createConn{f: f}, nil }
func (*createFixture) Driver() driver.Driver                          { return createDriver{} }

type createDriver struct{}

func (createDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type createConn struct {
	f                      *createFixture
	inTx, locked, inserted bool
}

func (*createConn) Close() error                        { return nil }
func (*createConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected prepare") }
func (c *createConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *createConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != driver.IsolationLevel(sql.LevelReadCommitted) {
		return nil, errors.New("conversion requires read committed")
	}
	c.f.mu.Lock()
	c.f.beginCount++
	fail := c.f.failure == "begin"
	c.f.mu.Unlock()
	if fail {
		return nil, errors.New("private begin detail")
	}
	c.inTx = true
	return &createTx{c: c}, nil
}
func (c *createConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	f := c.f
	if strings.HasPrefix(query, "INSERT INTO randevular") {
		if !c.inTx || !strings.Contains(query, "RETURNING rid") {
			return nil, errors.New("insert outside transaction")
		}
		f.mu.Lock()
		f.insertCount++
		f.insertArgs = append([]driver.NamedValue(nil), args...)
		fail := f.failure
		f.events = append(f.events, "insert")
		f.mu.Unlock()
		if fail == "insert" {
			return nil, errors.New("private insert detail")
		}
		if fail == "zero_rows" {
			return createRowsFor([]string{"rid"}, nil), nil
		}
		if fail == "zero_rid" {
			return createRowsFor([]string{"rid"}, [][]driver.Value{{int64(0)}}), nil
		}
		c.inserted = true
		return createRowsFor([]string{"rid"}, [][]driver.Value{{int64(91)}}), nil
	}
	if strings.Contains(query, "FROM subeler WHERE sid = $1") && !c.inTx {
		f.mu.Lock()
		f.postReads++
		fail := f.failure == "post_branch"
		empty := f.failure == "post_branch_empty"
		f.mu.Unlock()
		if fail {
			return nil, errors.New("private branch name detail")
		}
		if empty {
			return createRowsFor([]string{"name"}, nil), nil
		}
		return createRowsFor([]string{"name"}, [][]driver.Value{{"Branch"}}), nil
	}
	if strings.Contains(query, "FROM doktorlar WHERE drid = $1 AND sid = $2") && !c.inTx {
		f.mu.Lock()
		f.postReads++
		fail := f.failure == "post_doctor"
		empty := f.failure == "post_doctor_empty"
		f.mu.Unlock()
		if fail {
			return nil, errors.New("private doctor name detail")
		}
		if empty {
			return createRowsFor([]string{"title", "first_name", "last_name"}, nil), nil
		}
		return createRowsFor([]string{"title", "first_name", "last_name"}, [][]driver.Value{{"Dr.", "A", "B"}}), nil
	}
	if !c.inTx {
		return nil, errors.New("read outside transaction")
	}
	switch {
	case query == "SELECT sid FROM users WHERE uid = $1 FOR UPDATE":
		f.mu.Lock()
		f.sidReads++
		sid := f.userSID
		f.mu.Unlock()
		if len(args) != 1 || args[0].Value != int64(7) {
			return nil, errors.New("wrong user")
		}
		var v driver.Value
		if sid.Valid {
			v = sid.Int64
		}
		return createRowsFor([]string{"sid"}, [][]driver.Value{{v}}), nil
	case query == "SELECT can_view FROM user_branch_permissions WHERE uid = $1 AND sid = $2 FOR UPDATE":
		f.mu.Lock()
		f.permReads++
		canView, ok := f.perms[args[1].Value.(int64)]
		f.mu.Unlock()
		if len(args) != 2 || args[0].Value != int64(7) {
			return nil, errors.New("wrong permission key")
		}
		if !ok {
			return createRowsFor([]string{"can_view"}, nil), nil
		}
		return createRowsFor([]string{"can_view"}, [][]driver.Value{{canView}}), nil
	case strings.Contains(query, "FROM users"):
		if !strings.Contains(query, "FOR UPDATE") {
			return nil, errors.New("user not locked")
		}
		f.mu.Lock()
		f.userReads++
		fail, role, active := f.failure, f.role, f.active
		f.mu.Unlock()
		if fail == "user" {
			return nil, errors.New("private user detail")
		}
		if fail == "user_missing" {
			return createRowsFor([]string{"role", "is_active"}, nil), nil
		}
		return createRowsFor([]string{"role", "is_active"}, [][]driver.Value{{role, active}}), nil
	case strings.Contains(query, "FROM randevu_talepleri"):
		if !strings.Contains(query, "FOR UPDATE") {
			return nil, errors.New("request not locked")
		}
		f.mu.Lock()
		attempt := f.requestLockAttempt
		f.mu.Unlock()
		if attempt != nil {
			attempt <- struct{}{}
		}
		select {
		case <-f.requestLock:
			c.locked = true
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		f.mu.Lock()
		f.requestReads++
		fail, present, sid, nullSID := f.failure, f.requestPresent, f.requestSID, f.requestSIDNull
		f.mu.Unlock()
		if fail == "request" {
			return nil, errors.New("private request detail")
		}
		if !present {
			return createRowsFor([]string{"sid"}, nil), nil
		}
		if nullSID {
			return createRowsFor([]string{"sid"}, [][]driver.Value{{nil}}), nil
		}
		return createRowsFor([]string{"sid"}, [][]driver.Value{{sid}}), nil
	case strings.Contains(query, "FROM randevular WHERE rrid"):
		if !c.locked {
			return nil, errors.New("linked read without request lock")
		}
		f.mu.Lock()
		f.linkedReads++
		fail, linked := f.failure, f.linked
		f.mu.Unlock()
		if fail == "linked" {
			return nil, errors.New("private linked detail")
		}
		return createRowsFor([]string{"exists"}, [][]driver.Value{{linked}}), nil
	case strings.Contains(query, "FROM subeler"):
		f.mu.Lock()
		f.branchReads++
		fail, exists := f.failure, f.branchExists
		f.mu.Unlock()
		if fail == "branch" {
			return nil, errors.New("private branch detail")
		}
		return createRowsFor([]string{"exists"}, [][]driver.Value{{exists}}), nil
	case strings.Contains(query, "FROM doktorlar"):
		if !strings.Contains(query, "sid = $2") || !strings.Contains(query, "is_active = true") {
			return nil, errors.New("doctor branch predicate missing")
		}
		f.mu.Lock()
		f.doctorReads++
		fail, exists := f.failure, f.doctorExists
		f.mu.Unlock()
		if fail == "doctor" {
			return nil, errors.New("private doctor detail")
		}
		return createRowsFor([]string{"exists"}, [][]driver.Value{{exists}}), nil
	case strings.Contains(query, "COUNT(*) FROM randevular"):
		if !strings.Contains(query, "appointment_time < $4") {
			return nil, errors.New("overlap predicate missing")
		}
		f.mu.Lock()
		f.slotReads++
		fail, count := f.failure, f.conflicts
		f.mu.Unlock()
		if fail == "slot" {
			return nil, errors.New("private slot detail")
		}
		return createRowsFor([]string{"count"}, [][]driver.Value{{count}}), nil
	default:
		return nil, errors.New("unexpected query")
	}
}

type createTx struct{ c *createConn }

func (t *createTx) release() {
	if t.c.locked {
		t.c.f.requestLock <- struct{}{}
		t.c.locked = false
	}
	t.c.inTx = false
}
func (t *createTx) Commit() error {
	f := t.c.f
	if f.commitEntered != nil {
		close(f.commitEntered)
		<-f.commitGate
	}
	f.mu.Lock()
	f.commitCount++
	f.events = append(f.events, "commit")
	fail := f.failure == "commit"
	if !fail && t.c.inserted && t.c.locked {
		f.linked = true
	}
	f.mu.Unlock()
	t.release()
	if fail {
		return errors.New("private commit detail")
	}
	return nil
}
func (t *createTx) Rollback() error {
	t.c.f.mu.Lock()
	t.c.f.rollbackCount++
	t.c.f.events = append(t.c.f.events, "rollback")
	fail := t.c.f.failure == "rollback"
	t.c.f.mu.Unlock()
	t.release()
	if fail {
		return errors.New("private rollback detail")
	}
	return nil
}

type createRows struct {
	columns []string
	rows    [][]driver.Value
	cursor  int
}

func createRowsFor(columns []string, rows [][]driver.Value) driver.Rows {
	return &createRows{columns: columns, rows: rows}
}
func (r *createRows) Columns() []string { return r.columns }
func (*createRows) Close() error        { return nil }
func (r *createRows) Next(dest []driver.Value) error {
	if r.cursor >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.cursor])
	r.cursor++
	return nil
}

type createSnapshotReader struct {
	mu    sync.Mutex
	calls int
	fail  bool
	mail  bool
}

func (r *createSnapshotReader) ReadAppointmentWorkflowSnapshot(context.Context) (data.AppointmentWorkflowSnapshot, bool, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	if r.fail {
		return data.AppointmentWorkflowSnapshot{}, false, errors.New("private options detail")
	}
	snapshot := data.AppointmentWorkflowSnapshot{}
	if r.mail {
		snapshot.SMTPHost = "smtp.invalid"
		snapshot.SMTPPort = 25
		snapshot.SMTPUsername = "test"
		snapshot.SMTPPassword = "synthetic-secret"
	}
	return snapshot, true, nil
}
func (r *createSnapshotReader) count() int { r.mu.Lock(); defer r.mu.Unlock(); return r.calls }

func createHTTP(t *testing.T, f *createFixture, reader *createSnapshotReader, jwtRole, body string) (int, int, string) {
	t.Helper()
	t.Setenv("JWT_SECRET", "local-create-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	db.SetMaxOpenConns(3)
	t.Cleanup(func() { db.Close() })
	app := fiber.New()
	app.Post("/backend/add-randevu", AddRandevu(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}, AppointmentWorkflowSnapshotReader: reader}))
	request := httptest.NewRequest("POST", "/backend/add-randevu", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if jwtRole != "" {
		token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "7", Role: jwtRole, LastLogin: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Cookie", "n-hospital-auth="+token)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, result.Status, result.Message
}

const createManualBody = `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555","sid":"1","appointment_date":"2026-10-01T00:00:00Z","appointment_time":"2026-10-01T10:00:00Z","duration":30}`
const createRequestBody = `{"patient_first_name":"Ada","patient_last_name":"Yilmaz","patient_phone":"555","sid":"1","rrid":"11","drid":"5","appointment_date":"2026-10-01T00:00:00Z","appointment_time":"2026-10-01T10:00:00Z","duration":30}`

func TestCreateAppointmentCurrentRolesAndManualNullHTTP(t *testing.T) {
	t.Setenv("ROOT_DIRECTORY", t.TempDir())
	for _, role := range []string{"ik", "other"} {
		t.Run(role, func(t *testing.T) {
			f, reader := readyCreateFixture(1), &createSnapshotReader{mail: true}
			f.role = role
			original := addAppointmentMailHook
			t.Cleanup(func() { addAppointmentMailHook = original })
			mailCalls := 0
			addAppointmentMailHook = func(*models.EmailInfos) error { mailCalls++; return nil }
			body := strings.ReplaceAll(createRequestBody, `"patient_phone":"555"`, `"patient_phone":"555","patient_email":"synthetic@example.invalid"`)
			http, status, message := createHTTP(t, f, reader, "admin", body)
			if http != 403 || status != 403 || message != "Forbidden" || f.userReads != 1 || f.sidReads != 0 || f.permReads != 0 || f.requestReads != 0 || f.insertCount != 0 || f.postReads != 0 || f.rollbackCount != 1 || reader.count() != 0 || mailCalls != 0 {
				t.Fatalf("denial leaked work: http=%d status=%d fixture=%+v", http, status, f)
			}
		})
	}
	t.Run("inactive admin", func(t *testing.T) {
		f, reader := readyCreateFixture(1), &createSnapshotReader{}
		f.active = false
		_, status, _ := createHTTP(t, f, reader, "admin", createRequestBody)
		if status != 403 || f.requestReads != 0 || f.insertCount != 0 || reader.count() != 0 {
			t.Fatalf("inactive admin reached data: %+v", f)
		}
	})
	for _, sid := range []int64{1, 2} {
		t.Run("admin branch", func(t *testing.T) {
			f, reader := readyCreateFixture(sid), &createSnapshotReader{}
			body := strings.ReplaceAll(createManualBody, `"sid":"1"`, `"sid":"`+string(rune('0'+sid))+`"`)
			http, status, _ := createHTTP(t, f, reader, "moderator", body)
			if http != 200 || status != 201 || f.commitCount != 1 || f.insertCount != 1 || reader.count() != 1 || len(f.insertArgs) < 5 || f.insertArgs[1].Value != nil || f.insertArgs[4].Value != sid {
				t.Fatalf("manual SQL NULL or current admin failed: http=%d status=%d args=%v fixture=%+v", http, status, f.insertArgs, f)
			}
		})
	}
}

func TestCreateAppointmentBranchWriteAccessHTTP(t *testing.T) {
	t.Setenv("ROOT_DIRECTORY", t.TempDir())
	original := addAppointmentMailHook
	t.Cleanup(func() { addAppointmentMailHook = original })
	addAppointmentMailHook = func(*models.EmailInfos) error { return nil }
	null := sql.NullInt64{}
	for _, tc := range []struct {
		name  string
		setup func(*createFixture)
		want  int
		sidQ  int
		permQ int
	}{
		{"admin no extra queries", func(f *createFixture) { f.perms = nil }, 201, 0, 0},
		{"moderator can_view", func(f *createFixture) { f.role = "moderator"; f.userSID = null }, 201, 0, 1},
		{"moderator no permission row", func(f *createFixture) { f.role = "moderator"; f.perms = nil }, 403, 0, 1},
		{"moderator can_view false", func(f *createFixture) { f.role = "moderator"; f.perms = map[int64]bool{1: false} }, 403, 0, 1},
		{"moderator other branch only", func(f *createFixture) { f.role = "moderator"; f.perms = branchPerms(2) }, 403, 0, 1},
		{"santral matching sid and can_view", func(f *createFixture) { f.role = "santral" }, 201, 1, 1},
		{"santral sid mismatch", func(f *createFixture) { f.role = "santral"; f.userSID = sql.NullInt64{Int64: 2, Valid: true} }, 403, 1, 0},
		{"santral NULL sid", func(f *createFixture) { f.role = "santral"; f.userSID = null }, 403, 1, 0},
		{"santral sid ok but no can_view", func(f *createFixture) { f.role = "santral"; f.perms = nil }, 403, 1, 1},
		{"ik", func(f *createFixture) { f.role = "ik" }, 403, 0, 0},
		{"other", func(f *createFixture) { f.role = "other" }, 403, 0, 0},
		{"inactive moderator", func(f *createFixture) { f.role = "moderator"; f.active = false }, 403, 0, 0},
		{"missing user", func(f *createFixture) { f.failure = "user_missing" }, 403, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reader := readyCreateFixture(1), &createSnapshotReader{}
			tc.setup(f)
			http, status, _ := createHTTP(t, f, reader, "ik", createManualBody)
			if f.sidReads != tc.sidQ || f.permReads != tc.permQ {
				t.Fatalf("permission queries sid=%d perm=%d: %+v", f.sidReads, f.permReads, f)
			}
			if tc.want == 201 {
				if http != 200 || status != 201 || f.insertCount != 1 || f.commitCount != 1 {
					t.Fatalf("allowed create failed: %d/%d %+v", http, status, f)
				}
				return
			}
			if http != 403 || status != 403 || f.insertCount != 0 || f.commitCount != 0 || f.rollbackCount != 1 || f.postReads != 0 || reader.count() != 0 {
				t.Fatalf("denied create leaked work: %d/%d %+v", http, status, f)
			}
		})
	}
	t.Run("moderator converting request keeps branch binding", func(t *testing.T) {
		f, reader := readyCreateFixture(1), &createSnapshotReader{}
		f.role = "moderator"
		http, status, _ := createHTTP(t, f, reader, "ik", createRequestBody)
		if http != 200 || status != 201 || f.requestReads != 1 || f.insertCount != 1 {
			t.Fatalf("moderator conversion failed: %d/%d %+v", http, status, f)
		}
		f2 := readyCreateFixture(2)
		f2.role, f2.perms = "moderator", branchPerms(1)
		_, status, _ = createHTTP(t, f2, reader, "ik", createRequestBody)
		if status != 409 || f2.insertCount != 0 {
			t.Fatalf("request from other branch converted: %d %+v", status, f2)
		}
	})
}

func TestCreateAppointmentConversionGuardsHTTP(t *testing.T) {
	original := addAppointmentMailHook
	t.Cleanup(func() { addAppointmentMailHook = original })
	t.Setenv("ROOT_DIRECTORY", t.TempDir())
	for _, tc := range []struct {
		name, body, failure                    string
		sid                                    int64
		present, nullSID, linked, doctorExists bool
		conflicts                              int64
		want                                   int
	}{
		{"correct", createRequestBody, "", 1, true, false, false, true, 0, 201},
		{"correct second branch", strings.ReplaceAll(createRequestBody, `"sid":"1"`, `"sid":"2"`), "", 2, true, false, false, true, 0, 201},
		{"other branch", strings.ReplaceAll(createRequestBody, `"sid":"1"`, `"sid":"2"`), "", 1, true, false, false, true, 0, 409},
		{"missing request", createRequestBody, "", 1, false, false, false, true, 0, 404},
		{"unknown request branch", createRequestBody, "", 1, true, true, false, true, 0, 409},
		{"repeat", createRequestBody, "", 1, true, false, true, true, 0, 409},
		{"doctor not in primary branch", createRequestBody, "", 1, true, false, false, false, 0, 400},
		{"slot overlap", createRequestBody, "", 1, true, false, false, true, 1, 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, reader := readyCreateFixture(tc.sid), &createSnapshotReader{mail: true}
			mailCalls := 0
			addAppointmentMailHook = func(*models.EmailInfos) error { mailCalls++; return nil }
			f.requestPresent, f.requestSIDNull, f.linked, f.doctorExists, f.conflicts = tc.present, tc.nullSID, tc.linked, tc.doctorExists, tc.conflicts
			body := strings.ReplaceAll(tc.body, `"patient_phone":"555"`, `"patient_phone":"555","patient_email":"synthetic@example.invalid"`)
			http, status, message := createHTTP(t, f, reader, "admin", body)
			wantHTTP := tc.want
			if tc.want == 201 {
				wantHTTP = 200
			}
			if http != wantHTTP || status != tc.want {
				t.Fatalf("status http=%d json=%d want=%d", http, status, tc.want)
			}
			if tc.want == 201 {
				if f.requestReads != 1 || f.linkedReads != 1 || f.insertCount != 1 || f.commitCount != 1 || f.insertArgs[1].Value != int64(11) {
					t.Fatalf("conversion lost locked request: %+v args=%v", f, f.insertArgs)
				}
			} else if f.insertCount != 0 || f.postReads != 0 || reader.count() != 0 || f.rollbackCount != 1 || mailCalls != 0 || strings.Contains(message, "Ada") || strings.Contains(message, "555") || strings.Contains(message, "synthetic@example.invalid") {
				t.Fatalf("rejected conversion had side effects: %+v", f)
			}
		})
	}
	for _, rrid := range []string{"x", "0", "-1", "011"} {
		f, reader := readyCreateFixture(1), &createSnapshotReader{}
		body := strings.ReplaceAll(createRequestBody, `"rrid":"11"`, `"rrid":"`+rrid+`"`)
		_, status, _ := createHTTP(t, f, reader, "admin", body)
		if status != 400 || f.beginCount != 0 || reader.count() != 0 {
			t.Fatalf("invalid rrid reached DB: %s %+v", rrid, f)
		}
	}
}

func TestCreateAppointmentTransactionFailuresHTTP(t *testing.T) {
	for _, tc := range []struct {
		stage string
		want  int
	}{
		{"begin", 503}, {"user", 503}, {"user_missing", 403}, {"request", 503}, {"linked", 503}, {"branch", 503}, {"doctor", 503}, {"slot", 503}, {"options", 503}, {"insert", 500}, {"zero_rows", 500}, {"zero_rid", 500}, {"commit", 500},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			f, reader := readyCreateFixture(1), &createSnapshotReader{}
			if tc.stage == "options" {
				reader.fail = true
			} else {
				f.failure = tc.stage
			}
			_, status, message := createHTTP(t, f, reader, "admin", createRequestBody)
			if status != tc.want || strings.Contains(message, "private") || f.commitCount > 0 && tc.stage != "commit" || f.rollbackCount != f.beginCount && tc.stage != "commit" && tc.stage != "begin" {
				t.Fatalf("failed transaction lifecycle: status=%d message=%q fixture=%+v", status, message, f)
			}
			if tc.stage != "commit" && f.postReads != 0 {
				t.Fatal("read after failed transaction")
			}
		})
	}
}

func TestCreateAppointmentRollbackFailureDoesNotExposeDetailsHTTP(t *testing.T) {
	f, reader := readyCreateFixture(1), &createSnapshotReader{mail: true}
	f.role, f.failure = "ik", "rollback"
	http, status, message := createHTTP(t, f, reader, "admin", createRequestBody)
	if http != 403 || status != 403 || message != "Forbidden" || f.rollbackCount != 1 || f.insertCount != 0 || f.postReads != 0 || reader.count() != 0 {
		t.Fatalf("rollback failure changed denial: http=%d status=%d fixture=%+v", http, status, f)
	}
}

func TestCreateAppointmentCommitBeforeMailAndReadFailuresHTTP(t *testing.T) {
	original := addAppointmentMailHook
	t.Cleanup(func() { addAppointmentMailHook = original })
	t.Setenv("ROOT_DIRECTORY", t.TempDir())
	for _, stage := range []string{"", "post_branch", "post_branch_empty", "post_doctor", "post_doctor_empty", "mail"} {
		t.Run(stage, func(t *testing.T) {
			f, reader := readyCreateFixture(1), &createSnapshotReader{mail: true}
			if stage != "mail" {
				f.failure = stage
			}
			addAppointmentMailHook = func(*models.EmailInfos) error {
				f.record("mail")
				if stage == "mail" {
					return errors.New("private SMTP detail")
				}
				return nil
			}
			body := strings.ReplaceAll(createRequestBody, `"patient_phone":"555"`, `"patient_phone":"555","patient_email":"synthetic@example.invalid"`)
			http, status, message := createHTTP(t, f, reader, "admin", body)
			if http != 200 || status != 201 || strings.Contains(message, "private") || f.commitCount != 1 || f.rollbackCount != 0 || f.events[len(f.events)-1] != "mail" {
				t.Fatalf("post-commit contract failed: http=%d status=%d events=%v", http, status, f.events)
			}
			commitIndex, mailIndex := -1, -1
			for i, event := range f.events {
				if event == "commit" {
					commitIndex = i
				}
				if event == "mail" {
					mailIndex = i
				}
			}
			if commitIndex < 0 || mailIndex <= commitIndex {
				t.Fatalf("mail preceded commit: %v", f.events)
			}
		})
	}
}

func TestCreateAppointmentSecondConversionWaitsForRequestLockHTTP(t *testing.T) {
	f := readyCreateFixture(1)
	f.commitGate, f.commitEntered = make(chan struct{}), make(chan struct{})
	reader := &createSnapshotReader{}
	t.Setenv("JWT_SECRET", "local-create-test-key")
	t.Setenv("AUTH_COOKIE_NAME", "n-hospital-auth")
	db := sql.OpenDB(f)
	db.SetMaxOpenConns(3)
	defer db.Close()
	app := fiber.New()
	app.Post("/backend/add-randevu", AddRandevu(nil, &models.Utilities{Orm: &orm.Neorm{Pool: db}, AppointmentWorkflowSnapshotReader: reader}))
	token, err := lib.CreateJWT(models.AuthenticatedUser{Uid: "7", Role: "admin", LastLogin: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	send := func() int {
		req := httptest.NewRequest("POST", "/backend/add-randevu", strings.NewReader(createRequestBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", "n-hospital-auth="+token)
		response, err := app.Test(req)
		if err != nil {
			return 0
		}
		defer response.Body.Close()
		var body struct {
			Status int `json:"status"`
		}
		if json.NewDecoder(response.Body).Decode(&body) != nil {
			return 0
		}
		return body.Status
	}
	first, second := make(chan int, 1), make(chan int, 1)
	go func() { first <- send() }()
	select {
	case <-f.commitEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("first conversion did not reach commit")
	}
	f.mu.Lock()
	f.requestLockAttempt = make(chan struct{}, 1)
	attempt := f.requestLockAttempt
	f.mu.Unlock()
	go func() { second <- send() }()
	select {
	case <-attempt:
	case <-time.After(2 * time.Second):
		t.Fatal("second conversion did not attempt the request lock")
	}
	select {
	case status := <-second:
		t.Fatalf("second conversion bypassed request lock: %d", status)
	case <-time.After(30 * time.Millisecond):
	}
	close(f.commitGate)
	if a, b := <-first, <-second; a != 201 || b != 409 || f.insertCount != 1 || f.requestReads != 2 {
		t.Fatalf("concurrent conversions: first=%d second=%d fixture=%+v", a, b, f)
	}
}
