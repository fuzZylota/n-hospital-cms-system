# Nivgöz — architecture decisions

This register records accepted workflow decisions separately from unapproved technical designs. Source: accepted audit and the user's controlled engineering instructions, 2026-09-14.

## ADR-001 — Use the accepted audit as the baseline

Status: ACCEPTED.

Use baseline commit 22958244024a81ca69edb09f77a22aa60291ff6c and findings F01–F28. Do not repeat the full repository audit without an explicit request. Investigate new evidence narrowly and record any resulting finding with its evidence/confidence; do not silently relabel existing findings.

## ADR-002 — Deliver small, tested changes

Status: ACCEPTED.

Follow ANALYZE → PLAN → IMPLEMENT → TEST → REVIEW DIFF → COMMIT → UPDATE DOCUMENTATION. Normally solve one problem per task. Avoid unrelated cleanup, dependency churn and framework rewrites. Local tests accompany fixes; Phase C adds focused coverage.

Read CLAUDE.md and docs/ai/*.md before major work. The user's latest approved task scope takes precedence over CLAUDE.md's historical frontend-only restriction. Existing branding, Jet, asset separation and accessibility conventions still apply where relevant.

## ADR-003 — Preserve both product surfaces

Status: ACCEPTED.

Public website and Admin/CMS are first-class products. Address locally testable security/correctness risks before visual redesign. Admin improvement includes backend, permissions, database behavior and workflows as well as UI. Preserve the existing public performance work unless a regression is demonstrated.

## ADR-004 — Separate local engineering from production actions

Status: ACCEPTED.

No direct production changes. Explicit approval is required for production access, aaPanel, schema changes, destructive operations and deployment. Production verification does not imply permission to modify production.

Never use the existing destructive schema.sql as an upgrade migration. Future migration design must be reviewed independently of execution. Existing production versions/schema/commit remain UNKNOWN until verified with approval.

## ADR-005 — Business rules require explicit decisions

Status: ACCEPTED.

Appointment rule changes and unclear role/permission changes require approval. Do not derive policy from current bugs, hidden navigation or inconsistent checks.

Unresolved: branch authority, centerless-request ownership, status transitions, duplicate policy, scheduling conflict rules, deletion/archive semantics, assignment/history requirements and session lifetime. These are decision topics, not approved new features.

## ADR-006 — First implementation candidate

Status: PROPOSED — NOT APPROVED.

A01 would replace fatal exits in SendEmail with returned errors while retaining its signature and successful behavior. It would not introduce a mail queue, retries, appointment-rule changes or a notification redesign.

Reason: F08 is P1 / VERIFIED with a narrow local change and a deterministic local verification strategy.

Approval and actual test results must be recorded before this decision can be marked implemented.

## Future decision records

Create a new record only for a material choice reached during a scoped task. Include context, decision, alternatives when relevant, consequences, approval status and validation. Do not preselect a new framework, ORM, RBAC model, event system or design system.
