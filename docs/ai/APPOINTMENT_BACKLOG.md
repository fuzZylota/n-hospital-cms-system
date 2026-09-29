# Nivgöz — appointment and request backlog

Source: accepted Phase 1 audit. All findings OPEN. Business-rule and schema changes require approval.

## Existing end-to-end flow

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
| F10 / P1 / PARTIAL local fix | Detail, scheduled list, and edit-form GET use the approved read rule. Scheduled edit POST is limited to a current active DB admin with a locked real target. Other appointment POST handlers retain separate boundaries. | The temporary admin-only write rule can block legitimate moderator/santral editing; other mutation risks remain. | B02: decide non-admin edit rights and remaining action matrix; test each other object mutation. |
| F13 / P2 / VERIFIED | Status cycling trusts client state, lacks compare-and-set and overwrites status timestamps; scheduled creation is separate. | Stale/double actions, misleading lifecycle and missing history. | B05/B08: approve transitions and synchronization; test stale submissions and repeated actions. |
| F14 / P2 / VERIFIED | Queue JavaScript rebuilds pagination with defaults; search/status/query settings diverge; requested per_page does not control backend pagination. | Staff lose filters or see unexpected pages. | B03: define one query-state contract; test combined filters, sorting and page changes. |
| F15 / P2 / VERIFIED | Polling uses fixed pageSince, returns only 20 newest records and prepends without reconciling filters/sort/page; table/grid/counts diverge. | Missing bursts, stale counts and misleading filtered queues. | B04: test arrivals, reconnection and both views with active filters. |
| F16 / P2 / VERIFIED | Three public forms differ in captcha, loading/in-flight controls and feedback; backend validation is basic and consent is not persisted. | Duplicate actions and inconsistent submission experience; consent requirements need a product decision. | B06: align approved behavior; test keyboard and repeat submissions across all forms. |
| F17 / P2 / VERIFIED | Duplicate check compares raw phone strings over 24 hours, across statuses/centers, without an atomic guarantee. | Equivalent phones can bypass checks; valid follow-up requests can be rejected; races are possible. | B07: approve normalization/window/exemptions; reproduce concurrency locally before changes. |
| F18 / P2 / VERIFIED | Schema global date/time uniqueness differs from doctor-oriented overlap logic; start-time checks and cascading deletion do not define a consistent lifecycle. | Conflicting scheduling decisions and destructive lifecycle effects. | B08: agree conflict and deletion rules; schema changes remain separately approval-gated. |

### F10 / detail GET temporary product decision (2026-09-29)

**Approved for this endpoint by the user, not for other appointment operations.** The current active DB principal is authoritative: `admin` may view any branch; `moderator` needs current `can_view` on the target appointment's real `sid`; `santral` needs both current `users.sid = randevular.sid` and current `can_view`; `ik` and all other roles are denied. `rid` and branch come from the server DB, not the URL/body or an old JWT role. Denied and missing IDs share a PII-free 404/noindex response; permission/DB read errors return PII-free 503/noindex. Source: `controllers/panel/panel.go:RandevuPage`, `controllers/panel/appointment_detail_access.go`, and synthetic HTTP/real Jet tests in `controllers/panel/appointment_detail_http_test.go`. The authorization and PII detail read use one read-only repeatable-read snapshot plus an exact `rid`/`sid` predicate. Source/local-test confidence high: verified temporary cache, offline targeted and full panel/main package tests passed. Real DB, patient data, browser, and production **UNKNOWN**. List, edit GET, and appointment POST authorization remain **OPEN**.

### F10 / edit-form GET temporary product decision (2026-09-29)

**Approved for `GET /panel/randevular/:rid/duzenle` by the user.** The edit form now shares the detail GET's current active DB principal and target `rid → sid` decision: admin all branches; moderator current `can_view` on the target branch; santral both current `users.sid` match and `can_view`; ik/others denied. `controllers/panel/appointment_detail_access.go` performs authorization before the edit PII read and binds the final row to both `rid` and `sid` in one read-only repeatable-read snapshot. Denied/missing IDs receive the same PII-free 404/noindex; DB/permission errors return PII-free 503/noindex; anonymous behavior is unchanged. Synthetic Fiber HTTP and real Jet tests in `controllers/panel/appointment_edit_http_test.go` cover two branches, four roles, stale JWT role, a `rid` whose stored `sid` differs, denied/missing IDs, read failures, and unchanged form fields. Source/local-test confidence high after offline targeted and full panel/main tests; real DB, patient data, browser, and production **UNKNOWN**. The edit POST remains **OPEN**. Permission revocation during an already started snapshot takes effect on the next request.

### F10 / scheduled edit POST temporary write decision (2026-09-29)

**Approved by the user for `POST /backend/randevu/:rid/edit` only.** Current active DB `admin` may edit an existing real appointment in either branch; moderator, santral, ik and other roles are denied. `can_view` is not write permission. `controllers/post/randevular/appointment_edit_access.go` validates equal canonical URL/body `rid` before DB/options access, then locks the current user and real target `sid` in a request-local transaction; destination branch/doctor and doctor overlap checks precede a conditional update bound to the locked `rid` and original `sid`. The client `old_sid` grants no authority. Every pre-commit exit rolls back; mail is attempted only after successful commit and SMTP failure leaves the successful record response. Synthetic Fiber HTTP/fake transaction tests in `controllers/post/randevular/appointment_edit_http_test.go` cover two branches, stale JWT role, denied roles, mismatch/missing target, forged `old_sid`, target `sid` change, DB/transaction failures, rollback and mail order; targeted and full offline randevular/post/lib tests and `go mod verify` passed with a verified temporary cache. Source/local-test confidence high. This temporary rule may stop legitimate moderator/santral corrections; their write permission and branch-transfer policy need an explicit product decision. Concurrent insertion can still race with the doctor overlap count until F18's DB rule is resolved. Other appointment POST handlers remain **OPEN**; real DB, SMTP and production are **UNKNOWN**.

## Cross-cutting dependencies

- F08 email termination is the first safety proposal; do not mix it with notification policy or request transitions.
- F03 admin detail access and F05 notification trust must be handled with the security backlog.
- F20 query behavior and F21/F27 admin interactions affect daily staff efficiency.
- Targeted regression coverage includes submission, queue visibility, branch access, state changes and request-to-appointment behavior.
- Future assignment, notes/history, archive, export scope and duplicate policies need explicit product decisions; this document does not assert that those features have been approved.
