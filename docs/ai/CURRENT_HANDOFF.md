# Güncel durum ve devir

Tarih: 2026-09-30. Repo: `C:/Users/DELL/nivgoz/n-hospital-cms-system`.
Dal: `nivgoz-professional-v2`. Referans HEAD: `2d093b877d047e1096e92dfc3c97d98e949e6671`.
Bu tur yalnız üç durum belgesinin kaynak/yerel commit ile uzlaştırılmasıdır; test/build, gerçek DB/SMTP/tarayıcı/production doğrulaması yapılmadı. Fetch yok; remote/production güncelliği UNKNOWN. Önceki iki backlog'un unstaged içeriği birleştirildi; kökeni bilinmeyen değişikliklere yazar/onay/tamamlanma ataması yapılmadı.

## Belge sadeleştirmesi — 2026-09-30

Başlangıçta değişmiş SECURITY/APPOINTMENT backlog ve bu devir notunun geçerli içeriği korundu; bilinmeyen kökene yazar veya tamamlanma ataması yapılmadı. Eski proje/changelog kayıtları aşağıda, diğer taşınan kanıtların adresleri [ana dizinde](../../README.md#birleştirilen-belgelerin-yeni-adresleri). Tarihsel PASS/BLOCKED kanıtları tekrar koşulmuş değildir. Bu tur yalnız belge/bağlantı/satır sonu ve belge yolu bağımlılığı kontrolleridir; uygulama testi, gerçek DB/SMTP/tarayıcı/production doğrulaması değildir.

## Yerel yetki sınırları

| Yol grubu | Güncel yerel kural | Commit kanıtı |
| --- | --- | --- |
| Kesin randevu liste/detay/edit-form GET | Güncel aktif DB admin tüm şubeler; moderator gerçek şubede `can_view`; santral ayrıca güncel `users.sid` eşleşmesi; İK/diğerleri ret. | `08d4b93`, `9beedb3`, `03cbac4` |
| Randevu talebi liste/detay/XLSX/latest JSON GET | Aynı güncel aktör/gerçek şube okuma kuralı; PII, seçenek ve sayaç sorguları izin kapsamına bağlı. | `e796c57`, `527adfc`, `e3ac4bf`, `9e3b103` |
| Kesin randevu create/edit/delete ve talep status POST | Geçici güncel aktif DB admin; gerçek hedef/durum ve transaction kontrolleri. | `85911da`, `d1cc397`, `0fcd607`, `0356ca8` |
| Randevu talebi delete POST | Admin; moderator gerçek şubede `can_delete`; santral ayrıca `users.sid` eşleşmesi. Karar/kilit/DELETE aynı transaction. | `1169426` |
| İletişim liste/detay/yanıt-formu GET; respond/set-as-read/delete POST | Güncel aktif DB admin; gerçek kanonik hedef. Şube sütunu yok. Set-as-read boolean hedefi idempotent uygular; delete tam bir satır ve başarılı commit ister. | `98180ac`, `c45e851`, `f752342`, `2d093b8` |

Kaynak: `controllers/panel/appointment_*_access.go`, `contact_request_read_access.go`, `controllers/post/randevular/appointment_*_access.go` ve `controllers/post/contact_request_*` yardımcıları. Endpoint ayrıntıları [randevu](APPOINTMENT_BACKLOG.md) ve [güvenlik](SECURITY_BACKLOG.md) backlog'larında. Bu dar sınırlar tüm uygulamanın kabulü değildir. **Admin-only yazma kararları geçicidir; moderator/santral iş akışlarını durdurabilir.** Kalıcı rol/nesne/şube matrisi ayrı ürün kararıdır.

## Deploy kapısı ve açık işler

- `5fe3a6f`: kişisel bildirim şeması/owned writer temeli commitli, üreticilere bağlı değil; mevcut bildirim üretimi ve okuma legacy/paylaşımlı. Migration uygulanmadı.
- `97d533b`: `DeleteUser` hedef receipt'lerini sonra kullanıcıyı tek transaction'da siler; artık `notification_receipts` tablosuna bağımlıdır. **Migration ve gerçek PostgreSQL silme/FK doğrulaması olmadan deploy kapısı KAPALI.** Olaylar/diğer alıcılar korunur; son admin ve retention kararı açık.
- Ortak `notification=true` GET okundu yazımları kişisel receipt erişimi değildir. SMTP deadline/cancellation, outbox/retry ve tek-gönderim/teslim garantisi açık. Respond okuma transaction'ını SMTP'den önce kapatır; gönderim sonrası ayrı yazma hatası `email_sent_state_not_saved` kısmi başarısıdır. Rollback e-postayı geri alamaz; yetkili okuma sonrası rol iptali gönderimi kesin durdurmaz.
- İletişim silmesi notification/receipt temizlemez; eski link ve silme sonrası asenkron üretim yarışı açık. Bağlı randevu talebi silmesindeki kaynak `ON DELETE SET NULL` dönüşüm bağını koparabilir. Bildirim/bağlı kayıt silme, son admin ve audit/retention kararları açık; gerçek FK/trigger/commit davranışı UNKNOWN.
- Gerçek DB/SMTP/tarayıcı/production doğrulanmadı. Tarihli kayıtlardaki fake HTTP/transaction/Jet/Node ve ağsız test sonuçları **tarihsel kanıttır; bu turda yeniden çalıştırılmadı**. Belge uzlaştırması deploy veya production kabulü değildir.

## Sonraki tek kod işi

**İK başvurularının liste/detay/yanıt/silme yollarının PII ve nesne yetkisi envanteri.** Kayıtlı route → handler → güncel aktör/hedef kararı → PII/seçenek okuması → SMTP/silme/bağlı veri yan etkisi zincirini kaynakla çıkar; bu envanter yeni rol politikası onayı değildir. `c1e7b72` ve `lib/job_application_media.go:JobApplicationMediaHandler` içindeki admin-only CV/ek erişimi bütün başvuru akışının kabulü sayılmaz.

## Tarihsel devir kayıtları

Aşağıdaki HEAD/status/sonraki iş ve test sınırları kendi kayıt anlarına aittir; güncel öncelik yukarıdadır. Eski kayıtlar korunmuştur.

## WP-00 — Yerel başlangıç ve devir notu

Tarih: 2026-09-28
Repo: `C:/Users/DELL/nivgoz/n-hospital-cms-system`
Dal: `nivgoz-professional-v2`
HEAD: `67d8b7a63198050dba43a945e72abcf50318473b`

### Referans ve çalışma ağacı

Dış inceleme referansı `67d8b7a63198050dba43a945e72abcf50318473b` yerel HEAD ile aynıdır; yerel `origin/nivgoz-professional-v2` de aynı commit'i gösteriyor. Fetch yapılmadı; remote ref'in güncelliği doğrulanmadı. `401647b` HEAD'in atasıdır.

Başlangıçta mevcut ve korunmuş kullanıcı değişiklikleri:

- `fiber-v2/controllers/post/branslar/branslar.go`
- `fiber-v2/controllers/post/branslar/uploadpolicywiring/wiring_test.go`
- `fiber-v2/controllers/post/custommediauploadwiring/wiring_test.go`
- `fiber-v2/controllers/post/doctors/uploadpolicywiring/wiring_test.go`
- `fiber-v2/controllers/post/options/optionmediamutationwiring/wiring_test.go`
- `fiber-v2/database/postgres/options.go`
- `fiber-v2/database/postgres/options_test.go`
- Yeni: `fiber-v2/controllers/post/branslar/add_branch_upload_policy_test.go`

Başlangıç diff özeti: staged yok; unstaged 7 tracked dosyada 74 ekleme / 28 silme; ayrıca yukarıdaki bir untracked test dosyası.

### Son çalışmaların kanıtlı özeti

Son 12 commit yerel geçmişte mevcut. Başlıklar; appointment request/workflow/edit snapshot'larının sahipli wiring'i, appointment diagnostics güvenlik düzeltmeleri, options çağrı envanteri ve son olarak AddDoctor upload policy okumasının transaction'a bağlanmasını gösteriyor. `401647b` EditRandevu snapshot bağlantısı olarak geçmişte yer alıyor. Bu özet commit başlıkları ve yerel Git geçmişiyle sınırlı; ayrıca kaynak kodu davranış denetimi yapılmadı. Kod/commit varlığı test başarısı veya production yayını kanıtlamaz.

### Test, build ve deploy

Bu görevde test, build, uygulama, DB, SMTP veya migration çalıştırılmadı. Bu yerel snapshot için bu kontrollerin sonucu UNKNOWN. Deploy/production commit'i ve canlı davranış doğrulanmadı; UNKNOWN.

### State ve önerilen sonraki adım

`docs/ai/PROJECT_STATE.md` başlangıç tarihi `2026-09-14` ve “Next action” A01 e-posta hata önerisini gözden geçirme olarak eski kalmış. 28 Eylül HEAD'indeki sonraki iş akışını göstermiyor; bu görev plan dosyasını değiştirmiyor. Dokümanda settings sınırı A04/F03 olarak listelenmiş; SEC-002'nin güncel durumu bu dar kontrolde açık kanıtla saptanamadı.

Sonraki dar görev: SEC-002'nin güncel durumunu kontrol et; ayar response ve Jet template'lerinde secret alanlarının çıkarılmasını mevcut kanıtla doğrula. Eğer yapılmış görünüyorsa ilgili commit/dosya/test kanıtını kaydet ve çalışma zamanı/deploy belirsizliğini ayrıca belirt. Bu devir notu diğer talimatları veya tarihsel audit kayıtlarını geçersiz kılmaz.

### Tarihsel devir — 2026-09-28, `13b048e`

Yukarıdaki `67d8b7a` HEAD, status ve öneri ilk WP-00 anının tarihsel
fotoğrafıdır. Bu devir öncesi yerel HEAD `13b048ee8e26b70825f59d6cf8b6a8cebb921b46`, dal
`nivgoz-professional-v2` idi. Son yerel commitler `13b048e` (owned pilot
sözleşmeleri), `ba1494b` (randevu düzenleme tarayıcı taslağı), `c64aafa`
(iletişim), `af8e96b` (iş başvurusu), `c549a26` (randevu bildirimi) idi.
Remote güncelliği veya production yayını araştırılmadı. Bu turda test/build,
uygulama, DB, SMTP, migration veya tarayıcı çalıştırılmadı.

#### Kanıtlanan yerel durum ve açık sınır

- **SEC-002A / F03:** `0ac8fe7` kaynak ve hedefli testlerde ayar GET/POST için güncel admin sınırı, secret içermeyen read DTO/Jet alanları ve boş secret girişini koruyan güncelleme sağlıyor. Tam `baserouter` ve gerçek Jet yanıtı **BLOCKED**; diğer F03 endpoint/nesne izinleri **AÇIK**.
- **SEC-003A/B/C / F04:** `a6ab48d` etkin/güncel hesap ve rolü istek içinde doğruluyor; `06ec7c5` socket girişinde ve sıradaki canlı teslim öncesinde yeniden denetliyor. Dar ağsız testler geçmiş; tam `main` **BLOCKED**, token ömrü ve SEC-003D nesne/şube kapsamı **AÇIK**.
- **REL-001A / F08:** `f0a26a7` MIME/SMTP hatalarını proses sonlandırmadan döndürüyor ve DB sonrası e-posta çağıranları için yerel hata/süreç testleri geçmiş. F05'in randevu, iş başvurusu ve iletişim üreticileri sırasıyla `c549a26`, `af8e96b`, `c64aafa` ile kaydedilen nesneden üretiliyor; panel DOM metni `textContent` kullanıyor. F05 outbox/retry **AÇIK**; canlı teslim garantisi yok.
- **Dosyalar:** İK CV/ekleri `c1e7b72` ile private yol/korumalı indirme için kısmi yerel kabulde; eski İK dosyaları taşınmadı, admin dışı İK/moderator rol ve nesne kararı **AÇIK**. Genel dosya silme `8b719f2` ve upload URL'si `6af5993` yalnız genel `static/uploads` yolunda yerel testli; diğer silme çağrıları ve genel upload politikası **AÇIK**.
- **UI ve migration:** UI-001–005 dört yerel UI commit'inde kaynak, yapay DOM/Fiber düzeyinde düzeltildi; gerçek Jet ve oturumlu masaüstü/mobil/klavye tarayıcısı **BLOCKED/UNKNOWN**. Necoo N01–N04 yerel kabul; N05–N11 **AÇIK**. Tam `main`/`baserouter`/Jet, gerçek DB/şema, eski İK dosyaları, proxy/cache, tarayıcı ve production için ayrıca doğrulama gerekir; mevcut durumda **BLOCKED** veya **UNKNOWN**. Ayrıntılı kanıt ve sınırlar [güvenlik backlog'u](SECURITY_BACKLOG.md), [migration durumu](NECOO_MIGRATION_STATUS.md) ve [UI backlog'unda](UI_UX_IMPLEMENTATION_BACKLOG.md).

#### O tarihteki sonraki teknik iş

N05 için **yalnız options okuma seam'i**nin mevcut kaynak/çağıran sözleşmesini ve
SQL/NULL/hata/cache davranışını kirli options dosyalarına dokunmadan envanterle;
ayrı küçük paketin kapsamını ve kanıt eksiklerini çıkar. N05 kabulünü bu belge
güncellemesiyle ilan etme.

<a id="historical-project-state"></a>

## Arşiv kaydı: PROJECT_STATE.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### Nivgöz — project state

Updated: 2026-09-14.

#### Accepted baseline

The accepted Phase 1 Discovery & Architecture Audit is the technical baseline. This documentation is a working index of that audit, not a new audit.

- Repository: C:/Users/DELL/nivgoz/n-hospital-cms-system
- Branch at baseline: astra-full-audit
- Baseline commit: 22958244024a81ca69edb09f77a22aa60291ff6c
- Public website and Admin/CMS are equally important products.
- Current stage: planning documentation. No implementation task is approved.
- Findings remain open; documentation does not constitute remediation.
- Production synchronization, deployed commit and aaPanel configuration are UNKNOWN.

The audit inspected local source, configuration, templates, assets and history. It did not access production, run migrations, connect to the database or validate deployed behavior. Source-level UX findings require browser verification. Go was unavailable on PATH during the audit; compilation and Go tests were not run. Existing JavaScript syntax checks passed, but do not establish functional correctness.

#### Evidence conventions

- VERIFIED: direct repository evidence; does not imply deployed exploitation or a reproduced production incident.
- LIKELY: strong evidence requiring runtime verification.
- HYPOTHESIS: investigation needed; not an established defect.
- P0: catastrophic exposure requiring an immediate safety gate.
- P1: high severity; P2: material correctness/quality issue; P3: lower-priority consistency/maintenance issue.

Finding IDs F01–F28 retain the accepted audit identity. See [ROADMAP](PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-roadmap) for the complete register and ordered work.

#### Verified architecture

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

#### Protected previous work

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

#### Working contract

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

#### Next action

Review the A01 email failure proposal in [ROADMAP](PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-roadmap). Implementation is awaiting explicit approval.

<a id="historical-document-bootstrap"></a>

## Arşiv kaydı: CHANGELOG_AI.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### Nivgöz — AI change log

#### 2026-09-14 — Planning documentation bootstrap

Task: create the accepted audit's working documentation and propose the first implementation task.

Source baseline: 22958244024a81ca69edb09f77a22aa60291ff6c on astra-full-audit.

Created:

- PROJECT_STATE.md
- ROADMAP.md
- SECURITY_BACKLOG.md
- APPOINTMENT_BACKLOG.md
- ADMIN_BACKLOG.md
- PUBLIC_SITE_BACKLOG.md
- ARCHITECTURE_DECISIONS.md
- CHANGELOG_AI.md

Recorded the accepted F01–F28 findings, evidence limitations, protected previous work, approval gates and phases A–H. Selected A01 (F08: email failure terminates the process) as the first proposed implementation task.

Application behavior: unchanged.
Database/configuration/dependencies/production: unchanged.
New audit findings: none.
Implementation status: awaiting explicit approval; no fix implemented.

Validation: all eight requested Markdown files exist; local Markdown links resolve; the roadmap contains 28 unique finding entries; the documentation-only diff was reviewed. Trailing blank lines found during review were removed. Final staged whitespace and scope checks must pass before commit; the commit outcome is reported with task delivery.

No application tests are claimed for a documentation-only change. Go was unavailable during Phase 1; A01 requires a usable local toolchain before implementation can be verified.

#### Recording future work

After each task, append its problem/finding ID, approved scope, files/behavior changed, actual tests and results, diff/commit reference, remaining risks and rollback. Update the owning backlog and roadmap. Do not mark a finding resolved solely because a plan or test was added.
