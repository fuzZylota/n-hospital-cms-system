# Nivgöz — Admin/CMS backlog

Source: accepted Phase 1 audit. All implementation remains unapproved.

## Product baseline

Admin/CMS is a first-class product encompassing backend architecture, authorization, appointment workflows, database/query behavior, frontend architecture, UX, accessibility, responsive behavior and design consistency.

The admin uses server-rendered Jet layouts/components, panel CSS and JavaScript, Bootstrap and shared libraries. Large handlers combine data loading, authorization, business rules and rendering. Navigation visibility and endpoint authorization are inconsistent.

Audited modules include dashboard, appointment requests and scheduled appointments, doctors, departments/examinations, centers, content/news, media, users, settings, contacts and recruitment. Testimonials have dormant code/templates; some editor views are placeholders. There is no complete generalized permission model: role checks, legacy center membership and branch permission flags coexist.

## Correctness before redesign

Complete relevant security and appointment boundaries first:

- F02/F03/F04/F05/F06/F07/F12: [SECURITY_BACKLOG](SECURITY_BACKLOG.md).
- F09/F10/F13–F18: [APPOINTMENT_BACKLOG](APPOINTMENT_BACKLOG.md).
- F25: targeted tests in [ROADMAP](ROADMAP.md), Phase C.

## Architecture and product findings

| ID / severity / confidence | Evidence/current behavior | Risk | Future action |
| --- | --- | --- | --- |
| F19 / P2 / LIKELY | Shared model/database caches and ORM use in fiber-v2/models/models.go and fiber-v2/database/database.go lack demonstrated synchronization guarantees. | Concurrent access may produce stale/inconsistent behavior; dependency thread safety is unverified. | D01: focused dependency/source inspection and local race/concurrency checks before a fix. |
| F20 / P2 / VERIFIED patterns | Recruitment listing in panel.go loads results before slicing; exports are unbounded; dashboards/notifications repeat queries; some totals use page length. | Scaling and count correctness concerns; actual runtime cost is UNKNOWN. | D02: measure one path with isolated fixtures; test pagination/count correctness before optimization. |
| F21 / P2 / VERIFIED markup | Panel layout language is en; mouse-oriented sorting, unnamed icon actions, small text and uncertain modal focus behavior occur in templates/scripts. | Staff keyboard/screen-reader use and mobile operation may be difficult; visual/runtime impact needs testing. | E05: local browser audit of one workflow, then accessible interactions with keyboard/responsive verification. |
| F26 / P3 / VERIFIED | Dormant testimonials, editor placeholders and inconsistent naming remain in handlers/templates. | Confusing maintenance and unfinished workflows. | E06: establish usage/ownership before targeted cleanup; no blanket removal. |
| F27 / P3 / VERIFIED | Repeated styling/status strings and label defects, including an email sort mapped to surname, occur in panel templates/scripts. | Inconsistent feedback and misleading controls. | E01 and focused correctness tasks: verify each defect; establish reusable visual/interaction contracts later. |

## Phase E product scope

Prioritize daily appointment/request operations, then apply proven patterns to content CRUD.

The future system must cover navigation, dashboard, tables, filters, search, sorting, pagination, forms/validation, buttons, status badges, modals, confirmations, notifications and empty/loading/error/success states. Verify accessible names, focus behavior, keyboard interaction, contrast, touch targets and responsive layouts.

This is a product scope, not a declaration that every listed component has a verified defect. Source inspection cannot substitute for staff workflow observation and local browser review.

## Acceptance for later UI tasks

- Preserve approved authorization and appointment behavior.
- Provide clear feedback and prevent accidental repeated actions.
- Verify relevant desktop, tablet/mobile and keyboard paths.
- Keep public and panel styles isolated; avoid editing purchased vendor assets.
- Add tests appropriate to changed behavior; do not create tests that merely mirror CSS implementation.
