# WP-00 — Yerel başlangıç ve devir notu

Tarih: 2026-09-28
Repo: `C:/Users/DELL/nivgoz/n-hospital-cms-system`
Dal: `nivgoz-professional-v2`
HEAD: `67d8b7a63198050dba43a945e72abcf50318473b`

## Referans ve çalışma ağacı

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

## Son çalışmaların kanıtlı özeti

Son 12 commit yerel geçmişte mevcut. Başlıklar; appointment request/workflow/edit snapshot'larının sahipli wiring'i, appointment diagnostics güvenlik düzeltmeleri, options çağrı envanteri ve son olarak AddDoctor upload policy okumasının transaction'a bağlanmasını gösteriyor. `401647b` EditRandevu snapshot bağlantısı olarak geçmişte yer alıyor. Bu özet commit başlıkları ve yerel Git geçmişiyle sınırlı; ayrıca kaynak kodu davranış denetimi yapılmadı. Kod/commit varlığı test başarısı veya production yayını kanıtlamaz.

## Test, build ve deploy

Bu görevde test, build, uygulama, DB, SMTP veya migration çalıştırılmadı. Bu yerel snapshot için bu kontrollerin sonucu UNKNOWN. Deploy/production commit'i ve canlı davranış doğrulanmadı; UNKNOWN.

## State ve önerilen sonraki adım

`docs/ai/PROJECT_STATE.md` başlangıç tarihi `2026-09-14` ve “Next action” A01 e-posta hata önerisini gözden geçirme olarak eski kalmış. 28 Eylül HEAD'indeki sonraki iş akışını göstermiyor; bu görev plan dosyasını değiştirmiyor. Dokümanda settings sınırı A04/F03 olarak listelenmiş; SEC-002'nin güncel durumu bu dar kontrolde açık kanıtla saptanamadı.

Sonraki dar görev: SEC-002'nin güncel durumunu kontrol et; ayar response ve Jet template'lerinde secret alanlarının çıkarılmasını mevcut kanıtla doğrula. Eğer yapılmış görünüyorsa ilgili commit/dosya/test kanıtını kaydet ve çalışma zamanı/deploy belirsizliğini ayrıca belirt. Bu devir notu diğer talimatları veya tarihsel audit kayıtlarını geçersiz kılmaz.

## Güncel devir — 2026-09-28, `13b048e`

Yukarıdaki `67d8b7a` HEAD, status ve öneri ilk WP-00 anının tarihsel
fotoğrafıdır. Bu devir öncesi yerel HEAD `13b048ee8e26b70825f59d6cf8b6a8cebb921b46`, dal
`nivgoz-professional-v2` idi. Son yerel commitler `13b048e` (owned pilot
sözleşmeleri), `ba1494b` (randevu düzenleme tarayıcı taslağı), `c64aafa`
(iletişim), `af8e96b` (iş başvurusu), `c549a26` (randevu bildirimi) idi.
Remote güncelliği veya production yayını araştırılmadı. Bu turda test/build,
uygulama, DB, SMTP, migration veya tarayıcı çalıştırılmadı.

### Kanıtlanan yerel durum ve açık sınır

- **SEC-002A / F03:** `0ac8fe7` kaynak ve hedefli testlerde ayar GET/POST için güncel admin sınırı, secret içermeyen read DTO/Jet alanları ve boş secret girişini koruyan güncelleme sağlıyor. Tam `baserouter` ve gerçek Jet yanıtı **BLOCKED**; diğer F03 endpoint/nesne izinleri **AÇIK**.
- **SEC-003A/B/C / F04:** `a6ab48d` etkin/güncel hesap ve rolü istek içinde doğruluyor; `06ec7c5` socket girişinde ve sıradaki canlı teslim öncesinde yeniden denetliyor. Dar ağsız testler geçmiş; tam `main` **BLOCKED**, token ömrü ve SEC-003D nesne/şube kapsamı **AÇIK**.
- **REL-001A / F08:** `f0a26a7` MIME/SMTP hatalarını proses sonlandırmadan döndürüyor ve DB sonrası e-posta çağıranları için yerel hata/süreç testleri geçmiş. F05'in randevu, iş başvurusu ve iletişim üreticileri sırasıyla `c549a26`, `af8e96b`, `c64aafa` ile kaydedilen nesneden üretiliyor; panel DOM metni `textContent` kullanıyor. F05 outbox/retry **AÇIK**; canlı teslim garantisi yok.
- **Dosyalar:** İK CV/ekleri `c1e7b72` ile private yol/korumalı indirme için kısmi yerel kabulde; eski İK dosyaları taşınmadı, admin dışı İK/moderator rol ve nesne kararı **AÇIK**. Genel dosya silme `8b719f2` ve upload URL'si `6af5993` yalnız genel `static/uploads` yolunda yerel testli; diğer silme çağrıları ve genel upload politikası **AÇIK**.
- **UI ve migration:** UI-001–005 dört yerel UI commit'inde kaynak, yapay DOM/Fiber düzeyinde düzeltildi; gerçek Jet ve oturumlu masaüstü/mobil/klavye tarayıcısı **BLOCKED/UNKNOWN**. Necoo N01–N04 yerel kabul; N05–N11 **AÇIK**. Tam `main`/`baserouter`/Jet, gerçek DB/şema, eski İK dosyaları, proxy/cache, tarayıcı ve production için ayrıca doğrulama gerekir; mevcut durumda **BLOCKED** veya **UNKNOWN**. Ayrıntılı kanıt ve sınırlar [güvenlik backlog'u](SECURITY_BACKLOG.md), [migration durumu](NECOO_MIGRATION_STATUS.md) ve [UI backlog'unda](UI_UX_IMPLEMENTATION_BACKLOG.md).

### Tek sonraki teknik iş

N05 için **yalnız options okuma seam'i**nin mevcut kaynak/çağıran sözleşmesini ve
SQL/NULL/hata/cache davranışını kirli options dosyalarına dokunmadan envanterle;
ayrı küçük paketin kapsamını ve kanıt eksiklerini çıkar. N05 kabulünü bu belge
güncellemesiyle ilan etme.
