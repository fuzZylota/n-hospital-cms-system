# EditRandevu workflow snapshot — unwired reuse

Status: `EditRandevu` still calls `FetchOptionsForBackend`; no handler, `main`,
utility, schema, or dependency wiring changed. The existing
`data.AppointmentWorkflowSnapshot` and
`postgres.OptionsRepository.ReadAppointmentWorkflowSnapshot` are the prepared
contract and PostgreSQL reader. Their production consumer remains `AddRandevu`
only (one call); `EditRandevu` has zero snapshot-reader calls. No new reader is
needed because the existing projection matches this edit flow exactly.

## Source evidence and decision

- `fiber-v2/controllers/post/randevular/randevular.go:EditRandevu` reads options
  after authentication, role, body parsing, and nonempty `inputs.Rid`, before
  the appointment existence lookup, optional doctor availability check, and
  `Orm.Begin`. The route `rid` is read separately; the existence and update
  queries use `inputs.Rid`.
- The legacy projection asks for upload size, description, and social URLs in
  addition to mail and display fields. Actual `GetOptions` reads in the full
  edit handler are SMTP host, port, username, password, site name, contact
  email, contact phone, primary color, secondary color, and logo path. The
  existing `AppointmentWorkflowSnapshot` has exactly those ten values plus
  option-set identity. It does not add CAPTCHA or unrelated option values.
- `fiber-v2/database/database.go:FetchOptionsForBackend` filters
  `option_set_is_active = true`, left joins logo media, converts SQL NULL
  option/media values to zero values, errors when no row exists, and takes
  `rows[0]` if multiple active rows exist. The prepared PostgreSQL reader
  selects an active row and its logo in one statement. It returns a zero
  snapshot and `found=false` when missing; nullable option/media fields map
  to zero values; active and testing identity flags remain independent. It
  rejects duplicate active rows and invalid identity with a safe error,
  rather than selecting an arbitrary row. This stricter duplicate behavior
  matters only if a later task wires the reader; production row cardinality
  has not been observed.
- The edit handler checks the appointment and optional availability after
  options. It begins a transaction, updates, checks affected rows, and commits
  before evaluating email. Mail requires an appointment date/time/duration/
  doctor change, a nonempty nonzero doctor ID, all four SMTP settings, and a
  patient email. A missing `ROOT_DIRECTORY` suppresses mail; otherwise branch
  and doctor names are read and a logo attachment path is built. Mail failure
  is logged after commit and does not roll back the update. The logo value is
  application data, not filesystem-path authority.
- `fiber-v2/database/postgres/appointment_workflow_snapshot_test.go` already
  exercises exact SQL projection and Scan order, each nullable field, missing
  and duplicate rows, identity validation, caller context, and safe errors
  with a database-free driver. The edit-specific source test checks exact
  field reuse, the legacy read location, transaction/mail order, and the zero
  edit consumer boundary.

Decision: reuse the existing contract and reader. Wiring and any change to
the handler's current HTTP or duplicate-row behavior require a separate
review. Confidence is high for repository source and database-free reader
behavior. Production database, transaction, and SMTP behavior were not
observed; production state remains `UNKNOWN`.
