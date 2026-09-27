# AddRandevu workflow snapshot — UNWIRED

Status: review-ready, not connected to the production handler or application
startup. Production consumer count for `ReadAppointmentWorkflowSnapshot`: 0.

## Source evidence and decision

- `fiber-v2/controllers/post/randevular/randevular.go:AddRandevu` reads options
  after authentication, role, body, and required-field checks, but before doctor
  availability lookup and `Orm.Begin`. An options error returns a generic 500
  payload and stops the workflow.
- The handler's legacy `FetchOptionsForBackend` projection asks for upload size,
  description, and social URLs as well as mail/theme values. Following the
  actual reads of `GetOptions` through the whole function shows only SMTP host,
  port, username, password, site name, contact email, contact phone, primary
  color, secondary color, and logo path are consumed. It does not read CAPTCHA
  options or call CAPTCHA verification. The logo path is joined application
  data, not permission to read a filesystem path.
- The legacy fetch in `fiber-v2/database/database.go:FetchOptionsForBackend`
  filters `option_set_is_active = true`, left joins the logo media, converts SQL
  NULL option/media values to zero values, returns an error for no row, and
  takes the first row if multiple active rows exist. The new reader selects one
  active option row and its logo in a single statement, retains active/testing
  identity flags independently, returns `found=false` for missing, and rejects
  duplicate active rows as a cardinality error. This stricter duplicate policy
  is intentional for a future consumer and does not affect production now.
- The handler checks doctor availability after the options read, inserts in a
  transaction, and commits before considering email. Email is attempted only
  when all four SMTP values and patient email are nonempty, then `ROOT_DIRECTORY`
  must be present. Branch and doctor name reads happen during message building;
  send failure is logged and does not undo the committed appointment. No
  CAPTCHA action occurs in `AddRandevu`.
- Existing `AppointmentRequestWorkflowSnapshot` serves
  `AddRandevuRequest` and includes CAPTCHA, social, and description fields.
  Existing response snapshots omit secondary color. None matches this exact
  set, so `AppointmentWorkflowSnapshot` and its PostgreSQL reader were added
  without changing the handler, startup wiring, schema, or dependencies.

Confidence: high for repository source and DB-free reader behavior. Production
database contents and runtime behavior were not observed; production state
remains UNKNOWN.
