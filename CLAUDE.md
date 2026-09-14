# Nivgöz — proje notları

## Ne bu
Nivgöz göz sağlığı merkezlerinin kurumsal sitesi. Adana (Barajyolu,
Çukurova) ve Mersin'de üç merkez. Kurumsal tanıtım + online randevu talebi.
CMS: n-hospital-cms-system.

## Hedef kitle — bu her kararı etkiler
Ziyaretçilerin önemli bir kısmı görme sorunu yaşayan kişiler. Sitede
"Az Görenlere Yardım" hizmet sayfası var. Düşük görme, yüksek kontrast
ihtiyacı, tarayıcı zoom'u ve ekran okuyucu kullanımı burada istisna değil,
normal kullanım. Erişilebilirlik kozmetik değil, işin özü.
Ziyaretçi yaş ortalaması yüksek, trafik ağırlıklı mobil.

## Stack
- Go 1.25.1, go.work ile çok modüllü workspace (tek go.mod yok)
- Fiber v2.52.9
- Jet template engine (gofiber/template/jet/v2) — Go'nun html/template'i DEĞİL
- PostgreSQL 16, veritabanı adı nivgoz_db
- ORM: Necoo33/neormgo/v2
- Tema: satın alınmış Mediox HTML teması (/static/assets/)
- Banner: DB'de tek bir banner_page kaydı var, yani TEK slide.
  slick yine de "fade":true ile kuruluyor. Slide sayısı DB'den
  geliyor, kodda sınır yok.
- Canlı: aaPanel + Nginx reverse proxy → localhost:2000
- Binary: ./main/main, systemd değil, cron ile izleniyor
- ENVIRONMENT="prod" → Jet şablon önbelleği AÇIK, şablon değişikliği
  uygulama yeniden başlatılmadan görünmez

## Jet sözdizimi — EN SIK YAPILAN HATA
Jet, Go'nun standart template motorundan farklı. Go sözdizimi yazarsan
sayfa çalışma zamanında patlar, derleme hatası vermez.
- Değerler: {{ Value }} — {{ .Value }} DEĞİL
- Nokta yalnızca range içindeki struct üyelerinde
- {{ if X == "Y" }} kullan, {{ if eq X "Y" }} KULLANMA
- extend, block veya tanıtılmamış yapı KULLANMA
- Zaman: mthr fonksiyonu. Struct→JSON: stj fonksiyonu
- Kayıtlı fonksiyonlar: mthr, mthrwn, ctdi, ctdli, ctdf, cttf, ctdfm,
  ctdfd, sdti, stti, stj, contains, shorten

## CSS düzeni
- /static/assets/ klasörüne DOKUNMA. Satın alınmış tema orada.
- Yeni CSS/JS → /static/css/frontend/ ve /static/js/frontend/
- Dosya adı sayfa adıyla aynı: hakkimizda.css, hakkimizda.js
- Admin paneli ve frontend stilleri tamamen ayrı

## Ajax kuralları
- Başarı kontrolü yanıtın status özelliğine bakılarak yapılır.
  200 = okuma başarılı, 201 = yazma başarılı.
- Her istek için görsel geri bildirim göster, teknik dille değil.
- Bildirim arka planı 0.7 opaklıktan şeffaf olmasın.
- Sunucu hatalarında: "Server Hatası: Lütfen daha sonra tekrar deneyin."

## Mutlak kurallar
- Sen frontend geliştiricisisin. Backend kodu YAZMA, inisiyatif alma.
- Go handler, route, DB modeli, admin panel mantığına dokunma.
- Veritabanına hiçbir koşulda yazma.
- go.work veya go.mod'a bağımlılık ekleme, sürüm yükseltme.
- Marka renkleri, logo, tasarım kimliği değişmez.
- WCAG 2.2 AA altına düşme.
- Gövde metni 16px altına inmez, dokunma hedefi 44x44px altına inmez.
- outline: none yasak.

## Dosya düzeni
- Şablonlar (Jet): fiber-v2/static/html/ — views/, layouts/, components/
- Frontend CSS: fiber-v2/static/css/frontend/
- Panel CSS: fiber-v2/static/css/panel/
- Ortak: static/css/style.css, static/css/custom.css
- Tema (dokunma): fiber-v2/static/assets/
- Yüklenen medya: fiber-v2/static/files/

## Yapılanlar (13-14 Eylül 2026)
- lang="tr", skip link, <main> eklendi
- 16 sosyal bağlantıya ve 15 form alanına aria-label
- sr-only sınıfı style.css'e tanımlandı
- Sosyal linklerdeki çift https:// düzeltildi
- 26 sayfaya tek h1, hiyerarşi düzeltildi
- Mobilde 16px taban, dokunma hedefleri 44px
- focus-visible, prefers-reduced-motion
- Google Fonts 13 aile → dinamik 2 aile
- 13 kullanılmayan tema script'i kaldırıldı, 3'ü route'a bağlandı,
  kalanlara defer. Anasayfa JS 992 KB → 313 KB
- Preloader panelden kaldırıldı
- Nginx'te /static/html/ kapatıldı (şablon sızıntısı)

## Yapılanlar (14 Eylül 2026, CLS çalışması)
- Arapça dil seçeneği kaldırıldı (RTL desteği yok)
- Bayrak SVG'lerine width/height; Google Translate script'i async
- cls-guard: Owl/Slick kurulumundan önce yalnızca ilk öge görünür,
  böylece yükseklik baştan doğru rezerve ediliyor
- fixRootGap script'i silindi (300/1000/2000 ms'lik setTimeout'larla
  düzeni ilk boyamadan sonra değiştiriyordu); karşılığı CSS'e taşındı
- Ekran altı 30 görsele loading=lazy
- LCP banner görseline <link rel=preload>; mobilde masaüstü ve mobil
  sürümün birlikte inmesi durduruldu
- cls-guard ve root-gap main.jet'e inline gömüldü

## Bilinen sorunlar (backend — dokunma, sadece raporla)
- Uygulama root olarak çalışıyor
- 0.0.0.0:2000 dinliyor, 127.0.0.1 olmalı
- lib modülü terk edilmiş dgrijalva/jwt-go/v4 kullanıyor
- Nginx'te brotli kapalı (gzip AÇIK — 14 Eylül'de ölçüldü, 21 CSS
  dosyasının 20'si sıkıştırılmış geliyor; mediox.css 420 KB → 56 KB)
- /static/ 30 gün cache ama dosya adlarında hash yok
- TLSv1.1 hâlâ açık
- robots.txt route'u yok; 404 fallback sayfasının (fallback.jet)
  tüm HTML'ini text/plain gibi döndürüyor. PageSpeed 14 Eylül'de
  1.235 satır hata saydı (SEO 92'nin sebebi). Gerçek bir robots.txt
  route/dosyası backend tarafından eklenmeli.

## Güncel ölçüm (14 Eylül 11:35, PageSpeed mobil — düzeltmeler ÖNCESİ)
Performans 55, Erişilebilirlik 93, En İyi Uygulamalar 96, SEO 92
FCP 14,8 sn, LCP 21,0 sn, TBT 20 ms, CLS 0.011, Speed Index ölçülmedi

CLS TAMAM (0.715 → 0.011, hedef 0.1 idi).

## Yapılanlar (14 Eylül 2026, PageSpeed takip — resim/kontrast/CSS)
- Kontrast: çerez banner açıklaması (.82→.94 opaklık) ve banner
  arkaplanı (.92→.97), footer alt bilgi/telif metni (.55→.78 opaklık)
- animate.min.css render-engelleyen zincirden çıkarıldı
  (media=print + onload deseni, noscript yedeğiyle) — yalnızca
  WOW.js keyframe'leri, ilk boyamayı etkilemiyor
- Site logosu, merkez kartı ve tıbbi birim kart görsellerine
  gerçek orana göre width/height eklendi
- Görseller lossy yeniden kodlandı (dosya adı/uzantı aynı kaldı,
  Pillow ile 256 renk paletli dithered PNG / JPEG q82):
  - 10 tıbbi birim cover PNG'si: 2,76 MB → 1,67 MB (~%40)
  - şube fotoğrafları (barajyolu + mersin + çukurova, kopyalarıyla
    birlikte): ~628 KB → ~292 KB
  - banner smilepro_dt.png + smilepro_mb.png: 1.232 KB → 478 KB
  - 3 logo dosyası (6167x2500 → 1000x405): 585 KB → 217 KB
  ÖNEMLİ: fiber-v2/static/files/ ve fiber-v2/static/uploads/
  .gitignore'da — bu görsel değişiklikleri git'e girmedi, canlıya
  ayrıca (aaPanel dosya yöneticisi / rsync ile) yüklenmesi gerekiyor.
- Bu oturumda yerel Postgres/Go toolchain olmadığından siteyi
  çalıştırıp tarayıcıda elle test edemedim — bir sonraki oturumda
  veya deploy sonrası ana sayfa, footer ve çerez banner'ı elle
  kontrol edilmeli.

SIRADAKİ İŞ:
- Doktor fotoğrafları (12 adet, 1000x1000 P-mode PNG, ortalama
  ~210 KB) zaten önceden 60-70 renge kadar paletlenmiş; yeniden
  sıkıştırma denemesi gözle görülür kazanç vermedi (bkz. deneyler),
  dokunulmadı.
- 21 CSS dosyasından geri kalanı (bootstrap, fontawesome,
  mediox-icons, owl/slick) hâlâ senkron yükleniyor; ikonlar sayfanın
  her yerinde (header, kartlar) kritik olduğu için ertelemek risk —
  önce görsel test edilmeden dokunulmadı.
- robots.txt route'u backend tarafından eklenmeli (yukarıya bakın).

## Çalışma şekli
- Her değişiklik ayrı commit, mesajlar Türkçe
- Yapısal değişikliklerde önce plan sun, onay bekle
- Emin olmadığını sor, tahmin etme
- Ölçmediğin yere "ölçülmedi" yaz
- Jet şablon hatası çalışma zamanında çıkar — değiştirdiğin sayfayı
  tarayıcıda elle aç
