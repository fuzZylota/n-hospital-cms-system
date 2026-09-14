# Nivgöz — public website backlog

Source: accepted Phase 1 audit. No technical or visual implementation is approved.

## Baseline and protected work

Public pages use a shared Jet layout/components with page-specific templates and CSS/JavaScript. Page types include homepage, doctors/details, departments, examinations, centers/details/center doctors, appointments, contact/transportation, corporate pages, news, galleries, search and cookie pages.

Content is principally Turkish with Google Translate, rather than a full multilingual content/routing model. Public routes, database content and asset references are coupled in places.

Preserve prior lazy loading, responsive banner preload, dimensions, carousel/layout guards, root-gap CSS, font reductions, route-specific/deferred scripts and accessibility improvements. See the commit assessments in [PROJECT_STATE](PROJECT_STATE.md). Do not infer that historical production measurements are current.

## Findings

| ID / severity / confidence | Evidence/current behavior | Risk | Future action |
| --- | --- | --- | --- |
| F16 / P2 / VERIFIED | Appointment page, homepage and floating forms have divergent validation/captcha/loading behavior. | Inconsistent conversion and feedback. | B06, owned by [APPOINTMENT_BACKLOG](APPOINTMENT_BACKLOG.md); obtain approval for business-rule changes. |
| F21 / P2 / VERIFIED markup | Public feedback and accessibility gaps remain despite earlier semantic/label/focus improvements. | Visitors with impaired vision may encounter friction; browser impact unmeasured. | F-T04: test critical navigation and appointment flows with keyboard, zoom and assistive-technology checks. |
| F22 / P2 / VERIFIED source omissions | Public templates/controllers do not establish complete canonical/hreflang/structured-data coverage; doctor routes duplicate content; fallback does not explicitly return 404. No application sitemap/robots implementation established. | Ambiguous indexing and error semantics; deployed server behavior UNKNOWN. | F-T03: verify route/metadata contracts locally; agree SEO responsibilities without rewriting content. |
| F23 / P2 / VERIFIED source behavior | Consent code in the shared public layout gates initial analytics load; later revocation updates local storage without disabling an already loaded tracker. | The technical behavior may not match the user's withdrawal choice. | F-T05: verify tracker lifecycle and approve the expected consent contract. This is not a legal compliance conclusion. |
| F24 / P2 / VERIFIED patterns | Multiple blocking stylesheets, manual asset versions, database image references and hard-coded production asset URLs remain. | Performance/cache/environment coupling; actual current cost unmeasured. | F-T01/F-T02: measure first, then fix one demonstrated bottleneck while checking LCP/CLS and multiple slides. |
| F27 / P3 / VERIFIED | Public/panel styling and controls do not form complete documented design systems. | Inconsistent maintenance and interactions. | Phase G: define public tokens/components after technical stability; do not force public and admin into identical UI. |

## Ordered future scope

1. Phase F: establish repeatable performance evidence and verify existing performance work.
2. Address measured images, CSS, JavaScript and font costs.
3. Improve technical SEO, structured data, accessibility and multilingual architecture through scoped tasks.
4. Phase G: define typography, spacing, buttons, cards, forms, header/navigation and page patterns.
5. Apply the system to homepage, doctor, department, center and appointment experiences.
6. Add motion/microinteractions only with reduced-motion and performance verification.

Production PageSpeed, compression, caching, TLS and server headers belong to approved Phase H verification. Do not optimize based solely on historical CLAUDE.md metrics or overwrite existing SEO content.
