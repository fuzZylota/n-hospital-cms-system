# N04B — owned notification production wiring

## Güncel sonuç — onaylı kaynak hazırlığı sonrası devam, 2026-09-18

**Production kaynak geçişi uygulandı; tam runtime/release kabulü BLOCKED.**
Dal/HEAD değişmedi: `nivgoz-professional-v2`,
`ef3c78ba5c890f3e32dce8f07466862071718395`.
Devam başlangıcında yalnız N04A belge değişikliği ve bu untracked rapor vardı;
korundular. Önceki inceleme/matris aşağıda tarihsel kayıt olarak saklanmıştır;
oradaki “uygulanmadı” ve “kaynak yok” durumlarını bu güncel bölüm geçersiz kılar.
Kullanıcının devam talimatı yalnız mevcut üç transport/Fiber modülünün kesin
sürümlerle indirilmesini ve N04B uygulamasını onayladı. Başka dependency yok.

### Bağımsız inceleme sonrası dar UID düzeltmesi — 2026-09-18

Kullanıcının açık kimlik sözleşmesiyle `notificationws/identity.go` içindeki
tek production `authenticatedUserID` validator'ı eklendi. Yalnız named
`notify.UserID`, ASCII base-10, canonical decimal ve `1..MaxInt64` kabul edilir.
Digit kontrolü, hatası ele alınan `strconv.ParseInt(..., 10, 64)` ve
`FormatInt` round-trip uygulanır; trim, leading-zero normalizasyonu veya fallback
yoktur. Hata yalnız sabit `errIdentity`; ham UID/parse cause saklanmaz/loglanmaz.
Generic hub, consumer, CheckAuth, PanelAuth ve ban middleware değiştirilmedi.

HTTP `Handler`, production `authenticatedUpgrade` kapısından geçer:
auth callback → ortak validator → typed local → upgrade. Geçersiz UID'de
local/upgrade/register çağrısı yoktur; Fiber sınırı sabit Unauthorized döndürür.
Session `identity` aynı validator'ı tekrar çağırır; ret halinde metadata boş,
random reader/Register çağrısı sıfır ve socket close tam birdir. Tek watcher
yalnız başarılı Register sonrasında başlar; sıralama AST kontrolüyle de korunur.
Client JSON UID hiçbir zaman validator girdisi değildir.

Tracked normal login `post.go:AuthenticationController` DB `uid` sonucunu
`lib.String` ile JWT DTO'suna, `lib.CreateJWT` da Uid claim'ine aktarır;
`schema.sql` users.uid SERIAL tanımlar. Leading-zero üreten tracked login yolu
bulunmadı; production şeması/legacy ORM mapping'i doğrulanmış sayılmaz.

`identity_test.go` aynı production validator/gate/session üzerinde 3 pozitif
(`1`, `42`, MaxInt64) ve 22 negatif vakayı çalıştırır: boş, zero/negative,
işaret, whitespace, leading-zero, decimal/scientific, alphabetic, overflow,
non-ASCII/NUL, yanlış named/plain tip ve nil. Error formatları ile Unwrap/Is/As
ham girdiyi/cause'u açmaz. HTTP gate testinde ret: local/upgrade/Register/random
0; başarı: her biri 1, random 32 byte. Session ret: Register/random 0, close 1;
başarı: Register 1, canonical metadata ve ayrı crypto-random ConnectionID.
Watcher'ın ret yolunda başlamadığına dair kanıt guarded Register sırası/statiktir;
test-only watcher sayacı veya validation algoritması eklenmedi.

Saf adapter/helper/session/wiring testleri tek, 20 tekrar ve
`-count=20 -parallel=8 -cpu=1,4` geçti; notify/hub/event/lifecycle regresyonları
tek ve 20 tekrar, hedefli vet geçti. HTTP gate testi production karar sırasını
çalıştırır, gerçek Fiber context/handshake kanıtı değildir. Gerçek Fiber ret
testi de eklendi; mevcut integration testleri gibi eksik transitif girdiler
nedeniyle derleme/runtime **BLOCKED**. Gerçek servis başlatılmadı.

**SEC-003 freshness, SEC-007 ve FLOW-001 kapanmadı. Runtime/release BLOCKED
devam eder.** Bu düzeltmenin kapsamı yalnız fiber.go, session.go, üç mevcut
notificationws test dosyası, yeni identity.go/identity_test.go ve bu rapordur.
Manifest/checksum değişikliği, ağ/dependency indirme, DB/SMTP/migration,
stage/commit/push yoktur. Önceki N04B değişiklikleri korunmuştur.

### İndirme ve checksum kanıtı

Repository dışında `%TEMP%/nivgoz-n04b-source` alanında:
`manifest-before.json` (35 dosya SHA256), `status-before.txt`, `verified.json`
ve ayrı `modules` cache'i kullanıldı. `GOWORK=off`, `GOTOOLCHAIN=local`,
`GOPROXY=https://proxy.golang.org`, `GOSUMDB=sum.golang.org`; private/direct
bypass kapalı. İlk sandbox denemesi ağ bağlantısında reddedildi; izinli ağ
erişimiyle aynı dar komutlar başarıyla çalıştı. Automatic approval reddi yok.
Her modül ayrı `go mod download -json module@version` komutuyla indirildi.

| Modül | Sürüm | Module checksum (mevcut main/go.sum ile eşleşti) |
| --- | --- | --- |
| github.com/gofiber/contrib/websocket | v1.3.4 | `h1:tWeBdbJ8q0WFQXariLN4dBIbGH9KBU75s0s7YXplOSg=` |
| github.com/fasthttp/websocket | v1.5.8 | `h1:k5DpirKkftIF/w1R8ZzjSgARJrs54Je9YJK37DL/Ah8=` |
| github.com/gofiber/fiber/v2 | v2.52.9 | `h1:YjKl5DOiyP3j0mO61u3NTmK7or8GzzWzCFzkboyP5cw=` |

go.mod checksum'ları da aynı kayıtlarla eşleşti, sırasıyla:
`h1:kTFBPC6YENCnKfKx0BoOFjgXxdz7E85/STdkmZPEmPs=`,
`h1:d08g8WaT6nnyvg9uMm8K9zMYyDjfKyj3170AtPRuVU0=`,
`h1:YEcBbO/FB+5M1IZNBP9FO3J9281zgPAreiI1oqg8nDw=`.
İndirilen go.mod module bildirimleri ve sürümlü kaynak dizinleri doğrulandı.
Contrib adapter go.mod'u doğrudan fasthttp/websocket v1.5.8 kullanır;
Fiber alt sınırı v2.52.6 olsa da repository'nin mevcut seçimi v2.52.9'dur.
Başka sürüm indirilmedi. Ayrı cache'te yalnız bu üç kaynak modülü mevcut.
Origin alanındaki GitHub URL'leri metadata'dır; GitHub erişimi yapılmadı.

İndirme sonrası **35/35 hash aynıydı**. Kod geçişinden sonra yalnız üç
go.mod ve üç go.sum değişti: broadcaster'ın 3 require + 6 checksum satırı
silindi. Diğer 29 dosya, ORM kayıtları, go.work ve go.work.sum değişmedi.
Sonraki bütün inceleme/testler `GOPROXY=off`, `GOSUMDB=off`,
`GOTOOLCHAIN=local`, workspace'te `GOFLAGS=-mod=readonly` ile yürütüldü.

### Sabit kaynaklardan doğrulanan auth ve transport sözleşmesi

Kaynak yolları yukarıdaki cache içinde ilgili `module@version` altındadır.
Kanıt güveni yüksek/statik; gerçek transport testinin yerine geçmez.

- **contrib/websocket v1.3.4, websocket.go:New:** upgrade öncesinde
  `VisitUserValues` ile key/value ayrı `conn.locals` map'ine alınır. Interface
  değeri kopyalanır; pointer/map/slice içerikleri deep-copy değildir. Bu nedenle
  uygulama ortak validator'ın `strconv.FormatInt` ile ürettiği sahip olunan
  immutable `notify.UserID` değerini taşır; struct pointer'ı veya Fiber request
  context'i taşınmaz.
- **Aynı dosya, Conn/Locals/releaseConn:** fasthttp WebSocket Conn embed edilir;
  ReadMessage/WriteMessage/SetWriteDeadline/Close doğrudan promote edilir.
  Handler dönüşünde wrapper pool'a döner ve embedded Conn nil yapılır. Bu yüzden
  serve dönmeden writer ve watcher bitişi beklenir; pooled wrapper'a geç erişim
  yoktur. Cookie'ler de ayrı string map'e kopyalanır; yeni handler bunları tekrar
  parse etmez. UID yalnız HTTP CheckAuth sonucundan gelir.
- **Fiber v2.52.9, ctx.go:Locals/ReleaseCtx:** locals request UserValue'da tutulur;
  context pool'a geri verilir ve fasthttp alanı temizlenir. HTTP Ctx'yi socket
  ömrüne taşımak güvenli değildir. String anahtar, adapter'ın VisitUserValues
  kopyalamasıyla uyumludur; socket'te named UserID assertion zorunludur.
- **Upgrade başarısızlığı:** contrib `New` callback'i yalnız başarılı Upgrade
  yolunda çağrılır; registration/watchers henüz yoktur. Wrapper'ın acquireConn
  nesnesi başarısız upgrade yolunda pool'a iade edilmiyor; GC'ye bırakılır,
  owned socket/goroutine edinimi yoktur. Library davranışı değiştirilmedi.
- **fasthttp/websocket v1.5.8, doc.go:Concurrency:** bir reader + bir writer;
  Close ve WriteControl diğer metotlarla concurrent kullanılabilir.
- **conn.go:Close/write/WriteMessage/SetWriteDeadline:** Close doğrudan
  underlying net.Conn.Close çağırır; Go net.Conn sözleşmesi pending read/write'ı
  hata ile sonlandırır. WriteMessage text byte'larını frame'e yazar; WriteJSON
  ayrı encode yoludur ve kullanılmaz. Deadline önce writer alanına kaydolur,
  write sırasında underlying socket'e uygulanır. Timeout sonrası bağlantı
  yeniden kullanılamaz. Her Send deadline'ı yeniler; sıfırlama gerekmez.
  Uygulamanın underlying net.Conn'a doğrudan erişmesine gerek yoktur.
- **conn.go:SetPingHandler/SetCloseHandler/SetPongHandler:** varsayılan ping/close
  yanıtları WriteControl kullanır, pong no-op'tur. Ayrı application writer
  oluşturmazlar; kontrol yazımları library mutex'i ile koordine edilir.
- **server_fasthttp.go:FastHTTPUpgrader.Upgrade:** hijack sonrası WebSocket
  oluşturulur ve HTTP deadline'ları temizlenir. Owned adapter kendi deadline ve
  close sahipliğini kurar. Default contrib panic recovery ham panic/stack basıp
  WriteJSON çağırdığı için dar güvenli RecoverHandler ile değiştirildi.

### Yeni çağrı zinciri, kimlik ve registration

`main.run/openHub → notificationhub.New(32) → Utilities.NotificationHub
(notify.Hub) → notificationevent.Publish → Hub.Broadcast → registration writer
→ notificationws.transport.send → WriteMessage(TextMessage, payload)`.
Somut hub controller'a sızmaz; AppState.Broadcaster ve dış constructor yok.
Yeni modül/dependency yok; adapter mevcut post modülünün dar alt paketidir.

`PanelAuth → NotificationWebsocket auth callback → lib.CheckAuth(HTTP Ctx) →
positive canonical notify.UserID locals → contrib upgrade map copy → identity validation →
crypto/rand.Reader ile 32 byte / 64 hex ConnectionID → Hub.Register → read loop`.
Auth hatası veya geçersiz UID upgrade'i reddeder; socket tarafında ortak
positive canonical kontrolü register öncesinde tekrarlanır. Role ve BranchID snapshot'ı boş bırakılır;
alıcı rol/şube/permission sorguları güncel DB'den server UserID ile yapılır.
JWT rolü authoritative yapılmadı. UID hiçbir ConnectionID'den türetilmez.
Client JSON Uid alanı compatibility için kalır; dolu mismatch dahil **yok sayılır**.
Başka kullanıcı seçemez, kendi gönderimini engelleme/yönlendirme girdisi değildir.

Metadata.Protocol daima sunucu sabiti `kullanici`. Header yalnız eski event
seçimi için allowlist enum'a çevrilir; rol/UID değildir. Eski alıcı eşliğini
korumak için yalnız `kullanici` header modu `notifications` odasına kaydolur;
randevu/İK/iletişim/unknown gönderici bağlantıları `notification-events`
odasında aynı hub lifecycle'ına kaydolur. Bu ikinci odaya yayın yapılmaz.
Böylece producer'lar sabit protocol yüzünden yeni alıcıya dönüşmez; kapanışta
onlar da hub tarafından kapatılır. Event header'ının client kontrollü olması
SEC-007/N08 kapsamında açık kalır. Frontend/payload/rol politikası değiştirilmedi.

Register başarısızsa socket kapanır, watcher başlamaz. Başarıda tek watcher
hub Send context'inin iptalini veya Registration.Done'u izler; idle read'i de
kapatır. Send başına goroutine yoktur. Read disconnect, writer error, queue
overflow, text sonrası return, panic, explicit unregister ve shutdown aynı
idempotent cleanup'a gider. Cleanup unregister → physical close → writer Done
→ watcher Done sırasındadır; double physical close sync.Once ile önlenir.
Core'un generation-specific registration ve room boşaltma davranışı kullanılır.

### Alıcı matrisi, JSON ve davranış eşliği

Aşağıdaki tarihsel altı event matrisi halen alıcı iş kuralıdır; ConnectionID
bug'ları düzeltilerek uygulanır. `notificationevent` gerçek production
predicate'leri server UID alan Lookup/Permission callback'leriyle çalışır.
Appointment: admin/moderator veya santral+users.sid; application:
admin/moderator/ik; new request: admin/moderator veya UID+olay SID+can_view;
contact/delete/status: bütün authenticated kullanici subscriber'ları.
İK'nın permission satırıyla yeni randevu olayı alabilmesi ve moderator'ün İK
olayı alması korunmuştur; hedef güvenlik politikasına geçiş N08'dedir.

İK alıcı sorgusu Count(role allowlist) yerine UID ile Select(role) ve aynı
allowlist helper'ıdır. Repository schema.sql users.uid PRIMARY KEY tanımlar;
schema çalıştırılmadı ve production eşliği UNKNOWN. Sorgu sayısı değişmez.
Yeni talep yolunda önce yok sayılan Execute/Rows hataları artık deny/no publish
olur; hata içeriği taşınmaz. JSON parse hatasında socket event'i işlenmez.
Geçerli payload'ın bildirim metni, linki ve DB insert alanları korunur.

Outbound üç genel event Message(uid,message,request_link), NewRequest'in 10
alanı, Deleted(type,rrid), Status(type,rrid,new_status) aynı JSON alanları ve
type değerleriyle bir kez marshal edilir. Result wrapper yok; empty alanlar
kalır, helper'a typed nil verilirse JSON null olur (production caller'ları
dolu value verir). Map'ten struct'a geçiş property sırasını değiştirebilir;
JSON nesne/alan/değer eşliği test edilir, property byte sırası garantisi yoktur.
Hub byte kopyalar; transport yeniden encode etmez. Publish hatası sabit stage
olarak loglanır; primary DB sonucu rollback/HTTP failure'a çevrilmez.

Predicate başına sorgu maliyeti önceki matristeki gibidir (aynı kullanıcının
çoklu socket'lerinde tekrar edilir). Callbacks hub lock'u dışında; concurrent
broadcast'lar sorguları eşzamanlı çalıştırabilir. Shared legacy ORM concurrency
garantisi UNKNOWN; genel optimizasyon/ORM değişimi yapılmadı. Core panic'i
güvenli ErrPredicatePanic'e çevirir; callback'ler raw DB error loglamaz.

### Composition root ve shutdown

Edinim legacy DB → owned DB → hub → HTTP; cleanup HTTP → hub → owned DB →
legacy DB. Tek hub, 32 bekleyen mesaj + bir aktif write/client. Kapasite
best-effort kısa bildirim patlamalarını sınırlamak için açık sabittir;
production kapasite ölçümü değildir. Sıfır/geçersiz kapasite sabit startup hatası.
HTTP budget 10 saniye; hub için bağımsız yeni Background context ile 5 saniye.
Send deadline en fazla 5 saniye veya daha erken context deadline'ıdır.
Hub hatasında DB cleanup devam eder; listen → HTTP shutdown → hub shutdown
hataları sabit stage'lerle, primary önce olacak şekilde birleştirilir.
Her kaynak bir kez kapanır. Context'i ihlal eden sentetik sender timeout'u
başarı sayılmaz; testte kaynak serbest bırakılarak writer sonlandırılır.
Hub shutdown yalnız hub writer'larını bekler; mevcut ayrı DB broadcast
goroutine'leri veya hâlen callback içinde çalışan ORM işlerinin drain garantisi
yoktur. Bu process-local best-effort sınırı, tam runtime/DB kabulünde gözlenmelidir.

### Çalışan testler ile BLOCKED kontrollerin ayrımı

Tüm testler Go 1.25.1 Windows/amd64; gerçek DB/SMTP/production yok.
GOCACHE `%TEMP%/nivgoz-n04a-gocache`.

| Kontrol | Sonuç |
| --- | --- |
| notify, notificationhub, notificationevent testleri `-count=20 -timeout=60s` | PASS |
| Gerçek production identity.go + transport.go + session.go dosyaları, fake Socket ile testler; wiring/source AST/statik kontrolleri `-count=20` | PASS; Fiber runtime kanıtı değil |
| Hub concurrency ve adapter/session `-count=20 -parallel=8 -cpu=1,4` | PASS |
| lifecycle.go + lifecycle_test.go + notification_lifecycle_test.go `-count=20`; aynı CPU/paralellik koşumu | PASS; uygulama/DB başlatmadan gerçek lifecycle fonksiyonu |
| Yukarıdaki saf paketler/dosya listeleri için hedefli go vet | PASS |
| notify/hub/notificationevent `go list -deps` | PASS; owned + stdlib kapanışı |
| Tüm yerel production paket import graph'ı AST DFS | PASS; cycle yok; external derleme yerine geçmez |
| Native post, randevular, main, notificationws derleme kontrolü `-run '^$'` | BLOCKED: legacy neormgo/v2 v2.0.2 metadata'sı cache'te yok, GOPROXY=off |
| İzole gerçek `TestReal*` transport/Fiber testleri | BLOCKED: aşağıdaki izin dışı transitif girdiler eksik; hiçbiri indirilmedi |
| Race detector | BLOCKED: -race requires cgo; kurulum yapılmadı |
| Aktif dış broadcaster import/require/checksum, ID split/DB UID misuse taraması | PASS; sıfır |
| git diff --check / manifest diff | PASS; dependency farkı yalnız kanıtlanmış 9 satır |

Runtime saf testler: raw text, double encoding yokluğu, bounded deadline,
serial writer, context/hub shutdown/explicit unregister, idle shutdown,
write/deadline error ve panic, close yarışları, watcher bitişi, eksik/yanlış
typed UID, crypto-random hata, duplicate ID, producer izolasyonu, disconnect,
client mismatch, rol/şube/UID/protocol/permission allow-deny kombinasyonları,
marshal/nil/empty/JSON alanları, 8 birleşik lifecycle hata senaryosu, invalid
hub config, expired HTTP budget ve broken sender timeout'u kapsar.

`fiber_integration_test.go` gerçek API ile net.Pipe blocked read/write
cancellation, Fiber auth locals aktarımı, tek text encode, auth reddi ve
başarısız upgrade testlerini içerir; **çalışmadılar ve derlenmiş sayılmazlar**.
İzole GOPATH yalnız repository owned paketleri ile üç onaylı kaynağa junction
bağlar; kaynak kopyası/dependency taklidi yok. Eksik doğrudan girdiler:
`klauspost/compress/flate`, `savsgio/gotils/strconv`, `valyala/fasthttp`,
`golang.org/x/net/proxy`, `valyala/bytebufferpool`, `google/uuid`,
`mattn/go-colorable`, `mattn/go-isatty`, `mattn/go-runewidth`.
Bunların ek transitif kapanışı da henüz doğrulanmadı. Üç kaynak için verilen
izin bu eksikleri indirme izni olarak genişletilmedi.

### Dependency kesimi, kalan güvenlik ve kabul kararı

Retired/removed: dış broadcaster'ın dört production importu, somut tipi,
constructor/consumer API kullanımı ve üç go.mod + üç go.sum içindeki toplam
dokuz kayıt kaldırıldı. go.work.sum'da kayıt yoktu. Aktif kaynak/manifestte
sıfır referans; tarihsel migration planındaki retired/removed kayıt korunur.
ORM kayıtları değişmedi; bu tam paketlerin ayrı blokajıdır. Binary üretilmediği
için build metadata acceptance iddia edilmez.

SEC-003, SEC-007, FLOW-001 açık. Client event içeriği ve HTML/innerHTML,
notification DB metni, public JS'nin korunan WS'ye bağlanması ve session/role
yenilemesi N08/security kapsamındadır. UI ve DB şeması değişmedi.
N09 broadcaster kesimini bu kanıtla takip edebilir; kalan ORM kesimi ayrı.
N10 gerçek Fiber API/transport/auth runtime, race, tam paket ve izinli DB
kanıtını tamamlamalı. Rollback yalnız doğrulanmış owned sürüme dönüş/ileri
düzeltme; retired dependency yeniden eklenemez.

**Kaynak incelemesi ve kod review'ına hazır; N04B'nin tam test/kabul kapanışı
henüz hazır değil. Runtime/release BLOCKED, production UNKNOWN.**
Stage, commit, push, gerçek uygulama, DB, SMTP, migration veya deploy yapılmadı.

### Güncel dosya/diff teslim kaydı

Değişen tracked dosyalar (13): N04A raporu; post/post.go,
post/randevular/randevular.go; main/main.go, main/lifecycle.go,
main/lifecycle_test.go; models/models.go; post/main/models modüllerinin üç
go.mod ve üç go.sum dosyası. Silinen dosya yok.

Yeni dosyalar (12): bu N04B raporu; `post/notificationevent/events.go`,
`events_test.go`; `post/notificationws/transport.go`, `session.go`, `fiber.go`,
`transport_test.go`, `fiber_integration_test.go`, `wiring_test.go`,
`identity.go`, `identity_test.go`;
`main/notification_lifecycle_test.go`. Go yolları `fiber-v2/controllers/`
altındaki post ve `fiber-v2/` altındaki main'e göredir.

Son tracked `git diff --stat`: **13 files changed, 251 insertions(+),
325 deletions(-)**. Untracked 12 yeni dosya bu sayıya dahil değildir.
`git diff --check` temiz; yeni dosyalar ayrıca final LF/trailing whitespace
kontrolünden geçti. Gofmt değişen Go dosyalarında çalıştı; post.go içindeki
ilgili handler/importlar formatlı, görev dışındaki eski EditHomepageContent
girintisi baseline gibi korundu. Bu nedenle bütün post.go'ya gofmt-d uygulamak
yalnız o önceden var olan girinti farkını gösterebilir.

```text
 M docs/ai/N04A_NOTIFICATION_HUB.md
 M fiber-v2/controllers/post/go.mod
 M fiber-v2/controllers/post/go.sum
 M fiber-v2/controllers/post/post.go
 M fiber-v2/controllers/post/randevular/randevular.go
 M fiber-v2/main/go.mod
 M fiber-v2/main/go.sum
 M fiber-v2/main/lifecycle.go
 M fiber-v2/main/lifecycle_test.go
 M fiber-v2/main/main.go
 M fiber-v2/models/go.mod
 M fiber-v2/models/go.sum
 M fiber-v2/models/models.go
?? docs/ai/N04B_NOTIFICATION_HUB_WIRING.md
?? fiber-v2/controllers/post/notificationevent/
?? fiber-v2/controllers/post/notificationws/
?? fiber-v2/main/notification_lifecycle_test.go
```

---

## Tarihsel ilk inceleme — kaynak indirme onayından önce

Tarih: 2026-09-18. **N04B TAMAMLANMADI; IMPLEMENTATION BLOCKED.**
Başlangıç dalı `nivgoz-professional-v2`, HEAD
`ef3c78ba5c890f3e32dce8f07466862071718395`; çalışma ağacı temizdi.
Karar: Kullanıcının N04B görevindeki güvenli durma kuralı uygulandı.
Bu rapor bir production entegrasyonu veya güvenlik kapanışı değildir.

## 1. Blokaj ve kanıt sınırı

Görev kuralı: “Mevcut WebSocket library’sinin cancellation/close API’si
doğrulanamıyorsa” spekülatif kod yerine dur ve raporla.

Manifestteki mevcut transport sürümleri:
`github.com/gofiber/contrib/websocket v1.3.4` ve
`github.com/fasthttp/websocket v1.5.8`.
`go env GOMODCACHE` sonucu `C:\Users\DELL\go\pkg\mod`.
Bu cache içinde iki sürümün de kaynak dizini ve `cache/download` altındaki
sürüm ZIP'i yok (`Test-Path`: dört yol için false). Önceki izole
`%TEMP%/nivgoz-n03b-modules` ve `nivgoz-n03b-gopath/src/github.com`
alanlarında yalnız `lib` sağlayıcısı mevcut; transport kaynağı bulunmadı.

Bu yüzden upgrade locals/cookie kopyalaması, concurrent Close'un blocked I/O'yu
sonlandırması, kontrol frame handler'ları ve writer sahipliği kullanılan
sürümün kaynağından kanıtlanamadı. Native `go list` ayrıca eksik legacy ORM
metadata'sında duruyor; bu hata tek başına WebSocket kaynağının yokluğunun
kanıtı değildir, filesystem kontrolleri ayrı yapıldı.

Production Go kodu, frontend, manifest/checksum değiştirilmedi. Eski
broadcaster kaynağına erişilmedi; hiçbir dependency indirilmedi, `go mod tidy`
çalıştırılmadı. `.env`, secret değerleri, DB, SMTP ve production kullanılmadı.
Devam için mevcut sabit sürümlerin doğrulanabilir yerel kaynakları ve gerekli
izinli transitif girdileri gerekir; eski ORM/broadcaster temin edilmemelidir.

## 2. Çağrı zinciri ve authenticated identity

Mevcut zincir: `main.newHTTPServer → dış constructor → AppState.Broadcaster →
NotificationWebsocket / AddRandevuRequest / DeleteRandevuRequest /
ToggleRandevuRequestStatus → room.BroadcastIf`. N04A owned hub'ının production
consumer'ı yoktur.

Hedef, henüz uygulanmayan zincir: `main.run lifecycle → notificationhub.New →
Utilities içindeki notify.Hub → consumer Broadcast → registration writer →
dar notificationws transport → WebSocket text frame`.
Uygun aday yer `controllers/post/notificationws` (`post/notificationws`):
mevcut post modülü WebSocket dependency'sini zaten taşıyor; helper'ın kök
post/lib/models paketlerini import etmemesi gerekir. Bu aday için derlenmiş
adapter veya tamamlanmış import-cycle kanıtı yoktur.

Kaynak kanıtı (`fiber-v2/` altında):

- `main/main.go:148–157`: JWT, ban kontrolü ve handshake sırası.
- `baserouter/baserouter.go:169,272`: PanelAuth içeren backend grubu ve route.
- `lib/lib.go:GetJWT/CheckAuth`: HTTP veya WebSocket cookie'sinden JWT
  doğrulama ve UID çıkarma; client JSON kullanılmaz.
- `lib/lib.go:PanelAuthMiddleware`: JWT hatasında/boş UID'de devam etmez;
  başarılı UID'yi locals'a taşımaz.
- `lib/lib.go:WebsocketHandshake`: `protocol`, istemcinin
  `Sec-WebSocket-Protocol` header'ından locals'a yazılır; güvenilir rol değildir.
- `controllers/post/post.go:4443–4461`: socket cookie'sini yeniden okuma,
  hata halinde anonim random ID ile registration, başarıda `UID_random8`.

Dolayısıyla HTTP tarafındaki doğrulama kaynakta görülür, fakat güvenli
HTTP → socket aktarımı doğrulanmış sayılmaz. JWT rolü güncel DB rolü değildir;
SEC-003 açık kalır. Hedefte boş UID register edilmemeli; UserID doğrulanmış
server bağlamından gelmeli, ConnectionID bağımsız crypto/rand olmalı.
Client UID mismatch politikası ve crypto-random hata yolu henüz uygulanmadı
ve test edilmedi.

## 3. Kod değişikliğinden önce consumer matrisi

Tüm yolların odası `notifications`. Aşağıdaki “hedef” sütunu istenen geçişi
gösterir; mevcut kodda düzeltme yapılmış anlamına gelmez.

| Event / kaynak | Mevcut predicate ve kimlik kullanımı | Client kontrollü alanlar | Hedef typed karşılık / parity / açık bug |
| --- | --- | --- | --- |
| Genel randevu mesajı; `NotificationWebsocket`, 4495–4608 | Giriş protocol=randevu, connection ID != message.uid; alıcı Data=kullanici; users.uid=**tam c.Id** ile role/sid sorgusu; admin/moderator veya santral+users.sid=talep.sid | Header protocol, uid, iç message JSON, kişi/rrid/sid/tarih alanları | DB anahtarı Metadata.UserID; alıcı Metadata.Protocol; rol/şube DB'den. Tam ConnectionID/UID uyuşmazlığı N04B'de düzeltilecek. İçerik/event güveni N08 |
| Genel iş başvurusu mesajı; aynı handler, 4611–4681 | Giriş protocol=is-basvurusu, connection ID != message.uid; Data=kullanici; users.uid=**tam c.Id** ve role admin/moderator/ik count | Header protocol, uid, iç message JSON, ad/soyad/jaid | Metadata.UserID/Protocol; DB rol sorgusu korunmalı. Moderator alıcılığı hedef güvenlik politikasıyla uyuşmaz; N04B'de gizlice değiştirilmez |
| Genel iletişim mesajı; aynı handler, 4684–4729 | Giriş protocol=iletisim, **message.uid boş**; bütün Data=kullanici alıcıları; rol/şube sorgusu yok | Header protocol, uid, iç message JSON, ad/soyad/crid | Server metadata protokolü; client UID routing kararı kaldırılmalı. Yeni rol/şube filtresi varsayılmamalı; içerik/event yetkisi N08 |
| new_randevu_talebi; `AddRandevuRequest`, 532–578 | Data=kullanici; **Split(c.Id,"_")[0]** ile UID; DB user role admin/moderator veya aynı UID+talep SID+can_view=true izin satırı | HTTP form girdileri; yayın içeriği insert sonrası DB SELECT sonucundan | Metadata.UserID/Protocol; ConnectionID ayrıştırması kaldırılmalı. Fallback santral rolüne özgü değil: İK/başka rol de izin satırıyla seçilebilir; parity korunmalı |
| randevu_talebi_silindi; `DeleteRandevuRequest`, 722–733 | Bütün Data=kullanici; UID/rol/şube/permission filtresi yok | HTTP route rrid; mutation handler kontrolü ayrı | Metadata.Protocol; yeni alıcı iş kuralı eklenmemeli. Rol/şube genişliği N08 |
| randevu_talebi_status; `ToggleRandevuRequestStatus`, 907–918 | Bütün Data=kullanici; UID/rol/şube/permission filtresi yok | HTTP route rrid; mevcut handler'ın hesapladığı NewStatus | Metadata.Protocol; rol/şube genişliği N08; commit hatasının yok sayılması ayrı mevcut borç |

Client `Uid` üç socket event'inde broadcast'ın çalışıp çalışmayacağını
etkiliyor; doğrulanmış UID yerine kabul edilemez. Randevu/İK koşulu bir alıcı
self-exclusion filtresi değildir: sender connection ID'siyle client alanını
karşılaştırır. Bu fark yeni test beklentilerine doğru aktarılmalıdır.

### Alıcı politika tablosu (statik; runtime allow/deny testi değildir)

`K`: protocol=kullanici; `P`: DB'de alıcı UID + olay SID + can_view=true
izin satırı; `S`: DB users.sid olay SID'sine eşit. Sorgu hatası/boş kullanıcı
ilk iki socket olayında deny; Add yolunda Execute/Rows hataları yok sayılır.

| Event | Admin | Moderator | Santral aynı şube | Santral farklı şube | İK | Farklı UID / BranchID / permission | Protocol eşleşmez |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Socket randevu | K | K | K ve S | Hayır | Hayır | UID DB lookup anahtarı; permission kullanılmaz; SID DB'den | Hayır |
| Socket İK | K | K | Hayır | Hayır | K | Şube/permission etkisiz; UID DB lookup anahtarı | Hayır |
| Socket iletişim | K | K | K | K | K | UID/şube/permission filtresi yok | Hayır |
| new_randevu_talebi | K | K | K ve P | K ve P | K ve P | Permission true/false belirleyici; users.sid kontrolü yok; farklı UID yalnız kendi izin satırıyla | Hayır |
| Silme | K | K | K | K | K | UID/şube/permission filtresi yok | Hayır |
| Status | K | K | K | K | K | UID/şube/permission filtresi yok | Hayır |

İlk iki satır DB sorgusunun ifade ettiği politikadır; mevcut tam ConnectionID
lookup bug'ı gerçek teslimatı bozabilir. Eski broadcaster runtime davranışı
UNKNOWN. Metadata.Role/BranchID snapshot'ı authoritative yapılmamalı.

Maliyet: socket randevu/İK broadcast'ı seçime aday her bağlantı için bir DB
sorgusu; Add için bir talep sorgusu ve her aday bağlantıda bir rol + gerekirse
bir permission sorgusu. Aynı kullanıcıya ait sekmeler sorguyu tekrarlar.
N04A predicate'leri registry kilidi dışında çalıştırır; farklı broadcast'lar
concurrent olabilir. Paylaşılan legacy ORM'nin concurrency güvenliği UNKNOWN;
bu görevde geniş optimizasyon veya ORM taklidi yapılmadı.

## 4. JSON ve caller sözleşmesi

Genel üç event `models.WebsocketMessage`: `uid`, `message`, `request_link`;
outbound uid boş string; type alanı yok. Inbound message, ikinci JSON metnini
taşıyan string. `omitempty` yok; boş string alanları çıktıda kalır.

Add JSON alanları: `type,rrid,patient_first_name,patient_last_name,
patient_phone,message,created_at,status,sube_name,sid`; type=new_randevu_talebi.
Silme: `type,rrid`; status: `type,rrid,new_status`. Dört outbound şekil de
doğrudan üst nesnedir; `result` wrapper'ı yok. Dolu yerel struct/map marshal
edilir, nil üst payload üretilmez; nil DB değer dönüşümleri `lib.String` gibi
mevcut helper'lara bağlıdır. Yeni serialization parity/marshal hata testleri
BLOCKED / NOT IMPLEMENTED; mevcut kodun byte eşliği iddia edilmez.

Socket event'leri insert/LastInsertId/marshal hatasında loglayıp devam eder;
broadcast sonucu gözlenmez. Add, insert/e-posta sonrasında goroutine içinde
marshal hatasında döner. Silme commit başarısından sonra yayın goroutine'i
başlatır. Status, kontrol edilmeyen Commit çağrısından sonra başlatır.
Bildirim hatası HTTP caller'ın kalıcı kayıt başarı sözleşmesini değiştirmemeli.
Bu görevde payload, DB notification içeriği veya frontend rendering değişmedi.

## 5. Adapter, registration ve lifecycle — uygulanmadı

İstenen transport: registration başına fiziksel socket ve en fazla bir
cancellation watcher; hub writer'ından seri raw []byte text write, sınırlı
deadline, idempotent Close, blocked read/write iptali, ham hata/payload sızıntısı
olmaması. Bir reader + bir writer ve ping/pong/close kontrol yazımları mevcut
kütüphane kaynağı olmadan doğrulanamadı. Adapter/helper üretilmedi.

Mevcut socket handler text sonrasında explicit unregister olmadan dönebiliyor;
read error/close yolları RemoveById/RemoveIf kullanıyor. Hedef registration
handle ile defer cleanup olmalı; duplicate ID, register hata, panic, disconnect,
writer failure ve shutdown yarışları N04B'de henüz test edilmedi.

Mevcut edinim legacy DB → owned DB → HTTP; cleanup HTTP → owned DB → legacy.
Hub constructor HTTP kurulumunun içinde ve hub shutdown yok. Hedef edinim
legacy → owned → hub → HTTP; cleanup HTTP → hub → owned → legacy.
Hub için HTTP'den bağımsız bounded shutdown context'i, sabit hata stage'leri,
primary hata önceliği ve DB cleanup devamı uygulanmadı. Queue kapasitesi için
production değeri seçilmedi; N04A pozitif kapasite şartı korunuyor.

## 6. Dependency envanteri ve farklar

Retired/removed geçiş kaydı: kaldırılması istenen dış broadcaster bu blokaj
nedeniyle **henüz kaldırılmadı**. Aktif kaynak importu dört dosyada sürüyor:
`models/models.go`, `main/main.go`, `controllers/post/post.go`,
`controllers/post/randevular/randevular.go`.

Mekanik kaldırmaya aday kanıtlanmış kayıtlar: main, models ve controllers/post
altındaki üç go.mod'da birer v0.2.0 require; aynı üç dizinin go.sum dosyasında
ikişer checksum. `go.work.sum` dahil diğer taranan manifest/checksum'larda
broadcaster eşleşmesi yok. Kod geçişi yapılmadığı için dokuz satır korundu.
ORM kayıtları değişmedi. Manifest/checksum/workspace diff'i sıfır.

## 7. Çalıştırılan doğrulamalar

Go 1.25.1, Windows/amd64; `GOPROXY=off`, `GOSUMDB=off`,
`GOTOOLCHAIN=local`, `GOFLAGS=-mod=readonly`.
GOCACHE yalnız `%TEMP%/nivgoz-n04a-gocache` olarak ayarlandı.
Ağdan kaynak tamamlama yok. Mevcut production helper'ları çalıştırıldı;
test-only mantık kopyası veya yeni dependency yok.

| Kontrol | Sonuç ve kapsam |
| --- | --- |
| `go test -count=20 -timeout=60s ./models/notify ./lib/notificationhub` | PASS; mevcut N04A regresyonları |
| Hub `-run 'TestConcurrent\|TestPredicate\|TestShutdown\|TestTransport\|TestBounded\|TestPayload' -count=20 -parallel=8 -cpu=1,4 -timeout=60s` | PASS; fake Send ile core concurrency, gerçek socket değil |
| `go vet ./models/notify ./lib/notificationhub` | PASS |
| `go list -deps ./models/notify ./lib/notificationhub` | PASS; non-stdlib yalnız bu iki owned paket; cycle yok |
| main dizininde `go test -count=20 -timeout=60s lifecycle.go lifecycle_test.go` | PASS; mevcut DB/HTTP lifecycle, hub eklenmiş lifecycle değil |
| main dizininde `go vet lifecycle.go lifecycle_test.go` | PASS |
| `go test -race ./models/notify ./lib/notificationhub` | BLOCKED: -race requires cgo; kurulum yok |
| Native transport `go list -e` ve `go test -run '^$' ./controllers/post ./controllers/post/randevular ./main` | BLOCKED: eksik legacy ORM v2.0.2, module lookup disabled by GOPROXY=off |
| N04A API/import AST testleri | PASS; yeni production wiring veya Fiber auth runtime kanıtı değil |
| N04B adapter/auth/recipient/serialization/lifecycle testleri | NOT IMPLEMENTED; güvenli durma nedeniyle |
| Dış broadcaster sıfır referans ve kimlik misuse | FAIL / mevcut 4 import, 9 manifest/checksum satırı ve yukarıdaki kimlik hataları sürüyor |
| Tam import graph / binary metadata | BLOCKED; tam paket derlenemiyor, binary üretilmedi |
| gofmt | N/A; Go dosyası değişmedi |

Statik consumer incelemesi, runtime teslimat/authorization/JSON testi gibi
sunulmaz. Uygulama, gerçek WebSocket, DB, SMTP, migration, UI veya production
çalıştırılmadı. N04B'nin tam paketleri ve runtime/release kabulü BLOCKED.

## 8. Kalan riskler, N09/N10 ve rollback

SEC-007 authenticated cross-user XSS, client event metni, DB'ye yazılan
notification içeriği ve panel `innerHTML` hâlâ açık. FLOW-001 public JS'nin
PanelAuth korumalı endpoint'e bağlanma denemesi sürüyor. SEC-003 token/session
yenileme, güncel rol/aktif kullanıcı kararı çözülmedi. N08/security paketine
bırakılan bu konular N04B ile kapatılmış sayılmamalı. Bu duruşta N04B'nin
kimlik düzeltmeleri de henüz uygulanmadı.

N09: broadcaster kesimi tamamlanmamış olarak takip edilmeli; ORM temizliği bu
görevin kapsamı değil. N10: izinli offline girdilerle tam derleme, gerçek
transport/auth aktarımı, race ve bütün recipient/serialization/lifecycle
matrisi tamamlanmalı. Production bilgisi UNKNOWN; release/deployment yok.

Rollback: yalnız iki dokümantasyon dosyası değişti, runtime geri dönüşü yok.
İleride kod geçişinde hub+dört tüketici+manifest tek uyumlu birim olmalı;
kaldırılan dış dependency/import tekrar eklenemez. Doğrulanmış owned sürüme
dönüş veya ileri düzeltme sınırı geçerlidir.

## 9. Dosyalar ve teslim durumu

- Yeni: `docs/ai/N04B_NOTIFICATION_HUB_WIRING.md` (bu rapor).
- Değişen: `docs/ai/N04A_NOTIFICATION_HUB.md` (dar route/middleware açıklaması).
- Silinen: yok. Go/JS/manifest/checksum/workspace değişikliği: yok.
- Stage, commit, push, pull, merge veya deploy yapılmadı.

Son `git diff --check` temiz; yeni rapor için `git diff --no-index --check`
whitespace çıktısı boş (yeni dosya farkı nedeniyle exit 1). Tracked
`git diff --stat`: `docs/ai/N04A_NOTIFICATION_HUB.md | 12 ++++++++++++`,
`1 file changed, 12 insertions(+)`. Untracked yeni rapor bu stat'a dahil
değildir. Manifest/checksum/workspace `git diff --numstat` çıktısı boş.
Son çalışma ağacı:

```text
 M docs/ai/N04A_NOTIFICATION_HUB.md
?? docs/ai/N04B_NOTIFICATION_HUB_WIRING.md
```

N04B tamamlandı mı? **Hayır.** Ön inceleme ve consumer matrisi kaydedildi;
kullanıcının açık güvenli durma koşulu uygulandı. Dokümantasyon üretimi
production wiring kabulü veya dependency kaldırılması değildir.
