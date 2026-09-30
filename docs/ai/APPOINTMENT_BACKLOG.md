# Nivgöz — appointment and request backlog

Source: accepted Phase 1 audit. Initial findings and flow are historical; current partial fixes and remaining decisions are separated below. Business-rule and schema changes require approval.

## Current reconciliation — 2026-09-30

Reference HEAD `2d093b877d047e1096e92dfc3c97d98e949e6671`, branch `nivgoz-professional-v2`. This is a documentation/source/commit review only: no tests, build, DB, SMTP, browser or production verification ran. PASS/BLOCKED statements in dated records belong to their historical packages, not this turn. Earlier “remains OPEN” statements describe the point in time of each record; this summary defines the current scope.

The starting unstaged scheduled-list explanation matches `08d4b93` and `controllers/panel/appointment_list_access.go`; its valid source description is retained below. Its author/origin and historical test-success claim are not established by this reconciliation. The extra blank line and moved cross-cutting section have no behavioral meaning; their substantive content is retained without attributing authorship or completion.

- Definitive list/detail/edit-form GET (`08d4b93`, `9beedb3`, `03cbac4`) and request list/XLSX/latest JSON/detail GET (`e796c57`, `e3ac4bf`, `9e3b103`, `527adfc`) use the current active DB account and real branch before PII: admin all branches; moderator current `can_view`; santral also current `users.sid`; ik/others denied. See `controllers/panel/appointment_*_access.go` and the endpoint records below.
- Definitive create/edit/delete (`85911da`, `d1cc397`, `0fcd607`) and request status POST (`0356ca8`) temporarily require current active DB admin. These restrictions can halt legitimate moderator/santral work; permanent write rights and state-machine/assignment/history policy remain OPEN. Request deletion (`1169426`) is separate: moderator needs real-branch `can_delete`; santral also matching `users.sid`. Source `randevular.rrid ON DELETE SET NULL` still permits loss of conversion provenance; actual DB FK and linked-request deletion policy remain UNKNOWN/OPEN.
- Contact GET (`98180ac`), respond (`c45e851`), set-as-read (`f752342`) and delete (`2d093b8`) are locally restricted to current active DB admin; details and SMTP/notification lifecycle limits are in [SECURITY_BACKLOG](SECURITY_BACKLOG.md). This does not establish appointment-wide or application-wide acceptance.
- Personal notification foundations are committed in `5fe3a6f`, but producers/readers are not cut over and the migration is unapplied. `97d533b` makes `DeleteUser` depend on `notification_receipts`; migration plus real PostgreSQL deletion/FK validation is a **CLOSED deployment gate**. Receipt compatibility does not settle last-admin, retention or notification-event deletion policy.
- Shared GET read-mark writes, SMTP deadline, outbox/retry, notification deletion/asynchronous creation races and last-admin/retention decisions remain OPEN. Real DB/SMTP/browser/production remain UNKNOWN. Earlier fake/Jet test reports are historical and were not rerun here.
- Next single code task: inventory job-application list/detail/respond/delete PII and object authorization. The admin-only CV/media handler (`c1e7b72`, `lib/job_application_media.go`) is not acceptance of the whole application workflow. See the [current handoff](CURRENT_HANDOFF.md).

## Historical end-to-end flow — initial audit

1. A visitor uses the full appointment page, homepage strip or floating form.
2. Client validation requires identity/phone and KVKK confirmation; captcha behavior differs. Center selection is optional.
3. POST /backend/add-randevu-request binds a request model, checks required values and performs an exact-phone duplicate lookup over 24 hours. Captcha verification is conditional on configured keys.
4. The backend inserts name, surname, phone and optional email/date/center/message into randevu_talepleri; initial status is yeni. Doctor/preferred-time fields are not inserted by this path. Consent evidence is not persisted.
5. Optional email runs after insertion. Server notification paths and the floating form's client-generated persistent notification are inconsistent.
6. /panel/randevu-talepleri exposes search/status/sort/pagination, counters, table/grid views and XLSX export. Center joins can omit successfully inserted requests without a center.
7. WebSocket notifications and 30-second polling supplement the queue. Polling requests at most 20 newest records using a fixed page-start timestamp.
8. Staff cycle request status; submitted current state influences the next status. Last modifier/status timestamps do not provide durable transition history.
9. Creating a scheduled appointment inserts a separate randevular record linked by rrid. It does not reliably synchronize the request's state.
10. Deletion is hard deletion; assignment, archival and a complete immutable history were not established.

Request statuses: yeni → randevu-verildi → randevu-verilemedi → hasta-arandi → ulasilamadi → gelmedi → hasta-vazgecti → yeni.

Scheduled statuses: beklemede, onaylandi, iptal, tamamlandi, gelmedi.

These are observations, not an approved future state machine. No SMS implementation was found despite SMS wording in email content.

## Evidence and ordered backlog

Relevant source: fiber-v2/controllers/post/post.go, fiber-v2/controllers/post/randevular/randevular.go, fiber-v2/controllers/panel/panel.go, fiber-v2/models/models.go, schema.sql, and appointment templates/scripts under fiber-v2/static.

| ID / severity / confidence | Current behavior and evidence | Risk | Next action / verification |
| --- | --- | --- | --- |
| F09 / P1 / VERIFIED | Request insertion accepts no center; queue and related queries use inner joins to centers. | Valid requests can disappear from staff views. | B01: approve centerless-request routing, then test insertion through queue/count/export/polling visibility. |
| F10 / P1 / PARTIAL local fix | Detail, request detail, scheduled list, request list, request XLSX export, request latest JSON, and edit-form GET use the approved read rule. Scheduled edit, definitive appointment delete, and AddRandevu POST require a current active DB admin; AddRandevu separates manual creation from locked request conversion. Request delete uses current role and the real request branch: moderator needs can_delete; santral also needs matching users.sid. Request status POST temporarily requires a current active DB admin and checks the locked current state. Other appointment POST handlers retain separate boundaries. | Temporary admin-only rules can block legitimate moderator/santral work; deleting a linked request detaches its definitive appointment under the source FK; other mutation and cross-writer risks remain. | B02: decide remaining write rights and linked-request deletion lifecycle; test other object mutations. |
| F13 / P2 / PARTIAL local fix | `0356ca8` now locks current DB admin/request status and compares submitted state before a conditional UPDATE; the historical unguarded client-state claim is no longer current for this route. The seven-state cycle/timestamp mapping is retained, and scheduled creation stays separate. | Transition approval, immutable history, timestamp semantics and request/appointment synchronization remain open. | B05/B08: decide the state machine and lifecycle; actual DB/cross-writer behavior is UNKNOWN. Historical fake tests were not rerun here. |
| F14 / P2 / VERIFIED | Queue JavaScript rebuilds pagination with defaults; search/status/query settings diverge; requested per_page does not control backend pagination. | Staff lose filters or see unexpected pages. | B03: define one query-state contract; test combined filters, sorting and page changes. |
| F15 / P2 / VERIFIED | Polling uses fixed pageSince, returns only 20 newest records and prepends without reconciling filters/sort/page; table/grid/counts diverge. | Missing bursts, stale counts and misleading filtered queues. | B04: test arrivals, reconnection and both views with active filters. |
| F16 / P2 / VERIFIED | Three public forms differ in captcha, loading/in-flight controls and feedback; backend validation is basic and consent is not persisted. | Duplicate actions and inconsistent submission experience; consent requirements need a product decision. | B06: align approved behavior; test keyboard and repeat submissions across all forms. |
| F17 / P2 / VERIFIED | Duplicate check compares raw phone strings over 24 hours, across statuses/centers, without an atomic guarantee. | Equivalent phones can bypass checks; valid follow-up requests can be rejected; races are possible. | B07: approve normalization/window/exemptions; reproduce concurrency locally before changes. |
| F18 / P2 / VERIFIED | Schema global date/time uniqueness differs from doctor-oriented overlap logic; start-time checks and cascading deletion do not define a consistent lifecycle. | Conflicting scheduling decisions and destructive lifecycle effects. | B08: agree conflict and deletion rules; schema changes remain separately approval-gated. |

### F10 / detail GET temporary product decision (2026-09-29)

**Approved for this endpoint by the user, not for other appointment operations at that time.** The current active DB principal is authoritative: `admin` may view any branch; `moderator` needs current `can_view` on the target appointment's real `sid`; `santral` needs both current `users.sid = randevular.sid` and current `can_view`; `ik` and all other roles are denied. `rid` and branch come from the server DB, not the URL/body or an old JWT role. Denied and missing IDs share a PII-free 404/noindex response; permission/DB read errors return PII-free 503/noindex. Source: `controllers/panel/panel.go:RandevuPage`, `controllers/panel/appointment_detail_access.go`, and synthetic HTTP/real Jet tests in `controllers/panel/appointment_detail_http_test.go`. The authorization and PII detail read use one read-only repeatable-read snapshot plus an exact `rid`/`sid` predicate. Source/local-test confidence high: verified temporary cache, offline targeted and full panel/main package tests passed. Real DB, patient data, browser, and production **UNKNOWN**. The subsequent list decision is recorded below. Edit GET and appointment POST were still OPEN at the time of this detail record; later endpoint fixes and limits are summarized above.

### F10 / scheduled list GET temporary product decision (2026-09-29)

**Source reconciliation of the starting unstaged record (2026-09-30):** `08d4b93` and `controllers/panel/appointment_list_access.go` apply the current active DB role/branch matrix: admin all branches; moderator current `can_view`; santral also current `users.sid`; ik/others denied. Principal/permission reads precede PII, and allowed branches bind options, filtered count and paginated rows in one read-only repeatable-read transaction. User filters cannot broaden that scope; the earlier count ignored filters/branch and is now aligned. No permission returns PII-free 404, an authorized empty result returns 200/count zero, and read/commit failure returns PII-free 503. `appointment_list_http_test.go` contains synthetic SQL, Fiber and Jet fixtures for two branches, four roles, stale JWT, filters, pagination and failures. The previous unstaged text reported offline panel/main test success; that historical claim and the record's author/origin are not independently established in this turn. Real DB, patient data, browser and production remain UNKNOWN. Later edit GET/POST records define their separate boundaries; permission revocation during an existing snapshot is not a live cancellation guarantee.

### F10 / edit-form GET temporary product decision (2026-09-29)

**Approved for `GET /panel/randevular/:rid/duzenle` by the user.** The edit form now shares the detail GET's current active DB principal and target `rid → sid` decision: admin all branches; moderator current `can_view` on the target branch; santral both current `users.sid` match and `can_view`; ik/others denied. `controllers/panel/appointment_detail_access.go` performs authorization before the edit PII read and binds the final row to both `rid` and `sid` in one read-only repeatable-read snapshot. Denied/missing IDs receive the same PII-free 404/noindex; DB/permission errors return PII-free 503/noindex; anonymous behavior is unchanged. Synthetic Fiber HTTP and real Jet tests in `controllers/panel/appointment_edit_http_test.go` cover two branches, four roles, stale JWT role, a `rid` whose stored `sid` differs, denied/missing IDs, read failures, and unchanged form fields. Source/local-test confidence high after offline targeted and full panel/main tests; real DB, patient data, browser, and production **UNKNOWN**. The edit POST remains **OPEN**. Permission revocation during an already started snapshot takes effect on the next request.

### F10 / AddRandevu POST temporary write decision (2026-09-29)

**Approved for `POST /backend/add-randevu` in this task only.** Both blank `rrid` (manual SQL NULL) and positive `rrid` (request conversion) require a current active DB `admin`; moderator, santral, ik and other roles are denied. Neither `can_view` nor request `can_delete` grants creation. `controllers/post/randevular/appointment_create_access.go` locks the current user in a request-owned READ COMMITTED transaction. Conversion additionally locks the real `randevu_talepleri` row, requires a definite real `sid` equal to the body `sid`, and checks for an existing `randevular.rrid` after acquiring the request lock. The branch and active doctor are validated before the in-transaction overlap check and parameterized `INSERT ... RETURNING rid`; zero returned rows, read/insert/commit failures and denials cannot report success. Every pre-commit exit rolls back. Optional mail runs only after successful commit; SMTP failure keeps the existing HTTP 200 / JSON 201 partial-success contract. Post-commit branch/doctor name read failures omit that display detail without panicking.

**Evidence and limits:** `appointment_create_http_test.go` uses synthetic Fiber HTTP and a fake SQL transaction for four current roles, stale JWT role, two branches, SQL NULL/manual creation, real/fake/missing/repeated requests, body-branch mismatch, doctor/slot mismatch, transaction failures, lock ordering and commit-before-mail. `doktorlar.sid` is the structured primary branch used for validation; `doktorlar.calistigi_subeler_text` is free text, so secondary-branch eligibility is **UNKNOWN** and is rejected pending a product/data-model decision. The request-row lock serializes conversions through this handler under READ COMMITTED; other writers can bypass it. The source schema has no unique `randevular(rrid)` rule; no constraint or migration is part of this package. The existing date/time unique rule and doctor trigger do not establish general overlap or single-conversion safety. Request status remains separate from appointment creation. Moderator/santral creation policy remains **OPEN**. Real DB contents/constraints, SMTP and production are **UNKNOWN**.

### F10 / scheduled edit POST temporary write decision (2026-09-29)

**Approved by the user for `POST /backend/randevu/:rid/edit` only.** Current active DB `admin` may edit an existing real appointment in either branch; moderator, santral, ik and other roles are denied. `can_view` is not write permission. `controllers/post/randevular/appointment_edit_access.go` validates equal canonical URL/body `rid` before DB/options access, then locks the current user and real target `sid` in a request-local transaction; destination branch/doctor and doctor overlap checks precede a conditional update bound to the locked `rid` and original `sid`. The client `old_sid` grants no authority. Every pre-commit exit rolls back; mail is attempted only after successful commit and SMTP failure leaves the successful record response. Synthetic Fiber HTTP/fake transaction tests in `controllers/post/randevular/appointment_edit_http_test.go` cover two branches, stale JWT role, denied roles, mismatch/missing target, forged `old_sid`, target `sid` change, DB/transaction failures, rollback and mail order; targeted and full offline randevular/post/lib tests and `go mod verify` passed with a verified temporary cache. Source/local-test confidence high. This temporary rule may stop legitimate moderator/santral corrections; their write permission and branch-transfer policy need an explicit product decision. Concurrent insertion can still race with the doctor overlap count until F18's DB rule is resolved. Other appointment POST handlers remain **OPEN**; real DB, SMTP and production are **UNKNOWN**.

### F10 / definitive appointment delete POST temporary write decision (2026-09-29)

**Approved by the user for `POST /backend/randevu/:rid/delete` only.** Current active DB `admin` may delete a real definitive appointment in either branch; moderator, santral, ik and other roles are denied. The request-delete `can_delete` check belongs to the separate `randevu_talepleri` route and is not treated as definitive appointment permission. `controllers/post/randevular/randevular.go:DeleteRandevu` validates canonical URL `rid`, then uses a request-local SQL transaction and the edit POST's locked current-user/real-target decision; DELETE binds both `rid` and the locked original `sid`. Client branch/body values and old JWT role grant no authority. Pre-commit denial/failures roll back; zero deleted rows and commit failure do not report success. The definitive delete handler has no email, persistent notification or live broadcast; the request-delete notifier is untouched. Synthetic Fiber HTTP/fake transaction tests in `controllers/post/randevular/appointment_delete_http_test.go` cover two branches, four current roles, stale JWT, missing/invalid target, forged sid, role downgrade, DB/transaction errors, zero rows, rollback and no broadcast. Source/local-test confidence high; real DB, FK/trigger effects, and production **UNKNOWN**. This temporary rule may block legitimate moderator/santral deletion; their permanent-delete rights require a separate product decision. Other appointment POST handlers remain **OPEN**.

### F10 / request status POST temporary write decision (2026-09-29)

**Decision:** For registered `POST /backend/randevu-request/:rrid/toggle-status`, only a current active DB admin may change status until a separate `can_edit`/`can_status` product policy exists. The previous handler allowed moderator across branches and santral when its branch-count query passed; staff used the queue and detail page toggle buttons to move a request through the existing seven-state cycle. This temporary restriction can halt legitimate moderator/santral request follow-up and call-center tracking. `can_view`, request `can_delete`, client `sid` and old JWT role do not grant this write. The admin, real request `sid` and stored status are locked in a request-local transaction before any update. The submitted status must equal the stored status; the UPDATE binds `rrid`, old `sid` and old status and preserves the existing next-state/timestamp mapping. Invalid input is 400, stale state is 409, missing/denied target and zero affected rows are PII-free 404, DB/rollback/commit errors are PII-free 503. Only successful commit precedes the existing ID/status live event; no persistent notification or mail was found in this handler.

**Evidence and limits:** `controllers/post/randevular/randevular.go:ToggleRandevuRequestStatus`, `appointment_request_status_access.go`, synthetic Fiber/fake transaction `appointment_request_status_http_test.go`, both panel toggle callers and `schema.sql` status CHECK/updated_at trigger. The F13 client-state risk is addressed for this endpoint only; its durable transition/history risk remains open. The seven-step cycle is observed behavior, **not an approved future workflow**; timestamp columns record latest transition, not history. Current DB constraints and production behavior remain **UNKNOWN**. A permanent moderator/santral write matrix, permitted transitions and audit/history policy remain **OPEN**.

### F10 / appointment request list GET temporary read decision (2026-09-29)

**Approved for `GET /panel/randevu-talepleri` only.** A current active DB admin may see all branches; moderator needs current `can_view` on each visible branch; santral additionally needs the branch to equal current `users.sid`. İK and other roles are denied. `controllers/panel/appointment_request_list_access.go` reads actor, permissions, authorized branch choices, filtered counters and paginated patient rows in one read-only repeatable-read transaction. `sid`/`sube` query parameters, search and status cannot broaden the allowed set. Permission/read failures return PII-free 503; no permission returns PII-free 404 without a Jet list render; an authorized empty result retains HTTP 200 and the Jet empty state. The source panel page has no branch selector, but the branch choices supplied to its render data are scoped. `appointment_request_list_http_test.go` covers synthetic HTTP and real Jet with two branches, four roles, stale JWT, filters, page two and failures. The existing inner join continues to exclude requests with no `sid`; F09's centerless routing remains an **OPEN product decision**. Export, request detail and definitive appointment list are separate paths outside this package. Source/local-test confidence high; real DB, browser and production **UNKNOWN**.

### F10 / appointment request XLSX export GET temporary read decision (2026-09-29)

**Approved for GET /panel/randevu-talepleri/export only.** The registered PanelAuthMiddleware validates a JWT and can pass an old admin claim to this handler; it does not check the current DB role. The handler uses only the JWT UID, then uses the request list's shared current active DB principal/branch decision in the same read-only repeatable-read snapshot as all exported rows. Admin sees all branches; moderator needs can_view; santral needs both current users.sid and can_view; İK/others are denied. Date/status inputs only narrow the real rt.sid scope. No patient XLSX headers or bytes are sent until authorization, row reads, commit and workbook creation succeed. Authorized empty results retain a valid header-only XLSX and the existing filename/nine-column contract. Synthetic Fiber HTTP/fake DB tests reopen actual excelize bytes for two branches, four current roles, stale admin JWT, filters, empty/many rows and failures. Local excelize source writes SetCellValue(string) as a shared string; a formula-like patient value remains literal in the workbook test. The separate latest JSON poll and request detail GET remain **OPEN**. Real DB, patient data and production **UNKNOWN**.

### F10 / appointment request latest JSON GET temporary read decision (2026-09-29)

**Approved for `GET /panel/api/randevu-talepleri/latest` only.** The registered PanelAuthMiddleware validates the JWT, then the handler uses its UID for the shared current active DB role/branch decision: admin all branches; moderator current `can_view` branches; santral only current `users.sid` with `can_view`; İK/others denied. A valid old admin claim can reach the handler but gives no current DB privilege. Actor/permission reads and newest patient rows with real `rt.sid` scope and parameterized `since` share a read-only repeatable-read transaction. The existing descending `created_at` order, 20-row limit, eight JSON patient fields, and authorized empty `status:200,data:[]` contract remain. Denials return PII-free 404 and read/commit errors PII-free 503. Synthetic Fiber HTTP/fake DB tests cover two branches, four current roles, old JWT role, `since`, empty/one/over-20 rows, ordering, foreign PII exclusion and errors. The panel JS reads those same eight fields; its fixed page-start `since` and prepend/pagination divergence remain F15/B04 work, not resolved here. List, XLSX export and request detail are unchanged. Source/local-test confidence high; real DB and production **UNKNOWN**.

### F10 / appointment request detail GET object and branch decision (2026-09-29)

**Approved for `GET /panel/randevu-talepleri/:rrid` only.** After the registered panel JWT middleware, `RandevuTalebiPage` uses the UID, not the old JWT role. `appointment_request_detail_access.go` reads the real `randevu_talepleri.rrid → sid`, current active DB actor and shared `can_view` decision before patient data in one read-only repeatable-read transaction. Admin sees either branch; moderator needs target-branch `can_view`; santral additionally needs current `users.sid = target sid`; İK/others are denied. The patient/detail query binds both locked `rrid` and real `sid`; its linked definitive `randevular` join also requires that sid. Missing and unauthorized IDs share PII-free 404/noindex; read, permission, commit and options failures return PII-free 503/noindex. The actual middleware retains anonymous browser redirect; authorized Jet fields remain. Synthetic Fiber/fake DB and real Jet tests cover two branches, four roles, stale JWT, missing/foreign IDs, permissions, linked definitive appointment, query/commit errors, zero rejected-path PII and zero render. The existing `notification=true` link-based read mark is attempted only after authorization and options succeed; the source notification table has no per-user owner, so notification ownership remains a separate **OPEN** policy question. List, latest JSON, XLSX export and POST routes are unchanged. Source/local-test confidence high; real DB and production **UNKNOWN**.

### F10 / personal notification additive schema contract only (2026-09-29)

This first-phase record predates `97d533b`. Its warning about the old DeleteUser is historical; the receipt-aware handler and its new migration deployment dependency are recorded in the next section. The writer still has no producer cutover.

The local first-phase contract adds default-`legacy` `delivery_model`, nullable legacy `event_kind`/`subject_id`, a constrained personal event, a unique creation-event key and `(nid, recipient_uid)` receipts with explicit `ON DELETE RESTRICT` FKs. That choice preserves read history but will block the existing `DeleteUser` path for recipients with receipts and direct deletion of referenced notifications; retention/deletion policy is **OPEN**. The separate additive migration is **not applied**; the destructive `schema.sql` entry point must never upgrade an existing DB. Static schema and fake transaction tests cannot prove real PostgreSQL constraints or locks. No old row receives an inferred UID, no existing `is_read` value changes, and no producer uses the new owned writer. Request/contact/application producers, panel notification list/count and `notification=true` GET writes all remain **legacy and shared** after this package. Recipient policy, personal reader/mark-read endpoint, legacy compatibility and coordinated cutover remain **OPEN**; real DB and production **UNKNOWN**. Source: `fiber-v2/schema.sql`, `fiber-v2/migrations/20260929_01_personal_notification_contract.sql`, `fiber-v2/database/postgres/personal_notification.go` and focused tests.

### F10 / personal receipt compatibility with user deletion (2026-09-30)

For registered `POST /backend/user/:uid/delete` only, the user's temporary permanent-delete decision removes the target UID's receipts before exactly one target `users` row in one request-local transaction. The real active admin and target are locked in UID order; noncanonical UID, self-delete, current nonadmin/inactive account and absent target cannot remove receipts. Events and other recipients survive, and the authorized HTTP 200 / JSON 201 response follows commit. **Deployment gate:** without the unapplied migration, `notification_receipts` may not exist and deletion fails closed; do not deploy this handler until the migration and real PostgreSQL deletion/FK behavior are verified. No migration ran in this package. Synthetic Fiber/fake transaction tests passed offline, but real DB FK behavior, `user_branch_permissions` schema and production remain **UNKNOWN**. The last-admin and audit/retention policies remain **OPEN**. Appointment producers, shared notification list/count/read flows and notification-event deletion are unchanged; this local compatibility step does not authorize producer cutover.

### F10 / appointment request delete POST object and branch decision (2026-09-29)

**Approved for `POST /backend/randevu-request/:rrid/delete` only.** An active current DB admin may delete across branches, including a request whose `sid` is NULL as the previous handler did. A current moderator needs `can_delete` on the locked request's real `sid`; a current santral additionally needs matching locked `users.sid`. Other roles, inactive accounts, absent requests and unauthorized branches receive the same PII-free 404. User/request/permission reads and the conditional `rrid` plus null-safe real-`sid` DELETE share a request-local READ COMMITTED transaction. Read, delete, affected-row and commit errors do not publish; pre-commit failures roll back. Successful HTTP 200 / JSON 201 and the existing live `randevu_talebi_silindi` event are preserved, with publication only after commit. No client `sid` or old JWT role grants access. Source and synthetic Fiber/fake transaction evidence: `controllers/post/randevular/randevular.go:DeleteRandevuRequest`, `appointment_request_delete_access.go`, `appointment_request_delete_http_test.go`.

**Existing lifecycle rule and risk:** The old handler did not check for an existing `randevular.rrid` link. The source `schema.sql` FK declares `ON DELETE SET NULL`, so deleting a linked request keeps the definitive appointment but removes its request association. This behavior is retained, not presented as safe: it can erase conversion provenance and allow later workflows to treat the appointment as manual. Whether production has the same FK and whether linked requests should instead be archived/rejected are **UNKNOWN / OPEN product and migration decisions**. The request-delete event carries only type and ID; no persistent notification or email was found in this handler. Other appointment mutation routes, real DB and production remain **UNKNOWN / OPEN**.

## Cross-cutting dependencies

- F08 email termination is the first safety proposal; do not mix it with notification policy or request transitions.
- F03 admin detail access and F05 notification trust must be handled with the security backlog.
- F20 query behavior and F21/F27 admin interactions affect daily staff efficiency.
- Targeted regression coverage includes submission, queue visibility, branch access, state changes and request-to-appointment behavior.
- Future assignment, notes/history, archive, export scope and duplicate policies need explicit product decisions; this document does not assert that those features have been approved.

<a id="add-randevu-snapshot"></a>

## Arşiv kaydı: ADD_RANDEVU_WORKFLOW_SNAPSHOT.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### AddRandevu workflow snapshot — wired

Status: `AddRandevu` reads the owned `AppointmentWorkflowSnapshotReader` through
`appointmentworkflowsnapshot.Read`. Production reader caller count: 1. The
handler's legacy `FetchOptionsForBackend` call count: 0. The existing
`optionsRepository` is injected from `main`; no additional pool or repository
is created.

#### Source evidence and decision

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

#### Verification boundary

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

<a id="edit-randevu-snapshot"></a>

## Arşiv kaydı: EDIT_RANDEVU_WORKFLOW_SNAPSHOT.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### EditRandevu workflow snapshot — wired

Status: `EditRandevu` reads the existing `data.AppointmentWorkflowSnapshot`
through `appointmentworkflowsnapshot.Read` and
`utilities.AppointmentWorkflowSnapshotReader`. Its legacy
`FetchOptionsForBackend` call count is zero; its snapshot-helper call count is
one. `AddRandevu` retains its own call to the same helper and reader. `main`
already injects the single `optionsRepository` into the utility field, so no
new pool, repository, utility field, schema, or dependency was added.

#### Source evidence and behavior

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

#### Verification boundary

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
