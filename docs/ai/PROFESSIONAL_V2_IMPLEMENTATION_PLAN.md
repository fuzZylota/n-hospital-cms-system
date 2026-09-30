# Nivgöz Professional V2 — Faz 2B yetkilendirme, iş akışı ve uygulama planı

Tarih: 2026-09-17
Durum: PLAN — kodlama, migration veya deployment onayı değildir
Repository dalı: `nivgoz-professional-v2`
Plan HEAD'i: `ea1bfc9`

Bu planın ana kanıtı `docs/ai/VERIFIED_AUDIT_2026-09-17.md` dosyasıdır.
Denetim `18e98fd` HEAD'ini kaydeder; bu planda kullanılan ilgili route,
controller, model, şema, template ve middleware dosyalarında `18e98fd..ea1bfc9`
arasında değişiklik olmadığı ayrıca doğrulanmıştır. İnceleme statiktir; uygulama,
test, DB, SMTP, migration ve production bağlantısı çalıştırılmamıştır.

## 1. Amaç, kapsam ve kapsam dışı konular

### Amaç

Onaylanmış rol sınırlarını endpoint ve hedef nesne düzeyine indirmek; randevu
talebinin şubeden başlayıp kontrollü durum geçişleri, arşiv ve audit kaydıyla
ilerlediği uygulanabilir bir yol haritası oluşturmak; özel İK belgelerini public
static yüzeyden çıkarmak ve doğrulanmış bulguları küçük, geri alınabilir çalışma
paketlerine ayırmaktır.

### Kapsam

- `admin`, `moderator`, `santral` ve `ik` rollerinin panel/backend erişimleri.
- Kullanıcının kendi profil/parola işlemleri ile admin kullanıcı yönetiminin
  ayrılması.
- Public randevu talebinde zorunlu, sunucu tarafından doğrulanan aktif şube.
- Randevu talebi listeleme, detay, export, polling, durum değişikliği ve
  arşivlemede aynı nesne/şube kapsamı.
- Header ve şube galerisi yetkileri ile galeri upload güvenliği.
- İK ve iletişim kayıtlarının yönetimi; CV ve gelecekteki özel belgelerin
  private saklanması ve yetkili indirilmesi.
- Randevu, iletişim ve İK kayıtlarında soft-delete/arşiv, audit trail ve daha
  sonra belirlenecek kalıcı silme politikası.
- Secret değerlerinin response, template, log ve düşük yetkili endpoint'lerden
  çıkarılması.
- `VERIFIED_AUDIT_2026-09-17.md` içindeki doğrulanmış/kısmen doğrulanmış
  bulguların bağımlılık sırasına göre uygulanması.

### Kapsam dışı

- Bu belgede uygulama kodu, SQL veya migration yazılması.
- Production erişimi, deploy, backup/restore, gerçek DB/SMTP bağlantısı veya
  mevcut migration betiklerinin çalıştırılması.
- Yeni framework, ORM, genel service/repository katmanı, RBAC ürünü veya event
  sistemi seçimi.
- Görsel yeniden tasarım, genel performans/SEO çalışması ve satın alınmış
  `fiber-v2/static/assets/` varlıklarının değiştirilmesi.
- Duplicate randevu politikası, takvim çakışma politikası ve scheduled
  appointment yaşam döngüsünün bu planda kendiliğinden onaylanması.
- Denetimde doğrulanmamış production davranışını repository gerçeği gibi kabul
  etmek veya doğrulanmamış ASVS requirement numarası üretmek.

## 2. Onaylanmış ürün kararları

1. **Moderator:** Tüm şubelerde içerik ve randevu operasyonlarını yürütür;
   admin değildir. Kullanıcı rolü/aktifliği/şube izni ve hassas secret yönetemez,
   özel İK belgelerine erişemez.
2. **Santral:** Yalnız kendisine atanmış şubelerin randevu taleplerini görür ve
   yalnız onaylı durum geçişlerini yapar. Arşiv yetkisi, mevcut kullanıcı + hedef
   şube + `can_delete` ile doğrulanır. Başka panel modülüne erişemez.
3. **Public randevu:** Geçerli şube zorunludur. İstemciden gelen `sid` güven
   kararı değildir; insert öncesi sunucuda var ve aktif şube olarak doğrulanır.
   Şubesiz kayıt oluşturulmaz.
4. **İK:** İş başvuruları ve genel iletişim taleplerini yönetir. Randevu,
   kullanıcı yönetimi, secret ayarları ve genel içerik yönetimine erişmez.
5. **Özel belgeler:** İş başvurusu CV'si ve gelecekteki diploma benzeri özel İK
   belgeleri yalnız admin ve İK tarafından erişilebilir; public static kökte
   tutulmaz ve rol+nesne kontrolü yapan endpoint'ten indirilir.
6. **Header ve şube galerileri:** Ekleme, düzenleme, sıralama ve silme yalnız
   admin ve moderator içindir. Santral ve İK erişemez.
7. **Kullanıcı yönetimi:** Rol, aktiflik ve şube izinleri yalnız admin tarafından
   değiştirilir. Kullanıcı yalnız kendi temel profilini ve parolasını değiştirir;
   kendi rolünü, aktifliğini veya şube izinlerini değiştiremez. İstemci kontrollü
   eski değerler güvenlik kararı değildir.
8. **Randevu durumu:** İstemci kontrollü döngü kaldırılır. Sunucu kontrollü
   geçişler ve her değişiklik için eski/yeni durum, aktör, zaman ve mümkünse
   açıklama tutulur. Bölüm 6'daki matris öneridir ve kodlama öncesi ayrıca
   kullanıcı onayı gerektirir.
9. **Kayıt silme:** Randevu, iletişim ve İK günlük silme işlemleri arşivlemedir.
   Aktör ve zaman tutulur. Kalıcı silme yalnız admin bakım işlemidir; saklama
   süresi `TBD`'dir. Randevu bağlamındaki `can_delete`, kullanıcı arayüzünde ve
   policy'de “arşivleyebilme” anlamına çevrilir.

## 3. Rol ve endpoint yetki matrisi

Hedef politikayı gösterir; mevcut kodun yetkili olduğu anlamına gelmez.
`A` izinli, `K` yalnız kendi nesnesi, `Ş` atanmış hedef şubeyle sınırlı,
`—` reddedilir. Her izin endpoint'te uygulanır; menü görünürlüğü güvenlik
sınırı değildir.

| Endpoint/işlem ailesi | Public | Admin | Moderator | Santral | İK | Zorunlu nesne kuralı |
| --- | --- | --- | --- | --- | --- | --- |
| `POST /backend/add-randevu-request` | A | A | A | A | A | `sid` zorunlu; şube var+aktif olmalı |
| Panel ana giriş `GET /panel/` | — | A | A | A | A | Yalnız role uygun özet/veri; çapraz modül verisi yok |
| Randevu talebi liste/detay/export/latest API | — | A | A | Ş | — | Santral: mevcut UID + hedef `sid` + atanmış/`can_view` |
| Randevu talebi durum geçişi | — | A | A | Ş | — | Sunucu geçiş matrisi + hedef nesne + beklenen mevcut durum |
| Randevu talebi arşivleme | — | A | A | Ş | — | Santral ayrıca aynı UID+`sid` satırında `can_delete=true` |
| Scheduled appointment `/panel/randevular*`, `/backend/randevu*`, `/backend/add-randevu` | — | A | A | — | — | Moderator tüm şubeler; hedef nesne sunucuda yüklenir |
| İletişim talepleri paneli ve backend işlemleri | — | A | — | — | A | Kayıt varlığı; arşiv dahil aktör kaydı |
| İş başvuruları paneli ve backend işlemleri | — | A | — | — | A | Kayıt varlığı; arşiv dahil aktör kaydı |
| Özel İK belge indirme endpoint'i | — | A | — | — | A | Attachment, başvuruya bağlı olmalı; fiziksel yol istemciden alınmaz |
| Genel içerik CRUD: şube, kurum, uzmanlık, branş, doktor, tıbbi birim, ana/özel içerik, haber, tetkik | — | A | A | — | — | Route ve mutation düzeyinde rol; gerekli yerde hedef nesne |
| Header sayfaları ve `add/edit/delete/change-order` mutation'ları | — | A | A | — | — | Hedef `hbid` sunucuda çözülür; istemci eski değeri karar değildir |
| Şube galerisi görüntüleme/yönetim mutation'ları | — | A | A | — | — | URL `sid`, galeri kaydının gerçek `sid`'siyle eşleşmeli |
| Genel içerik medyası ekleme/silme | — | A | A | — | — | DB medya kimliği + yetkili kök + sahiplik; fiziksel path alınmaz |
| Kullanıcı listeleme/ekleme/başka kullanıcı düzenleme/silme/aktiflik | — | A | — | — | — | Aktör güncel DB rolü admin; self-target kısıtları ayrıca uygulanır |
| Kendi temel profilini düzenleme | — | K | K | K | K | Route UID aktör UID olmalı; dar DTO; rol/aktiflik/şube alanı yok |
| Kendi parolasını değiştirme | — | K | K | K | K | Route UID aktör UID; parola doğrulama/session politikası |
| Şube izinlerini (`can_view`, `can_delete`) değiştirme | — | A | — | — | — | Hedef UID+SID sunucuda doğrulanır; audit olayı yazılır |
| Ayar/secret sayfaları ve option mutation'ları | — | A | — | — | — | Mevcut karma endpoint'ler admin-only; secret hiçbir read DTO'da yok |
| `GET /backend/notifications` WebSocket | — | A | A | Ş | A | Event allowlist; alıcı kapsamı rol/şube/nesneye göre sunucuda belirlenir |

Notlar:

- Public form endpoint'inin panel rolleri tarafından teknik olarak çağrılabilmesi,
  panel yetkisi değildir; endpoint anonim form sözleşmesiyle aynı doğrulamayı
  uygular.
- Santral için “atanmış şube” kaynağı hedef mimaride
  `user_branch_permissions` satırıdır. `users.sid`, JWT'deki `sid` veya hedef
  şubede herhangi bir santral kullanıcısının bulunması yetki kanıtı değildir.
- `can_view`, şube üyeliği/okuma kapsamını; mevcut `can_delete` ise yalnız
  arşivleyebilme yetkisini temsil eder. Kalıcı silme yetkisi vermez.
- Current options endpoint'leri secret ve secret olmayan ayarları karıştırdığı
  için ayrıştırılana kadar tamamı admin-only olmalıdır. Moderatorün gelecekte
  secret olmayan ayarlara erişip erişmeyeceği Bölüm 12'de `TBD`'dir.

## 4. Nesne/şube yetkilendirme kuralları

1. **Güncel aktör:** JWT yalnız kimlik taşıyıcısıdır. Her korumalı request'te
   kullanıcı DB'den yüklenir; varlık, aktiflik ve güncel rol doğrulanır. Silinmiş,
   pasif veya çözülemeyen aktör için request durdurulur; response cookie silmek
   tek başına yeterli değildir.
2. **Hedefi önce çöz:** Mutation'da route/body ID'siyle hedef kayıt ve gerçek
   `sid` sunucuda yüklenir. İstemcinin `sid`, rol, eski durum veya eski alan
   değerleri authorization girdisi değildir.
3. **Tek sorgusal bağ:** Santral randevu okuma/değiştirme kararı
   `actor_uid + target_request_id + target_sid + required_permission` bağını
   aynı yetki değerlendirmesinde kurar. “Kullanıcının herhangi bir şubesi var”
   ve “hedef şubede herhangi bir santral var” kontrolleri kabul edilmez.
4. **Moderator kapsamı:** İçerik ve randevu hedeflerinde global şube kapsamı
   vardır; admin-only kullanıcı, secret ve özel İK alanlarına bu kapsam taşınmaz.
5. **İK kapsamı:** İletişim ve iş başvurusu kayıtları ile bu başvuruların özel
   attachment'ları. Başka içerik veya randevu nesnesine rol geçişi yapılamaz.
6. **Self-service:** `uid` aktörden türetilir veya route UID ile kesin eşleştirilir.
   Temel profil DTO'sunda `role`, `is_active`, `sid`, `can_view`, `can_delete`
   bulunmaz. Admin DTO'su ayrıdır ve eski değerleri istemciden güven kaynağı
   olarak almaz.
7. **Fail closed:** Yetki sorgusu/aktör yenileme hatasında hassas işlem ilerlemez.
   `404` ile varlık gizleme ve `403` kullanım standardı endpoint ailelerinde
   tutarlılaştırılır; karar kullanıcı rolüne göre veri sızdırmamalıdır.
8. **Liste-detay-mutation eşliği:** Aynı nesne kapsamı liste, detail, export,
   polling, notification ve mutation'da paylaşılır. Bir kayıt listede gizliyken
   doğrudan URL/API ile açılamaz.
9. **TOCTOU önleme:** Yetki ve mutation mümkünse aynı transaction içinde;
   durum değişimi `WHERE id=? AND status=? AND archived_at IS NULL` benzeri
   compare-and-set davranışıyla yapılır. Sıfır satır güncellenmesi stale/denied
   sonuç olarak ele alınır.
10. **Audit:** Rol/aktiflik/şube izni, durum, arşiv, restore, private belge
    indirme ve secret rotasyonu için aktör UID, zaman, hedef tür/ID ve sonuç
    kaydedilir. Secret değeri audit/log içine yazılmaz.

## 5. Randevu talebi yaşam döngüsü

1. **Form verisi alınır:** Ad, soyad, telefon ve `sid` zorunlu doğrulanır;
   mevcut captcha/duplicate davranışı bu paket tarafından kendiliğinden
   değiştirilmez.
2. **Şube doğrulanır:** `sid`, `subeler` içinde bulunan ve aktif bir kayda
   çözülür. Eksik, parse edilemeyen, pasif veya bulunmayan şube için insert
   yapılmadan kontrollü `4xx` döner.
3. **Kayıt oluşturulur:** Talep doğrulanmış `sid` ve başlangıç durumu `yeni`
   ile tek kez yazılır. Başarılı yanıt, kayıt gerçekten kalıcı olduktan sonra
   verilir. Şubesiz legacy kayıt üretimine devam edilmez.
4. **Operasyon kuyruğu:** Admin/moderator tüm aktif talepleri; santral yalnız
   atanmış şubeleri görür. Liste, sayaç, detail, export, polling ve notification
   aynı arşiv/şube predicate'ini kullanır.
5. **Durum değişimi:** İstemci yalnız hedef durum, beklediği mevcut durum ve
   opsiyonel açıklama gönderir. Sunucu DB'deki mevcut durumu yükler, Bölüm 6
   matrisini ve rol/şube kapsamını uygular, ana kayıt ile history satırını aynı
   transaction'da yazar.
6. **Scheduled appointment:** `randevu-verildi` ile ayrı `randevular` kaydı
   arasındaki atomiklik/senkronizasyon henüz onaylanmış değildir. Bu bağ Bölüm
   12'de `TBD` kalır; geçiş implementasyonu bu kararı varsaymaz.
7. **Arşiv:** Günlük “sil” işlemi ana kaydı fiziksel olarak kaldırmaz;
   `archived_at`, `archived_by_uid` ve varsa açıklama yazar. Santral için
   `can_delete` aynı hedef şubede doğrulanır.
8. **Kalıcı silme:** Normal panel akışında yoktur. Yalnız admin bakım süreci,
   onaylı saklama süresi, backup/restore ve audit gereksinimleri tamamlandıktan
   sonra ayrı araç/işlem olarak tasarlanır.

## 6. Önerilen durum geçiş matrisi

> **UYGULAMA ÖNCESİ KULLANICI ONAYI GEREKLİDİR.** Aşağıdaki matris mevcut durum
> adlarını (`yeni`, `hasta-arandi`, `ulasilamadi`, `randevu-verildi`,
> `randevu-verilemedi`, `gelmedi`, `hasta-vazgecti`) koruyan bir öneridir;
> onaylanmış iş kuralı değildir.

| Mevcut durum | Önerilen hedefler | Standart işlemi yapabilen rol | Açıklama kuralı |
| --- | --- | --- | --- |
| `yeni` | `hasta-arandi`, `ulasilamadi`, `randevu-verildi`, `randevu-verilemedi`, `hasta-vazgecti` | Admin, moderator; atanmış şubede santral | `randevu-verilemedi` ve `hasta-vazgecti` için önerilen zorunlu |
| `hasta-arandi` | `ulasilamadi`, `randevu-verildi`, `randevu-verilemedi`, `hasta-vazgecti` | Admin, moderator; atanmış şubede santral | Başarısız/sonlandıran sonuçlarda önerilen zorunlu |
| `ulasilamadi` | `hasta-arandi`, `randevu-verilemedi`, `hasta-vazgecti` | Admin, moderator; atanmış şubede santral | Yeniden arama dışında önerilen zorunlu |
| `randevu-verildi` | `gelmedi`, `hasta-vazgecti` | Admin, moderator; atanmış şubede santral | Zorunlu; scheduled appointment bağı ayrıca onaylanmalı |
| `randevu-verilemedi` | `yeni` (yeniden açma) | Admin, moderator | Zorunlu yeniden açma gerekçesi; santral yapamaz |
| `gelmedi` | `yeni` (yeniden açma) | Admin, moderator | Zorunlu yeniden açma gerekçesi; santral yapamaz |
| `hasta-vazgecti` | `yeni` (yeniden açma) | Admin, moderator | Zorunlu yeniden açma gerekçesi; santral yapamaz |

Kurallar:

- Generic `toggle-status` ve istemcinin eski duruma göre sıradaki durumu
  seçmesi kaldırılır; endpoint açık bir `target_status` kabul eder.
- İstek `expected_status` taşır; DB durumu farklıysa `409 Conflict` benzeri
  stale yanıt verilir ve hiçbir audit/history satırı yazılmaz.
- Başarılı değişiklikte history en az `request_id`, `old_status`, `new_status`,
  `actor_uid`, `changed_at`, `reason`, `source` alanlarını taşır.
- Aynı durumdan aynı duruma no-op reddedilir veya audit politikasına göre açıkça
  tanımlanır; sessizce başarılı sayılmaz.
- `gelmedi` bilgisinin randevu talebinde mi, scheduled appointment kaydında mı
  tutulacağı Bölüm 12'de onay bekler. Bu karar çıkmadan çift kayıt güncellemesi
  yapılmaz.

## 7. Özel belge saklama ve indirme modeli

- İş başvurusu CV/diploma attachment'ları `fiber-v2/static/` ve doğrudan static
  route'ların dışında, runtime tarafından yapılandırılan private bir kökte
  saklanır. Repository içine gerçek belge konmaz.
- DB, kullanıcı kontrollü fiziksel yolu değil; opaque attachment ID, storage
  key, bağlı nesne türü/ID, güvenli sunucu üretimli dosya adı, doğrulanmış medya
  türü, boyut, checksum, oluşturma zamanı ve oluşturan akışı tutar.
- Önerilen indirme yüzeyi
  `GET /backend/job-applications/:jaid/attachments/:attachment_id/download`
  biçimindedir. Route adı uygulamada ayrıca gözden geçirilebilir; güvenlik
  sözleşmesi değişmez.
- Endpoint sırasıyla güncel kullanıcıyı, admin/İK rolünü, başvurunun varlığını,
  arşiv politikasını ve attachment'ın tam olarak bu başvuruya bağlı olduğunu
  doğrular. İstemciden filesystem yolu veya dosya adıyla çözüm yapılmaz.
- Yanıt doğrulanmış `Content-Type`, güvenli `Content-Disposition`,
  `X-Content-Type-Options: nosniff` ve hassas içeriğe uygun cache politikası
  kullanır. Inline gösterim tür bazında allowlist gerektirir; varsayılan indirme
  attachment'tır.
- Upload boyut, uzantı, magic-byte/MIME uyumu ve aktif içerik açısından
  doğrulanır. Orijinal ad yalnız kullanıcıya gösterilecek metadata olabilir;
  storage key olamaz.
- Başarılı ve reddedilen indirme denemeleri hedef attachment/başvuru, aktör,
  zaman ve sonuçla log/audit edilir; belge içeriği ve fiziksel yol loglanmaz.
- Mevcut `static/files/job-applications` dosyalarının taşınması, URL'lerin
  kapatılması ve orphan temizliği ancak production envanteri/backup planı
  onaylandıktan sonra ayrı migration/runbook paketidir.

## 8. Soft-delete, audit trail ve kalıcı silme modeli

### Günlük arşiv modeli

Randevu talebi, iletişim talebi ve iş başvurusu için önerilen ortak kavramlar:

- `archived_at`: arşiv zamanı; `NULL` aktif kaydı ifade eder.
- `archived_by_uid`: işlemi yapan kullanıcı.
- `archive_reason`: opsiyonel/role göre zorunlu açıklama.
- Aktif liste/detail/export/polling/notification sorguları varsayılan olarak
  arşivli kayıtları dışlar.
- Arşiv endpoint'i idempotency politikasını açıkça uygular; zaten arşivli kayıt
  ikinci kez fiziksel silinmez.
- Arşivli kayıt restore yetkisi ve reason kuralı Bölüm 12'de onay bekler.

### Audit ve history

- Randevu durum değişiklikleri için append-only durum history kaydı gerekir.
- Çapraz modül audit olayı en az aktör, hedef tür/ID, eylem, zaman, sonuç ve
  açıklama/metadata taşır. Rol/izin değişimi, arşiv/restore, private download ve
  secret rotasyonu kapsanır.
- Audit satırları uygulama ekranından sıradan CRUD ile değiştirilemez/silinemez.
- Hasta/başvuran kişisel verisi, secret, dosya içeriği ve fiziksel path audit
  metadata'sına kopyalanmaz.
- Ana mutation başarısızsa başarı audit'i yazılmaz; güvenlik reddi için ayrı,
  veri minimizasyonlu olay politikası uygulanabilir.

### `can_delete` dönüşümü

- Randevu bağlamında mevcut DB alanı geçiş sürecinde uyumluluk için kalabilir,
  ancak policy/helper/UI etiketi `can_archive` anlamını kullanır.
- Alanı fiziksel olarak yeniden adlandırma kararı production şema kanıtı ve
  incremental migration planından sonra verilir. Ad değişikliği olmadan da
  kalıcı silme yetkisi vermediği sunucu policy'sinde garanti edilir.

### Kalıcı silme

- Yalnız admin bakım işlemi; normal kayıt ekranlarında bulunmaz.
- Saklama süresi, hukuki/operasyonel dayanak, legal hold, backup'tan düşme,
  attachment temizliği ve geri dönüş penceresi `TBD`'dir.
- Purge; önce dry-run envanteri, açık hedef listesi, onay, ölçülebilir sonuç ve
  audit gerektirir. Mevcut `schema.sql` veya migration betikleri purge aracı
  değildir.

## 9. Secret ve ayar yönetimi

1. Secret alanları read sorgularında `o.*` ile çekilmez; açık allowlist DTO
   kullanılır. Template modeline, JSON'a, hidden/password input'a veya log'a
   mevcut secret değeri konmaz.
2. Mevcut karışık ayar sayfaları ve `/backend/option*` mutation'ları ayrıştırma
   tamamlanana kadar admin-only olur. Menü gizleme yeterli değildir.
3. Secret güncelleme “write-only replace” davranışıdır: response mevcut değeri
   dönmez. Boş alanın “değiştirme” mi “sil” mi olduğu ayrı açık komutla ayırt
   edilir; istemci eski secret'ı geri göndermez.
4. Secret rotasyonu aktör, hedef ayar ve zamanla audit edilir; eski/yeni değer,
   credential parçası veya hash audit/log içine alınmaz.
5. Public akışların captcha/SMTP kullanması secret okuma yetkisi değildir;
   yalnız sunucu içi dar kullanım servisidir. Handler/template DTO'suna taşınmaz.
6. Moderator için secret olmayan site ayarlarının ayrı bir DTO/endpoint ile
   açılıp açılmayacağı kullanıcı kararı bekler. Karar çıkana kadar moderator
   options alanına erişemez.

## 10. Bağımlılık sırasına göre uygulama dalgaları

Her dalga içindeki `WP` çalışma paketleri ayrı, bağımsız review/commit adayıdır.
Bir dalga tek commit anlamına gelmez. Paket başına mümkünse 1–3 ilişkili bulgu
sınırı korunur. `OPS-001` ve `DATA-001` kapanmadan hiçbir dalga production'a
çıkamaz; şema gerektiren paketler tasarım aşamasından uygulamaya geçemez.

FAZ 3B'nin onaylanan veri katmanı ve notification hub geçişi,
[Owned Data Layer Migration Plan](OWNED_DATA_LAYER_MIGRATION_PLAN.md) içindedir.
`REL-001A`, migration kodlamasından önceki kritik dar görev olarak kalır;
`N01–N11` programı ancak `REL-001A` kabul/test kapısı geçildikten sonra başlar.
Mevcut dalga sırası ve güvenlik kapıları korunur. Geçişin release/build kabulü
için 21 modülün doğrulanması, offline/boş-cache build/test kanıtı ve aktif
kaynak/import/manifest/checksum/build çıktısı/binary metadata'da sıfır eski
bağımlılık referansı gerekir; tarihsel belge istisnası yeni planda tanımlıdır.
Bu ek, migration kodlaması veya deployment yürütme onayı değildir.

### İçindekiler — uygulama dalgaları

| Dalga | Başlık | Zorunlu sıra notu |
| --- | --- | --- |
| 0 | Deployment ve veri sözleşmesi kapıları | Tüm release/migration'lar için kapı |
| 1 | Zorunlu ilk güvenlik paketleri ve güncel kullanıcı sınırı | Önce `SEC-001A`, hemen ardından `REL-001A`; ikisi bitmeden geniş güvenlik dalgaları veya release yok |
| 2 | Endpoint rol sınırları, secret izolasyonu ve içerik yönetimi | Dalga 1 geçiş kapısından sonra |
| 3 | Dosya/media sınırları ve özel İK belgeleri | Dalga 2 rol sınırlarından sonra |
| 4 | Public randevu şubesi ve nesneye bağlı operasyon yetkisi | Dalga 1 principal/izin sınırından sonra |
| 5 | Randevu durum makinesi, history ve arşiv | Durum matrisi ve veri kapısı onayından sonra |
| 6 | İK ve iletişim iş akışı, arşiv ve pagination güvenliği | Rol ve private attachment sınırlarından sonra |
| 7 | Notification güvenliği ve public yardımcı sınırları | Rol/şube policy helper'larından sonra |
| 8 | E-posta teslimat dayanıklılığı (`REL-001B`) | `REL-001A` sonrası iyileştirme; `REL-001A` için önkoşul değildir |

### Dalga 0 — Deployment ve veri sözleşmesi kapıları

- **Kapsanan bulgular:** `OPS-001`, `DATA-001`.
- **Muhtemel alanlar:** `fiber-v2/schema.sql`, `fiber-v2/migrate.sh`,
  `fiber-v2/production-migrate.sh`, sorgu/model sözleşmeleri ve ayrı migration
  tasarımları. Bu plan bu dosyaları değiştirmez veya çalıştırmaz.
- **Önkoşullar:** Onaylı, disposable olmayan DB'ye dokunmayan inceleme;
  production şema envanteri için ayrıca açık erişim onayı; backup/restore sahibi.
- **Kabul kriterleri:** Fresh install ve upgrade girişleri ayrılmış; destructive
  yol production'da fail-closed; repository query beklentileri ile onaylı
  non-production şema farkı kayıtlı; incremental migration sırası ve rollback
  taslağı review edilmiş.
- **Gerekli test türleri:** Betik refusal-path testleri; destructive SQL
  çalıştırmadan static/shell kontrolleri; yalnız onaylı disposable DB'de clean
  install ve incremental upgrade sözleşme testleri.
- **Geri dönüş:** Guard/migration tasarım commit'lerini bağımsız geri alma;
  uygulanmış migration için ileri-düzeltme veya önceden test edilmiş restore.
  `schema.sql` ile geri dönüş yapılmaz.
- **Geçiş kapısı:** `OPS-001` ve `DATA-001` deployment gate olarak kapanmadan
  hiçbir release/migration yoktur. DB'siz yerel güvenlik paketleri geliştirilebilir
  fakat deploy edilemez.

### Dalga 1 — Zorunlu ilk güvenlik paketleri ve güncel kullanıcı sınırı

- **Kapsanan bulgular:** `SEC-001` (`SEC-001A` ilk paket), `REL-001`
  (`REL-001A` ikinci zorunlu paket), ardından `SEC-003` ve kalan kullanıcı
  güven sınırı işi.
- **Zorunlu paket sırası:**
  1. `WP-01 / SEC-001A`: şube izin yazımını admin sınırına al.
  2. `WP-02 / REL-001A`: e-posta hata yollarında proses sonlandırmayı kaldır.
  3. İlk iki paket kapandıktan sonra `SEC-003` principal yenileme ve kalan
     kullanıcı sınırı paketlerine geç.
- **Muhtemel alanlar:** `fiber-v2/main/main.go`, `fiber-v2/lib/lib.go`,
  `fiber-v2/controllers/post/users/users.go`, `fiber-v2/baserouter/baserouter.go`,
  `SendEmail` çağıran randevu/iletişim/İK handler'ları, kullanıcı panel
  template/JS'leri ve dar test dosyaları.
- **Önkoşullar:** `SEC-001A` için self-service alan allowlist'i ve admin yönetim
  sınırı; `REL-001A` için mevcut caller'ların DB insert/commit ile e-posta çağrı
  sırasının salt okunur belgelenmesi ve production SMTP kullanmayan kontrollü
  test yaklaşımı. Retry, idempotency, queue/outbox, gözlemlenebilirlik ve teslimat
  takibi `REL-001A` önkoşulu değildir.
- **Kabul kriterleri:** `SEC-001A` sonunda non-admin şube izni yazamaz.
  `REL-001A` sonunda `SendEmail` ve ilgili alt hata yolları hiçbir durumda
  `log.Fatal`, `log.Fatalf`, `os.Exit` veya panic ile prosesi sonlandırmaz ve
  caller'a normal `error` döner. Her caller'ın partial-success sözleşmesi açıkça
  belirlenir: DB kaydı tamamlandıktan sonra e-posta başarısızsa kullanıcıya
  yanlışlıkla “kayıt oluşmadı” sonucu verilmez; kalıcı kayıt sonucu ile e-posta
  teslimat sonucu ayrı ele alınır. Paket queue/outbox veya geniş mimari refactor
  içermez. Sonraki `SEC-003` işinde silinmiş/pasif/çözülemeyen kullanıcı aynı
  request'te durdurulur, güncel DB rolü kullanılır ve istemci eski değerleri
  güvenlik kararı olmaz.
- **Gerekli test türleri:** `SEC-001A` rol/self-edit allow/deny testleri;
  `REL-001A` için fake SMTP başarı ve bağlantı/hata senaryoları, hatalı MIME veya
  eksik attachment hata yolu, subprocess/process-survival regresyonu ve DB
  kaydı tamamlanmış caller'larda doğru partial-success sonucu; `SEC-003` için
  aktif/pasif/silinmiş kullanıcı ve DB lookup hata yolu.
- **Geri dönüş:** `SEC-001A`, `REL-001A`, `SEC-003` ve kalan kullanıcı sınırı
  işleri ayrı commit'lerdir. `REL-001A` library error-return ile gerekli caller
  uyarlamalarını birlikte uyumlu tutar; yalnız bir tarafın eski sürümüne dönülmez.
  Güvenlik/proses-sonlandırma riskini yeniden açan geri dönüş release'i durdurur.
- **Geçiş kapısı:** `SEC-001A` ve `REL-001A` tamamlanıp hedefli testleri geçmeden
  Dalga 2 veya sonraki geniş güvenlik dalgalarına başlanmaz ve hiçbir release
  yapılmaz. Sonraki rol matrisleri ayrıca güncel, aktif aktöre dayanmalıdır.

### Dalga 2 — Endpoint rol sınırları, secret izolasyonu ve içerik yönetimi

- **Kapsanan bulgular:** `SEC-002`, `SEC-008`.
- **Muhtemel alanlar:** `fiber-v2/baserouter/baserouter.go`,
  `fiber-v2/controllers/panel/panel.go`,
  `fiber-v2/controllers/post/options/`,
  `fiber-v2/controllers/post/headerbuttons/headerbuttons.go`,
  `fiber-v2/static/html/views/panel/secenek-sayfalari/` ve header/galeri panel
  template/JS'leri.
- **Önkoşullar:** Bölüm 3 matrisi kabul edilmiş; Dalga 1 güncel principal sınırı
  hazır; moderator non-secret settings kararı çıkmadıysa fail-closed admin-only.
- **Kabul kriterleri:** Settings read/write ve secret response yalnız admin
  sınırında ve secret hiçbir response'ta yok; header ve galeri mutation'ları
  yalnız admin/moderator; santral/İK doğrudan URL/API ile reddedilir; galeri URL
  `sid` ile gerçek nesne `sid` eşleşir.
- **Gerekli test türleri:** Her endpoint ailesi için dört rol allow/deny matrisi;
  doğrudan URL testleri; rendered HTML/JSON'da secret alanı/değeri bulunmama;
  farklı `sid`/`sgid` negatif nesne testi.
- **Geri dönüş:** Settings-secret ve header/galeri yetki paketleri ayrı
  commit'lerdir. UI menü değişimi backend sınırıyla aynı pakette geri alınır;
  backend olmadan menü değişimi güvenlik düzeltmesi sayılmaz.
- **Geçiş kapısı:** Admin/moderator içerik pozitif yolu ve santral/İK negatif
  yolu kanıtlanmış; secret read yüzeyi sıfırlanmış olmalı.

### Dalga 3 — Dosya/media sınırları ve özel İK belgeleri

- **Kapsanan bulgular:** `SEC-004`, `SEC-005`, `SEC-006`.
- **Muhtemel alanlar:** `fiber-v2/controllers/post/post.go`,
  `fiber-v2/controllers/panel/panel.go`, `fiber-v2/lib/lib.go`,
  `fiber-v2/main/main.go`, `fiber-v2/models/models.go`, iş başvurusu ve galeri
  template/JS'leri; gelecekteki incremental migration dosyaları.
- **Önkoşullar:** Dalga 2 rol sınırları; private storage kökü/izinleri için
  runtime kararı; legacy dosya envanteri; şema işi için Dalga 0 veri kapısı.
- **Kabul kriterleri:** Custom delete fiziksel path değil DB ID alır ve canonical
  containment uygular; galeri upload admin/moderator ve güvenli tür/boyut/içerik
  allowlist'iyle çalışır; İK attachment'ları static dışındadır ve yalnız
  admin/İK indirme endpoint'iyle erişilir; eski public URL yeni upload üretmez.
- **Gerekli test türleri:** Geçerli dosya, traversal, absolute path, separator ve
  symlink fixture testleri; MIME/magic-byte/uzantı uyuşmazlığı; aktif HTML/SVG
  reddi; admin/İK indirme pozitif, moderator/santral/public negatif; IDOR testi.
- **Geri dönüş:** Containment, gallery validation ve private attachment ayrı
  paketlerdir. Storage migration geri dönüşü metadata+dosya kopyasının doğrulanmış
  planını gerektirir; eski public sunuma sessiz geri dönüş yapılmaz.
- **Geçiş kapısı:** Public static kökte yeni özel İK belgesi üretilememeli;
  legacy taşıma runbook'u onaylanmadan production geçişi yoktur.

### Dalga 4 — Public randevu şubesi ve nesneye bağlı operasyon yetkisi

- **Kapsanan bulgular:** `FLOW-002`, `FLOW-003`.
- **Muhtemel alanlar:**
  `fiber-v2/controllers/post/randevular/randevular.go`,
  `fiber-v2/controllers/panel/panel.go`, `fiber-v2/models/models.go`, public
  randevu template/JS'leri ve panel liste/detail/export/polling sorguları.
- **Önkoşullar:** Dalga 1 principal ve şube izin sınırı; aktif şube tanımı;
  legacy NULL/invalid `sid` production envanteri `TBD` olarak ayrılmış olmalı.
- **Kabul kriterleri:** Eksik/pasif/geçersiz şube insert öncesi reddedilir;
  oluşturulan her yeni talep geçerli şubeye bağlıdır; santral tüm read/mutation
  yüzeylerinde aynı UID+hedef SID kapsamında kalır; moderator globaldir; İK
  randevuya erişemez; `can_delete` yalnız arşiv yetkisi anlamındadır.
- **Gerekli test türleri:** Public valid/invalid/missing/inactive şube;
  admin/moderator global allow; santral atanmış allow ve atanmamış deny;
  İK deny; liste/detail/export/latest/notification kapsam eşliği; concurrent
  permission change sınırı.
- **Geri dönüş:** Public doğrulama ile panel authorization ayrı commit'lerdir.
  Geri dönüş şubesiz kayıt riskini açarsa release durur; veri rollback'i yoktur.
- **Geçiş kapısı:** Yeni şubesiz kayıt üretilemediği ve santral IDOR/cross-branch
  negatif testleri geçtiği kanıtlanmalı.

### Dalga 5 — Randevu durum makinesi, history ve arşiv

- **Kapsanan bulgular:** Onaylı ürün kararları 8–9; legacy backlog eşlemesi
  `F13`; `FLOW-003` mutation kapsamının devamı.
- **Muhtemel alanlar:**
  `fiber-v2/controllers/post/randevular/randevular.go`,
  `fiber-v2/controllers/panel/panel.go`, `fiber-v2/models/models.go`, randevu
  panel template/JS'leri ve onaylanacak incremental migration'lar.
- **Önkoşullar:** Bölüm 6 matrisi kullanıcı tarafından onaylanmış; Dalga 0 veri
  kapıları kapanmış; scheduled appointment senkronizasyon kararı kaydedilmiş.
- **Kabul kriterleri:** Generic cycle yok; sunucu allowlist geçişi ve
  compare-and-set var; ana durum+append-only history atomik; eski/yeni/aktör/zaman
  ve gerekli reason tutulur; hard delete yerine arşiv uygulanır; tüm read
  yüzeyleri arşiv predicate'inde tutarlıdır.
- **Gerekli test türleri:** Her izinli/yasak geçiş; stale/double submit;
  transaction rollback; actor/reason/history doğruluğu; santral şube matrisi;
  archive idempotency ve list/detail/export/polling görünürlüğü.
- **Geri dönüş:** Handler/UI geçişi ile migration ayrı adımlardır. Migration
  sonrası kod geri dönüşü uyumlu çift-okuma/feature flag planı gerektirir;
  history verisi silinmez.
- **Geçiş kapısı:** Onaylı matrisin tüm kenarları test edilmiş, audit/history
  kaybı olmadan rollback yaklaşımı prova edilmiş olmalı.

### Dalga 6 — İK ve iletişim iş akışı, arşiv ve pagination güvenliği

- **Kapsanan bulgular:** `FORM-001`, `REL-002`; ürün kararı 4 ve 9'un
  iletişim/İK kısmı.
- **Muhtemel alanlar:** `fiber-v2/controllers/post/post.go`,
  `fiber-v2/controllers/panel/panel.go`, İK/iletişim template ve JS'leri,
  `fiber-v2/models/models.go` ve onaylanacak incremental migration'lar.
- **Önkoşullar:** Dalga 2 rol matrisi ve Dalga 3 private attachment modeli;
  şema işleri için Dalga 0; archive/restore policy kararı.
- **Kabul kriterleri:** Yalnız admin/İK erişir; İK formunun tek submit sahibi
  vardır; server-side dosya doğrulama insert öncesidir ve kısmi kayıt/dosya
  bırakmaz; pagination sıkı parse/range ve DB pagination uygular; hard delete
  yerine aktör+zamanlı arşiv vardır.
- **Gerekli test türleri:** Admin/İK allow, moderator/santral deny; duplicate
  listener/network request kontrolü; dosyasız/geçersiz dosyalı başvuru rollback;
  `page`/`per_page` sınır ve kötü girdi testleri; iletişim ve İK arşiv görünürlüğü.
- **Geri dönüş:** Form listener, pagination ve her kayıt türünün arşivi ayrı
  paketlerdir. Attachment/DB tutarlılığı korunmadan eski hard-delete'a dönülmez.
- **Geçiş kapısı:** Yetkisiz özel veri görünürlüğü yok; invalid pagination panic
  üretmiyor; arşivli kayıt normal operasyon yüzeylerinden tutarlı biçimde çıkıyor.

### Dalga 7 — Notification güvenliği ve public yardımcı sınırları

- **Kapsanan bulgular:** `SEC-007`, `FLOW-001`.
- **Muhtemel alanlar:** `fiber-v2/controllers/post/post.go`,
  `fiber-v2/baserouter/baserouter.go`, `fiber-v2/static/js/panel/notifications.js`
  ve ilgili public JS dosyaları.
- **Önkoşullar:** Rol/şube policy helper'ları ve event alıcı matrisi; public read
  verisinin açık allowlist'i.
- **Kabul kriterleri:** Client UID/rol/event metni güven kaynağı değildir;
  event türü ve hedef alıcı sunucuda belirlenir; notification içeriği text API
  ile render edilir; santral yalnız atanmış şube olaylarını, İK yalnız
  iletişim/başvuru olaylarını alır; public UI panel JWT/WS endpoint'ine bağlı
  değildir.
- **Gerekli test türleri:** Hostile payload'ın metin olarak render edilmesi;
  anonymous/yanlış rol/yanlış şube event deny; recipient izolasyonu; anonim
  public sayfa entegrasyon testi; reconnect davranışı.
- **Geri dönüş:** Safe rendering, server event authorization ve public helper
  ayrıştırması ayrı commit'lerdir; unsafe `innerHTML` davranışına geri dönülmez.
- **Geçiş kapısı:** Cross-user payload yürütme yolu kapanmış ve public temel
  akış panel oturumu olmadan çalışıyor olmalı.

### Dalga 8 — E-posta teslimat dayanıklılığı (`REL-001B`)

Bu dalga prosesin hayatta kalması düzeltmesi değildir; o zorunlu düzeltme
`REL-001A` olarak Dalga 1'de tamamlanır. `REL-001B`, `REL-001A` için önkoşul
değildir ve eksikliği `REL-001A`nın erken uygulanmasını geciktiremez.

- **Kapsanan bulgular:** `REL-001B` — `REL-001A` sonrası teslimat dayanıklılığı
  ve operasyonel izlenebilirlik iyileştirmeleri.
- **Muhtemel alanlar:** E-posta çağrı sınırı, onaylanırsa retry/idempotency ve
  queue/outbox bileşenleri, teslimat durum modeli, gözlemlenebilirlik ve ilgili
  kontrollü test altyapısı. Somut mimari ayrıca onaylanmadan seçilmez.
- **Önkoşullar:** `REL-001A` tamamlanmış olmalı; teslimat garantisi, retry
  sayısı/backoff, idempotency anahtarı, kalıcı queue/outbox seçimi ve operasyonel
  sahiplik ayrıca kararlaştırılmalı. Bu kararların hiçbiri `REL-001A` önkoşulu
  değildir.
- **Kabul kriterleri:** Onaylanan kapsamda retry aynı mesajı kontrolsüz çoğaltmaz;
  idempotency duplicate teslimatı sınırlar; queue/outbox kayıt ile teslimat
  durumunu izlenebilir kılar; kalıcı başarısızlık ve yeniden deneme sonucu
  secret/kişisel veri sızdırmadan ölçülebilir. Caller'ın Dalga 1'de tanımlanan
  partial-success sözleşmesi korunur.
- **Gerekli test türleri:** Fake SMTP ile geçici/kalıcı hata ayrımı, retry ve
  backoff, duplicate/idempotency, worker/process yeniden başlatma, queue/outbox
  recovery ve teslimat durumu/gözlemlenebilirlik testleri. Production SMTP
  kullanılmaz.
- **Geri dönüş:** Retry, idempotency, queue/outbox, gözlemlenebilirlik ve teslimat
  takibi bağımsız küçük paketlere ayrılır. Geri dönüş `REL-001A`nın normal error
  dönüşünü veya process-survival güvencesini geri alamaz.
- **Geçiş kapısı:** Bu dalganın seçilen özelliği production'a alınacaksa ilgili
  dayanıklılık ve recovery testleri geçmelidir. Ancak `REL-001B`, Dalga 1'deki
  `REL-001A` process-survival release kapısının önkoşulu değildir.

## 11. Her dalga için zorunlu içerik ve çalışma paketi kuralları

Bölüm 10'daki her dalga; kapsanan bulgular, değişecek muhtemel alanlar,
önkoşullar, kabul kriterleri, gerekli test türleri, geri dönüş yaklaşımı ve
sonraki dalgaya geçiş kapısı ile tanımlanmıştır. Uygulama sırasında bu yedi alan
ilgili WP'ye özgü somut dosya/test/rollback bilgileriyle yeniden doğrulanır.

- Her WP tek güvenlik/iş akışı amacı taşır; mümkünse 1–3 ilişkili bulguyu aşmaz.
- Birbirinden bağımsız düzeltmeler aynı commit'e konmaz. Örneğin secret response,
  gallery upload ve SMTP error aynı commit olamaz.
- Her WP başlangıcında temiz çalışma ağacı, dal ve HEAD doğrulanır; sonunda ilgili
  test, `git diff`, `git diff --check` ve `git status --short` kaydedilir.
- Mevcut test kapsamı kritik akışları karşılamaz; yalnız
  `fiber-v2/test/all_test.go` içindeki dar yardımcı testi görülmüştür. Planlanan
  authorization/integration testleri henüz mevcutmuş gibi kabul edilmez; her WP
  kendi en küçük gerekli test seam/fixture'ını tanımlar.
- DB gerektiren test, ancak izole/disposable test DB'si ve güvenli config açıkça
  hazırlandıktan sonra çalışır. Gerçek kullanıcı verisi veya production DB/SMTP
  kullanılmaz.
- Şema migration'ı ayrı review/approval paketidir. Repository `schema.sql`'ının
  production ile aynı olduğu varsayılmaz ve upgrade için çalıştırılmaz.
- UI değişikliği backend policy ile birlikte doğrulanır; desktop, mobil ve
  klavye yolu etkileniyorsa ilgili erişilebilirlik kontrolü WP kabulüne eklenir.

## 12. Production/runtime bilgisi veya kullanıcı onayı gerektiren `TBD` kararlar

| TBD | Gereken karar/kanıt | Bloke ettiği alan |
| --- | --- | --- |
| Production DB şeması ve manuel migration geçmişi | Açık onaylı, değer/secret göstermeyen şema envanteri | `DATA-001`, tüm migration'lar ve deploy |
| Fresh install/upgrade/rollback operasyon sahibi | Hedef doğrulama, backup ve restore kanıtı | `OPS-001`, tüm release'ler |
| Randevu durum geçiş matrisi | Bölüm 6'nın ürün onayı veya revizyonu | Dalga 5 kodlaması |
| `gelmedi` kaydının sahibi | Talep mi scheduled appointment mı; çift yazım/senkronizasyon kuralı | `randevu-verildi` sonrası akış |
| Talep → scheduled appointment atomikliği | Aynı transaction, eventual sync veya manuel süreç kararı | Durum ve `randevular` oluşturma |
| Durum açıklaması zorunluluğu | Hangi geçişlerde reason zorunlu | API/DB validation ve UI |
| Legacy NULL/geçersiz `sid` kayıtları | Production sayısı ve karantina/atama/arşiv kararı | Yeni NOT NULL/foreign-key önerisi ve queue |
| `can_delete` fiziksel rename | Kolonu koruyup anlamı mı değiştireceğiz, incremental rename mi | Şema/UI uyumluluğu |
| Arşiv restore yetkisi | Admin-only mi, domain rolü de mi; reason/audit kuralı | Arşiv ekranları ve endpoint'ler |
| Kalıcı silme saklama süresi | Süre, legal hold, backup'tan düşme ve onay akışı | Purge; şu an `TBD` |
| Private storage runtime'ı | Yerel private disk mi object storage mı; kök, sahiplik, izin, encryption/backup | Özel belge migration/indirme |
| Legacy public İK dosyaları | Dosya/DB eşleşme envanteri, orphan'lar, taşıma ve eski URL kapatma | `SEC-006` production kapanışı |
| Reverse proxy/static header davranışı | `/files`, CSP, `nosniff`, cache ve Content-Disposition gerçek davranışı | Legacy exposure ve upload savunması |
| Moderator non-secret settings | Ayrı non-secret ayar subset'ine erişecek mi | Options DTO/route ayrıştırması |
| Session ömrü ve role-change invalidation | Token TTL, yenileme, logout/revocation standardı | `SEC-003` tam kapanışı |
| Audit retention ve erişim | Audit'i kim görür, ne kadar tutulur, export/PII politikası | Ortak audit modeli |
| Runtime process manager ve health-check | Fatal/panic sonrası gerçek davranış | `REL-001A` process-survival ve `REL-002` production doğrulaması |
| Sunucu OS/filesystem/symlink durumu | Onaylı read-only runtime kanıtı | `SEC-004` production doğrulaması |

## 13. Definition of Done

Bu Professional V2 güvenlik/iş akışı planı ancak aşağıdakiler birlikte
sağlandığında uygulanmış sayılır:

- Onaylı rol/endpoint matrisi route, handler ve hedef nesne düzeyinde uygulanmış;
  her rol için pozitif ve negatif testler vardır.
- Korumalı request'ler güncel, aktif DB kullanıcısına dayanır; stale JWT rolü,
  menü görünürlüğü ve istemci eski değerleri güvenlik kararı değildir.
- Public randevu geçerli aktif şube olmadan yazılamaz; santral liste/detail/
  export/polling/notification/mutation yüzeylerinin tamamında atanmış şubeyle
  sınırlıdır.
- Kullanıcı temel self-service ile admin güvenlik alanları ayrılmış; non-admin
  rol, aktiflik veya şube izinlerini etkileyemez.
- Secret hiçbir read DTO/template/JSON/log/audit çıktısında bulunmaz; write-only
  admin rotasyonu uygulanır.
- Header/galeri işlemleri yalnız admin/moderator; upload ve delete dosya
  sınırları allowlist+containment+nesne yetkisiyle korunur.
- Özel İK attachment'ları public static kökün dışındadır; yalnız admin/İK,
  nesneye bağlı download endpoint'i üzerinden erişir; legacy taşıma doğrulanır.
- Kullanıcı tarafından onaylanmış durum matrisi server-side çalışır; her başarılı
  geçiş append-only history üretir ve stale update atomik olarak reddedilir.
- Randevu, iletişim ve İK günlük silmeleri arşivdir; aktör/zaman tutulur;
  production hard-delete retention kararı çıkana kadar purge yoktur.
- `SEC-001A` ve `REL-001A` ilk iki zorunlu kodlama paketi olarak tamamlanmış;
  `SendEmail`/alt yollar normal `error` dönüyor, proses hayatta kalıyor ve
  DB-sonrası e-posta hatalarında partial-success sonucu doğru veriliyor olmalıdır.
  Bu ikisi tamamlanmadan geniş güvenlik dalgaları veya release yoktur.
- `REL-001B`; retry, idempotency, queue/outbox, gözlemlenebilirlik ve teslimat
  takibi için sonraki dayanıklılık paketidir; `REL-001A`nın önkoşulu veya proses
  hayatta kalma düzeltmesini geciktiren bir release bağı değildir.
- `REL-002`, `SEC-007`, `FLOW-001` dahil kalan doğrulanmış bulgular hedefli
  testlerle kapanmış veya açık risk/TBD olarak release'i bloke eder.
- `OPS-001` ve `DATA-001` kapıları kapanmış; destructive `schema.sql` upgrade
  olarak kullanılmamış; onaylı incremental migration ve geri dönüş kanıtı vardır.
- Her WP'de çalıştırılan gerçek kontroller, sınırlamalar, diff ve commit referansı
  owning backlog/roadmap/changelog'a yazılmıştır. Plan yazılması tek başına
  bulguyu çözülmüş yapmaz.
- Production doğrulaması ve deployment yalnız ayrı açık onayla yapılmıştır.

## 14. Önerilen ilk kodlama görevi

### WP-01 — `SEC-001A`: şube izin yazımını admin sınırına al

**Neden ilk:** `SEC-001`, düşük yetkili kullanıcının kendi yetki kapsamını
genişletip sonraki şube tabanlı işlemlere etki edebildiği doğrulanmış bir
yükseltme yoludur. Değişiklik tek handler çevresinde, geri alınabilir ve DB
migration gerektirmez. `OPS-001`/`DATA-001` deployment kapıları açık kalırken
yerel olarak hazırlanabilir; deploy edilemez.

**Dar kapsam:**

- `fiber-v2/controllers/post/users/users.go` içindeki `EditUser` akışında
  `user_branch_permissions` okuma/yazma bloğunu yalnız güncel admin aktöre aç.
- Non-admin self-edit için `name`, `surname`, `email`, `phone`, `timezone`
  allowlist'i uygula; permission/protected alan gönderimini reddet ve bu istekte
  hiçbir permission query/mutation çalıştırma.
- Route UID ile aktör UID eşliğini sunucuda doğrula. `role`, `is_active`, `sid`
  ve izin karşılaştırmalarında istemcinin `old_*` alanlarını güven kararı yapma.
- Admin kullanıcı yönetiminin kapsamını genişletme; endpoint ayrıştırması ve
  genel principal yenileme Dalga 1'in sonraki ayrı paketleridir.

**Muhtemel dosyalar:** Yalnız handler ve dar test için
`fiber-v2/controllers/post/users/users.go` ile aynı modülde eklenecek hedefli
bir `_test.go` dosyası. Route/template değişikliği gerekirse kapsam yeniden
gözden geçirilir; sessizce büyütülmez.

**Kabul kriterleri:**

- Moderator, santral ve İK kendi UID'siyle dahi `can_view`/`can_delete` veya
  şube ataması yazamaz.
- Permission alanı gönderen non-admin request, DB permission döngüsüne girmeden
  reddedilir; alan göndermeyen self-profile request izin satırlarını sıfırlamaz.
- Admin için mevcut şube permission yönetim yolu korunur; hedef UID+aktif SID
  doğrulanır ve hata yutulmaz.
- Temel profil allowlist'i dışındaki güvenlik alanları DB'deki mevcut değeri
  değiştiremez.

**Test yaklaşımı:** Mevcut kritik authorization test altyapısı yoktur. En küçük
pakette rol/alan policy'si için tablo testleri ve DB'ye ulaşmadan non-admin deny
handler testleri eklenmelidir. Admin pozitif DB yolu ancak açıkça hazırlanmış
izole fixture ile entegrasyon testi yapılabiliyorsa tamamlanır; production DB
kullanılmaz. Build/test bu plan görevinde çalıştırılmamıştır.

**Geri dönüş:** Tek bağımsız commit geri alınabilir; şema/config rollback'i
yoktur. Geri alma self-permission yükseltme riskini yeniden açacağından ilgili
release de geri çekilmeli veya durdurulmalıdır.

### WP-02 — `REL-001A`: e-posta hatalarında prosesin hayatta kalmasını sağla

**Sıra ve kapı:** `SEC-001A`dan hemen sonraki ikinci zorunlu kodlama görevidir.
Bu iki görev tamamlanıp hedefli testleri geçmeden sonraki geniş güvenlik
dalgalarına veya release'e geçilmez.

**Dar kapsam:**

- `fiber-v2/lib/lib.go` içindeki `SendEmail` ve çağırdığı/bağlı hata yollarında
  `log.Fatal`, `log.Fatalf`, `os.Exit` ve panic ile proses sonlandırmayı kaldır;
  hatayı caller'a normal `error` olarak döndür.
- Mevcut randevu, iletişim ve İK caller'larında DB insert/commit ile e-posta
  çağrısının sırasını kaydet ve her caller için partial-success sözleşmesini
  açıklaştır.
- DB kaydı tamamlandıktan sonra e-posta başarısız olduğunda kullanıcıya “kayıt
  oluşmadı” sonucu verme; kalıcı kayıt başarısı ile e-posta teslimat başarısını
  ayrı ve yanıltıcı olmayan sonuçlar olarak ele al.
- Queue/outbox, retry, idempotency, geniş mimari refactor, gözlemlenebilirlik ve
  teslimat takibi bu pakete eklenmez. Bunlar `REL-001B` kapsamıdır ve
  `REL-001A` önkoşulu değildir.

**Muhtemel dosyalar:** `fiber-v2/lib/lib.go`, yalnız davranışı uyarlaması gereken
mevcut caller handler'ları ve aynı modüllerde eklenecek dar test dosyaları.
Kapsam caller envanterinde büyürse dosyalar açıkça listelenir; başka iş akışı
değişikliği pakete eklenmez.

**Kabul kriterleri:**

- Hatalı MIME, eksik attachment, SMTP bağlantı/teslimat hatası ve beklenmeyen
  alt hata yolları prosesi sonlandırmaz; `SendEmail` normal `error` döndürür.
- Her mevcut caller error'ı bilinçli olarak ele alır; error yutulmaz ve proses
  sonlandırmaya çevrilmez.
- DB işlemi tamamlanmış caller, e-posta hatasında kaydın oluşmadığını iddia etmez;
  kayıt sonucu ile teslimat sonucu ayrıdır.
- Başarılı e-posta davranışı korunur; queue/outbox veya retry bu pakete girmez.

**Test yaklaşımı:** Production SMTP kullanılmadan fake SMTP ile başarılı ve
başarısız teslimat; hatalı MIME/eksik attachment; subprocess üzerinden hata
yolunda prosesin yaşadığını doğrulama; DB işlemi tamamlanmış caller için
partial-success response regresyonu. Mevcut test altyapısı yeterli değilse en
küçük seam/fixture bu pakette ve yalnız bu amaçla eklenir.

**Geri dönüş:** Library error-return ve zorunlu caller uyarlamaları tek uyumlu
paket olarak geri alınabilir; yalnız bir taraf eski sürüme döndürülemez. Geri
dönüş proses-sonlandırma riskini yeniden açacağından release durdurulur.

## Kanıt ve kaynak yolları

- `docs/ai/VERIFIED_AUDIT_2026-09-17.md`
- `docs/ai/SECURITY_BACKLOG.md`
- `docs/ai/APPOINTMENT_BACKLOG.md`
- [ADMIN_BACKLOG.md arşivi](UI_UX_IMPLEMENTATION_BACKLOG.md#historical-admin-backlog)
- [ROADMAP.md arşivi](PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-roadmap)
- [PROJECT_STATE.md arşivi](CURRENT_HANDOFF.md#historical-project-state)
- [ARCHITECTURE_DECISIONS.md arşivi](PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-architecture-decisions)
- `fiber-v2/baserouter/baserouter.go`
- `fiber-v2/main/main.go`
- `fiber-v2/lib/lib.go`
- `fiber-v2/models/models.go`
- `fiber-v2/schema.sql`
- `fiber-v2/controllers/panel/panel.go`
- `fiber-v2/controllers/post/post.go`
- `fiber-v2/controllers/post/users/users.go`
- `fiber-v2/controllers/post/randevular/randevular.go`
- `fiber-v2/controllers/post/headerbuttons/headerbuttons.go`
- `fiber-v2/controllers/post/options/`
- `fiber-v2/static/html/views/panel/secenek-sayfalari/`
- `fiber-v2/static/html/views/panel/job-applications-sayfalari/`
- `fiber-v2/static/js/panel/notifications.js`
- `fiber-v2/test/all_test.go`

<a id="historical-roadmap"></a>

## Arşiv kaydı: ROADMAP.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### Nivgöz — ordered roadmap

Source: accepted Phase 1 audit, baseline 22958244024a81ca69edb09f77a22aa60291ff6c.
All implementation items are OPEN / NOT APPROVED. Ordering does not authorize execution.

#### Delivery rules

Each task needs a bounded plan, appropriate local tests, reviewed diff, commit and documentation update. Tests accompany safety fixes; Phase C extends coverage rather than postponing it. Split any row further if implementation would address unrelated problems.

Before executing code that opens a database or invokes a migration, establish an isolated local test configuration. Never use schema.sql as an upgrade. No production access is approved.

#### Phase A — safety foundation

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

#### Phase B — appointment integrity

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

#### Phase C — targeted regression protection

C01–C08 are small test tasks around existing approved behavior: authentication, authorization, branch permissions, appointment submission, queue visibility, status changes, uploads/files and critical admin endpoints. Source: F25 and the corresponding Phase A/B findings.

Use deterministic local fixtures. Do not create a broad test framework or introduce dependencies without a demonstrated need and approval. A usable Go toolchain and isolated test configuration are prerequisites; neither was validated during Phase 1.

#### Phase D — incremental backend architecture

- D01 (F19): investigate shared cache/ORM concurrency with focused local evidence; do not assume a race has been proven.
- D02 (F20): profile one expensive query path at a time, starting with staff queues/exports and recruitment pagination.
- D03: extract authorization boundaries already specified and tested in A/B; avoid a new framework.
- D04: reduce duplicated validation and query mapping only around touched, regression-protected behavior.
- D05: separate the next oversized controller responsibility when its safety or maintenance benefit can be demonstrated.

The architecture work derives from the audited large controllers and duplicated logic. It does not authorize an application rewrite, generalized service-layer migration or speculative optimization.

#### Phase E — Admin/CMS product improvement

- E01 (F21, F27): establish a reviewed admin interaction/design inventory and accessible component contracts.
- E02: improve appointment queue tables, filters, search, pagination, status feedback and confirmation interactions.
- E03: improve appointment forms and detail hierarchy without changing business rules.
- E04: apply the approved system incrementally to navigation, dashboard and content CRUD.
- E05: verify keyboard access, responsive layouts, modals, notifications and empty/loading/error/success states.
- E06 (F26): confirm ownership/use before removing or replacing dormant modules and placeholders.

These are future product workstreams grounded in the audit; each requires a concrete task before implementation.

#### Phase F — public technical quality

- F-T01 (F24): establish a repeatable local/browser performance baseline, protecting existing CLS/LCP work.
- F-T02: address measured image, CSS, JavaScript and font costs individually.
- F-T03 (F22): verify canonical routes, fallback status, metadata, sitemap/robots responsibilities and structured data; do not rewrite SEO content.
- F-T04 (F21, F16): verify accessibility and submission feedback on critical public flows.
- F-T05 (F23): verify consent revocation behavior and agree the technical tracking contract.
- F-T06: assess localized content/routing requirements before replacing translation behavior.

Production PageSpeed and server behavior remain Phase H work requiring approval.

#### Phase G — public visual design

After technical foundations are stable, define a reviewed public design system and improve typography, spacing, controls, cards, header/navigation, homepage, doctor/department/center pages and appointment experience. Add motion only with accessibility and performance checks. Preserve branding and existing performance work. No redesign is approved by this roadmap.

#### Phase H — production/server verification

Explicit approval required before any access. Verify aaPanel, Nginx, PostgreSQL version/schema, deployed commit, process manager, environment, compression, caching, TLS, security headers, backup/restore, monitoring and production PageSpeed. Source: F01/F11/F12/F22/F24/F28 and the audit's production unknowns.

Read-only verification does not authorize deployment, server changes or restore drills.

#### Complete audit finding register

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

#### First implementation proposal — A01

- TASK: Return MIME/SMTP errors from SendEmail without terminating the process.
- WHY FIRST: F08 is P1 / VERIFIED, with a small change surface and low implementation risk. It can be tested locally without changing schemas, appointment rules or permission rules. F01 is already subject to an immediate no-execution gate.
- FILES: fiber-v2/lib/lib.go; proposed new fiber-v2/lib/email_test.go. Read existing callers and module configuration for test compatibility; caller changes are not assumed.
- PLANNED CHANGE: Replace the two log.Fatalf paths in SendEmail with ordinary error returns, preserving its signature and successful-send behavior. Add narrowly scoped regression tests. Do not add retries, a queue or notification-policy changes.
- RISKS: Callers may ignore returned errors or expose existing partial-success behavior; check and document that behavior. Go toolchain/test-environment availability must be resolved before verification. If isolated tests require broader refactoring or dependency changes, stop and revise the plan.
- TEST PLAN: With an approved Go toolchain, test MIME assembly failure using a missing temporary attachment; SMTP failure against a local controlled server; successful delivery to a local fake SMTP server; and a subprocess regression that fails if SendEmail exits. No production SMTP or database connection. Review callers and run relevant package checks.
- ROLLBACK: Revert the isolated implementation commit if necessary; no database or configuration rollback. Reverting restores the fatal-exit risk and must be recorded.
- STATUS: PROPOSED — awaiting explicit approval. No implementation or tests have been performed for A01.

<a id="historical-architecture-decisions"></a>

## Arşiv kaydı: ARCHITECTURE_DECISIONS.md

Bu bölüm taşınan tarihsel kaydın tamamını korur; o tarihteki öneri/onay, rol, çağrı sırası ve test durumu güncel kabul değildir. Güncel sınırlar için [devir notuna](CURRENT_HANDOFF.md), alan backlog’una ve tarihli commit kayıtlarına bakın.

### Nivgöz — architecture decisions

This register records accepted workflow decisions separately from unapproved technical designs. Source: accepted audit and the user's controlled engineering instructions, 2026-09-14.

#### ADR-001 — Use the accepted audit as the baseline

Status: ACCEPTED.

Use baseline commit 22958244024a81ca69edb09f77a22aa60291ff6c and findings F01–F28. Do not repeat the full repository audit without an explicit request. Investigate new evidence narrowly and record any resulting finding with its evidence/confidence; do not silently relabel existing findings.

#### ADR-002 — Deliver small, tested changes

Status: ACCEPTED.

Follow ANALYZE → PLAN → IMPLEMENT → TEST → REVIEW DIFF → COMMIT → UPDATE DOCUMENTATION. Normally solve one problem per task. Avoid unrelated cleanup, dependency churn and framework rewrites. Local tests accompany fixes; Phase C adds focused coverage.

Read CLAUDE.md and docs/ai/*.md before major work. The user's latest approved task scope takes precedence over CLAUDE.md's historical frontend-only restriction. Existing branding, Jet, asset separation and accessibility conventions still apply where relevant.

#### ADR-003 — Preserve both product surfaces

Status: ACCEPTED.

Public website and Admin/CMS are first-class products. Address locally testable security/correctness risks before visual redesign. Admin improvement includes backend, permissions, database behavior and workflows as well as UI. Preserve the existing public performance work unless a regression is demonstrated.

#### ADR-004 — Separate local engineering from production actions

Status: ACCEPTED.

No direct production changes. Explicit approval is required for production access, aaPanel, schema changes, destructive operations and deployment. Production verification does not imply permission to modify production.

Never use the existing destructive schema.sql as an upgrade migration. Future migration design must be reviewed independently of execution. Existing production versions/schema/commit remain UNKNOWN until verified with approval.

#### ADR-005 — Business rules require explicit decisions

Status: ACCEPTED.

Appointment rule changes and unclear role/permission changes require approval. Do not derive policy from current bugs, hidden navigation or inconsistent checks.

Unresolved: branch authority, centerless-request ownership, status transitions, duplicate policy, scheduling conflict rules, deletion/archive semantics, assignment/history requirements and session lifetime. These are decision topics, not approved new features.

#### ADR-006 — First implementation candidate

Status: PROPOSED — NOT APPROVED.

A01 would replace fatal exits in SendEmail with returned errors while retaining its signature and successful behavior. It would not introduce a mail queue, retries, appointment-rule changes or a notification redesign.

Reason: F08 is P1 / VERIFIED with a narrow local change and a deterministic local verification strategy.

Approval and actual test results must be recorded before this decision can be marked implemented.

#### Future decision records

Create a new record only for a material choice reached during a scoped task. Include context, decision, alternatives when relevant, consequences, approval status and validation. Do not preselect a new framework, ORM, RBAC model, event system or design system.
