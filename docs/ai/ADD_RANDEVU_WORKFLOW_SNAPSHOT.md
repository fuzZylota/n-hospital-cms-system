# AddRandevu workflow snapshot — wired

Status: `AddRandevu` reads the owned `AppointmentWorkflowSnapshotReader` through
`appointmentworkflowsnapshot.Read`. Production reader caller count: 1. The
handler's legacy `FetchOptionsForBackend` call count: 0. The existing
`optionsRepository` is injected from `main`; no additional pool or repository
is created.

## Source evidence and decision

- `fiber-v2/controllers/post/randevular/randevular.go:AddRandevu` reads options
  after authentication, role, body, and required-field checks, but before doctor
  availability lookup and `Orm.Begin`. The new helper preserves that position.
  Missing options and backend failure return the same generic JSON 500 payload
  with the existing HTTP 200 behavior. The log records only the fixed operation
  and stage; backend details and partial secret-bearing snapshots are discarded.
- The former `FetchOptionsForBackend` projection asked for upload size,
  description, and social URLs as well as mail/theme values. Following the
  actual reads of `GetOptions` through the whole function shows only SMTP host,
  port, username, password, site name, contact email, contact phone, primary
  color, secondary color, and logo path are consumed. It does not read CAPTCHA
  options or call CAPTCHA verification. The logo path is joined application
  data, not permission to read a filesystem path.
- The legacy fetch in `fiber-v2/database/database.go:FetchOptionsForBackend`
  filters `option_set_is_active = true`, left joins the logo media, converts SQL
  NULL option/media values to zero values, returns an error for no row, and
  takes the first row if multiple active rows exist. The owned reader selects
  one active option row and its logo in a single statement, retains
  active/testing identity flags independently, returns `found=false` for
  missing, and rejects duplicate active rows as a cardinality error. With the
  handler now wired, duplicate active rows result in a safe 500 rather than
  choosing one arbitrarily. Production row cardinality has not been observed.
- The handler checks doctor availability after the options read, inserts in a
  transaction, and commits before considering email. Email is attempted only
  when all four SMTP values and patient email are nonempty, then `ROOT_DIRECTORY`
  must be present. Branch and doctor name reads happen during message building;
  send failure is logged and does not undo the committed appointment. No
  CAPTCHA action occurs in `AddRandevu`.
- Existing `AppointmentRequestWorkflowSnapshot` serves
  `AddRandevuRequest` and includes CAPTCHA, social, and description fields.
  Existing response snapshots omit secondary color. None matches this exact
  set. `AppointmentWorkflowSnapshot` supplies only the `AddRandevu` values.
  `fiber-v2/main/main.go` assigns the same `optionsRepository` to its utility
  field. The schema and dependency manifests are unchanged.

## Verification boundary

- `fiber-v2/controllers/post/appointmentworkflowsnapshot/decision_test.go`
  covers found/missing/backend/nil reads, secret-safe errors, and context
  identity. `fiber-v2/controllers/post/randevular/appointment_workflow_test.go`
  checks actual early HTTP statuses and JSON, source-level operation order,
  field use, main injection, and production reader caller count.
- `fiber-v2/database/postgres/appointment_workflow_snapshot_test.go` covers
  SQL, Scan, NULL, active identity, missing, duplicate, context, and safe
  errors using a DB-free driver. These tests do not exercise a real PostgreSQL
  transaction, SMTP send, or full application runtime. The local `main` test
  could not complete because required modules were absent from the offline
  cache; source-level main injection checks ran through the handler tests.

Confidence: high for repository source, early handler HTTP behavior, and DB-free
reader behavior. Production database contents, SMTP delivery, and runtime
behavior were not observed; production state remains UNKNOWN.
