# Nivgöz Pro v3 — yol haritası

Tarih: 2026-10-03 · Dal: `nivgoz-professional-v2` · Durum: **YÖN ONAYLANDI (2026-10-03)** — backend kapsamda, kendi tasarım sistemi, panel eksiksiz Tabler, gövde fontu Atkinson Hyperlegible Next. ONAY etiketli görevler yine ayrıca sorulur.
Çalışma protokolü: `.claude/skills/nivgoz-pro/SKILL.md` (spec → uygula → doğrula → commit).

Hedef: nivgoz.com'un ön yüzü, randevu akışı ve yönetim paneli; tasarım,
erişilebilirlik, güvenlik, SEO ve performansta "üst düzey bir ürün ekibi
yapmış" seviyesine çıkması. Kitle düşük görmeli, yaşı yüksek, mobil ağırlıklı
ziyaretçi → **erişilebilirlik ve okunabilirlik tasarımın kendisidir.**

## Başlangıç durumu (2026-10-03 ölçümü)
- `go build` (main) yerel olarak geçiyor; Go 1.25.1, psql ve Node 22 mevcut.
- Panel, ayrı bir admin teması değil; public Mediox temasının Bootstrap 5.3 +
  14 eklenti CSS'iyle çalışıyor (owl, slick, nouislider, jarallax… panelde
  gereksiz). → Tabler (Bootstrap 5 tabanlı) geçişi doğal ve düşük riskli.
- Public: Mediox + 18 sync CSS, jQuery + 7 eklenti. Marka: `#0A2241` / `#df0024`.
- Önceki (ChatGPT) çalışması: 50 commit, ağırlıkla yetki/güvenlik sertleştirme
  ve geniş belge. Deploy kapısı KAPALI (bildirim migration'ı, bkz. README).
- 130 Jet şablonu, ~92 bin satır Go.

## Önceki çalışmanın değerlendirmesi (ChatGPT/Codex, 14–30 Eylül)
Dallar: `main` (15 Eylül, canlıya en yakın), `codex-performance-v1` (10 commit,
bu dala zaten birleşmiş), `nivgoz-professional-v2` (main'in 114 commit önünde,
**hiçbiri canlıda değil**, deploy kapısı KAPALI).

| Dönem | Ne yapıldı | Değer |
| --- | --- | --- |
| 14–15 Eyl (Codex) | Font preload, ikon CSS erteleme; robots.txt + sitemap.xml, OG/Twitter, canonical, JSON-LD (MedicalOrganization, Physician, Breadcrumb); body limit 50 MB, JWT exp, cookie Secure/SameSite, global rate limit (100/dk/IP), upload whitelist, graceful shutdown | Yüksek — korunur |
| 17–27 Eyl | 35 "architecture" commit: neormgo yerine parça parça "owned data layer" (snapshot/repository), notification hub | Orta — yarım geçiş (N05–N11 açık) |
| 28–30 Eyl | 32 "security" commit: endpoint bazlı rol + şube yetkisi, randevu/iletişim/İK PII sınırları, CV'ler private, ayar secret'ları panelden çıkarıldı | Yüksek — korunur |
| Tüm dönem | 5 küçük "ui" commit; görsel/tasarım işi neredeyse yok. Panel hâlâ Mediox. | — |

Ölçüm (2026-10-03, bu oturum): eklenen satırların **48.288'i test**, 11.154'ü
Go, 2.317'si UI, 4.586'sı belge (`docs/ai` toplam 670 KB). 34 test dosyası
`go/ast` ile kaynağın *yapısını* sabitliyor (ör. "handler bu fonksiyonu tam 1
kez çağırmalı"). Go testleri: 40 paket geçti, **1 başarısız**
(`controllers/post` → `TestGeneralFileUploadFilesystemFailureLeavesNoResult`);
Node testleri 18/18 geçti; `main` build geçiyor.

Riskler:
- **Geçici admin-only yazma kuralları**: moderatör/santral kesin randevu
  oluşturamaz/düzenleyemez, talep durumunu değiştiremez. Bu dal bu hâliyle
  canlıya çıkarsa çağrı merkezi iş akışı durur → rol matrisi kararı (P1-0).
- AST "wiring" testleri davranışı değil kod biçimini test ediyor; UI/handler
  yeniden yazımında toplu kırılır. Yaklaşım: yeşil kaldıkça korunur, yeniden
  yazılan handler'da davranış testine (HTTP + sahte DB) çevrilir (P0-5).
- Canlıdaki kod `main` + sunucuda elle yapılmış düzeltmeler; bu dalla farkı
  deploy öncesi ayrıca doğrulanmalı.

## Tasarım kararları (onaylandı)
| Konu | Öneri | Gerekçe |
| --- | --- | --- |
| Gövde fontu | **Atkinson Hyperlegible Next** (self-host, woff2, Türkçe glifler) | Braille Institute'un düşük görme için tasarladığı aile; kitleyle birebir uyumlu, "jenerik AI fontu" değil |
| Başlık fontu | Aynı ailenin 700/800 ağırlığı (tek aile = tek istek) | Performans + tutarlılık |
| Renk | Marka token'ları + AA/AAA doğrulanmış ton skalası (`--nv-navy-50…900`, `--nv-red-…`) | Kırmızı yalnız vurgu/CTA; metin kontrastı ≥7:1 hedef |
| Tip ölçeği | 18px gövde (mobil 17px), 1.6 satır, `clamp()` akışkan başlıklar | Yaşlı/az gören kitle |
| Panel | **Tabler** (sürüm sabit, `static/vendor/` altına) | Bootstrap 5 uyumlu, açık kaynak, eksiksiz admin bileşenleri |
| İkon | Tabler Icons (SVG sprite, yalnız kullanılanlar) | Font Awesome 1 MB'ı yerine |
| Erişilebilirlik aracı | Header'da kalıcı "Aa+ / Yüksek kontrast" düğmesi (localStorage) | Kitleye özgü ayırt edici özellik |

## Fazlar ve görevler
Biçim: `[ ] ID — iş · Dosyalar · Kabul`. **ONAY** = başlamadan kullanıcı onayı.

### P0 — Doğrulama altyapısı (her şeyin önkoşulu)
- [ ] P0-1 — Yerel geçici ortam: tek komutla boş Postgres (port 55432) + `schema.sql` + örnek seed + uygulama. · `tools/dev/` · Production'a hiçbir bağlantı yok; `tools/dev/up.sh` ile ana sayfa 200 döner.
- [ ] P0-2 — QA betiği: Playwright + axe-core ile kritik 10 sayfanın 390/1280px ekran görüntüsü, a11y ihlal sayısı, konsol hatası. · `tools/qa/` · Tek komut, JSON + PNG çıktı; baseline kaydı.
- [ ] P0-4 — Başarısız testi kök nedeniyle düzelt (`TestGeneralFileUploadFilesystemFailureLeavesNoResult`).
- [ ] P0-5 — AST wiring testleri envanteri: hangileri davranış testine çevrilecek, hangileri silinecek. **ONAY** (test silme).
- [ ] P0-3 — CI: GitHub Actions'ta `go build`, `go test`, `node --test`. · `.github/workflows/ci.yml` · PR'da yeşil.

### P1 — Güvenlik ve backend sertleştirme
- [ ] P1-0 — **ONAY** Kalıcı rol matrisi (admin / moderatör / santral / İK × randevu, talep, iletişim, İK, içerik) → geçici admin-only kurallar kaldırılır. Deploy öncesi zorunlu.
- [ ] P1-1 — Güvenlik başlıkları middleware'i (CSP report-only → enforce, HSTS, X-Content-Type-Options, Referrer-Policy, Permissions-Policy, frame-ancestors). · `main/` · securityheaders.com ≥ A.
- [ ] P1-2 — Public formlara (randevu, iletişim, İK) form bazlı sıkı hız sınırı (global 100/dk zaten var) + honeypot + sunucu tarafı doğrulama (Fiber yerleşik `limiter`, yeni bağımlılık yok). · `baserouter/`, ilgili controller · 429 + kullanıcı dostu mesaj.
- [ ] P1-3 — Panel oturumu: CSRF koruması (Fiber yerleşik `csrf`), giriş deneme sınırı (cookie bayrakları + JWT exp Codex'te yapıldı; doğrulanacak). · `main/`, `lib/` · Testli.
- [~] P1-4 — `robots.txt` + `sitemap.xml` route'ları var (`18e98fd`); yerel ortamda 200/content-type doğrulanacak.
- [ ] P1-5 — **ONAY** `dgrijalva/jwt-go` → `golang-jwt/jwt/v5` (bağımlılık değişimi). · `lib/` · Eski token'lar geçiş süresince okunur.
- [ ] P1-6 — **ONAY** Sunucu: `127.0.0.1:2000` bind, root olmayan systemd servisi, TLS 1.2+, brotli. · deploy notu + `tools/deploy/` · Runbook.
- [ ] P1-7 — Açık deploy kapısı: bildirim migration'ını geçici DB'de uygula/test et, production runbook'u hazırla. · `migrations/` · Kapı AÇIK.

### P2 — Tasarım sistemi (public)
- [ ] P2-1 — Token katmanı: renk/tip/boşluk/gölge/radius, açık + yüksek kontrast modu. · `static/css/frontend/tokens.css` · Kontrast tablosu belgelenmiş.
- [ ] P2-2 — Font: Atkinson Hyperlegible Next self-host, `font-display: swap`, preload; Google Fonts kaldırılır. · `static/fonts/` · CLS artmaz.
- [ ] P2-3 — Bileşenler: buton (3 varyant, 48px), form alanları (görünür etiket, hata metni `aria-describedby`), kart, rozet, bildirim/toast (0.7+ opak), breadcrumb. · `components.css` · Bileşen sayfası (`/_ui`, yalnız dev).
- [ ] P2-4 — Header + mega menü yeniden: tek satır, büyük dokunma hedefleri, sabit "Randevu Al" + "Ara" CTA, erişilebilirlik düğmesi, mobil tam ekran menü (focus trap). · `components/frontend-header*.jet` · Klavye ile tam gezilebilir.
- [ ] P2-5 — Footer yeniden: 3 merkez kartı (adres/telefon/harita linki), çalışma saatleri, KVKK, sosyal. · `frontend-footer.jet` · AA.
- [ ] P2-6 — Hero/slider: tek slide gerçeğine göre statik hero (otomatik kayan yok), net değer önerisi + 2 CTA; slick anasayfadan kalkar. · `components/home/` · LCP < 2.5 sn hedef.

### P3 — Public sayfalar
- [ ] P3-1 — **Randevu talep akışı** (sıfırdan): 4 adım (merkez → bölüm → doktor/tarih tercihi → iletişim), ilerleme göstergesi, her adımda doğrulama, onay ekranı + talep numarası, KVKK onayı. · `views/frontend/randevu.jet`, `js/frontend/randevu-page.js` · Ekran okuyucuyla uçtan uca tamamlanır.
- [ ] P3-2 — **Randevu takip** (public): talep no + telefon ile durum sorgulama. · yeni route + view · **ONAY** (yeni public endpoint, PII riski — yalnız durum döner).
- [ ] P3-3 — Anasayfa: hero, hızlı erişim (Randevu / Doktorlar / Merkezler / Ara), hizmetler, doktorlar, merkezler, güven kanıtları, SSS.
- [ ] P3-4 — Doktor liste/detay: filtre (merkez, bölüm), kart, profil + "bu doktordan randevu".
- [ ] P3-5 — Merkez (şube) liste/detay: harita (lazy), yol tarifi, saatler, telefonla ara.
- [ ] P3-6 — Tıbbi birim/tetkik sayfaları, haberler, iletişim, İK, KVKK, 404.

### P4 — SEO
- [~] P4-1 — (OG/canonical temeli var, `7ea8480`) Her sayfaya benzersiz title/description, canonical, OG/Twitter. · layout + controller verisi.
- [~] P4-2 — (MedicalOrganization/Physician/Breadcrumb temeli var) JSON-LD tamamla: `MedicalOrganization` + 3 `MedicalClinic` (adres, saat, telefon), `Physician`, `BreadcrumbList`, `FAQPage`.
- [ ] P4-3 — Türkçe slug'lar, kırık link taraması, 301 eşlemesi.

### P5 — Performans
- [ ] P5-1 — Kritik CSS inline + kalan CSS tek paket; kullanılmayan tema CSS'i public'ten çıkar. Hedef: mobil Lighthouse Perf ≥ 90.
- [ ] P5-2 — Responsive görseller (`srcset`/`<picture>`): Jet'e yardımcı fonksiyon kaydı (backend). · `main/`
- [ ] P5-3 — Statik dosya adlarına içerik hash'i / `?v=` otomatik, `immutable` cache.
- [ ] P5-4 — jQuery eklentilerini kaldır (owl/slick/wow/jquery-ui) → native.

### P6 — Yönetim paneli → Tabler
- [ ] P6-1 — Tabler'ı vendor'la, yeni `layouts/panel/panel.jet` (sidebar, topbar, kullanıcı menüsü, karanlık mod, breadcrumb). Mediox panelden tamamen çıkar.
- [ ] P6-2 — Dashboard: bugünkü talepler, bekleyen/onaylı/iptal KPI'ları, şube bazlı grafik, son talepler.
- [ ] P6-3 — **Randevu talepleri**: filtreli/sıralı tablo (durum, şube, tarih), durum rozetleri, satır içi hızlı aksiyonlar, detay offcanvas, XLSX dışa aktarma.
- [ ] P6-4 — **Randevu takip**: takvim + günlük liste görünümü, doktor/şube filtresi.
- [ ] P6-5 — Diğer CRUD sayfaları (doktor, şube, birim, tetkik, haber, içerik, kullanıcı, ayarlar) ortak form/tablo bileşenleriyle; ~60 panel CSS dosyası 1 dosyaya iner.
- [ ] P6-6 — Bildirim merkezi, toast'lar, boş/yükleniyor/hata durumları.
- [ ] P6-7 — Giriş ekranı (Tabler auth), şifre politikası, oturum zaman aşımı uyarısı.

### P7 — Operasyon
- [ ] P7-1 — Deploy betiği + geri dönüş (rollback), yedek doğrulama.
- [ ] P7-2 — Uptime/hata izleme, Search Console, Web Vitals RUM.

## Sıra ve maliyet stratejisi
P0 → P1-0 (karar) → P1 (1–4) → P2 → P3-1 → P6-1/3/4 → P3 kalan → P4 → P5 → P6 kalan → P7.
Her görev tek commit; görev başına yalnız ilgili dosyalar okunur; büyük
belgeler (`docs/ai/*`, 670 KB) yeniden okunmaz — bu dosya tek kaynak.

## Ajan sistemi (2026-10-03 araştırması sonucu)
| Repo dosyası | Uyarlandığı kaynak | Ne zaman |
| --- | --- | --- |
| `.claude/skills/nivgoz-pro/SKILL.md` | GSD / spec-kit döngüsü, obra/superpowers (plan + görev başına alt ajan + inceleme) | Her yol haritası görevi |
| `.claude/skills/nivgoz-design/SKILL.md` | Anthropic `frontend-design` skill, UI UX Pro Max (Healthcare kuralları) | Her yeni ekran/bileşen |
| `.claude/agents/design-reviewer.md` | OneRedOak/claude-code-workflows design-review (7 aşama, Playwright) | UI görevi commit'inden önce |
| `.claude/agents/seo-auditor.md` | AgriciDaniel/claude-seo (teknik + yerel + schema) | P4 ve şablon değişikliği |
| yerleşik `/security-review`, `/code-review` | Claude Code | Güvenlik/backend değişikliği |
Kurulmayanlar ve nedeni: UI UX Pro Max (100+ stil/161 palet veritabanı; marka
ve kitle zaten sabit), claude-seo tam paket (18 ajan, harici API'ler), GSD/BMAD
(tek geliştiricili, tek repoluk iş için fazla tören). Yöntemleri alındı.

## Araştırma kaynakları
- Spec odaklı ajan iş akışları (GSD, spec-kit, BMAD): planı tek kaynak dosyada tut, görev = kabul ölçütü.
- Anthropic `frontend-design` skill: jenerik font/renk/düzenden kaçın, net estetik yön seç.
- Tabler (tabler.io), WCAG 2.2, Atkinson Hyperlegible (Braille Institute).
