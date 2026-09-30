# Nivgöz UI/UX uygulama backlog'u

**Kaynak ve sınır — 2026-09-28.** [UI/UX araştırması](UI_UX_RESEARCH.md) A–O ile mevcut çalışma ağacı route/render kontrolünden türetilmiş 27 işlik liste. Durum yerel kaynak içindir; canlı yayın, görsel doğrulama veya klinik doğruluk anlamına gelmez. P0 güven sınırı, P1 temel görev, P2 kullanılabilirlik/erişilebilirlik, P3 bakım önceliğidir. UI-001–005 yerel UI commit'leriyle düzeltildi (`48cb6a9`, `edb776a`, `f0780e3`, `401546c`); satırlarındaki kalan doğrulamalar açıktır. Diğer işlerin durumu kendi satırlarında belirtilir. İlk beş satır küçük, bağımsız UI paketleridir; randevu iş kuralı, hekim–merkez bağı veya rol politikası kararı içermez.

| ID | Etkilenen aile; kanıt | Kullanıcı etkisi | Öncelik | Bağımlılık / iş kararı | Test edilebilir kabul | Durum |
| --- | --- | --- | --- | --- | --- | --- |
| UI-001 | Panel randevu talepleri; araştırma C, randevu-talepleri.jet | Yanlış “Hasta Soy Adı” sıralama etiketi ve yanlış boş tablo sütun sayısı listeyi yanıltır. | P2 | Yok; mevcut sütun anlamı korunur. | E-posta sıralama seçeneği doğru adlanır, boş satır 7 başlığa oturur; 0/1 sonuçta tablo anlaşılır. | yerelde düzeltildi; Jet render ve oturumlu panel UNKNOWN |
| UI-002 | Panel ortak layout; I, panel.jet:2 | Yanlış sayfa dili Türkçe içeriğin telaffuzunu bozar. | P2 | Yok. | HTML dili tr; panel ve oturumlu temel sayfa ekran okuyucu dil algısı izole ortamda doğrulanır. | yerelde düzeltildi; Jet render ve oturumlu tarayıcı UNKNOWN |
| UI-003 | Public merkez hekim listesi; D, doktorlar.jet:34,92 ve baserouter.go:52-55 | Kart/boş durum bağlantısı kayıtlı olmayan yola götürür. | P1 | Mevcut merkez bağlamlı route kullanılır; hekim–merkez veri ilişkisi değiştirilmez. | Yapay geçerli kart ve boş durum bağlantısı doğru kayıtlı hedefe gider; yanlış yol üretilmez. | yerelde düzeltildi; eksik slug kartı bağlantısız, Jet render ve tarayıcı UNKNOWN |
| UI-004 | Panel dokümantasyon; N, panel.go:7461-7469, dokumantasyon.jet, dokumantasyon.js | “Dosya Ekle” başlığı ve fareye bağlı yardım akordeonu görevi karıştırır. | P2 | Yardım metninin rol içeriği ayrıca sahip onayı ister; bu paket başlık/kontrol semantiğidir. | Sayfa başlığı “Dokümantasyon”; her bölüm düğmeyle Tab/Enter/Space/Escape yolunda ve durum duyurusuyla açılıp kapanır. | yerelde düzeltildi; handler/layout başlığı ve 18 akordeon kaynakta doğrulandı, yapay DOM kontrolü geçti; Jet render ve oturumlu tarayıcı UNKNOWN |
| UI-005 | Public genel fallback; H kapsam düzeltmesi, main.go:198, frontend.go:2406-2431, fallback.jet | Bulunamayan URL'nin HTTP durumu açıkça 404 yapılmıyor; geçmişsiz “Önceki sayfa” çıkmaz olabilir. | P1 | Bilinmeyen route için mevcut bulunamadı anlamı korunur. | Bilinmeyen public/panel URL 404 döner, ana sayfa yolu çalışır, geçmişsiz geri eylemi çıkmaz yaratmaz; başlık/odak ve indekslenmeme doğrulanır. | Yerelde düzeltildi; kalıcı `frontend/fallback_http_test.go` gerçek handler'larla yapay Fiber HTTP 404 (bilinmeyen public/panel URL ve boş haber detayı), 200 (`robots.txt`), 302 (`giris`), 500 (sorgu/seçenek/render hatası), sabit `/` bağlantısı ve `X-Robots-Tag` için geçti. Test üretim route kalıplarını yineler; doğrudan `baserouter` paket testi yerel `excelize` bağımlılığı yokluğundan BLOCKED. Gerçek Jet, oturumlu tarayıcı ve production UNKNOWN. |
| UI-006 | Ortak public header/arama; B, frontend-header.jet ve frontend-header-init.jet | Mobil arama/menü klavye ve adlandırma belirsizliği. | P1 | Menü sırası ve randevu CTA metni ürün sahibi kararı; önce mevcut kontrol semantiği. | Ara/menü düğmeleri adlandırılmış, durum ve odak dönüşü tutarlı; 390 px ve yalnız klavyede temel bağlantılar bulunur. | Yerelde düzeltildi; yapay DOM etkileşim testi ve JS sözdizimi geçti. 390 px/masaüstü/200% tarayıcı, gerçek Jet render ve production UNKNOWN. |
| UI-007 | Ana sayfa koşullu popup; A kapsam düzeltmesi, home/popup.jet:134-149,356-400 | Otomatik açılan pencere odak ve kapatma bağlamını kaybettirebilir. | P1 | Popup yayın zamanlaması/içeriği editör kararı; mevcut koşul korunur. | Yapay popup açık/kapalı durumda erişilebilir ad, açılış odağı, Escape ve tetikleyiciye dönüş; arka plan odak dışı. | AÇIK |
| UI-008 | Ana sayfa yorum karuseli; A, home/testimonials.jet:1-106 | Otomatik hareket ve görsel bağlantı adları okunabilirliği etkiler. | P2 | Yorum rızası, doğruluk ve yayın kaynağı içerik sahibi onayı. | Yapay 0/1/çok yorumda hareket durdurulur, kontroller adlandırılır, odak görünür, veri yokken boş karusel yok. | AÇIK |
| UI-009 | Kurumsal çerez politikası; F, frontend.go:1185-1253, kvkk.jet | İki URL aynı içeriği tutarsız koşulla sunar. | P2 | Kanonik URL/yönlendirme içerik ve SEO kararı; mevcut metin teyit edilmeli. | Her iki URL'de doğru çerez başlığı/metni; onaylı kanonik/redirect ve sitemap sonucu, boş içerik/hata ayrı. | AÇIK |
| UI-010 | Public haber ve galeriler; G, foto-galeri.jet, video-galeri.jet | Kırık tema görselleri ve fareye bağlı filtreler medya bulmayı engeller. | P1 | Gerçek medya, telif, açıklama ve video yayını içerik sahibi onayı. | İzinli yapay medya ile 0/çok kart, çalışan bağlantı, klavye filtresi ve anlamlı alt metin; kırık URL yok. | AÇIK |
| UI-011 | Public randevu talebi; A, üç form ve ilgili JS | Talep ile kesin randevu izlenimi karışır; hata toparlama parçalıdır. | P1 | CTA/yanıt sözü ve form alanları işletme/hukuk onayı; akış sessizce birleştirilmez. | Üç mevcut girişte talep sonucu doğru adlanır; yapay boş/CAPTCHA/ağ/başarı durumları alan yanında ve odakla duyurulur. | AÇIK |
| UI-012 | Public hekim/merkez/birim/tetkik; D/E/O | İlişki ve klinik iddia doğrulanmadan hizmete geçiş yanıltabilir. | P1 | Klinik sahip, hekim–merkez ilişkisi ve yayın kuralı onayı. | Yapay aktif/pasif ve ilişkili/ilişkisiz kayıtta yalnız onaylı iddia, doğru merkez/hekim, ilgili talep bağlantısı ve 404/SEO davranışı görünür. | AÇIK |
| UI-013 | Public kurum/başvuru/politika; F | Anlaşma, başvuru ve politika metni eski/yer tutucu olabilir. | P1 | Kurum/sözleşme, İK ve hukuk sahipleri metin/kapsam/sürüm onayı. | Yer tutucu yok; her iddianın sahibi ve tarih/kapsamı kayıtlı; boş/hata durumda yanıltmayan iletişim yolu. | AÇIK |
| UI-014 | Public iletişim, ulaşım, arama, giriş; H | Boş sonuç, harita yüklenmeme ve form hatasında yol belirsiz. | P2 | İletişim kanalı/saat vaadi ve giriş güvenlik metni sahip onayı. | Yapay 0/1/11+ arama, harita yok, geçersiz/başarılı form ve login hatasında görünür, odaklı geri dönüş. | AÇIK |
| UI-015 | Panel talep ve kesin randevu listeleri; C/I | Filtre, sayfalama, tablo ve işlem geri bildirimi gerçek işi izlemeyi zorlaştırır. | P1 | Durum geçişi ve rol/şube yetkisi ayrı onay; UI paketi bunları değiştirmez. | Yapay 0/1/çok kayıtta doğru toplam, klavye sıralama, boş/hata ayrımı ve işlem sonrası gerçek durum. | AÇIK |
| UI-016 | Panel merkez/branş/uzmanlık/hekim; J | İlişki seçimi ve medya adımları tutarsız sonuç verebilir. | P1 | Branş–merkez–hekim modeli ve CV sınırı/izin politikası sahip onayı. | Yapay iki şube/ilişki ve 0/çok medya ile oluştur-düzenle-geri çekme; ekle/düzenle dosya sınırı metni eşit, hata alanla ilişkili. | AÇIK |
| UI-017 | Panel içerik, haber, header menüsü; K | Editör yayın önizlemesi, sıra ve public sonuç arasında kopukluk olabilir. | P1 | İçerik sahibi yayın onayı ve HTML/CSS/JS güven sınırı. | Yapay taslak/aktif/pasif, sıra ve medya değişimi sonrası panel önizlemesi public sonucu, boş/hata ve menü sırası doğru. | AÇIK |
| UI-018 | Panel iletişim/İK kuyruğu; L | Yanıt, okundu, sil ve ek erişimi için iş sonucu/nesne sınırı belirsiz. | P0 | Rol/şube/nesne matrisi, İK özel dosya politikası ve SMTP onayı. | Yapay doğru/yanlış rol/şube/nesne GET/POST, yanıt başarısızlığı ve ek indirmede veri sızmaz; başarı/hata durumu duyurulur. | AÇIK |
| UI-019 | Panel kullanıcı ve izinler; M | Hesap açma ile izin tamamlama arası ve parçalı kayıt yanlış erişime neden olabilir. | P1 | Son admin, self-edit ve şube/nesne yetki matrisi kararı. | Yapay dört rol/iki şubede ilk anda güvenli izin, atomik/uzlaştırılmış değişim, audit ve anlaşılır hata/başarı. | AÇIK |
| UI-020 | Panel ayar düzenleme; M | Rotasyon sonucu ve ayar hatası editöre açık değil. | P2 | SEC‑002A admin kapısı mevcut yerel düzeltmedir; secret değeri gösterilmez. | Sahte değerle boş/değişti/başarısız rotasyon ayrılır; mevcut secret DOM/log/yanıtta yok; odak ve hata metni doğru. | AÇIK |
| UI-021 | Genel dosya yöneticisi; N | Public/özel sınıfı, yükleme kısmi sonucu ve silme hedefi güven sınırıdır. | P0 | Dosya sahibi, rol/nesne, upload politikası ve fiziksel yol tasarımı onayı. | Yapay dosyada izinli/izinsiz yükle-listele-indir-sil, kök dışı ad reddi, kısmi başarı ve disk hatası açık; yanlış dosya değişmez. | AÇIK |
| UI-022 | Panel birim/tetkik/kurum yayını; O | Aktiflik, slug, gerçek toplam ve iki ayrı kurum public kaynağı editörü yanıltır. | P1 | Klinik ve sözleşme sahibi onayı, yayın/geri çekme, şube yetkisi ve SEO kararı. | Yapay üç ailede taslak→onay→yayın→geri çekme; liste/menü/arama/detay/merkez, slug, medya ve sayfalama aynı kuralı izler. | AÇIK |
| UI-023 | Panel ayar güven sınırı; M, SECURITY_BACKLOG SEC‑002A, options_access.go, options_read.go | Yetkisiz ayar okuma ve secret render eski bulgudur; yerel kapı ve güvenli view modeli vardır. | P1 | Daha geniş ayar rolü politikası onay bekler; production UNKNOWN. | Yapay admin/diğer rol GET/POST matrisi ve DOM/yanıtta secret yokluğu; dağıtım ayrıca doğrulanır. | yerelde düzeltildi |
| UI-024 | Oturum/rol yenileme ve bildirim; M/N, SECURITY_BACKLOG SEC‑003A/B/C | Pasif/stale rol ve canlı bildirim eski bulgularının yerel düzeltmesi vardır; nesne sınırı ayrı iş. | P1 | SEC‑003D rol/şube/nesne matrisi UI-018/019; production UNKNOWN. | Aynı tokenla rol düşürme/yükseltme, pasifleştirme, WebSocket inbound/queued olay yapay testte reddedilir; canlı dağıtım sonucu ayrıca kaydedilir. | yerelde düzeltildi |
| UI-025 | Panel randevu düzenleme belleği; I, randevu-duzenle.js:20-66 | Eski localStorage hasta taslağı artık açılışta temizlenir ve otomatik yazım kaldırılmıştır. | P1 | Gerçek tarayıcı, ayrı hesap ve yayın doğrulaması eksik. | Yapay hasta alanı localStorage/sessionStorage'a yazılmaz; eski anahtar temizlenir, başarısız kayıt sonrası form açık oturumda kullanılabilir. | yerelde düzeltildi |
| UI-026 | İK özel CV/diploma erişimi; N, SECURITY_BACKLOG A09, lib/job_application_media.go | Private yazma/static blok/yetkili endpoint yerelde var; eski dosya ve rol kapsamı tamamlanmadı. | P0 | Eski kopya envanteri/migrasyon, İK/moderatör/şube politikası, proxy/cache onayı. | Yapay eski/yeni CV ve diplomada eski URL ve yanlış rol/nesne reddedilir, izinli preview/indirme çalışır, production ayrıca doğrulanır. | doğrulama bekliyor |
| UI-027 | Panel ortak profil/çıkış; I kapsam düzeltmesi, panel-header.jet:81-84, post.go:177-199 | Çıkış bağlantısının klavye ve geri düğmesi sonrası oturum sonucu görsel doğrulanmadı. | P2 | GET/POST yöntemi ve oturum politikası güvenlik sahibi kararı; bu işte değiştirilmez. | Yapay kullanıcı menüden yalnız klavyeyle çıkar; geriyle korunan içerik açılamaz, girişe dönüş ve odak açık. | doğrulama bekliyor |

## Ayrı doğrulama kuyruğu

| ID | Eksik kanıt | Kabul / ortam |
| --- | --- | --- |
| V-01 | Oturumlu panel masaüstü ve 390 px görünümü | Yalnız yetkili yapay veri ortamında A–O panel görevleri ekran ve hata durumlarıyla kaydedilir; production görseli UNKNOWN kalır. |
| V-02 | 200% zoom ve gerçek klavye odağı | Public/panel ilk beş paket dahil, 200% ve 390 px'de taşma, Tab sırası, modal/karusel odak dönüşü ölçülür. |
| V-03 | Gerçek ekran okuyucu | Türkçe dil, başlık, form hata özeti, tablo ve dinamik durum NVDA/VoiceOver benzeri yardımcı teknolojiyle görev bazında doğrulanır. |
| V-04 | Core Web Vitals ve ağ maliyeti | Public örnek sayfalarda izinli ölçümle LCP/CLS/INP ve medya transferi önce/sonra kaydedilir; kaynak gözleminden hız kazanımı çıkarılmaz. |
| V-05 | Yapay verili uçtan uca güvenli görevler | İki şube/dört rol; 0/1/çok kayıt, aktif/pasif, ağ/DB/disk hata, medya ve reddedilen nesne yolları izole ortamda sınanır; gerçek hasta/CV/secret kullanılmaz. |

**Uygulama sırası:** UI-001 → UI-002 → UI-003 → UI-004 → UI-005; her paket kendi dar kabulüyle incelenir. UI-011/012/018/019/022 gibi iş kuralı ve klinik/yetki kararı isteyen işler sahip kararı gelmeden çözülmüş sayılmaz. Route'suz panel Jet'leri ve kullanılmayan home/randevu.jet backlog'da etkin sayfa uygulaması olarak yer almaz; bakım kararı N ve A kapsam notlarındadır.

<a id="historical-admin-backlog"></a>

## Arşiv kaydı: ADMIN_BACKLOG.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### Nivgöz — Admin/CMS backlog

Source: accepted Phase 1 audit. All implementation remains unapproved.

#### Product baseline

Admin/CMS is a first-class product encompassing backend architecture, authorization, appointment workflows, database/query behavior, frontend architecture, UX, accessibility, responsive behavior and design consistency.

The admin uses server-rendered Jet layouts/components, panel CSS and JavaScript, Bootstrap and shared libraries. Large handlers combine data loading, authorization, business rules and rendering. Navigation visibility and endpoint authorization are inconsistent.

Audited modules include dashboard, appointment requests and scheduled appointments, doctors, departments/examinations, centers, content/news, media, users, settings, contacts and recruitment. Testimonials have dormant code/templates; some editor views are placeholders. There is no complete generalized permission model: role checks, legacy center membership and branch permission flags coexist.

#### Correctness before redesign

Complete relevant security and appointment boundaries first:

- F02/F03/F04/F05/F06/F07/F12: [SECURITY_BACKLOG](SECURITY_BACKLOG.md).
- F09/F10/F13–F18: [APPOINTMENT_BACKLOG](APPOINTMENT_BACKLOG.md).
- F25: targeted tests in [ROADMAP](PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-roadmap), Phase C.

#### Architecture and product findings

| ID / severity / confidence | Evidence/current behavior | Risk | Future action |
| --- | --- | --- | --- |
| F19 / P2 / LIKELY | Shared model/database caches and ORM use in fiber-v2/models/models.go and fiber-v2/database/database.go lack demonstrated synchronization guarantees. | Concurrent access may produce stale/inconsistent behavior; dependency thread safety is unverified. | D01: focused dependency/source inspection and local race/concurrency checks before a fix. |
| F20 / P2 / VERIFIED patterns | Recruitment listing in panel.go loads results before slicing; exports are unbounded; dashboards/notifications repeat queries; some totals use page length. | Scaling and count correctness concerns; actual runtime cost is UNKNOWN. | D02: measure one path with isolated fixtures; test pagination/count correctness before optimization. |
| F21 / P2 / VERIFIED markup | Panel layout language is en; mouse-oriented sorting, unnamed icon actions, small text and uncertain modal focus behavior occur in templates/scripts. | Staff keyboard/screen-reader use and mobile operation may be difficult; visual/runtime impact needs testing. | E05: local browser audit of one workflow, then accessible interactions with keyboard/responsive verification. |
| F26 / P3 / VERIFIED | Dormant testimonials, editor placeholders and inconsistent naming remain in handlers/templates. | Confusing maintenance and unfinished workflows. | E06: establish usage/ownership before targeted cleanup; no blanket removal. |
| F27 / P3 / VERIFIED | Repeated styling/status strings and label defects, including an email sort mapped to surname, occur in panel templates/scripts. | Inconsistent feedback and misleading controls. | E01 and focused correctness tasks: verify each defect; establish reusable visual/interaction contracts later. |

#### Phase E product scope

Prioritize daily appointment/request operations, then apply proven patterns to content CRUD.

The future system must cover navigation, dashboard, tables, filters, search, sorting, pagination, forms/validation, buttons, status badges, modals, confirmations, notifications and empty/loading/error/success states. Verify accessible names, focus behavior, keyboard interaction, contrast, touch targets and responsive layouts.

This is a product scope, not a declaration that every listed component has a verified defect. Source inspection cannot substitute for staff workflow observation and local browser review.

#### Acceptance for later UI tasks

- Preserve approved authorization and appointment behavior.
- Provide clear feedback and prevent accidental repeated actions.
- Verify relevant desktop, tablet/mobile and keyboard paths.
- Keep public and panel styles isolated; avoid editing purchased vendor assets.
- Add tests appropriate to changed behavior; do not create tests that merely mirror CSS implementation.

<a id="historical-public-backlog"></a>

## Arşiv kaydı: PUBLIC_SITE_BACKLOG.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### Nivgöz — public website backlog

Source: accepted Phase 1 audit. No technical or visual implementation is approved.

#### Baseline and protected work

Public pages use a shared Jet layout/components with page-specific templates and CSS/JavaScript. Page types include homepage, doctors/details, departments, examinations, centers/details/center doctors, appointments, contact/transportation, corporate pages, news, galleries, search and cookie pages.

Content is principally Turkish with Google Translate, rather than a full multilingual content/routing model. Public routes, database content and asset references are coupled in places.

Preserve prior lazy loading, responsive banner preload, dimensions, carousel/layout guards, root-gap CSS, font reductions, route-specific/deferred scripts and accessibility improvements. See the commit assessments in [PROJECT_STATE](CURRENT_HANDOFF.md#historical-project-state). Do not infer that historical production measurements are current.

#### Findings

| ID / severity / confidence | Evidence/current behavior | Risk | Future action |
| --- | --- | --- | --- |
| F16 / P2 / VERIFIED | Appointment page, homepage and floating forms have divergent validation/captcha/loading behavior. | Inconsistent conversion and feedback. | B06, owned by [APPOINTMENT_BACKLOG](APPOINTMENT_BACKLOG.md); obtain approval for business-rule changes. |
| F21 / P2 / VERIFIED markup | Public feedback and accessibility gaps remain despite earlier semantic/label/focus improvements. | Visitors with impaired vision may encounter friction; browser impact unmeasured. | F-T04: test critical navigation and appointment flows with keyboard, zoom and assistive-technology checks. |
| F22 / P2 / VERIFIED source omissions | Public templates/controllers do not establish complete canonical/hreflang/structured-data coverage; doctor routes duplicate content; fallback does not explicitly return 404. No application sitemap/robots implementation established. | Ambiguous indexing and error semantics; deployed server behavior UNKNOWN. | F-T03: verify route/metadata contracts locally; agree SEO responsibilities without rewriting content. |
| F23 / P2 / VERIFIED source behavior | Consent code in the shared public layout gates initial analytics load; later revocation updates local storage without disabling an already loaded tracker. | The technical behavior may not match the user's withdrawal choice. | F-T05: verify tracker lifecycle and approve the expected consent contract. This is not a legal compliance conclusion. |
| F24 / P2 / VERIFIED patterns | Multiple blocking stylesheets, manual asset versions, database image references and hard-coded production asset URLs remain. | Performance/cache/environment coupling; actual current cost unmeasured. | F-T01/F-T02: measure first, then fix one demonstrated bottleneck while checking LCP/CLS and multiple slides. |
| F27 / P3 / VERIFIED | Public/panel styling and controls do not form complete documented design systems. | Inconsistent maintenance and interactions. | Phase G: define public tokens/components after technical stability; do not force public and admin into identical UI. |

#### Ordered future scope

1. Phase F: establish repeatable performance evidence and verify existing performance work.
2. Address measured images, CSS, JavaScript and font costs.
3. Improve technical SEO, structured data, accessibility and multilingual architecture through scoped tasks.
4. Phase G: define typography, spacing, buttons, cards, forms, header/navigation and page patterns.
5. Apply the system to homepage, doctor, department, center and appointment experiences.
6. Add motion/microinteractions only with reduced-motion and performance verification.

Production PageSpeed, compression, caching, TLS and server headers belong to approved Phase H verification. Do not optimize based solely on historical CLAUDE.md metrics or overwrite existing SEO content.
