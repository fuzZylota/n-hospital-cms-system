# Nivgöz — security backlog

Source: accepted Phase 1 audit. All items OPEN; proposed actions are not approvals.
VERIFIED means repository evidence. Production exposure and exploitability were not tested.

## Safety gate

**Never execute the existing schema.sql as an upgrade migration.** Do not run either migration script during ordinary verification. Production access, schema changes, destructive operations and deployment require approval.

## Findings and future work

| ID / severity / confidence | Evidence and current behavior | Why it matters / risk | Future action and local verification |
| --- | --- | --- | --- |
| F01 / P0 / VERIFIED | schema.sql contains DROP SCHEMA public CASCADE; migrate.sh and production-migrate.sh load the schema. | Catastrophic loss if used against an existing database; no execution or loss was observed. | A02: distinguish fresh-install and upgrade entry points; test refusal paths without destructive SQL. |
| F02 / P1 / VERIFIED | fiber-v2/controllers/post/users/users.go: self-edit handling reaches branch-permission writes without an admin-only boundary; omitted fields may reset permissions. | Privilege changes through a self-service path. | A03: approve permission-management semantics and test self-edit versus authorized management. |
| F03 / P1 / VERIFIED | fiber-v2/controllers/panel/panel.go: settings and appointment-detail boundaries differ from navigation; settings templates contain secret fields; header/gallery operations have authentication-only protection. | Navigation does not protect sensitive data or operations; possible cross-branch disclosure. | A04/A13: enforce approved endpoint authorization one module at a time; test reads and writes for each role. |
| F04 / P1 / VERIFIED | fiber-v2/lib/lib.go: token construction lacks expiry claims; revocation/account lookup paths do not consistently stop processing. | Stale access can persist; invalid-account handling may fail open. | A10/A11: approve lifetime/revocation rules; test expiration, banning, deleted users and lookup failure. |
| F05 / P1 / VERIFIED | fiber-v2/controllers/post/post.go: client WebSocket events can persist/broadcast notifications; panel notifications.js interprets notification content with innerHTML. | Forged notifications, unsafe rendering and recipient metadata exposure; runtime delivery/exploitation not tested. | A06/A07: server-verified event authorization and safe text rendering, separately tested. |
| F06 / P1 / VERIFIED | DeleteCustomMedia in fiber-v2/controllers/post/post.go and DeleteFile in fiber-v2/lib/lib.go join supplied paths without proven containment before removal. | An authenticated operation can target files outside its intended directory. | A05: enforce authorized-root containment; use temporary fixtures for traversal and legitimate paths. |
| F07 / P1 / VERIFIED | Generic upload in post.go and gallery handling in panel.go lack consistent type/size/role checks; main/main.go serves the static tree containing recruitment attachments. | Unsafe content/storage use and potentially public sensitive attachments; deployed exposure unknown. | A08/A09: separate upload validation from private-attachment access policy; verify locally. |
| F08 / P1 / VERIFIED | fiber-v2/lib/lib.go:894 SendEmail calls log.Fatalf on MIME assembly and smtp.SendMail errors. | An ordinary email failure can terminate the application. | A01: return errors; verify MIME failure, SMTP failure and local success without process exit. First proposed task. |
| F11 / P1 / VERIFIED | Repository schema omits structures/columns expected by models and handlers, including user_branch_permissions, sube_galerileri and status timestamps. Seed password representation differs from bcrypt-only login. | Fresh reconstruction or an assumed upgrade may fail; deployed schema state unknown. | A14: reconcile against an approved non-production schema; design incremental migration proposals, never execute the destructive schema as upgrade. |
| F12 / P1 / VERIFIED source gaps | Authentication/startup code lacks established CSRF and login throttling controls; cookie Secure/SameSite are not explicit; differentiated login errors and very large body limit appear in source. | Authentication abuse and request risks require local verification; reverse-proxy protections are unknown. | A12: separate cookie, CSRF, rate-limit and response tasks; verify normal login/forms and rejection paths. |

Appointment authorization finding F10 is owned by [APPOINTMENT_BACKLOG](APPOINTMENT_BACKLOG.md). Shared cache/ORM concern F19 is LIKELY, not a verified race; see [ADMIN_BACKLOG](ADMIN_BACKLOG.md).

## Boundaries for implementation

- Do not infer intended role permissions solely from hidden menu items.
- Do not change unclear role/permission rules without approval.
- Do not print credentials, copy production secrets into fixtures or send real mail during tests.
- Document existing caller behavior before expanding a security fix into workflow changes.
- Keep each fix independently reviewable; production effectiveness remains unverified until approved Phase H work.
