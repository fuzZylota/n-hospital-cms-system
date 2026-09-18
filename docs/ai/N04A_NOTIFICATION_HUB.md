# N04A — notification sözleşmesi ve concurrency çekirdeği

Tarih: 2026-09-18. Başlangıç: `nivgoz-professional-v2`,
`6aa46ceac934d64dbc157da389f88d6ea8357588`, temiz çalışma ağacı.
Karar durumu: Kullanıcının N04A görevi kapsamında uygulama kararı;
production entegrasyonu veya alıcı iş kuralı değişikliği onayı değildir.

## Koddan önce çıkarılan tüketici envanteri

Kanıtlar mevcut tracked consumer kaynaklarıdır; güven yüksek (statik).
Eski paketin iç davranışı, teslimat/sıra/close garantileri ve production UNKNOWN.
Retired/removed geçiş kaydı: kaldırılması kararlaştırılan dış broadcaster hâlâ
production consumer'larında bulunur; N04A bu bağımlılığı kaldırmaz veya indirmez.
Doğrudan import eden dosyaların tamamı: `fiber-v2/models/models.go`,
`fiber-v2/main/main.go`, `fiber-v2/controllers/post/post.go`,
`fiber-v2/controllers/post/randevular/randevular.go`.

| Dosya / fonksiyon (fiber-v2 altında) | Oda | Connection kimliği / metadata | Register / unregister | Broadcast koşulu / mesaj | Sıralama / hata / belirsizlik |
| --- | --- | --- | --- | --- | --- |
| `main/main.go:newHTTPServer`, `models/models.go:AppState` | Başlangıçta yok | Dış somut broadcaster pointer'ı | `New()`; hub shutdown çağrısı yok | Yok | Hub lifecycle composition root'a bağlanmamış |
| `controllers/post/post.go:NotificationWebsocket` 4429–4488, 4730–4756 | `notifications`; `Handle`, `RoomById` | Auth başarılı: `UID_random8`; başarısız: `random10`. `Data` yalnız protocol string'i | Auth/protocol okunduktan sonra `room.Handle(c,id,protocol)`; read error/close frame: `RemoveById`, sonra `RemoveIf(IsRoomEmpty)` | Aşağıdaki üç text yolu | Read döngüsü seri. Text sonrası `break Loop` var, bu yolda explicit unregister/defer yok: doğrulanmış cleanup açığı; dış wrapper cleanup davranışı UNKNOWN. Binary/ping/pong işlenmiyor |
| Aynı fonksiyon, randevu text yolu 4495–4608 | Aynı | `Data == kullanici`; `c.Id` doğrudan DB UID olarak kullanılıyor | Yukarıdaki lifecycle | Protocol `randevu` ve `id != message.uid`; DB role admin/moderator veya santral + `users.sid == request.sid`; JSON `uid,message,request_link` | `UID_random8` ile UID eşitliği kurulması doğrulanmış kimlik uyuşmazlığı. JSON parse hataları yok sayılıyor; insert/marshal hatası loglanıp devam ediliyor; send sonucu gözlenmiyor |
| Aynı fonksiyon, iş başvurusu text yolu 4611–4681 | Aynı | `Data == kullanici`; `c.Id` doğrudan UID | Aynı | Protocol `is-basvurusu`, `id != message.uid`; DB admin/moderator/ik; aynı üç JSON alanı | Aynı kimlik uyuşmazlığı. Moderator alıcılığı hedef İK politikasından farklı; N04A düzeltmez |
| Aynı fonksiyon, iletişim text yolu 4684–4729 | Aynı | Yalnız protocol | Aynı | Protocol `iletisim`, `message.uid == ""`; bütün `kullanici` bağlantıları; aynı üç JSON alanı | Rol/şube filtresi yok; client event metni güven kaynağı. Mevcut güvenlik borcu |
| `controllers/post/randevular/randevular.go:AddRandevuRequest` 532–578 | Aynı; `Handle`, `RoomById`, nil guard | Protocol `kullanici`; UID `strings.Split(c.Id,"_")[0]` ile türetiliyor | Register/unregister yok | Insert/e-posta sonrası goroutine; DB admin/moderator veya UID+SID+can_view izin satırı; `new_randevu_talebi` JSON | Diğer goroutine broadcast'larıyla sıra tanımlı değil. UID ayrıştırma bağı; non-admin fallback açıkça santral rolünü aramıyor. DB hataları yok sayılabiliyor, marshal hatasında return |
| Aynı dosya: `DeleteRandevuRequest` 722–733 | Aynı | Protocol `kullanici` | Yok | Commit sonrası goroutine; tüm kullanici; `type:randevu_talebi_silindi,rrid` | Rol/şube filtresi yok; marshal/send hata sonucu ele alınmıyor |
| Aynı dosya: `ToggleRandevuRequestStatus` 907–918 | Aynı | Protocol `kullanici` | Yok | Commit çağrısı sonrası goroutine; tüm kullanici; `type:randevu_talebi_status,rrid,new_status` | Rol/şube filtresi yok; commit/marshal/send hata sonucu ele alınmıyor |

`new_randevu_talebi` alanlarının tamamı: `type,rrid,patient_first_name,
patient_last_name,patient_phone,message,created_at,status,sube_name,sid`.
`WebsocketMessage` alanları `uid,message,request_link`; inbound `message` içinde
ikinci bir JSON string'i var. JSON core dışında kalır.

Ek kanıt: `lib/lib.go:WebsocketHandshake` protocol'ü request header'dan locals'a
koyar; `baserouter/baserouter.go:272` route'u bağlar. Public
`fast-randevu.js`, `iletisim.js`, `insert-job-application.js` ilgili protocol ile
`uid:null,message:JSON.stringify(...)` gönderir, sonra close ister.
`static/js/panel/notifications.js` `kullanici` ile bağlanır; üç typed randevu
olayını ve genel message/request_link yapısını tüketir, close sonrası reconnect
yapar. Runtime teslimat sırası consumer'lardan kanıtlanamaz.

N04B route/middleware düzeltmesi (2026-09-18): Bu endpoint anonim değildir.
`main/main.go` global JWT/ban middleware'lerini ve notification handshake'ini
kaydeder; `baserouter/baserouter.go:BackendRouter` `/backend` grubunu
`lib.PanelAuthMiddleware()` ile kurar ve `/notifications` route'unu bu gruba
ekler. PanelAuth, `GetJWT(c)` hatası veya boş UID için isteği sonlandırır;
UID'yi locals'a yazmaz. N04A baseline handler'ı upgrade sonrasında cookie'yi
yeniden parse ediyordu. N04B'de `notificationws.Handler`, HTTP `CheckAuth(c)`
sonucunu upgrade öncesi cloned `notify.UserID` olarak string anahtarlı locals'a
yazar; socket handler yalnız bu typed değeri kabul eder. Kullanıcının dar kaynak
indirme onayıyla incelenen contrib/websocket v1.3.4 `New`, locals'ı ayrı map'e
kopyalar; Fiber context saklanmaz, socket içinde JWT tekrar parse edilmez.
İlk incelemenin kaynak yokluğu engeli kalktı; gerçek Fiber integration testleri
eksik transitif girdiler nedeniyle hâlâ BLOCKED. Public JS'nin bu korunan route'a bağlanması
`FLOW-001` borcudur; anonim handler erişimi kanıtı değildir. Ayrıntı ve güvenli
durma/devam kaydı: [N04B wiring raporu](N04B_NOTIFICATION_HUB_WIRING.md).

## Yerleşim ve karar

Saf sözleşme `models/notify`, somut çekirdek `lib/notificationhub`.
`notify` adı onaylı migration planı §5 ile aynıdır; yeni modül yoktur.
Import yönü yalnız `lib/notificationhub → models/notify → stdlib`.
Kök models/lib, Fiber, DB, ORM, SMTP ve WebSocket core'a girmez.

Bağlantı, kullanıcı, şube, rol ve protocol ayrı named tiplerdir. Metadata
değer olarak kopyalanır; map/slice/any yoktur. Role ve BranchID, consumer DB
lookup'larında görülen alanların typed temsilidir; register snapshot'ı güncel
yetkilendirme kanıtı değildir. Çoklu şube `can_view` politikası hub'a taşınmaz;
N04B uygulama katmanı güncel UID/izin kümesiyle predicate oluşturmalıdır.
ConnectionID hiçbir zaman parçalanmaz veya UserID olarak yorumlanmaz.

Transport, context alan ve error döndüren tek Send fonksiyonudur. Socket ve
close sahipliği adapter'dadır; adapter cancellation'ı send'i sonlandıracak
deadline/close ile uygulamak zorundadır. Hub keyfi bir callback'i zorla
sonlandıramaz. Shutdown timeout'u başarı sayılmaz; sonra yeniden beklenebilir.

## Exported API ve gerekçesi

| API | Gerekçe / sözleşme |
| --- | --- |
| `notify.RoomID` | Room-scoped dağıtım; kullanıcı/bağlantı kimliğinden ayrı |
| `notify.ConnectionID` | Tek oturum kimliği; UID dönüşümü veya parçalama yapılmaz |
| `notify.UserID` | Mevcut UID tabanlı alıcı filtrelerinin doğru girdisi |
| `notify.BranchID` | SID ile şube kapsamını açık temsil eder; BRID değildir |
| `notify.Role`, `notify.Protocol` | Mevcut role ve protocol kontrollerinin ayrı typed alanları; enum/rol politikası dayatmaz |
| `notify.Metadata` | `UserID,BranchID,Role,Protocol` value snapshot'ı; serbest Data/any veya pointer/slice içermez |
| `notify.Client` | `ID` ile `Metadata` ayrımı; register ve predicate girdisi |
| `notify.Predicate` | Typed client filtreleme; nil tümünü seçer; lock dışında çalışır |
| `notify.Send` | Context, opaque byte payload, error sınırı; serialization ve fiziksel close adapter'da |
| `notify.Hub.Register` | Client yaşam süresini başlatır; kayıt başına tek writer ve tek room |
| `notify.Hub.Broadcast` | Room-scoped admission, alıcı filtresi ve payload sahipliği |
| `notify.Hub.Shutdown` | Kalıcı kapatma ve writer bitişini context ile bekleme |
| `notify.Registration.Unregister` | ID yerine kayıt nesnesiyle idempotent cleanup; eski handle yeni kaydı silemez |
| `notify.Registration.Done` | Son send/queue cleanup tamamlanmasının gözlenmesi, adapter kaynak yaşam döngüsü |
| `notify.Registration.Err` | İlk terminal nedenin güvenli gözlenmesi; aktif veya explicit unregister için nil |
| `notify.BroadcastResult` | `Enqueued` queue kabulü; `Disconnected` bu çağrıda taşan client sayısı; delivery ACK değildir |
| `notify.ErrorCode`, `Error()` | Mutable error global'i yerine immutable, karşılaştırılabilir sabit hata; errors.Is desteği |
| `ErrInvalidConfig`, `ErrInvalidArgument` | Geçersiz kapasite; nil context/send veya boş room/connection kimliği |
| `ErrDuplicateClient`, `ErrClosed` | ID hâlâ writer'a ait; hub kalıcı kapalı |
| `ErrSlowClient`, `ErrSendFailed`, `ErrSendPanic`, `ErrPredicatePanic` | Kuyruk taşması ve callback failure sınıfları; hassas cause/panic değeri tutulmaz |
| `notificationhub.New(int)` | Pozitif queue kapasitesi zorunlu tek construction API; implementation tipi private |

Private implementation tiplerinin `String/GoString` metotları yalnız güvenli
formatlama içindir; `%v`, `%+v`, `%#v`, `%s` metadata/payload göstermez.
Somut tipler için compile-time Hub ve Registration assertion'ları vardır.
Room listesi, arbitrary Data, RemoveIf, RoomById veya dış API uyumluluk yüzeyi yoktur.

## Queue, ordering ve lifecycle modeli

- Client başına `capacity` bekleyen mesaj + en fazla bir dispatched send.
  Kapasite <= 0 reddedilir; sessiz default veya unbounded seçenek yoktur.
- Broadcast input byte'larını predicate öncesinde kopyalar; seçilen her client
  ayrıca ayrı kopya alır. Caller broadcast sürerken input'u değiştiremez;
  dönüşten sonra serbesttir. Send kendi kopyasını saklayabilir/değiştirebilir.
- Tek hub mutex'i registry, room üyeliği, admission ve terminal nedeni korur.
  Client mutex'i yoktur; ters lock sırası oluşmaz. Predicate, send ve context
  cancel çağrıları bu mutex dışında çalışır. Predicate'lerin kısa ve concurrency
  güvenli olması caller sözleşmesidir; tercihen yetki kümesi önceden hesaplanır.
- Snapshot → predicate → bütün admission aşamaları ayrıdır. Concurrent
  broadcast'ların sırası admission lock'una giriş sırasıdır; çağrı başlama
  sırası değildir. Aynı alıcılara kabul edilen concurrent mesajların göreli
  sırası tutarlıdır. Sequential broadcast FIFO korunur.
- Snapshot'tan sonra eklenen client mesajı almaz. Snapshot'taki kayıt kaldırılıp
  aynı ID tekrar kullanılırsa eski mesaj yeni generation'a gitmez.
- Full queue'da send beklenmez: ilgili kayıt routing'den ayrılır, cancellation
  gönderilir, pending mesajlar bırakılır; diğer client'ların admission'ı sürer.
  Bu removal `Disconnected`, terminal neden `ErrSlowClient` olarak gözlenir.
- Predicate panic'inde o broadcast hiçbir mesaj enqueue etmez; güvenli
  `ErrPredicatePanic` döner. Caller callback yan etkileri geri alınmaz.
  Send error/panic'i yalnız kendi client'ını çıkarır; ham error/cause saklanmaz.
- Data channel **kapatılmaz**. Böylece send/close yarışı yoktur. Routing'den
  ayrıldıktan sonra yeni enqueue olmaz; writer cancellation ile çıkar,
  buffered payload referanslarını temizler, queue/send referanslarını bırakır.
  `Done` ve hub completion kanalları mutex altında yalnız bir kez kapanır.
- Duplicate ConnectionID aynı/farklı odada reddedilir; retiring writer da ID'yi
  `Done` kapanana kadar tutar. Register idempotent replacement değildir.
  Unregister idempotenttir; ilk terminal neden korunur. Already-dispatched send
  unregister ile yarışarak dönebilir; Done sonrası send yoktur.
- Son client ayrılınca room kaydı silinir. Olmayan room başarılı sıfır sonuçtur;
  boş room ID ise invalid argument'tır.
- Shutdown yeni kayıt/gönderimi kalıcı kapatır, tüm kayıtları çıkarır, cancel
  eder ve writer bitişini bekler. Tekrarlanabilir; timeout sonrası yeniden
  beklenebilir. Non-nil canceled context de kapatmayı başlatır; nil context
  geçersizdir ve kapatmaz. Argument validation lifecycle kontrolünden önce gelir.
  Geçerli kapalı-hub işlemi `ErrClosed`; client context iptali standard
  `context.Canceled` / `context.DeadlineExceeded` kimliğini korur.
- Writer başına bir goroutine vardır; hub koordinatör goroutine'i, send başına
  goroutine, global registry, retry veya durable queue yoktur. Callback'in kendi
  Done'unu veya başarılı Shutdown'ı beklemesi yasaktır: kendi dönüşünü bekler.
  Shutdown caller-owned, hâlâ çalışan predicate'leri beklemez; admission aşamasına
  vardıklarında `ErrClosed` dönerler.

Kilit incelemesi: Registry/channel send/stop işlemleri aynı mutex altında;
writer'ın dequeue sonrası stop kontrolü de aynı mutex altında. Hiçbir blocking
transport çağrısı kilit içinde değil. `stopLocked` idempotent olduğundan eski
handle yeni generation'ın room üyeliğini silemez. `finish`, registry'den son
writer'ı silme ve Done kapatmayı aynı kritik bölgede yapar; duplicate register
eski send bitmeden kabul edilmez. Shutdown completion yalnız ilk boş kapanışta
veya son writer cleanup'ında kapanır. Bu statik kanıt race detector yerine
eşdeğer kanıt olarak sunulmaz.

## Runtime test matrisi

Testler gerçek `notificationhub.New` implementasyonunu kullanır; yalnız Send
uçları sentetiktir. Test-only hub algoritması yoktur. Sabit sleep yoktur;
kanal/barrier ve iki timeout/lifecycle ailesinde `testing/synctest` kullanılır.
5 saniyelik gerçek test timeout'ları sadece deadlock guard'dır, sıralama kanıtı
değildir. Parallel alt testlerin her birinin ayrı hub'ı vardır.

| İstenen senaryo | Runtime test (Test önekiyle) |
| --- | --- |
| 1. Tek client register/broadcast | `RegisterBroadcastAndRoomIsolation` |
| 2. Aynı room'da çok client | `RegisterBroadcastAndRoomIsolation` |
| 3. Farklı room izolasyonu | `RegisterBroadcastAndRoomIsolation` |
| 4. Typed metadata predicate | `TypedMetadataAndIdentityIsolation` |
| 5. UserID / ConnectionID ayrımı | `TypedMetadataAndIdentityIsolation`; ek reflection sözleşme testi |
| 6. Mesaj sırası | `FIFOAndSerialTransport`, `ConcurrentBroadcastOrderIsConsistentAcrossClients` |
| 7. Client'lar birbirini bloklamaz | `BoundedQueueSlowClientAndIndependentDelivery` |
| 8. Bounded queue | `BoundedQueueSlowClientAndIndependentDelivery` |
| 9. Slow client çıkarılır | `BoundedQueueSlowClientAndIndependentDelivery` |
| 10. Diğer client etkilenmez | `BoundedQueueSlowClientAndIndependentDelivery` |
| 11. Send hatasında çıkarma | `SendFailuresAreSafeAndRemoveClient` error/panic alt vakaları |
| 12. Explicit unregister | `UnregisterDuplicateAndGenerationIsolation` |
| 13. Tekrarlı unregister | `UnregisterDuplicateAndGenerationIsolation`, `ConcurrentLifecycleAndChannelSafety` |
| 14. Duplicate ID | `UnregisterDuplicateAndGenerationIsolation`: aynı/farklı room, retiring writer, güvenli reuse |
| 15. Room boşaldığında davranış | `UnregisterDuplicateAndGenerationIsolation`, `ConcurrentLifecycleAndChannelSafety` |
| 16. Olmayan room | `MissingRoomAndPredicatePanicAreNoOps` |
| 17. Hub shutdown | `ShutdownActiveSendAndRepeatedShutdown`, `ShutdownAllIdleWritersComplete` |
| 18. Tekrarlı shutdown | `ShutdownActiveSendAndRepeatedShutdown`, `ConcurrentShutdownAndAdmission` |
| 19. Shutdown sırasında send | `ShutdownActiveSendAndRepeatedShutdown`, `ShutdownTimeoutCanBeRetried` |
| 20. Shutdown sonrası register | `ShutdownActiveSendAndRepeatedShutdown`, `ConcurrentShutdownAndAdmission` |
| 21. Shutdown sonrası broadcast | `ShutdownActiveSendAndRepeatedShutdown`, `ShutdownDoesNotWaitForPredicateOrHoldItsLock` |
| 22. Concurrent lifecycle | `ConcurrentLifecycleAndChannelSafety`, `ConcurrentShutdownAndAdmission` |
| 23. Predicate lock dışında | `PredicateOutsideLockAndSnapshotGeneration`, `ShutdownDoesNotWaitForPredicateOrHoldItsLock` |
| 24. Payload/metadata mutation izolasyonu | `PayloadMutationIsolation`, `TypedMetadataAndIdentityIsolation` |
| 25. Transport seri | `FIFOAndSerialTransport`, `ConcurrentLifecycleAndChannelSafety`; reentrant `TransportCanUnregisterWithoutHubLock` |
| 26. Queue/close yarışında panic yok | `ConcurrentLifecycleAndChannelSafety`, `ConcurrentShutdownAndAdmission` |
| 27. Error/String veri sızdırmaz | `SendFailuresAreSafeAndRemoveClient`; ek `ErrorCodesAreComparableAndFixed` |
| 28. Writer tamamlanması | `ShutdownAllIdleWritersComplete` (64 idle writer), active-send/slow-client/unregister Done kontrolleri |
| 29. Invalid config | `InvalidConfigurationAndArguments` |
| 30. Context/lifecycle cancellation | `ClientContextCancellation` idle/active, `ContextDeadlineAndCanceledShutdown`, shutdown testleri |

Toplam 24 top-level test: hub paketinde 20 runtime + 1 AST/import testi;
sözleşmede 2 tip/hata kontrolü + 1 AST/API/import testi. Subtest'ler bu sayıya
eklenmemiştir. API şekil/import testleri gerçek WebSocket veya auth runtime
kanıtı değildir. Hub testleri de consumer davranışının düzeltildiğini iddia etmez.

## Çalıştırılan doğrulamalar

Ortam: Go `1.25.1 windows/amd64`, `CGO_ENABLED=0`.
Bütün Go doğrulamaları `fiber-v2` workspace'inde, `GOPROXY=off`, `GOSUMDB=off`,
`GOTOOLCHAIN=local`, `GOFLAGS=-mod=readonly` ile çalıştı. Default GOCACHE konumu
filesystem izni nedeniyle açılamadı; yalnız build cache TEMP altındaki
`nivgoz-n04a-gocache` dizinine yönlendirildi. Modül/workspace değişikliği veya
GOPATH izolasyonu gerekmedi; gerçek yeni paketler workspace içinde derlendi.

| Kontrol | Sonuç |
| --- | --- |
| `go test -timeout=45s -v ./models/notify ./lib/notificationhub` | PASS (ilk test turu) |
| `go test -count=20 -timeout=60s ./models/notify ./lib/notificationhub` | PASS (tam testler, son davranış değişikliği sonrası) |
| `go test -run 'TestConcurrent\|TestPredicate\|TestShutdown\|TestTransport\|TestBounded\|TestPayload' -count=20 -parallel=8 '-cpu=1,4' -timeout=60s ./lib/notificationhub` | PASS; PowerShell'de CPU argümanı tırnaklıdır |
| `go vet ./models/notify ./lib/notificationhub` | PASS |
| `go list -deps` (iki yeni paket) | PASS; non-stdlib kapanış yalnız `models/notify`, `lib/notificationhub`; cycle yok |
| `go test -race ./models/notify ./lib/notificationhub` | BLOCKED: Go `-race requires cgo` döndürdü; gcc PATH'te yok; kurulum yapılmadı |
| Yeni production import/API AST testleri | PASS; sözleşme yalnız context; core yalnız context, sync, models/notify |
| `gofmt -d` yalnız 5 yeni Go dosyası | PASS; çıktı boş |
| `git diff --check`, untracked whitespace/final-LF | PASS; trailing whitespace yok, final LF mevcut |
| Manifest/checksum/workspace SHA256 | Başlangıç snapshot'ındaki 35 dosyada 0 değişiklik |
| Yeni dosyalarda yasaklı dış import taraması | PASS; 0 eşleşme |

Race detector başarısı iddia edilmez. Alternatif kanıt: statik tek-kilit/channel
incelemesi, runtime barrier testleri ve farklı CPU/paralellikte tekrarlı koşumlar.
Application, gerçek WebSocket, DB, SMTP, migration veya production çalıştırılmadı.
Tam workspace build/test N04A kapsamı değildir. Production durumu UNKNOWN kalır.

## Teslim kapsamı ve N04B

Yeni dosyalar:

1. `fiber-v2/models/notify/contracts.go` — saf sözleşme.
2. `fiber-v2/models/notify/contracts_test.go` — tip/hata/API/import kontrolleri.
3. `fiber-v2/lib/notificationhub/hub.go` — gerçek concurrency çekirdeği.
4. `fiber-v2/lib/notificationhub/hub_test.go` — runtime testleri.
5. `fiber-v2/lib/notificationhub/imports_test.go` — statik sınır testi.
6. `docs/ai/N04A_NOTIFICATION_HUB.md` — envanter, karar ve kanıt kaydı.

Mevcut tracked dosyalar değiştirilmedi. Main wiring, models.Utilities/AppState,
controller/route, JSON, DB sorguları, recipient/auth/public WebSocket davranışı,
eski dependency/import kaldırılması, Fiber adapter ve socket close/deadline
uygulaması N04B'ye bırakıldı. İş kuralı uyuşmazlıkları envanterde açık kalır;
N04B bunları sessizce değiştiremez, kendi açık kapsam/kararına bağlamalıdır.
Yeni rol kararı, retry/durable queue, dış dependency, schema veya UI yoktur.

N04A sözleşme ve çekirdek kapsamı tamamlandı; production N04 geçişi tamamlanmadı.
Stage, commit, push yapılmadı.

Son tracked `git diff` boştur; eklemeler stage edilmemiş 6 yeni dosyadır.
Eşdeğer diff yalnız eklemelerden oluşur, silinen satır yoktur. `git status --short`:

```text
?? docs/ai/N04A_NOTIFICATION_HUB.md
?? fiber-v2/lib/notificationhub/
?? fiber-v2/models/notify/
```
