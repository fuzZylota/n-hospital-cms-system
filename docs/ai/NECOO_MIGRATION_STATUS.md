# Necoo33 neormgo kaldırma envanteri

İlk envanter: 2026-09-28, statik kaynak incelemesi. Eski modül kaynağına
erişim/indirme, DB veya production doğrulaması yapılmadı.

## Güncel devir — 2026-09-28, HEAD `13b048e`

- **Yerel kapı kararı / yüksek güven:** N01 sözleşmeleri, N02 fake/seam, N03 header okuma pilotu ve N04 notification hub için aşağıdaki hedefli kaynak/test kanıtları yerel kabulü destekler. `13b048e` pilot sözleşme testlerini kaydetti. Bunlar tam workspace veya production kabulü değildir; eski başlangıç HEAD'i ve ilk envanter aşağıda tarihsel kanıt olarak korunur. Bu belge güncellemesinde Go testleri yeniden çalıştırılmadı.
- **N04 sonrası güvenlik bağlantısı:** `06ec7c5` canlı oturum/alıcı yeniden yetkilendirmesini, `c549a26`, `af8e96b`, `c64aafa` üç kayıt kaynaklı F05 üreticisini yerel geçmişe ekledi. DOM bildirim metni `textContent` ile yazılır. Hub kabulü F05 outbox/retry teslim garantisini, İK admin dışı rol/nesne kararını veya genel dosya politikası işini kapatmaz; ayrıntı [güvenlik backlog'unda](SECURITY_BACKLOG.md).
- **Açık kapılar:** N05 ortak veri erişimi henüz değerlendirilmedi; seçenek/ban okuma ve InsertMedia transaction eşliği için ayrı sözleşme/çağıran kanıtı gerekir. N06–N11 de **AÇIK**. Mevcut owned snapshot ve kirli options kodu N05 kabulü değildir. Tam `main`/`baserouter`/Jet kontrolleri eksik bağımlılıklar nedeniyle **BLOCKED**; gerçek DB/şema, eski İK dosyaları, proxy/cache, tarayıcı ve production **UNKNOWN**. N09 sıfır eski import/manifest kapısı ve N10 tam workspace testi geçilmiş sayılmaz.

## N01 kapısı — 2026-09-28 yerel kabul

- **GEÇTİ, yerel/yüksek güven.** REL-001A F08/A01 kanıtı ve N00 envanteri mevcut.
- `models/data/contracts.go`, `header_buttons.go`: saf DTO/reader; boş liste,
  güvenli hata ve NULL ayrımı. `contracts_test.go`, `imports_test.go`,
  `package_shape_test.go` geçti; ağsız `go test -count=1 ./data` GEÇTİ.
- `go list ./data` yalnız `context` importunu ve cycle olmadığını gösterdi;
  `parentread` tüketicisi ile `postgres` header testleri de ağsız geçti.

## N02 kapısı — 2026-09-28 yerel kabul

- **GEÇTİ, yerel/yüksek güven.** Aşağıdaki testlerin hepsi ağsız geçti.
- SQL/arg sırası ve tipi: `dbtest:TestQueryRecordsSQLAndOrderedArguments`; gerçek header SQL'i/sıfır arg: `postgres:assertHeaderButtonQuery`.
- Boş/çoklu satır, scan/NULL: `dbtest:TestRowsScanMultipleTypesAndNull`, `TestScanTypeMismatchUsesDatabaseSQLConversionError`; `postgres:TestHeaderButtonRepositoryQueryAndMapping`, `TestHeaderButtonRepositoryEmptyResultIsNonNil`.
- Hata/Close: `dbtest:TestRowsIterationErrorAndCloseAreVisible`, `TestRowsCloseErrorLifecycle`; `postgres:TestHeaderButtonRepositoryQueryErrorIsSafe`, `TestHeaderButtonRepositoryScanErrorReturnsNilAndClosesRows`.
- Tx: `dbtest:TestTransactionCommitSequence`, `TestTransactionQueryErrorRollbackSequence`, `TestTransactionExecErrorRollbackSequence`, `TestTransactionLifecycleErrors`.
- Context: `dbtest:TestContextBeforeDriverEntry`, `TestContextAfterDriverEntry` ve `postgres` header iptal testleri.
- Reader fake'i: `models/data:TestHeaderButtonReaderOutcomes`, `parentread:TestResponseContract` (yetkisiz sıfır okuma, boş/hata). Yeni `postgres:TestHeaderParentReadSeam` SQL + satır + consumer cevabını birlikte doğrular.
- `go list` import grafiği/cycle kontrolü geçti; `dbtest` yalnız stdlib kullanır. Fake, PostgreSQL constraint/trigger/isolation veya production şemasını kanıtlamaz.

## N03 kapısı — 2026-09-28 DB'siz pilot kabul

- **GEÇTİ, yerel/yüksek güven.** `schema.sql:124–136` hbid/title/nullable parent_id ve active/sort kolonlarını gösterir; canlı şema/DB parity **UNKNOWN**.
- `postgres/header_buttons.go` yalnız `hbid,title,parent_id WHERE parent_id IS NULL` sorgular; `is_active` ve `ORDER BY` yoktur. `TestHeaderButtonRepositoryKeepsUnorderedParentRows`, exact SQL/args ve NULL scan testleri geçti; önceki ORM sorgusu da active/order uygulamıyordu.
- `headerbuttons:TestGetMainHeaderButtonsHTTPContract` gerçek Fiber handler'da anonim sıfır okuma, yetkili başarı/boş liste, hata, JSON 403/200/500, HTTP 200 ve nil parent `""` yollarını geçti.
- `parentread:TestHandlerUsesProductionResponseAfterAuthWithoutLegacyFallback`, `main/userstatuswiring:TestOwnedUserStatusComposition` ve `models.Utilities.HeaderButtonReader` repository → injection → handler bağını doğruladı; pilotta ORM fallback/ikinci sorgu yok.
- Ağsız data, postgres header, headerbuttons/parentread ve main/userstatuswiring testleri ile ilgili `go list -test -deps` kontrolleri geçti. Native `main` tam testi uncached godotenv/Jet/excelize/msgp ve cache-lock erişim hatasıyla **BLOCKED**; tam route middleware ve DB parity test edilmedi, N10 borcu.

## N04 kapısı — 2026-09-28 güncel yerel kanıt

- **GEÇTİ (yerel N04 kabulü).** N04A/N04B tarihsel raporları tamamlanma kanıtı sayılmadı; güncel üretim Go kodunda dış broadcaster import/tipi, manifest/checksum kaydı bulunmadı.
- `models/notify/contracts.go`, `lib/notificationhub/hub.go`, `main/main.go`, `models/models.go`, `post/post.go`, `post/randevular/randevular.go`: typed kimlik, tek hub ve shutdown, dört tüketici bağlı. `notificationws/wiring_test.go` import/cycle ve kimlik sınırını geçti.
- Ağsız `go test -count=1`: `models/notify`, `lib/notificationhub`, `post/notificationevent`, `post/notificationws`, `post`, `randevular`, `main/userstatuswiring` GEÇTİ; ayrı `main/lifecycle.go` + iki testi GEÇTİ. Fake sink, FIFO/seri yazma, bounded queue/yavaş istemci, disconnect, allow/deny, Fiber upgrade ve transport testleri bu paketlerde çalıştı.
- SEC-003C: `CurrentRecipients` güncel durum/rolü admission ve teslim öncesi denetler (`live_recipients_test.go`, `live_authorization_test.go`, `hub/live_authorization_test.go`); inbound yeniden yetkilendirme `post/notification_authorization_test.go` ve `notificationws/live_authorization_test.go` ile geçti. F05 üretici nesne yetkisi **AÇIK**.
- Race **GEÇTİ**: resmî MSYS2 UCRT64 (kurucu 20260927), GCC 16.2.0, yeni terminalde `GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=C:\msys64\ucrt64\bin\gcc.exe`; `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOFLAGS=-mod=readonly` ile `fiber-v2` kökünde `go test -race -count=1 ./lib/notificationhub ./controllers/post/notificationws` iki pakette `ok` döndü. İlk sandbox cgo derlemesi başarısızdı; aynı ağsız komut izinli yürütmede geçti. Native `main` uncached godotenv/Jet/excelize/msgp nedeniyle **BLOCKED** (N10 borcu); gerçek DB, servis, canlı şema/production **UNKNOWN**. N05 kapısı artık ayrı değerlendirilebilir.

## Git başlangıç kanıtı

- Repo: `C:/Users/DELL/nivgoz/n-hospital-cms-system`; dal `nivgoz-professional-v2`; HEAD `67d8b7a63198050dba43a945e72abcf50318473b`.
- `origin`: `https://github.com/fuzZylota/n-hospital-cms-system.git`; fetch yapılmadı.

## Dört sınıf

### Aktif çalışma zamanı

- Gerçek üretim importları: `fiber-v2/database/database.go` (ORM tipi/
  bağlantı adaptörü), `fiber-v2/models/models.go` (Utilities/FrontendOptions
  ORM alanları), `fiber-v2/controllers/post/options/options.go` (Neorm tipi).
- Bu sınırları kullanan eski ORM sorgu akışları dosya/fonksiyon grupları:
  `controllers/frontend/frontend.go` içindeki ana sayfa, içerik, haber,
  testimonial, tedkik/tıbbi birim, şube ve doktor okuma fonksiyonları;
  `controllers/panel/panel.go` içindeki panel liste/detay/query oluşturma
  fonksiyonları; `controllers/post/**` içindeki domain CRUD ve sıralama
  fonksiyonları; ayrıca `baserouter/options_access.go`, `database/database.go`
  ve `lib/lib.go` yardımcı/kimlik ve ortak veri yolları. Bu gruplar bağımsız
  akış sayısı değildir; dosya ve fonksiyon bazında migration öncesi ayrıştırma
  gerekir. N00 planındaki 41 metot/4.649 çağrı eski baseline sayımıdır, runtime
  ölçümü değildir.
- Yeni durum: kirli, untracked `controllers/post/branslar/
  add_branch_upload_policy_test.go` de neormgo import ediyor; bu üretim kodu
  değil ama N09 sıfır-import kapısını engeller.

### Manifest / checksum

- `go.mod`: `baserouter`, `database`, `lib`, `models` içinde doğrudan veya
  dolaylı require kayıtları.
- `go.sum`: aynı dört modülde neormgo/v2 v2.0.2 checksumları; `baserouter`
  ayrıca eski v1.0.1 checksumlarını taşıyor.
- `go.work.sum`: v1 için v1.1.3, v1.3.1–v1.9.0; v2 için v2.1.0 ve v2.4.0
  checksum kayıtları. Değerler kasıtlı olarak kopyalanmadı.

### Emekli paket testi

- Neormgo için “emekli paket” yalnız test ifadesi tespit edilmedi. Yukarıdaki
  branslar testi eski modülü gerçekten import ediyor; salt metinsel anma değil.
- `notificationws/wiring_test.go` Necoo33 fiber-ws-broadcaster yolunu
  parçalı string olarak testte anıyor; neormgo kaldırma kapısına dahil değil.

### Tarihsel belge

- `docs/ai/OWNED_DATA_LAYER_MIGRATION_PLAN.md` (retired hedef ve N00–N11),
  `docs/ai/PROJECT_STATE.md`, `docs/ai/N04B_NOTIFICATION_HUB_WIRING.md` ve
  `CLAUDE.md` eski paketi anıyor. `AGENTS.md` de mevcut ORM mimarisini anlatıyor;
  güncel işletim gerçeği olarak yenilenmesi N11 kapsamındadır. Bu belge
  migration'ın tamamlandığına kanıt değildir.

## N07 ilk paket adayı ve engeller

- Tek akış adayı: public Tedkikler listesi, `controllers/frontend/frontend.go`
  içindeki `TedkiklerPage`; mevcut sorgu `tedkikler` + `medias` LEFT JOIN,
  `is_active=true`, sayfalama yok, `models.Tedkikler` eşlemesi ve mevcut
  hata/redirect davranışı. Liste sayfası ile detay, anasayfa veya URL kontrolü
  aynı pakete katılmamalı.
- Owned `models/data` ve `database/postgres` katmanları ile option/appointment/
  contact/job snapshot okuyucuları vardır; ancak Tedkikler için DTO, reader,
  repository veya bağlı snapshot yok. Snapshot bağlantısı bu akışa davranış
  kanıtı sağlamaz.
- Kesin uygulama dosyaları: `models/data/tedkikler.go`, `database/postgres/tedkikler.go` ve testi, `controllers/frontend/frontend.go` (yalnız handler), `models/models.go`, `main/main.go` (injection). Kirli dosya tek başına **BLOCKED** kanıtı değildir; N07 değerlendirilmedi.
- Dar testler: data reader sözleşme testi; postgres SQL/args, aktif filtre,
  LEFT JOIN, scan/NULL ve hata testleri; TedkiklerPage handler fake ile başarı,
  boş liste, repository hatasında redirect ve aynı render alanları. DB'siz.
  Çalıştırılmadı; N06 çıkış kanıtı ve şema/NULL eşliği ayrıca doğrulanmalıdır.
  REL-001A yerel kabulü geçti; N07 davranış eşliği henüz kanıtlanmadı.

## N09 kalan kullanıcılar / kapılar

- Çalışma zamanı: `database/database.go`, `models/models.go`,
  `controllers/post/options/options.go` ve eski ORM çağrılarını kullanan yukarıdaki
  frontend/panel/post akışları. Ayrıca `branslar/add_branch_upload_policy_test.go`.
- Manifestler: `baserouter/go.mod`, `database/go.mod`, `lib/go.mod`,
  `models/go.mod`; checksumlar: neormgo içeren dört `go.sum` ile
  `fiber-v2/go.work.sum`.
- N09 önkoşulu N08 caller-by-caller tamamlanması; N07 için N06 çıkışı gerekir.
  N01–N04 yerel kabulü geçti; N05–N06 değerlendirilmedi. Tam workspace N10
  build/test ve N11 hash/provenance/offline/zero-reference kanıtı yok: **BLOCKED**.
