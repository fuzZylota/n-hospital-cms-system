# Nivgöz çalışma sözleşmesi

Bu dosya, bu repository üzerinde çalışan ajanlar ve geliştiriciler için proje
özelindeki çalışma kurallarını içerir. Kullanıcının güncel görevi ve güvenlik
sınırları her zaman önceliklidir.

## Mimari özeti

- `fiber-v2/`, Go/Fiber tabanlı public website ve CMS uygulamasıdır.
- Public sayfalar ve panel, Jet şablonlarıyla server-render edilir.
- PostgreSQL erişimi neormgo ve raw SQL ile yapılır; tek, ayrık bir service veya
  repository katmanı yerleşik değildir.
- Kimlik doğrulama JWT cookie ile yapılır. Roller `admin`, `moderator`,
  `santral` ve `ik` olarak görülür; şube yetkileri ayrıca değerlendirilir.
- E-posta SMTP, canlı bildirimler WebSocket üzerinden yürür. Production
  topolojisi ve aktif altyapı ayarları ancak açıkça onaylanmış kanıtla bilinir.

## Önemli dizinler

| Konum | Sorumluluk |
| --- | --- |
| `fiber-v2/main/` | Başlatma, global middleware ve static route'lar |
| `fiber-v2/baserouter/` | Public/panel/backend route grupları |
| `fiber-v2/controllers/` | Public, panel ve mutation handler'ları |
| `fiber-v2/models/`, `fiber-v2/database/`, `fiber-v2/lib/` | Model, veri erişimi ve ortak yardımcılar |
| `fiber-v2/static/html/` | Jet views, layouts ve components |
| `fiber-v2/static/css/frontend/`, `fiber-v2/static/js/frontend/` | Public yüzeye özel ekler |
| `fiber-v2/static/css/panel/`, `fiber-v2/static/js/panel/` | CMS/panel yüzeyine özel ekler |
| `fiber-v2/static/assets/` | Satın alınmış tema varlıkları; zorunlu olmadıkça değiştirme |
| `docs/ai/` | Denetim, karar, backlog, roadmap ve değişiklik kayıtları |

## Talimat önceliği

1. Kullanıcının güncel, açık görevi ve güvenlik sınırları.
2. Bu dosya ile ilgili görevdeki yerel talimatlar.
3. Görevle ilgili bölümleriyle kök `CLAUDE.md`, `fiber-v2/.cursorrules` ve
   `docs/ai/` belgeleri.

Çelişkide, daha dar ve daha güvenli kural uygulanır. Tarihsel belgeler
production gerçeğinin kanıtı değildir. Belirsiz iş kuralları varsayımla
değiştirilmez; karar veya onay istenir. `AGENTS.md` temel repository
talimatıdır; diğer belgeler yalnız görevle ilişkili olduklarında okunur.

## Git ve dal güvenliği

- Göreve başlamadan önce repository kökü, aktif dal, HEAD ve `git status --short`
  kontrol edilir.
- Kullanıcının mevcut değişiklikleri kullanıcıya aittir; silinemez, geri
  alınamaz, formatlanamaz veya taşınamaz.
- `git reset --hard`, force push, kontrolsüz `checkout`/`restore`, `clean` ve
  benzeri yıkıcı işlemler yasaktır.
- Kullanıcı açıkça istemeden commit, push, pull, merge, rebase veya tag yapılmaz.
- Görev sonunda değişen dosyalar, `git diff`, `git diff --check` ve çalışma ağacı
  durumu raporlanır.

## Güvenli çalışma sınırları

- `fiber-v2/schema.sql`, `fiber-v2/migrate.sh` ve
  `fiber-v2/production-migrate.sh`, disposable olmayan hiçbir DB'de
  çalıştırılamaz.
- Uygulama, DB, SMTP veya migration kullanıcı açıkça istemeden çalıştırılamaz.
- Production erişimi, deploy, yedek/geri yükleme ve altyapı değişiklikleri ayrı
  açık onay gerektirir.
- Secret değerleri okunamaz, raporlanamaz, fixture'a kopyalanamaz veya commit
  edilemez. Yalnız varlıkları ve etkileri, değer göstermeden belirtilebilir.
- Dosya yolu, upload, private belge ve secret alanları güven sınırıdır.
- Güvenlik kararı menü görünürlüğüne dayanamaz; endpoint seviyesinde rol ve hedef
  nesne/şube yetkisi gerekir.

## Dependency sahipliği

- Yeni veya değiştirilen dış bağımlılıklar açık kullanıcı onayı gerektirir.
- Kaldırılması kararlaştırılmış dependency kaynaklarına ağ erişimi veya indirme
  yapılamaz; bunların module/import yolları yeniden eklenemez.
- Proje sahipliğindeki veri katmanı ve notification hub tercih edilir;
  [onaylı geçiş planı](docs/ai/OWNED_DATA_LAYER_MIGRATION_PLAN.md) uygulanır.
- Tarihsel audit/migration belgeleri eski bağımlılıkları açıklama amacıyla açık
  `retired/removed` bağlamında anabilir; bu istisna aktif kod, import, manifest,
  checksum veya build/binary metadata için geçerli değildir.

## Kod değişikliği kuralları

- Görevi dar tut; ilgisiz cleanup, framework değişimi, bağımlılık güncellemesi
  veya geniş yeniden yazım yapma.
- Public ve panel CSS/JS yüzeyleri ayrıdır; yeni dosyaları ilgili yüzeye koy.
- Satın alınmış `fiber-v2/static/assets/` varlıklarını zorunlu değilse
  değiştirme.
- Jet, Go `html/template` değildir. Mevcut Jet sözdizimi ve kayıtlı yardımcıları
  kullan; şablon değişikliklerini runtime davranışı açısından doğrula.
- Erişilebilirlik korunur: görünür odak, anlamlı etiketler, klavye kullanımı,
  mobil dokunma hedefleri ve düşük görme gereksinimleri geriletilmez.
- Backend, şema, rol politikası veya randevu iş kuralları ancak kullanıcının
  kapsamı açıkça bunları içerdiğinde değiştirilir.

## Test ve doğrulama

- Kod değişikliğinden sonra ilgili testler ve `git diff` kontrolü zorunludur.
- Değişikliğe uygun en dar testleri çalıştır; çalıştırılmayan kontrolleri ve
  nedenini açıkça raporla.
- Güvenlik düzeltmeleri için olumlu ve reddedilen yetki yollarını doğrula.
- DB veya SMTP gerektiren testler yalnız açıkça onaylanmış, izole test ortamında
  çalıştırılır. Gerçek kullanıcı verisi, gerçek SMTP veya production DB kullanılmaz.
- UI değişikliklerinde etkilenen desktop, mobil ve klavye yolunu kontrol et.

## Güvenlik değişmezleri

- Authentication, authorization ve object/branch ownership her mutation'da
  sunucuda doğrulanır.
- JWT claim'i tek başına güncel rol veya iptal kararı için yeterli kabul edilmez.
- Kullanıcıdan gelen dosya adı veya yol, fiziksel dosya sistemi yolu değildir.
- Upload türü, boyutu, içeriği, hedefi ve erişim politikası ayrı ayrı doğrulanır.
- Private belgeler public static kökte tutulmaz; indirmenin yetkisi ayrıca
  denetlenir.
- Secret alanları response, template, log ve istemci tarafına taşınmaz.
- Hata yolları süreci sonlandırmaz; hatalar kontrollü biçimde çağırana döner.

## Dokümantasyon güncelleme kuralları

- Her görevde önce bu dosya okunur. `CLAUDE.md`, `fiber-v2/.cursorrules` ve `docs/ai/`
  belgeleri yalnız görev mimariyi, güvenliği, veri modelini, deployment'ı veya
  birden fazla modülü etkiliyorsa ve ilgili oldukları ölçüde okunur.
- Tüm `docs/ai/*.md` belgelerini her görevde okumak zorunlu değildir.
- Hassas değer içerebilecek config bölümleri gereksiz yere açılmaz,
  kopyalanmaz veya raporlanmaz. Secret/config incelemesi gerekiyorsa yalnız
  değişken adları ve yapı değerlendirilir; değerler ekrana veya dosyalara
  taşınmaz.
- Tamamlanan işte ilgili backlog/roadmap/changelog yalnız görev kapsamı bunu
  içeriyorsa güncellenir; plan yazmak bulguyu çözülmüş yapmaz.
- Yeni kanıt, ilgili belgeye kanıt yolu, güven düzeyi, sınırlama ve karar olarak
  eklenir. Production bilgisi doğrulanmadıkça `UNKNOWN` olarak kalır.
- Yeni mimari veya iş kuralı kararı için ayrı, onay durumunu belirten bir kayıt
  oluşturulur.

## Görev sonu raporu

Kısa rapor şunları içerir:

1. Ne değişti ve kullanıcı açısından sonucu.
2. Değiştirilen dosyalar.
3. Çalıştırılan doğrulamalar ve sonuçları.
4. Çalıştırılmayan kontroller, sınırlamalar veya açık riskler.
5. `git diff --check`, diff özeti ve çalışma ağacı durumu. Salt-okunur
   görevlerde bunun yerine tracked/untracked değişiklik oluşmadığı doğrulanır.
6. Commit/push yapılmadıysa bunu; yapıldıysa yalnız kullanıcı onayıyla yapıldığını.
