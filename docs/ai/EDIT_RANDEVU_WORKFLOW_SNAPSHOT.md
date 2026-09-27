# EditRandevu workflow snapshot — wired

Status: `EditRandevu` reads the existing `data.AppointmentWorkflowSnapshot`
through `appointmentworkflowsnapshot.Read` and
`utilities.AppointmentWorkflowSnapshotReader`. Its legacy
`FetchOptionsForBackend` call count is zero; its snapshot-helper call count is
one. `AddRandevu` retains its own call to the same helper and reader. `main`
already injects the single `optionsRepository` into the utility field, so no
new pool, repository, utility field, schema, or dependency was added.

## Source evidence and behavior

- `fiber-v2/controllers/post/randevular/randevular.go:EditRandevu` reads the
  snapshot after authentication, role, body parsing, and nonempty `inputs.Rid`,
  at the former options-read point. Appointment existence and optional doctor
  availability queries, `Orm.Begin`, update, affected-row check, and commit
  remain afterward. The route `rid` and body `inputs.Rid` behavior is unchanged.
- The handler consumes only SMTP host, port, username, password, site name,
  contact email, contact phone, primary color, secondary color, and site logo
  path. These are exactly the existing appointment snapshot fields. It does
  not read CAPTCHA or the upload, description, social, or other unused options.
  `SMTPPassword` stays server-internal. `SiteLogoPath` is application data,
  not filesystem-path authority.
- The shared PostgreSQL reader uses one active-row statement, maps nullable
  option/media values to zero values, retains active and testing identity
  flags independently, and returns `found=false` for missing. It rejects
  duplicate active rows and invalid identity with a safe error. The legacy
  fetch chose the first duplicate row; now duplicate active rows reach the
  handler's existing generic options-read JSON 500 path. Production row
  cardinality has not been observed.
- The snapshot helper discards partial values and backend details on missing
  or failure. Only the caller context determines canceled/deadline error
  identity. The handler keeps the existing generic JSON payload and HTTP 200
  behavior for early validation and options failures.
- Conditional mail remains after commit. Its guards still require an
  appointment date/time/duration/doctor change, a nonempty nonzero doctor ID,
  four SMTP values, and a patient email. Missing `ROOT_DIRECTORY` suppresses
  mail. Branch/doctor reads and logo attachment building remain in the mail
  branch; send failure is logged without rolling back the committed edit.

## Verification boundary

- `fiber-v2/controllers/post/randevular/edit_appointment_workflow_snapshot_test.go`
  checks actual early HTTP/JSON outcomes and caller-context forwarding, exact
  option fields, source operation order, post-commit mail behavior, one edit
  helper call, zero legacy calls, and reuse of the single main repository.
  Existing AddRandevu tests retain the other helper call.
- `fiber-v2/controllers/post/appointmentworkflowsnapshot/decision_test.go`
  covers found, missing, backend, nil, and context behavior with safe errors.
  `fiber-v2/database/postgres/appointment_workflow_snapshot_test.go` covers
  SQL/Scan, NULL, missing, duplicate, identity, context, and safe reader errors
  with a database-free driver.

Confidence is high for repository source and database-free test behavior.
Production database contents, transaction execution, SMTP delivery, and full
application runtime were not observed; production state remains `UNKNOWN`.
