# FAZ 3B — Proje sahipliğinde veri katmanı ve notification hub geçişi

Tarih: 2026-09-17. Karar durumu: **hedef mimari C kullanıcı tarafından onaylandı**.
Kanıt baseline'ı: `nivgoz-professional-v2`, `a5e9bfc`.
Uygulama durumu: **PLAN; bağımlılıklar henüz kaldırılmadı**.
Bu belgeyi kalıcılaştıran görev yalnız dokümantasyondur; kodlama, test/build,
DB/SMTP, dependency indirme, migration veya deployment yürütme onayı değildir.

## 1. Amaç, kapsam ve sıra

Uygulamaya ait veri erişimini ve bildirim dağıtımını
`fuzZylota/n-hospital-cms-system` sahipliğine taşımak; küçük, geri alınabilir
paketlerle eski dış paketlerin kaynak, dağıtım ve build gereksinimini kaldırmak.
Yeni kod ve bakım sorumluluğu bu repository'de kalır.

[Ana uygulama planındaki](PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md) sıra korunur:
`SEC-001A` dar güvenlik işi → **`REL-001A` kabul/test kapısı** → `N01–N11`.
`N00` envanteri ve bu karar kaydı önceden yapılabilir. `REL-001A` migration'a
bağlanmaz; gereken en küçük fake/seam kendi dar kapsamındadır. Eksik eski paket
temin edilmez; çalışamayan kontrol kaydedilir, başarı veya kapanış sayılmaz.
N programı geniş güvenlik dalgalarının sırasını değiştirmez; rol/iş kuralı
değişikliği ilgili ana plan paketinde kalır.

Kanıt: [doğrulanmış audit](VERIFIED_AUDIT_2026-09-17.md),
`fiber-v2/go.work`, alt modüllerin manifestleri ve mevcut caller kodu.
N00 statik envanteri: 21 modül, iki aktif dış import yolu, 18 Go dosyasında
41 ORM metodu / 4.649 çağrı noktası. Bunlar runtime veya production kanıtı değildir.
Production şeması, eski paketlerin iç davranışı ve lisansları **UNKNOWN**.

## 2. Onaylanan mimari ve kaldırılacak paketler

- Standart `database/sql`, mevcut `github.com/lib/pq v1.10.9` ve kademeli domain
  repository'leri. Mevcut sürücü kanıtı: `fiber-v2/lib/go.mod`, `fiber-v2/lib/lib.go`.
- Yeni ORM, eski 41 metotlu API'nin kopyası veya genel uyumluluk adapter'ı yok.
- Gerekirse köprü yalnız adlandırılmış tüketiciye özeldir: sahibi, kullanım
  noktaları, kaldırılacağı N paketi ve silinme testi kaydedilir; yeni tüketici alamaz.
- Notification/WebSocket broadcaster, bize ait dar hub ile değiştirilir.
- Eski kaynağa ağ erişimi, kaynak/release/module indirme veya kaynak kopyalama yok.
  Eksik kanıt `UNKNOWN` olarak kalır; Git geçmişi yeniden yazılmaz.

**Retired/removed geçiş kaydı:** Aşağıdaki paketlerin kaldırılması onaylanmıştır;
baseline'da hâlâ aktiftirler. Bu liste tamamlanmış kaldırma iddiası değildir.

| Kaldırılacak aktif paket | Başlıca sınır |
| --- | --- |
| `github.com/Necoo33/neormgo/v2` | `fiber-v2/database/database.go`, `fiber-v2/models/models.go`, controller sorguları |
| `github.com/Necoo33/fiber-ws-broadcaster` | `fiber-v2/main/main.go`, `fiber-v2/controllers/post/post.go`, `fiber-v2/controllers/post/randevular/randevular.go`, ortak modeller |

## 3. Aktif sıfır referans ve tarihsel belge ayrımı

N11'de `Necoo33` ve `neormgo` için aktif kaynak, import, module/replace/vendor
girdileri, tüm manifest/checksum'lar, üretilen build çıktıları ve binary module
metadata'sı sıfır referans vermelidir. Eski sürümlerin yalnız checksum'da kalan
`github.com/Necoo33/neormgo` yolu da kapsamdadır. Aktif kaynak yorumları istisna değildir.

Tarihsel audit/migration belgeleri adları yalnız açık **retired/removed** açıklaması
olarak taşıyabilir; gerçekleşmiş kaldırma tarihi/kanıtı ancak N11'de eklenir.
N11 kontrolü bu belgeleri adlandırarak ayırır; bütün Markdown dosyalarını otomatik
muaf tutmaz. Mevcut işletim talimatları güncel mimariyi anlatmalıdır. Eski Git
nesneleri/reflog sıfır-tarama kapsamında değildir; geçmiş değiştirilmez.

## 4. N00–N11 çalışma paketleri

Tablodaki kapsamlar sonraki ayrı uygulama görevlerinin sınırıdır; toplu yürütme
izni değildir. Her pakette temiz başlangıç, diff incelemesi ve yalnız izinli
ortamda ilgili test kanıtı gerekir. Test bağımlılığı eksikse ağdan tamamlanmaz.
Şema gerektiren uygulama, ana planın `OPS-001`/`DATA-001` kapılarına tabidir.

| Paket / amaç | İzin verilen dosya/paket kapsamı | Önkoşul | Kabul kriteri | Test türü | Geri dönüş sınırı | Sonraki pakete geçiş kapısı |
| --- | --- | --- | --- | --- | --- | --- |
| N00 — Envanter | Mevcut kaynak ve Git metadata; karar belgesi | Baseline ve temiz ağaç | İki paket, 21 modül, API/caller ve UNKNOWN listesi kayıtlı | Statik tarama, yol doğrulama | Yalnız belge düzeltmesi | Envanter tamam; N01 için ayrıca REL-001A geçmeli |
| N01 — Sözleşmeler | §5'teki iki yeni veri sözleşmesi dosyası | REL-001A kabul/testi; N00 | Saf DTO/arayüz, hata/null sözleşmesi; somut altyapı veya init yok | Statik import/cycle; DB'siz sözleşme kontrolleri | Yeni tüketicisiz paketi geri al; manifestlere dokunma | N02 testlenebilir sözleşme review'ı |
| N02 — Fake/seam | `models/data` testleri; `database/postgres` içinde test yardımcıları | N01 | SQL/args, satır/hata, tx yaşam döngüsü gözlenebilir; harici mock paketi yok | Repository fake'i, test driver'ı, hata enjeksiyonu | Test desteği tüketicileriyle birlikte geri alınır | N03 başarı/boş sonuç/hata senaryoları hazır |
| N03 — Header okuma pilotu | `database/postgres`; `fiber-v2/controllers/post/headerbuttons/headerbuttons.go`; yalnız gerekli injection için `fiber-v2/models/models.go`, `fiber-v2/main/main.go` | N02; pilot şema sözleşmesi | Yalnız GetMainHeaderButtons verisi taşınmış; auth/JSON/NULL eşliği | SQL/args, scanner ve fake handler; izole DB varsa sözleşme | Handler ve injection birlikte; diğer header işlemleri korunur | Pilot kabulü; tam paket engelleri N10'a kayıtlı |
| N04 — Notification hub | §5 notify/hub paketleri; `fiber-v2/models/models.go`, `fiber-v2/main/main.go`, `fiber-v2/controllers/post/post.go`, `fiber-v2/controllers/post/randevular/randevular.go` | N03; alıcı ve bağlantı sözleşmesi review'ı | Dış broadcaster tip/importları kaldırılmış; yetki ve bağlantı ömrü korunmuş | Fake sink, allow/deny, disconnect, yavaş istemci, race | Hub ve dört tüketici birlikte: yalnız önceden doğrulanmış repository sahipliğindeki hub sürümüne dönüş veya ileri düzeltme (§8); eski dependency/import geri gelmez, rol politikası geri alınmaz | Hub kontratları geçer; stale manifest temizliği N09'a kayıtlı |
| N05 — Ortak veri erişimi | `database/postgres`; `fiber-v2/database/database.go`, `fiber-v2/lib/lib.go`; gerekli model/injection noktaları | N04; mapping ve tx sözleşmesi | Options/ban okumaları taşınmış; InsertMedia'nın açık tx alan karşılığı hazır; köprü kaydı var | Typed scan, NULL/ID, hata, tx ve cache testleri | Bir yardımcı ve tüm ilgili çağrıları birlikte | N06 users erişim seam'leri hazır |
| N06 — SEC-001A bağlantısı | `fiber-v2/controllers/post/users/`; `fiber-v2/controllers/panel/` yalnız kullanıcı düzenleme yolları; ilgili repository/sözleşmeler | N05; mevcut policy beklentileri sabit | Güncel aktör/hedef/şube sınırı; yetkisiz izin yazımı yok; tam test listesi kayıtlı | Policy, fake handler/repository; N10 tam paket + izole DB | Güvenlik sınırı gevşetilemez; eski savunmasız davranışa dönüş yok | DB'siz kabul geçer; tam testler N10 release borcu olarak açık |
| N07 — Basit domain'ler | Workspace matrisindeki haberler, tedkikler, tibbibirimler, anlasmali_kurumlar, testimonials; ilgili frontend/panel okumaları ve yeni repository'ler | N06; domain başına sorgu/şema listesi | Her alt pakette bir domain/okuma akışı; filtre/sıralama/DTO sonucu kayıtlı; §10 identifier/order sözleşmesi: repository'ye özgü sabit domain allowlist'i, yalnız ASC/DESC ve SQL/args negatif testleri geçmiş | SQL/args, mapping, liste/count, handler; izinli izole DB | Tek domain ve injection; başka domain'e rollback yayılmaz | Bir alt paket kapanmadan aynı akışın yazmaları genişletilmez; tüm N07 kabulü N08'i açar |
| N08 — Karmaşık akışlar | Şube/branş/doktor, header/options, randevu, İK/iletişim ve kalan frontend/panel/post yolları; ilgili repository'ler | N07; akış başına tx, şema/fonksiyon ve yetki kanıtı | §10 riskleri akış bazında kapanmış; eski/yeni bağlantıda bölünmüş tx yok; §10 identifier/order sözleşmesi: repository'ye özgü sabit domain allowlist'i, yalnız ASC/DESC ve SQL/args negatif testleri geçmiş | Tx hata matrisi, SQL/sonuç, yetki, eşzamanlılık; izinli disposable PostgreSQL | Tek tam işlem ve yardımcıları; veri/schema rollback yok | Tüm eski caller'lar N09 kaldırmasına hazır |
| N09 — Bağımlılık kesimi | Eski köprü/tipler; matris kapsamındaki Go importları ve 21 `go.mod`/mevcut checksum'lar; `fiber-v2/go.work`, `fiber-v2/go.work.sum` | N08; sıfır kalan runtime tüketici | İki paket ve eski yol/checksum kalıntıları yok; sahip olunan module path'ler; 21 modül korunur | Statik graph/tarama; offline çözümleme ve hedefli derleme | Import/manifest/source değişimi uyumlu birimdir; yasaklı yol geri eklenmez | N10 için tam workspace çözümlenebilir |
| N10 — Workspace kanıtı | §11 matrisi; mevcut ve eklenen hedefli testler, kanıt kaydı | N09; izinli dependency girdileri; izole DB için ayrı onay | 21/21 build/test; users/panel tam testleri; cycle yok; açık parity borcu yok | Birim, handler, race, gerekli disposable DB entegrasyonu | Başarısız domain'i düzelt; test engelini gizleme veya kapıyı gevşetme yok | N11'e eksiksiz modül/test kanıtı |
| N11 — Kalıcı yasak/offline kabul | `AGENTS.md`, bu plan ve gerekli güncel talimatlar; onaylı build girdileri/kanıtı | N10; hash/lisans/provenance kayıtları | §3 sıfır referans; §12 boş-cache kanıtı; geçici köprü yok | Offline build/test, binary metadata ve kapsamlı son tarama | Yalnız bağımlılığı geri getirmeyen sürüme dönüş; Git geçmişi sabit | Definition of Done ve ayrıca deployment onayı |

N07/N08 tek büyük rewrite değildir. Her alt görev en fazla bir domain veya
tam transaction akışıdır; başlamadan kesin dosya listesi çıkarılır. N08'de
sıralama fonksiyonları, option aktivasyonu/cache, belge toplu insert'i,
doktor ilişkileri, randevu liste/export/latest, randevu mutation'ları ve
İK/iletişim ayrı alt paketlerdir. Şema değişikliği gerekirse bu program içinde
sessizce yapılmaz; ayrı onaylı görev ve veri kapısı gerekir.

## 5. N01 kesin yerleşimi ve import-cycle sınırı

Mevcut gerçek module path'ler `models`, `database`, `lib` şeklindedir; dizin adı
otomatik olarak module path değildir. **22. modül gerekmiyor.** Aşağıdaki yeni
yollar PLAN'dır, bu belge görevi sırasında oluşturulmaz.

| Planlanan repository yolu | İlk import path / paket | Paket ve zaman |
| --- | --- | --- |
| `fiber-v2/models/data/contracts.go` | `models/data` / `data` | N01: tüketiciye özel repository arayüzü, uygulama hataları; yalnız standart kütüphane |
| `fiber-v2/models/data/header_buttons.go` | `models/data` / `data` | N01: HeaderParent DTO'su ve HeaderButtonReader; NULL parent ayrı temsil edilir |
| `fiber-v2/database/postgres/` | `database/postgres` / `postgres` | N02 test desteği; N03 somut pilot, N05+ diğer PostgreSQL uygulamaları |
| `fiber-v2/models/notify/` | `models/notify` / `notify` | N04: event, bağlantı kimliği, Sink/Hub sözleşmeleri |
| `fiber-v2/lib/notificationhub/` | `lib/notificationhub` / `notificationhub` | N04: hub; mevcut WebSocket bağımlılığına transport bağlama |

N01 üretim dosyaları yalnız ilk iki satırdır; somut PostgreSQL veya hub kodu
N01'e girmez. DTO/arayüzler `sql.DB`, `sql.Tx`, `sql.Rows`, `pq`, Fiber, WebSocket,
eski ORM veya kök `models` tiplerini taşımaz. İşlemler `context.Context` ve domain
verisi alır; `not found`, boş liste ve nullable alanlar açık tanımlanır.

İzinli yönler: `database/postgres → models/data`; `lib/notificationhub → models/notify`;
controller'lar → sözleşmeler; `main → somut uygulamalar + controller'lar`.
Yeni alt paketler kök `database`, `lib`, `models` paketlerini import etmez;
`models/data` ve `models/notify` birbirlerini de import etmez. Böylece mevcut
`database → lib → models` zincirine geri kenar eklenmez. Sibling paket olması
kök paketi import etmek değildir; bu statik tasarım değerlendirmesidir, build kanıtı değildir.

Geçişte `models.Utilities`/`AppState` yalnız yeni sözleşmelere alan ekleyebilir;
eski alanlar son tüketicide silinir. Somut nesneler `main` içinde birleştirilir,
`init()` DB/SMTP açmaz. N05'teki yeni InsertMedia uygulamasına eski transaction'ın
ortasından geçilmez; mevcut yardımcı, ilgili tam işlem N08'de taşınana kadar
kalır. Transaction somut PostgreSQL uygulamasının içinde yönetilir; N01 genel
DBTX/ORM API'si üretmez. Parent modüllerin eski require kayıtları tam testleri
hâlâ engelleyebilir; alt paket ayrımı tek başına dependency kaldırılması değildir.
N09'da 21 modül ve local importlar, repository dizinlerini izleyen
`github.com/fuzZylota/n-hospital-cms-system/` önekiyle taşınır; alt paket ayrımı
ve bağımlılık yönleri korunur. Bu mekanik işlem domain davranış değişikliğinden ayrıdır.

## 6. N02 fake/test seam

- Handler testleri tüketiciye özel repository fake'i kullanır; çağrı/argüman ve
  özellikle yasaklı yazma çağrısının sıfır oluşunu kaydeder. Gerçek DB/SMTP yoktur.
- `database/postgres` testleri standart `database/sql/driver` ile yalnız testte
  açılan küçük bağlantı/rows/tx fake'i kullanır; yeni mock dependency eklenmez.
  SQL metni, parametre sırası/tipi, satır/scan/iterator hatası, Close, Begin,
  Commit, Rollback ve context iptali gözlenir. Bu fake eski ORM emülatörü değildir.
- SQL sözleşmesi + sentetik satır fixture'ı + handler sonucu birlikte test edilir.
  Fake PostgreSQL constraint, trigger veya isolation davranışını kanıtlamaz.
- N09 öncesi yalnız bağımlılık kapanışı temiz test kapsamı çalışabilir. Gerekirse
  ayrı görevde açıkça listelenmiş kaynaklarla izole test derlemesi yapılır; bu
  21 modülün veya tam users/panel paketinin geçtiği anlamına gelmez. Yasaklı paket
  eksikliğinde indirme yerine kontrol BLOCKED kaydedilir.

## 7. N03 GetMainHeaderButtons pilotu

Kanıt: `fiber-v2/controllers/post/headerbuttons/headerbuttons.go`,
`GetMainHeaderButtons`. Sorgu yalnız `header_buttons` üzerindeki `hbid`, `title`,
`parent_id` kolonları ve `parent_id IS NULL` filtresidir; yeni sıralama/aktiflik
filtresi eklenmez. DTO, handler sınırında mevcut JSON modeline çevrilir.

Auth başarısızlığında sorgu yok; JSON `status: 403`, başarıda `status: 200`,
DB/okuma hatasında `status: 500` sözleşmesi korunur. Bu alanlar HTTP status kodu
ile karıştırılmaz. Başarıdaki `header_buttons: []`, ID biçimleri, parent NULL
dönüşümü ve mevcut mesajlar test edilir. Sıralama garantisi olmayan sonuç için
test yapay ORDER BY zorlamaz. Header yazma/silme/sıralama, stored function,
cache, route ve rol politikası değişikliği pilot dışındadır.

## 8. N04 notification hub sözleşmesi

Hub sözleşmesi: Register(connection identity, Sink), idempotent Unregister,
Publish(context, event, yetkili alıcı kimlikleri), Close. Kimlik UID, bağlantı ID'si
ve mevcut protokol bilgisini açık taşır; UID bağlantı string'inden türetilmez.
Sink byte mesajı gönderir/kapatır; sözleşmede somut WebSocket veya DB tipi yoktur.

Güncel kullanıcı/şube yetkisini uygulama katmanı çözer; hub menü görünürlüğü,
istemci rolü veya yalnız JWT iddiasından yetki üretmez. Event alanları ve mevcut
protokoller korunur; kullanıcı verisi loglanmaz. DB sorgusu/socket yazımı hub
registry kilidi altında yapılmaz; bağlantı başına seri yazma ve sınırlı kuyruk
kullanılır. Yavaş/kapalı istemci davranışı N04'te açık karar ve testle sabitlenir;
eski paketin teslimat, sıra veya retry garantisi varsayılmaz. Kuyruk/outbox,
kalıcı teslimat ve REL-001B kapsamı bu hub'a eklenmez. Güvenlik politikasındaki
farklar ana planın ilgili dalgasına taşınır; N04 bunları kapatılmış saymaz.

N04 geri dönüşü, `AGENTS.md` dependency yeniden-ekleme yasağına tabidir:
eski dış broadcaster importlarını veya kaldırılması kararlaştırılmış dependency'yi
geri getiren rollback yasaktır. Kabul edilen yollar, repository sahipliğindeki
daha önce doğrulanmış hub sürümüne dönüş veya ileri düzeltmedir. Önceden
doğrulanmış sahip olunan hub sürümü yoksa eski broadcaster'a dönülemez;
değişiklik düzeltilmeli veya release bloke edilmelidir.

## 9. N06 SEC-001A tam test bağlantısı

Baseline'da `fiber-v2/controllers/post/users/users_test.go` içinde 12,
`fiber-v2/controllers/panel/user_edit_page_policy_test.go` içinde bir policy
testi vardır; bunlar tam paket/DB entegrasyon kanıtı değildir.
N06 güncel aktör lookup hatası, pasif/silinmiş kullanıcı, admin pozitif yolu,
non-admin self-edit allowlist, farklı UID/SID ve reddedilen permission yazımı
senaryolarını repository/handler seam'ine bağlar. Gönderilmeyen izin alanları
sıfırlanmamalı; DB hatası başarı gibi sunulmamalıdır.

N06 çıkışında DB'siz testler ve tam test senaryoları hazır olur. Mevcut workspace
için tam users/panel çalıştırma kapısı **N09 sonrası N10**'dur: `models`, `lib`,
`database` ve panelin diğer handler bağımlılıkları da temizlenmelidir. Admin yazma,
constraint/rollback testi yalnız açıkça onaylı disposable PostgreSQL'de çalışır.
Eksik tam test N10 release engelidir; N06 veya bu belge SEC-001A'yı otomatik kapatmaz.

## 10. Davranış riskleri ve kanıt sınırı

| Risk / önem | Repository kanıtı ve güven | Azaltma / kapanış |
| --- | --- | --- |
| Transaction — kritik | `fiber-v2/main/main.go` ortak ORM pointer'ı; caller örüntüsü yüksek güven, eski tx izolasyonu UNKNOWN | Request/işlem kapsamlı tx; bütün yardımcılar aynı executor; Begin/ara hata/commit hata testleri. Eski-yeni havuzda bölünmüş atomiklik yok. N05/N08/N10 |
| Mapping — yüksek | `fiber-v2/lib/lib.go` dönüşümleri; `fiber-v2/database/database.go` satır mapping'i; yüksek güven | NULL/zero ayrımı, bool/int/string/byte/time/array fixture'ları; typed scan, Rows.Err ve Close. RETURNING/etkilenen satır açık sözleşme. N02/N05–N08 |
| SQL parity — yüksek | `fiber-v2/controllers/panel/panel.go` GetFullQuery/parçalama; `fiber-v2/controllers/post/subeler/subeler.go` VALUES birleştirme; yüksek güven | Parametreli SQL ve ortak predicate; aşağıdaki identifier/order allowlist sözleşmesi ve SQL/args negatif testleri; LIKE kaçışları, boş IN, AND/OR, join, sayfalama/count, fonksiyon sonuçları testli. N07/N08/N10 |
| Authorization — kritik | `fiber-v2/controllers/post/users/users.go`, panel ve randevu şube filtreleri; yüksek güven | Güncel aktör + hedef UID/SID; allow/deny; hatada kapalı davranış ve sıfır yetkisiz mutation. N04/N06/N08/N10 |
| Şema ve yan etkiler — yüksek | Audit DATA-001; seçenek cache'i ve randevu bildirimleri kaynakta doğrulanmış; production UNKNOWN | Onaylı secretsız DDL/fonksiyon kanıtı; sentetik disposable fixture; cache/bildirim başarılı commit sonrasında. N05/N08/N10 |

Zorunlu SQL güvenliği ve test sözleşmesi: Dinamik tablo, tablo takma adı, kolon
ve ORDER BY alanları kullanıcı girdisinden doğrudan SQL'e eklenemez. Her
repository kendi sabit ve domain'e özgü allowlist eşlemesini kullanır.
Sıralama yönü yalnız ASC veya DESC olabilir. Geçersiz identifier/order girdisi
ya açıkça reddedilir ya da belgelenmiş güvenli varsayılana döner.
Reddetme/varsayılan davranışı, üretilen SQL ve args üzerinde negatif testlerle
doğrulanır. Reddetme yolunda sorgunun çalıştırılmadığı da doğrulanır.
SQL value placeholder'ları identifier güvenliğinin yerine geçmez; yalnız
sözdizimi kontrolü domain kolon allowlist'i sayılmaz. Kullanıcı girdisi tablo,
kolon veya takma ad olarak doğrudan birleştirilemez. Bu madde mevcut kodda yeni
bir SQL injection doğrulaması değildir; migration'ın zorunlu güvenlik kabul kriteridir.

Eski kaynak veya onaylı eski runtime çıktısı olmadan byte-for-byte ORM parity
iddia edilmez. SQL/args ve uygulama sonuç testleri yeni sözleşmeyi doğrular;
gerçek PostgreSQL testleri yalnız onaylı şema üzerindeki davranışı kanıtlar.
Repository `fiber-v2/schema.sql` production eşdeğeri veya güvenli upgrade aracı
değildir. Şema kanıtı gerektirmeyen sözleşme/fake/kod/path işleri DDL migration
olmadan ilerleyebilir; bilinmeyen şemaya bağlı uygulama ve release ilerleyemez.

## 11. 21 modüllü workspace doğrulama matrisi

Kaynak: `fiber-v2/go.work` ve her satırdaki dizinin `go.mod` bildirimi.
Tüm sonuçlar bu belge hazırlanırken **NOT RUN**. Her satır için N10 kaydı:
hedef Go/toolchain/OS, package/import graph (cycle yok), build ve test sonucu;
eşzamanlılık içeren paketlerde desteklenen ortamda race testi. Tam workspace
modunda modüller açıkça tek tek hedeflenir; kökte tek `./...` yeterli sayılmaz.

| Mevcut module path | Repository dizini | Ek kabul odağı |
| --- | --- | --- |
| `baserouter` | `fiber-v2/baserouter/` | Route ve bağımlılık bağlama |
| `main` | `fiber-v2/main/` | Derleme, composition; uygulama başlatılmaz |
| `models` | `fiber-v2/models/` | Saf data/notify paketleri, eski dış tip yok |
| `database` | `fiber-v2/database/` | PostgreSQL/scan/tx/cache sözleşmeleri |
| `frontend` | `fiber-v2/controllers/frontend/` | Public okuma, arama ve mapping |
| `panel` | `fiber-v2/controllers/panel/` | Tam paket; kullanıcı, filtre/count/şube kapsamı |
| `post` | `fiber-v2/controllers/post/` | İçerik/İK/iletişim ve bildirim caller'ları |
| `options` | `fiber-v2/controllers/post/options/` | Aktivasyon, medya, cache |
| `users` | `fiber-v2/controllers/post/users/` | SEC-001A tam test matrisi |
| `testimonials` | `fiber-v2/controllers/post/testimonials/` | CRUD ve medya |
| `headerbuttons` | `fiber-v2/controllers/post/headerbuttons/` | Pilot + ayrı sıralama/yazma testleri |
| `subeler` | `fiber-v2/controllers/post/subeler/` | Belge/galeri, toplu insert |
| `anlasmali_kurumlar` | `fiber-v2/controllers/post/anlasmali_kurumlar/` | Liste/filtre/CRUD |
| `branslar` | `fiber-v2/controllers/post/branslar/` | Şube ve başhekim ilişkileri |
| `doctors` | `fiber-v2/controllers/post/doctors/` | Şube/uzmanlık/medya transaction'ları |
| `tibbibirimler` | `fiber-v2/controllers/post/tibbibirimler/` | Liste/CRUD/medya |
| `tedkikler` | `fiber-v2/controllers/post/tedkikler/` | Liste/CRUD/medya |
| `randevular` | `fiber-v2/controllers/post/randevular/` | Yetki, tx, durum ve bildirim |
| `haberler` | `fiber-v2/controllers/post/haberler/` | Liste/CRUD ve mapping |
| `lib` | `fiber-v2/lib/` | Ban lookup, hub race, REL-001A regresyonları |
| `tests` | `fiber-v2/test/` | Mevcut yardımcı testi |

N09 sonrasında her satırın yeni sahiplik path'i ayrıca kaydedilir; 21 sayısı ve
dizin kapsamı korunur. Testi olmayan modülde build başarısı, davranış testi
başarısı gibi yazılmaz. Modül başına eksik kontrol ve nedeni açık kalır.

## 12. Offline/boş-cache build kapısı

N11 yalnız önceden onaylanan, hash/lisans/provenance kayıtlı izinli dependency
girdileriyle yürütülür. Eski dependency'yi içeren bundle/vendor kabul edilmez.
Başlangıçta hem GOMODCACHE hem GOCACHE gerçekten boştur; eski cache kopyalamak
boş-cache kanıtı değildir. İzinli kaynaklar onaylı workspace vendor/bundle'dan
sağlanır. Bu girdiler mevcut değilse **BLOCKED**; ağdan eksik tamamlama yapılmaz.

Go 1.25.1 workspace sözleşmesine uygun, önceden kurulu toolchain kullanılır;
`GOTOOLCHAIN=local`, `GOPROXY=off`, dış checksum erişimi kapalı ve ayrıca ağ
engelli ortam gerekir. Ağ ayarları tek başına kaynak doğrulaması yerine geçmez.
21 modülün dependency çözümleme/build/DB'siz testleri, izinli izole testlerin
ayrı kaydı ve binary module metadata incelemesi arşivlenir. Build çıktıları,
generated kaynaklar ve dağıtılan dependency girdileri §3 taramasına dahildir.
Toolchain/OS/arch, kaynak HEAD, girdi hash'leri, komutlar ve exit code'lar
secretsız kaydedilir. Production uygulaması başlatılmaz.

## 13. Release engelleri ve kapsam dışı işler

REL-001A kabul/testi yoksa N01 başlamaz. N10 tam users/panel testleri, 21/21
modül kanıtı, N11 aktif sıfır referans veya boş-cache kanıtından biri eksikse
geçiş release'i yoktur. UNKNOWN kalan gerekli şema/fonksiyon, authorization,
transaction veya dependency provenance kanıtı da engeldir. Ana plandaki
SEC-001A, OPS-001, DATA-001 ve diğer açık güvenlik/release kapıları korunur.

Kapsam dışı: production erişimi/deploy/backup/restore; DDL/migration betiklerini
çalıştırmak; yeni ORM veya dependency yükseltmek; genel adapter; eski kaynak
temini/kopyası; Git geçmişi değişimi; rol/randevu iş kuralını yeniden tasarlamak;
UI/template/JavaScript değişikliği; REL-001B queue/outbox/retry sistemi.
Bu plan hiçbir stage/commit/push yetkisi vermez.

## 14. Definition of Done

- N00–N11 ve domain alt paketleri gerçek kanıtla kapanmış; plan yazımı kapanış sayılmamış.
- Veri katmanı ve hub bu repository'de; DTO/arayüzler altyapıdan bağımsız;
  21 sahip olunan modül path'i ve import graph doğrulanmış, cycle yok.
- Eski iki paket, tarihsel checksum yolu, tüm geçici köprüler ve dış somut tipler
  aktif kapsamdan kaldırılmış; yeniden ekleme/ağ yasağı AGENTS'da kalıcı.
- Transaction/mapping/SQL/authorization ve cache/bildirim kabulü, SEC-001A tam
  users/panel testleri ve gereken izole PostgreSQL testleri geçmiş.
- §10 dinamik identifier/order allowlist sözleşmesi tüm ilgili repository'lerde
  uygulanmış; yalnız ASC/DESC, reddetme veya belgelenmiş güvenli varsayılan
  davranışı SQL/args negatif testleriyle doğrulanmış.
- 21/21 modül ve ağ kapalı boş-cache build/test kanıtı; binary metadata dahil
  aktif sıfır referans; tarihsel retired/removed istisnaları adlandırılmış.
- REL-001A ve ana planın release kapıları geçilmiş; production bilgisi kanıtsız
  doğrulandı sayılmamış; deployment yalnız ayrı açık onayla yapılabilir.
