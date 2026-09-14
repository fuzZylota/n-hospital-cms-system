# Nivgöz — ordered roadmap

Source: accepted Phase 1 audit, baseline 22958244024a81ca69edb09f77a22aa60291ff6c.
All implementation items are OPEN / NOT APPROVED. Ordering does not authorize execution.

## Delivery rules

Each task needs a bounded plan, appropriate local tests, reviewed diff, commit and documentation update. Tests accompany safety fixes; Phase C extends coverage rather than postponing it. Split any row further if implementation would address unrelated problems.

Before executing code that opens a database or invokes a migration, establish an isolated local test configuration. Never use schema.sql as an upgrade. No production access is approved.

## Phase A — safety foundation

The F01 destructive-schema prohibition is an immediate operating gate. A01 is the first proposed code change because its high-confidence failure is narrow and does not require database or business-rule decisions.

| Order / task | Source | Concrete scope and completion evidence |
| --- | --- | --- |
| A01 — Return email failures without exiting | F08 | Replace fatal exits in SendEmail with returned errors; local MIME and SMTP failure regressions plus successful delivery to a local test server. Awaiting approval. |
| A02 — Guard destructive database entry points | F01 | Plan explicit fresh-install versus upgrade safeguards for schema-loading scripts; validate guards without executing destructive SQL. Any destructive validation needs approval. |
| A03 — Protect permission writes | F02 | Constrain branch-permission changes to the approved permission-management boundary; test self-edit and authorized management. Confirm unclear role semantics before coding. |
| A04 — Protect settings operations | F03 | Enforce the approved settings read/write boundary; test each role and secret-field responses. |
| A05 — Enforce filesystem containment on delete | F06 | Restrict custom-media deletion to its authorized root; test traversal and legitimate deletion with temporary fixtures. |
| A06 — Authenticate WebSocket operations | F05 | Establish server-verified identity and event authorization; test anonymous and unauthorized events locally. |
| A07 — Render notification content safely | F05 | Remove unsafe notification HTML interpretation; test hostile text renders as text. |
| A08 — Apply upload validation | F07 | Define and enforce approved type/size checks for one upload entry point at a time; use local fixtures. |
| A09 — Protect recruitment attachments | F07 | Agree the access policy, then verify authorized retrieval and denial; separate from generic upload handling. |
| A10 — Enforce token lifetime | F04 | Define session lifetime, enforce expiration and test invalid/expired tokens. |
| A11 — Enforce account revocation | F04 | Deny banned/invalid accounts and stale access according to approved rules; test failure paths. |
| A12 — Harden authentication transport and abuse controls | F12 | Separate tasks for cookie policy, CSRF, login throttling and error responses; verify compatibility locally. |
| A13 — Complete other admin endpoint protection | F03 | Review the existing endpoint/role matrix and fix one missing boundary per task, including header/gallery operations. |
| A14 — Reconcile schema evidence safely | F11 | Compare source expectations with a non-production schema supplied/approved for review; design incremental migration proposals only. Schema changes require approval. |

Exit gate: critical boundaries and crash/file/notification risks have targeted regression protection; unresolved access-policy and schema questions are explicit.

## Phase B — appointment integrity

| Order / task | Source | Concrete scope and completion evidence |
| --- | --- | --- |
| B01 — Keep accepted requests visible | F09 | Agree routing/visibility of requests without centers; verify insertion, staff queue, counts, polling and export against that rule. |
| B02 — Reconcile branch authorization | F10 | Agree users.sid versus branch-permission authority, then enforce object-level rules for request and scheduled-appointment operations. |
| B03 — Preserve queue query state | F14 | Carry search/status/sort/page settings consistently; test filter and pagination combinations. |
| B04 — Reconcile live queue updates | F15 | Preserve active filters, ordering, view mode and counts; test more than 20 arrivals and reconnect behavior. |
| B05 — Define safe status transitions | F13 | Obtain approved transition rules; plan server-owned transitions, stale-update handling and required history. Schema/history changes need approval. |
| B06 — Make public submission behavior consistent | F16 | Align validation, in-flight handling, feedback and captcha behavior across the three forms; obtain decisions for consent/business-rule changes. |
| B07 — Review duplicate-request policy | F17 | Agree normalization, time window and exemptions; reproduce concurrency behavior locally before proposing a fix. |
| B08 — Reconcile request-to-appointment lifecycle | F18, F13 | Agree scheduling conflicts, request-state synchronization and deletion/archive behavior; separate schema proposals from code changes. |

Exit gate: accepted requests remain visible to the correct staff, queue results are reliable, and lifecycle rules are documented and locally tested. Approval is required before changing appointment rules.

## Phase C — targeted regression protection

C01–C08 are small test tasks around existing approved behavior: authentication, authorization, branch permissions, appointment submission, queue visibility, status changes, uploads/files and critical admin endpoints. Source: F25 and the corresponding Phase A/B findings.

Use deterministic local fixtures. Do not create a broad test framework or introduce dependencies without a demonstrated need and approval. A usable Go toolchain and isolated test configuration are prerequisites; neither was validated during Phase 1.

## Phase D — incremental backend architecture

- D01 (F19): investigate shared cache/ORM concurrency with focused local evidence; do not assume a race has been proven.
- D02 (F20): profile one expensive query path at a time, starting with staff queues/exports and recruitment pagination.
- D03: extract authorization boundaries already specified and tested in A/B; avoid a new framework.
- D04: reduce duplicated validation and query mapping only around touched, regression-protected behavior.
- D05: separate the next oversized controller responsibility when its safety or maintenance benefit can be demonstrated.

The architecture work derives from the audited large controllers and duplicated logic. It does not authorize an application rewrite, generalized service-layer migration or speculative optimization.

## Phase E — Admin/CMS product improvement

- E01 (F21, F27): establish a reviewed admin interaction/design inventory and accessible component contracts.
- E02: improve appointment queue tables, filters, search, pagination, status feedback and confirmation interactions.
- E03: improve appointment forms and detail hierarchy without changing business rules.
- E04: apply the approved system incrementally to navigation, dashboard and content CRUD.
- E05: verify keyboard access, responsive layouts, modals, notifications and empty/loading/error/success states.
- E06 (F26): confirm ownership/use before removing or replacing dormant modules and placeholders.

These are future product workstreams grounded in the audit; each requires a concrete task before implementation.

## Phase F — public technical quality

- F-T01 (F24): establish a repeatable local/browser performance baseline, protecting existing CLS/LCP work.
- F-T02: address measured image, CSS, JavaScript and font costs individually.
- F-T03 (F22): verify canonical routes, fallback status, metadata, sitemap/robots responsibilities and structured data; do not rewrite SEO content.
- F-T04 (F21, F16): verify accessibility and submission feedback on critical public flows.
- F-T05 (F23): verify consent revocation behavior and agree the technical tracking contract.
- F-T06: assess localized content/routing requirements before replacing translation behavior.

Production PageSpeed and server behavior remain Phase H work requiring approval.

## Phase G — public visual design

After technical foundations are stable, define a reviewed public design system and improve typography, spacing, controls, cards, header/navigation, homepage, doctor/department/center pages and appointment experience. Add motion only with accessibility and performance checks. Preserve branding and existing performance work. No redesign is approved by this roadmap.

## Phase H — production/server verification

Explicit approval required before any access. Verify aaPanel, Nginx, PostgreSQL version/schema, deployed commit, process manager, environment, compression, caching, TLS, security headers, backup/restore, monitoring and production PageSpeed. Source: F01/F11/F12/F22/F24/F28 and the audit's production unknowns.

Read-only verification does not authorize deployment, server changes or restore drills.

## Complete audit finding register

| ID | Severity / confidence | Finding | Owning documentation / phase |
| --- | --- | --- | --- |
| F01 | P0 / VERIFIED, conditional execution risk | Destructive schema used by migration scripts | SECURITY / A02; immediate prohibition |
| F02 | P1 / VERIFIED | Self-edit permission writes | SECURITY / A03 |
| F03 | P1 / VERIFIED | Missing admin endpoint boundaries | SECURITY / A04, A13; appointment cross-reference |
| F04 | P1 / VERIFIED | Token expiry/revocation gaps | SECURITY / A10–A11 |
| F05 | P1 / VERIFIED | WebSocket trust and unsafe rendering | SECURITY / A06–A07 |
| F06 | P1 / VERIFIED | File deletion lacks containment | SECURITY / A05 |
| F07 | P1 / VERIFIED | Upload and attachment access weaknesses | SECURITY / A08–A09 |
| F08 | P1 / VERIFIED | Email failure terminates application | SECURITY / A01 |
| F09 | P1 / VERIFIED | Centerless requests omitted from queue joins | APPOINTMENT / B01 |
| F10 | P1 / VERIFIED | Inconsistent appointment branch authorization | APPOINTMENT / B02 |
| F11 | P1 / VERIFIED | Schema/source reconstruction mismatch | SECURITY / A14 |
| F12 | P1 / VERIFIED source gaps; exploitation LIKELY/unverified | Missing authentication/request safety controls | SECURITY / A12 |
| F13 | P2 / VERIFIED | Client-driven status cycle and missing durable history | APPOINTMENT / B05, B08 |
| F14 | P2 / VERIFIED | Filter/pagination state divergence | APPOINTMENT / B03 |
| F15 | P2 / VERIFIED | Live queue update inconsistency | APPOINTMENT / B04 |
| F16 | P2 / VERIFIED | Divergent public request forms/validation | APPOINTMENT / B06; PUBLIC |
| F17 | P2 / VERIFIED | Non-normalized, non-atomic duplicate check | APPOINTMENT / B07 |
| F18 | P2 / VERIFIED | Scheduling constraints/lifecycle inconsistencies | APPOINTMENT / B08 |
| F19 | P2 / LIKELY | Shared cache/ORM concurrency concerns | ADMIN / D01 |
| F20 | P2 / VERIFIED patterns; cost UNKNOWN | Unbounded/repeated queries and pagination patterns | ADMIN / D02 |
| F21 | P2 / VERIFIED markup; runtime impact unmeasured | Accessibility/interaction gaps | ADMIN, PUBLIC / E05, F-T04 |
| F22 | P2 / VERIFIED source omissions; production UNKNOWN | Technical SEO/routing gaps | PUBLIC / F-T03 |
| F23 | P2 / VERIFIED source behavior | Consent revocation does not disable loaded tracker | PUBLIC / F-T05 |
| F24 | P2 / VERIFIED patterns; performance unmeasured | Asset loading/coupling concerns | PUBLIC / F-T01–F-T02 |
| F25 | P2 / VERIFIED | Missing critical regression tests | ROADMAP / C |
| F26 | P3 / VERIFIED | Dormant code/placeholders/naming | ADMIN / E06 |
| F27 | P3 / VERIFIED | Design tokens/labels/status consistency | ADMIN, PUBLIC / E01, G |
| F28 | P3 / VERIFIED repository limitations | Operations/logging evidence gaps | ROADMAP / H |

## First implementation proposal — A01

- TASK: Return MIME/SMTP errors from SendEmail without terminating the process.
- WHY FIRST: F08 is P1 / VERIFIED, with a small change surface and low implementation risk. It can be tested locally without changing schemas, appointment rules or permission rules. F01 is already subject to an immediate no-execution gate.
- FILES: fiber-v2/lib/lib.go; proposed new fiber-v2/lib/email_test.go. Read existing callers and module configuration for test compatibility; caller changes are not assumed.
- PLANNED CHANGE: Replace the two log.Fatalf paths in SendEmail with ordinary error returns, preserving its signature and successful-send behavior. Add narrowly scoped regression tests. Do not add retries, a queue or notification-policy changes.
- RISKS: Callers may ignore returned errors or expose existing partial-success behavior; check and document that behavior. Go toolchain/test-environment availability must be resolved before verification. If isolated tests require broader refactoring or dependency changes, stop and revise the plan.
- TEST PLAN: With an approved Go toolchain, test MIME assembly failure using a missing temporary attachment; SMTP failure against a local controlled server; successful delivery to a local fake SMTP server; and a subprocess regression that fails if SendEmail exits. No production SMTP or database connection. Review callers and run relevant package checks.
- ROLLBACK: Revert the isolated implementation commit if necessary; no database or configuration rollback. Reverting restores the fatal-exit risk and must be recorded.
- STATUS: PROPOSED — awaiting explicit approval. No implementation or tests have been performed for A01.
