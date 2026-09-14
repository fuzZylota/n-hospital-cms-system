# Nivgöz — project state

Updated: 2026-09-14.

## Accepted baseline

The accepted Phase 1 Discovery & Architecture Audit is the technical baseline. This documentation is a working index of that audit, not a new audit.

- Repository: C:/Users/DELL/nivgoz/n-hospital-cms-system
- Branch at baseline: astra-full-audit
- Baseline commit: 22958244024a81ca69edb09f77a22aa60291ff6c
- Public website and Admin/CMS are equally important products.
- Current stage: planning documentation. No implementation task is approved.
- Findings remain open; documentation does not constitute remediation.
- Production synchronization, deployed commit and aaPanel configuration are UNKNOWN.

The audit inspected local source, configuration, templates, assets and history. It did not access production, run migrations, connect to the database or validate deployed behavior. Source-level UX findings require browser verification. Go was unavailable on PATH during the audit; compilation and Go tests were not run. Existing JavaScript syntax checks passed, but do not establish functional correctness.

## Evidence conventions

- VERIFIED: direct repository evidence; does not imply deployed exploitation or a reproduced production incident.
- LIKELY: strong evidence requiring runtime verification.
- HYPOTHESIS: investigation needed; not an established defect.
- P0: catastrophic exposure requiring an immediate safety gate.
- P1: high severity; P2: material correctness/quality issue; P3: lower-priority consistency/maintenance issue.

Finding IDs F01–F28 retain the accepted audit identity. See [ROADMAP](ROADMAP.md) for the complete register and ordered work.

## Verified architecture

| Area | Baseline |
| --- | --- |
| Backend | Go 1.25.1 workspace; Fiber v2.52.9 in the main module |
| Templates | Server-rendered Jet through gofiber/template/jet/v2; not Go html/template |
| Database | PostgreSQL with neormgo and raw SQL; deployed PostgreSQL version UNKNOWN |
| Frontend | Vanilla JavaScript, jQuery, Bootstrap, Mediox assets, Owl/Slick |
| Authentication | JWT cookie, bcrypt; admin/moderator/santral/ik roles and branch permissions |
| Notifications | SMTP email and WebSocket messaging; no implemented SMS integration identified |
| Localization | Turkish source pages and Google Translate; no complete localized route/content architecture |
| Build | Multi-module Go workspace; no verified CI pipeline |
| Tests | Small encryption/decryption test file; critical workflow regression coverage absent |

Key source boundaries:

- [Startup](../../fiber-v2/main/main.go), [routing](../../fiber-v2/baserouter/baserouter.go).
- [Public controllers](../../fiber-v2/controllers/frontend/frontend.go).
- [Admin controllers](../../fiber-v2/controllers/panel/panel.go).
- [Backend handlers](../../fiber-v2/controllers/post/post.go) and domain handler directories.
- [Database helpers](../../fiber-v2/database/database.go), [models](../../fiber-v2/models/models.go), [shared library](../../fiber-v2/lib/lib.go).
- Templates and components under fiber-v2/static/html; separate public and panel CSS/JavaScript under fiber-v2/static.
- Controllers contain substantial business/query logic. A distinct service/repository layer was not established.
- Domain groups: centers, doctors, specialties/departments, examinations, content/media, users/permissions, appointment requests, scheduled appointments, contacts, recruitment and notifications.

Appointment flow: public form → backend validation and phone duplicate check → request insert → optional email and notification paths → staff queue → status updates → optional separate scheduled appointment. Request and scheduled-appointment state are not reliably synchronized. See [APPOINTMENT_BACKLOG](APPOINTMENT_BACKLOG.md).

## Protected previous work

Preserve the existing public performance and accessibility work unless a regression is demonstrated.

| Commit | Accepted assessment |
| --- | --- |
| 2295824 | Carousel geometry alignment: GOOD BUT REQUIRES VERIFICATION |
| c156735 | Responsive first-banner preload: GOOD BUT REQUIRES VERIFICATION |
| 3e802db | Deferred remaining banner backgrounds: PARTIAL; verify multiple slides |
| 8b5def3 | Below-fold image lazy loading: GOOD / KEEP |
| e70bb8e | Delayed root-gap JavaScript replaced by CSS: GOOD / KEEP; verify offsets |
| abbeb43 | Inline layout guards: GOOD / KEEP |
| c19b6f7 | Doctor accessibility labels: GOOD / KEEP |
| 24c34e7 | Flag dimensions and asynchronous translation loading: GOOD / KEEP |
| 3f225e7 | Arabic option removal: QUESTIONABLE product decision; do not reverse without agreement |
| 8f8246b | Broad earlier snapshot: PARTIAL; preserve useful changes and address findings individually |

Keep semantic landmarks, headings, labels, focus visibility, reduced-motion support, image dimensions, route-specific assets and font reductions. Historical PageSpeed/CLS claims in CLAUDE.md are not fresh measurements.

## Working contract

1. Read CLAUDE.md and every docs/ai/*.md before each major task.
2. Do not repeat a broad audit unless explicitly requested. New evidence warrants focused investigation.
3. Use small, independently reviewable tasks, normally one problem each.
4. Follow ANALYZE → PLAN → IMPLEMENT → TEST → REVIEW DIFF → COMMIT → UPDATE DOCUMENTATION.
5. Avoid unrelated cleanup, application rewrites and unapproved dependency changes.
6. Preserve public performance work unless a regression is proven.
7. Do not change production directly or run production migrations.
8. Never run the destructive schema.sql as an upgrade migration.
9. Obtain approval before production access, aaPanel changes, schema changes, destructive operations, deployment, appointment business-rule changes, or unclear role/permission business-rule changes.
10. Implement locally verifiable security fixes before visual redesign.
11. Treat Admin/CMS backend, authorization, workflows, queries, frontend, UX, accessibility, responsiveness and design consistency as product work.
12. Public work later includes performance, Core Web Vitals, SEO, accessibility, frontend quality, design, responsive UX and conversion flows.
13. Commit messages follow the existing Turkish convention. Document tests actually run and limitations.
14. Update the owning backlog, roadmap and changelog after each task. Record architectural decisions separately.

The user's controlled engineering instructions supersede CLAUDE.md's historical frontend-only remit for specifically approved backend tasks. They do not authorize blanket backend work. Other applicable project conventions remain in force.

## Next action

Review the A01 email failure proposal in [ROADMAP](ROADMAP.md). Implementation is awaiting explicit approval.
