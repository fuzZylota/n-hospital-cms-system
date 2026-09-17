# Nivgöz — doğrulanmış kritik bulgu denetimi

Tarih: 2026-09-17
Tür: FAZ 1B bağımsız, salt okunur statik kod incelemesi
Repository: `n-hospital-cms-system`
Dal: `nivgoz-professional-v2`
HEAD: `18e98fd`
Baseline etiketi: `pre-professional-audit-2026-09-17`

## Kapsam ve sınırlar

Bu kayıt, önceki denetimin kritik/yüksek iddialarını bağımsız olarak kaynak
koddan, route/middleware sırasından ve veri/çağrı zincirinden yeniden
değerlendirir. Uygulama başlatılmadı; test, migration, DB, SMTP, production
erişimi veya exploit denemesi yapılmadı. Bu nedenle bulgular repository
düzeyindeki davranışı gösterir; deployment, reverse proxy ve production veri
durumu için kanıt değildir.

Secret değerleri ve `.env` içerikleri incelenen dosyalarda mevcut olsa dahi bu
rapora alınmamıştır. OWASP ASVS 5.0 için yalnız bölüm adları kullanılmıştır;
doğrulanmamış requirement numarası verilmemiştir.

## Karar özeti

| Kimlik | Karar | Önem | Güven | Kısa sonuç |
| --- | --- | --- | --- | --- |
| OPS-001 | DOĞRULANDI | Kritik | Yüksek | Yıkıcı şema yükleme betikleri hedef DB'ye uygulanabilir. |
| SEC-001 | DOĞRULANDI | Yüksek | Yüksek | Non-admin kullanıcı kendi şube izinlerini yazabilir. |
| SEC-002 | DOĞRULANDI | Yüksek | Yüksek | Ayar secret'ları düşük yetkili panel HTML'ine taşınır. |
| SEC-003 | DOĞRULANDI | Yüksek | Yüksek | Ban/silme aynı request'i durdurmaz; token rolü DB'den yenilenmez. |
| SEC-004 | DOĞRULANDI | Yüksek | Yüksek | Kullanıcı kontrollü yol containment olmadan silme sink'ine ulaşır. |
| SEC-005 | DOĞRULANDI | Yüksek | Yüksek | Galeri upload yetki ve içerik doğrulaması olmadan public static köke yazar. |
| SEC-006 | KISMEN DOĞRULANDI | Yüksek | Yüksek | CV public erişimi doğrulandı; diploma fiziksel akışı doğrulanmadı. |
| SEC-007 | KISMEN DOĞRULANDI | Yüksek | Yüksek | Anonim WS değil; authenticated cross-user gerçek zamanlı XSS yolu var. |
| SEC-008 | KISMEN DOĞRULANDI | Yüksek | Yüksek | Bazı header ve tüm galeri mutation'ları her panel rolüne açık. |
| REL-001 | DOĞRULANDI | Yüksek | Yüksek | SMTP hatası uygulama sürecini sonlandırabilir. |
| REL-002 | DOĞRULANDI | Yüksek | Yüksek | Pagination girdileri panic ve aşırı veri yükleme yolları açar. |
| DATA-001 | DOĞRULANDI | Yüksek | Yüksek | Şema, model ve handler sözleşmeleri temel olarak uyumsuzdur. |
| FLOW-001 | DOĞRULANDI | Orta | Yüksek | Public JS yalnız panel JWT'siyle açılan yardımcı endpoint'lere bağlı. |
| FLOW-002 | DOĞRULANDI | Yüksek | Yüksek | Şubesiz randevular operasyonel queue yüzeylerinden elenir. |
| FLOW-003 | KISMEN DOĞRULANDI | Yüksek | Yüksek | Silme bağlı; santral status değişikliği hedef şubeye doğru bağlı değil. |
| FORM-001 | DOĞRULANDI | Orta | Yüksek | İK formu iki listener çalıştırır; insert dosya doğrulamasından öncedir. |

Toplam: 12 doğrulandı, 4 kısmen doğrulandı. Bu kararlar remediation veya
production istismarı kanıtı değildir.

## Bulgular

### OPS-001 — yıkıcı schema uygulaması

- **Doğrudan kanıt:** `fiber-v2/schema.sql` içindeki `DROP SCHEMA public
  CASCADE`; `fiber-v2/migrate.sh` ve `fiber-v2/production-migrate.sh` içindeki
  şema yükleme çağrıları.
- **Çağrı zinciri:** Operatör → betik → ortamdan DB hedefi → `psql` → yıkıcı
  schema SQL'i.
- **Mevcut korumalar:** Değişken/bağlantı kontrolleri vardır; production
  reddi, hedef doğrulama, yazılı onay, backup veya restore doğrulaması yoktur.
- **Önkoşul ve etki:** Yetkili erişimle betiğin değerli DB'ye çalıştırılması;
  public şemadaki veri ve bağımlı nesnelerin kaybı.
- **Düzeltme:** Betiğin varlığı otomatik veri kaybı değildir; operatör çalışması
  gerekir. Koşullu olması önemini düşürmez.
- **Kabul kriterleri:** Fresh install ve upgrade ayrımı, production fail-closed,
  hedef/backup doğrulaması, açık yıkıcı onay ve test edilmiş geri dönüş.
- **ASVS:** Configuration; Data Protection.

### SEC-001 — self-edit ile şube izni yazımı

- **Doğrudan kanıt:** `fiber-v2/baserouter/baserouter.go` içindeki
  `POST /backend/edit-user`; `fiber-v2/controllers/post/users/users.go` içindeki
  `EditUser` ve `user_branch_permissions` update/insert döngüsü.
- **Çağrı zinciri:** Panel JWT → `PanelAuthMiddleware` → `EditUser` → body/form
  permission alanları → aktif şubeler için izin yazımı.
- **Mevcut korumalar:** Body UID'si token UID'siyle eşleşmelidir. Eski/yeni
  rol, aktiflik ve şube alanları karşılaştırılır; ancak her ikisi de istemci
  kontrollüdür. Permission bloğunda admin/rol kontrolü yoktur.
- **Önkoşul ve etki:** `moderator`, `santral` veya `ik` hesabı; kendi UID'si;
  değişmemiş görünen eski/yeni alanlar ve en az bir izin/değişiklik alanı.
  Kullanıcı kendi şube view/delete kapsamını genişletebilir.
- **Düzeltme:** Doğrudan admin rolü yükseltmesi kanıtlanmadı. Gönderilmeyen
  aktif şube izinleri satır silinmese de `false` olarak sıfırlanabilir.
- **Kabul kriterleri:** Permission yazımı ayrı yetkili endpoint'te; hedef ve
  eski durum sunucuda yüklenmiş; allowlist DTO ve UID+şube yetki testleri.
- **ASVS:** Authorization; Validation and Business Logic.

### SEC-002 — ayarlardan secret sızıntısı

- **Doğrudan kanıt:** `fiber-v2/baserouter/baserouter.go` ayar route'larını
  yalnız panel auth altında tutar. `fiber-v2/controllers/panel/panel.go`
  içindeki `SecenekPage` ve `SecenekDuzenlePage` `o.*` alanlarını template
  modeline taşır; `fiber-v2/static/html/views/panel/secenek-sayfalari/secenek.jet`
  ve `fiber-v2/static/html/views/panel/secenek-sayfalari/secenek-duzenle.jet`
  secret değerleri hidden/password alanlarında response'a koyar.
- **Çağrı zinciri:** Doğrudan ayar URL'si → panel auth → DB ayar satırı → Jet
  template → HTML response.
- **Mevcut korumalar:** Ayar menüsü admin/moderator için gizlenir; yazma
  handler'larında admin kontrolü vardır. Menü gizleme endpoint koruması değildir.
- **Önkoşul ve etki:** Her geçerli panel rolüyle doğrudan URL; SMTP veya captcha
  secret'larının response içinden okunması.
- **Düzeltme:** Maskeli input veya gizli span, değeri response'dan çıkarmaz.
- **Kabul kriterleri:** Secret'lar hiçbir read DTO/template'e konmaz; secret
  read/write ayrı role bağlanır; düzenleme formu mevcut secret'ı göstermez.
- **ASVS:** Authorization; Data Protection; Configuration.

### SEC-003 — ban ve token revocation zinciri

- **Doğrudan kanıt:** `fiber-v2/main/main.go` sırası `JWTMiddleware` ardından
  `HandleUserBanning`dir. `fiber-v2/lib/lib.go` içindeki `HandleUserBanning`, silinmiş
  veya pasif kullanıcı için expired response cookie yazıp `c.Next()` çağırır.
  Sonraki `PanelAuthMiddleware` request cookie'sini yeniden okur.
- **Çağrı zinciri:** Request JWT → token doğrulama/yenileme → kullanıcı DB
  kontrolü → response cookie silme → `c.Next()` → route auth/handler.
- **Mevcut korumalar:** Kullanıcının varlığı ve aktifliği DB'den sorgulanır.
  DB hatası fail-open ilerler; token rolü DB'den yeniden yüklenmez.
- **Önkoşul ve etki:** Süresi dolmamış eski token ve silinmiş/pasif hesap ya da
  DB kontrol hatası. Aynı request tamamlanabilir; rol değişimi token ömrünce
  gecikebilir.
- **Düzeltme:** Sonraki tarayıcı request'inde cookie sırası runtime doğrulaması
  ister; aynı request'in devam etmesi kaynak kodunda kesindir.
- **Kabul kriterleri:** Denetim sonrası fail-closed abort; DB rol/session
  doğrulaması; token yenilemenin revocation kontrolünden sonra yapılması.
- **ASVS:** Session Management; Self-contained Tokens; Authorization.

### SEC-004 — custom media silmede yol containment eksikliği

- **Doğrudan kanıt:** `fiber-v2/controllers/post/post.go` içindeki `DeleteCustomMedia`
  body'den `FileName` alır, `filepath.Join` ile yol üretir ve
  `fiber-v2/lib/lib.go` içindeki `DeleteFile` üzerinden `os.Remove` çağırır.
- **Çağrı zinciri:** `POST /backend/delete-file` → panel auth → `CheckAuth` →
  FileName → Join → Stat → Remove.
- **Mevcut korumalar:** JWT, sabit başlangıç dizini ve varlık kontrolü var;
  DB sahiplik çözümü, canonical containment, `Rel`, `EvalSymlinks` veya base
  name allowlist yok.
- **Önkoşul ve etki:** Geçerli panel hesabı, hedef yol bilgisi ve OS yazma
  yetkisi. `../` ile intended kök dışındaki yazılabilir dosyalar hedeflenebilir.
- **Düzeltme:** Mutlak yolun tüm platformlarda prefix'i doğrudan attığı ayrıca
  kanıtlanmadı. Windows ayırıcıları OS'ye bağlıdır; ara symlink kaçışı ancak
  mevcut symlink önkoşuluyla mümkündür.
- **Kabul kriterleri:** İstemciden fiziksel yol alma; DB dosya ID'si çözümü;
  canonical root containment; symlink engeli; sahiplik/rol denetimi.
- **ASVS:** File Handling; Authorization.

### SEC-005 — şube galerisi upload sınırı

- **Doğrudan kanıt:** `fiber-v2/controllers/panel/panel.go` içindeki `SubeGaleriEkle`
  yalnız `CheckAuth` uygular, orijinal uzantıyı korur ve
  `fiber-v2/static/files/subeler/galeri` altına yazar. `fiber-v2/main/main.go` `/files` static
  route'unu auth middleware'lerinden önce kaydeder.
- **Çağrı zinciri:** Gallery POST → panel auth → multipart file → uzantı →
  üretilen dosya adı → same-origin public static storage.
- **Mevcut korumalar:** Global body limiti ve üretilmiş isim traversal/olağan
  çakışma olasılığını azaltır. Rol/şube/nesne, MIME, magic-byte, allowlist,
  per-file limit, içerik/uzantı ve no-follow kontrolleri yoktur.
- **Önkoşul ve etki:** Her panel rolü ve dosyanın kurbana açtırılması. HTML/SVG
  gibi aktif içerikler kabul edilebilir; aynı origin'de kalıcı içerik ve XSS
  teslim noktası riski doğar.
- **Düzeltme:** Tek başına JS upload'ı otomatik çalışmaz. Static Content-Type
  dosya türü çözümlemesinden gelir; çift/alışılmadık uzantı davranışı runtime
  doğrulaması ister.
- **Kabul kriterleri:** Rol+şube yetkisi, dar allowlist, magic-byte/MIME uyumu,
  aktif içerik reddi/dönüşümü, güvenli storage ve response header politikası.
- **ASVS:** File Handling; Web Frontend Security; Authorization.

### SEC-006 — İK belgelerinin public erişimi

- **Doğrudan kanıt:** `fiber-v2/controllers/post/post.go` içindeki `AddJobApplication`
  CV dosyasını `fiber-v2/static/files/job-applications/` altında runtime
  oluşturulan başvuru-ID dizininde saklar. `/files`
  static route'u auth öncesidir.
- **Çağrı zinciri:** Public job application POST → DB insert → sıralı `jaid` →
  CV dosyası → public `/files/job-applications/...` URL'si.
- **Mevcut korumalar:** CV için boyut ve uzantı kontrolü var; magic-byte/MIME
  doğrulaması ve erişim kontrolü yok. Dizin sıralı ID, ad başvuran basename'idir.
- **Önkoşul ve etki:** Tahmin edilen/bilinen başvuru ID'si ve ad. CV'ler kaynak
  kodu düzeyinde authentication olmadan sunulabilir.
- **Düzeltme:** `DiplomaFileMid` istemciden alınsa da bu handler'da diploma için
  fiziksel upload yolu bulunmadı. Tüm diploma dosyaları için aynı iddia yoktur.
- **Kabul kriterleri:** Private storage, yetkili download endpoint'i,
  tahmin edilemez kimlik, sahiplik/medya türü doğrulaması ve erişim logları.
- **ASVS:** File Handling; Authorization; Data Protection.

### SEC-007 — WebSocket bildirim XSS'i

- **Doğrudan kanıt:** `/backend/notifications` route'u panel auth grubundadır.
  `fiber-v2/controllers/post/post.go` içindeki `NotificationWebsocket`, contact
  mesajından bildirim metni üretip kaydedebilir/yayınlayabilir.
  `fiber-v2/static/js/panel/notifications.js` mesajı `innerHTML` ile render eder.
- **Çağrı zinciri:** Panel JWT → WebSocket upgrade → client event → bildirim
  metni/DB broadcast → bağlı panel → `innerHTML` sink.
- **Mevcut korumalar:** Panel auth ve origin listesi vardır. Repository'de
  mesaj sanitization'ı veya CSP bulunmadı. Bazı randevu/İK broadcast sorguları
  hatalı recipient kimliği nedeniyle teslimi sınırlayabilir.
- **Önkoşul ve etki:** Geçerli panel hesabı, izinli origin/istemci ve bağlı bir
  panel kurbanı. Authenticated kullanıcı başka kullanıcıya gerçek zamanlı XSS
  payload'ı ulaştırabilir.
- **Düzeltme:** Endpoint anonim değildir. Klasik sayfa yenileme sonrası stored
  XSS ayrıca kanıtlanmadı; doğrulanan davranış cross-user real-time XSS'tir.
- **Kabul kriterleri:** Event schema/role doğrulaması, client UID'sine güvenmeme,
  text DOM API/escaping, CSP, event allowlist ve rol bazlı WS testleri.
- **ASVS:** API and Web Service; Encoding and Sanitization; Web Frontend Security.

### SEC-008 — header ve galeri mutation yetkilendirmesi

- **Doğrudan kanıt:** `fiber-v2/baserouter/baserouter.go` header ekleme/düzenleme ve galeri
  ekleme/silme/açıklama route'larını panel auth altında tanımlar. Header silme
  ve sıralama `fiber-v2/controllers/post/headerbuttons/headerbuttons.go` içinde admin
  kontrolüne sahiptir. Galeri silme/açıklama handler'ları route `sid` değerini
  nesne yetkisine bağlamaz.
- **Mevcut korumalar:** Header silme ve order admin-only'dir; diğer sayılan
  mutation'lar için açık rol/nesne denetimi yoktur. CSRF token, Origin/Referer
  doğrulaması bulunmadı; `SameSite=Lax` tam CSRF koruması değildir.
- **Önkoşul ve etki:** Her panel rolü header ekleme/düzenleme ve tüm galeri
  işlemlerini yapabilir; başka şube galerisi değiştirilebilir.
- **Düzeltme:** “Giriş yapan herkes tüm header işlemlerini yapar” yanlıştır.
  Header silme/sıralama admin ile sınırlıdır; galeri mutation'ları değildir.
- **Kabul kriterleri:** Endpoint bazlı role matrix, hedef nesne+şube eşlemesi,
  CSRF/origin politikası ve yetki regresyon testleri.
- **ASVS:** Authorization; Web Frontend Security.

### REL-001 — e-posta hatasında proses sonlanması

- **Doğrudan kanıt:** `fiber-v2/lib/lib.go` içindeki `SendEmail` hata yollarında
  `log.Fatalf` çağırır; bu tüm Go sürecini `os.Exit(1)` ile sonlandırır.
- **Çağrı zinciri:** Public randevu/iletişim/İK ve admin randevu/yanıt
  handler'ları → senkron `SendEmail` → fatal hata → proses sonu.
- **Mevcut korumalar:** Bazı akışlarda e-posta ayarları koşulludur; error return,
  retry, outbox veya request izolasyonu yoktur.
- **Önkoşul ve etki:** E-posta tetikleyen akış ve MIME/SMTP hatası. Public
  insertler ve admin randevu commitleri genellikle e-postadan önce kalıcıdır;
  contact/İK yanıt durumları e-postadan sonra yazılır.
- **Düzeltme:** Çağrılar şu an request goroutine'inde görünür; goroutine içinde
  olsa dahi `log.Fatalf` tüm prosesi sonlandırır. Restart davranışı runtime'dır.
- **Kabul kriterleri:** Hata dönüşü, kontrollü HTTP sonucu, transaction sonrası
  outbox/queue politikası, idempotency ve SMTP hata regresyonları.
- **ASVS:** Configuration; Security Logging and Error Handling.

### REL-002 — İK pagination panic'i

- **Doğrudan kanıt:** `fiber-v2/controllers/panel/panel.go` içindeki
  `JobApplicationsPage`, `Atoi` hatasını yok sayar; `per_page` ile bölme ve
  hesaplanan indekslerle slice yapar. Global recover middleware bulunmadı.
- **Çağrı zinciri:** `GET /panel/is-basvurulari?page=&per_page=` → panel auth →
  parse → tüm kayıtları yükleme → bölme/slice.
- **Mevcut korumalar:** `santral` reddedilir. Parametre alt/üst sınırı, SQL
  pagination ve panic recovery yoktur.
- **Önkoşul ve etki:** `admin`, `moderator` veya `ik` JWT'si. `per_page=0`,
  sayısal olmayan değer veya `page=0` gerçek divide-by-zero/negative-slice
  panic yoludur; büyük değerler belleği ve sorguyu büyütebilir.
- **Düzeltme:** Bu teorik değil, statik olarak ulaşılabilir panic yoludur.
- **Kabul kriterleri:** Sıkı parse/range, SQL LIMIT/OFFSET, recover, sınır ve
  hata-path testleri.
- **ASVS:** Validation and Business Logic; Security Logging and Error Handling.

### DATA-001 — schema, model ve handler uyumsuzluğu

- **Doğrudan kanıt:** `fiber-v2/schema.sql` içinde `user_branch_permissions` ve
  `sube_galerileri` yoktur; kod bunları sorgular. Randevu status timestamp ve
  İK kimlik/adres/eğitim/departman alanları handler/modelde bulunup şemada yoktur.
- **Çağrı zinciri:** Fresh schema yükleme → uygulama SQL sorguları → eksik
  tablo/kolon hatası. Çalıştırılabilir alternatif migration/AutoMigrate bulunmadı.
- **Mevcut korumalar:** `değiştirilen-eklenen-sutunlar` açıklayıcı nottur,
  executable migration değildir.
- **Önkoşul ve etki:** Repository şemasıyla yeni kurulum veya destructive
  yeniden kurulum. Runtime SQL hataları ve özellik kaybı doğabilir.
- **Düzeltme:** `users.sid` şemada vardır; eksik diye sayılmamalıdır. Seed
  parolasının biçimi bcrypt-only login ile uyumlu görünmez; seed değeri yazılmaz.
- **Kabul kriterleri:** Tek şema kaynağı, sıralı incremental migration'lar,
  temiz DB sözleşme testi, güvenli seed üretimi ve schema-query doğrulaması.
- **ASVS:** Configuration; Authentication; Secure Coding and Architecture.

### FLOW-001 — public JS'nin korunan endpoint bağımlılığı

- **Doğrudan kanıt:** `fiber-v2/static/js/frontend/fast-randevu.js` şube endpoint'ini,
  `fiber-v2/static/js/frontend/anlasmali-kurumlar.js` pagination endpoint'ini, public form
  script'leri WebSocket endpoint'ini çağırır. İlgili backend route'ları
  `PanelAuthMiddleware` grubundadır.
- **Çağrı zinciri:** Anonim public sayfa → fetch/WebSocket → panel JWT denetimi
  → redirect/401 veya başarısız upgrade.
- **Mevcut korumalar:** Endpoint'ler yetkisiz erişime kapalıdır; public UI için
  yanlış güven sınırı kullanılmıştır.
- **Önkoşul ve etki:** Anonim ziyaretçi. Hızlı randevu şube yükleme, anlaşmalı
  kurum “daha fazla” ve bazı live notification yolları çalışmayabilir.
- **Düzeltme:** Tüm randevu akışı kapalı değildir; server-rendered şube verisi
  ve public form POST'ları vardır. Bozuk olan JS yardımcı bağımlılıklarıdır.
- **Kabul kriterleri:** Public read endpoint veya server-rendered alternatif;
  panel WS bağımlılığının kaldırılması; anonim entegrasyon testi.
- **ASVS:** Authorization; API and Web Service.

### FLOW-002 — şubesiz randevunun operasyonel kaybı

- **Doğrudan kanıt:** `fiber-v2/controllers/post/randevular/randevular.go` şube ID'sini
  yalnız doluysa insert eder. `fiber-v2/controllers/panel/panel.go` liste, polling,
  export ve istatistik sorgularında şubeye `INNER JOIN` uygular.
- **Çağrı zinciri:** Public form → nullable `sid` ile insert → queue/poll/export
  inner join → kaydın elenmesi.
- **Mevcut korumalar:** Şube gönderilirse normal ilişki kuruluyor; null kayıt
  için sahiplik veya atanacak queue yok.
- **Önkoşul ve etki:** Public istekte `sid` gönderilmemesi. Kayıt başarılı
  görünür fakat liste, polling, bildirim ve export'ta görünmez. Detail/aggregate
  yüzeyleri kapsamlarına göre kaydı görebilir.
- **Düzeltme:** Kayıt DB'den silinmez; ana operasyonel yüzeylerden kaybolur.
- **Kabul kriterleri:** Şube zorunluluğu veya atanamamış queue kararı; aynı
  görünürlük kuralının liste/detail/poll/export/bildirimde uygulanması.
- **ASVS:** Validation and Business Logic; Data Protection.

### FLOW-003 — randevu mutation şube yetkisi

- **Doğrudan kanıt:** `DeleteRandevuRequest`, hedef kaydın şubesi ile mevcut
  UID'nin `can_delete` iznini birlikte sorgular. `ToggleRandevuRequestStatus`
  santral için önce mevcut kullanıcının herhangi bir şubesini, sonra hedef
  şubede herhangi bir santral kullanıcısını kontrol eder.
- **Çağrı zinciri:** Protected mutation → RRID → rol/şube sorguları →
  delete veya status update.
- **Mevcut korumalar:** Delete target+UID'ye bağlıdır. Listede view izni,
  detailde santral UID+SID ilişkisi vardır. Moderator için bu yüzeyler global/
  izinli davranış açısından tutarsızdır.
- **Önkoşul ve etki:** Bir şubede yetkili santral ve hedef şubede başka bir
  santral kullanıcısı. Hedef dışı randevu status'u değiştirilebilir.
- **Düzeltme:** Delete için UID bağının olmadığı ifadesi yanlıştır; eksik bağ
  santral status sorgusundadır.
- **Kabul kriterleri:** Tek authorization scope, hedef RRID+mevcut UID+SID
  bağlamı, moderator politikası ve allow/deny test matrisi.
- **ASVS:** Authorization.

### FORM-001 — İK formu çift listener ve insert sırası

- **Doğrudan kanıt:** `fiber-v2/static/html/views/frontend/insan-kaynaklari.jet`
  bir submit script'i yükler; layout ayrıca
  `fiber-v2/static/js/frontend/insan-kaynaklari.js` yükler. Her
  ikisi `preventDefault` sonrası fetch yapar. İkinci script
  `/backend/make-job-application` çağırır; repository'de route bulunmadı.
  `AddJobApplication` DB insertini dosya doğrulamasından önce yapar.
- **Çağrı zinciri:** Submit → iki listener → tanımlı public endpoint ve
  tanımsız endpoint → ilk handlerda DB insert → dosya/root kontrolleri.
- **Mevcut korumalar:** Client-side validation var; `stopImmediatePropagation`,
  tek owner veya server-side zorunlu CV kontrolü yok.
- **Önkoşul ve etki:** Normal form gönderimi. İki fetch oluşabilir; ikinci
  kayıt oluşturmaz. İlk kayıt başarılı olup ikinci yanıt hata gösterebilir;
  geçersiz/dosyası olmayan başvuru kaydı kalabilir.
- **Düzeltme:** İki başarılı DB insert kanıtlanmadı; doğrulanan iki request ve
  dosya kontrolünden önce kalıcı insert'tir.
- **Kabul kriterleri:** Tek submit sahibi, tanımsız endpoint'in kaldırılması,
  server-side validationın insert önüne alınması, tutarlı cleanup/transaction
  politikası ve duplicate submit testleri.
- **ASVS:** Validation and Business Logic; File Handling; Web Frontend Security.

## İlk denetimde daraltılan ifadeler

- WebSocket endpoint'i anonim değildir; doğrulanan risk authenticated
  cross-user gerçek zamanlı XSS'tir.
- Header mutation'larının tümü her kullanıcıya açık değildir; delete ve order
  admin kontrolündedir.
- Randevu silme sorgusu mevcut UID'ye bağlıdır.
- Dosya silmede mutlak yol davranışı tüm platformlar için ayrıca kanıtlanmadı;
  containment/traversal açığı kanıtlandı.
- Diploma dosyaları için CV ile aynı fiziksel upload akışı kanıtlanmadı.
- İK formu iki request üretir; iki başarılı DB kaydı kanıtlanmadı.
- Şubesiz randevu DB'de kalır; temel operasyonel sorgularda elenir.

## Operasyonel yasaklar

1. `fiber-v2/schema.sql`, `fiber-v2/migrate.sh` veya
   `fiber-v2/production-migrate.sh` disposable olmayan hiçbir DB'de
   çalıştırılmamalı.
2. Gerçek İK belgeleri public `/files` kökünde tutulmamalı; mevcut erişimler
   altyapı politikasında kısıtlanana kadar paylaşılmamalı.
3. Gallery upload ve custom-file delete, düşük yetkili panel hesaplarına
   açılmamalı.
4. SMTP hata yolunun prosesi öldürmeyeceği düzeltilmeden e-posta akışları
   kontrollü olmayan production trafiğine açılmamalı.
5. Menü gizliliği role güvenlik sınırı sayılmamalı; panel hesabı dağıtımı
   minimumda tutulmalı.
6. Bu repository şemasıyla yeni production kurulumu veya geri dönüş denemesi
   yapılmamalı.

## Bulgular arası bağımlılıklar

- OPS-001 ve DATA-001: Yıkıcı şema yükleme veri kaybına ek olarak uyumsuz
  şema ile işlev kaybı yaratabilir.
- SEC-001 ve FLOW-003: Self-edit izni, şube kapsamlı appointment işlemlerini
  genişletebilir.
- SEC-003 tüm panel endpoint'lerinin token/revocation güven sınırını etkiler.
- SEC-005, SEC-008 ile erişilebilir hale gelir ve SEC-007 payload teslimini
  kolaylaştırabilir.
- FLOW-001 public WebSocket işlevini bozar; SEC-007 ise authenticated bağlama
  ihtiyaç duyar.
- FLOW-002 nullable şube ve inner join bileşiminin sonucudur.
- FORM-001 ile SEC-006, İK veri bütünlüğü ve belge gizliliğini birlikte etkiler.

## Runtime veya production kanıtı gerektiren noktalar

- Gerçek production DB şeması ve manuel migration geçmişi.
- Reverse proxy'nin static dosya erişimi, CSP, `nosniff` ve Content-Disposition
  header'ları.
- HTML/SVG/çift uzantılı dosyaların gerçek static response türleri.
- Ban response'unda birden çok `Set-Cookie` davranışının tarayıcıdaki sonucu.
- Process manager'ın fatal/panic sonrası restart ve health-check davranışı.
- Sunucu OS'i, filesystem izinleri ve mevcut symlink'ler.
- Moderator rolünün iş kuralı kapsamı ve secret/attachment erişim politikası.
- Seed dönüşümünün deployment'ta harici yapılıp yapılmadığı.

## Geçici ilk düzeltme dalgası

Bu sıralama uygulama onayı değildir:

1. OPS-001: Yıkıcı DB giriş noktalarını güvenli hale getirme.
2. SEC-001: Şube permission write sınırını sunucuda kurma.
3. SEC-002: Secret'ları tüm response ve düşük yetkili okuma yollarından çıkarma.
4. SEC-005: Gallery upload doğrulaması ve yetkilendirmesi; SEC-008 ile birlikte.
5. REL-001: `log.Fatalf` süreç sonlandırmasını kaldırma.

DATA-001 bu dalganın deployment kapısıdır: şema sözleşmesi uzlaştırılmadan
migration veya yayın kabul edilmemelidir.

## Kapanış

Bu belge yalnız doğrulanmış kaynak kodu davranışını kaydeder. Hiçbir bulgu bu
belgenin eklenmesiyle düzeltilmiş sayılmaz. Her remediation; onaylı kapsam,
hedefli test, diff incelemesi ve ilgili backlog/roadmap güncellemesi gerektirir.
