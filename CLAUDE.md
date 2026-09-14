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

## Bilinen sorunlar (backend — dokunma, sadece raporla)
- Uygulama root olarak çalışıyor
- 0.0.0.0:2000 dinliyor, 127.0.0.1 olmalı
- lib modülü terk edilmiş dgrijalva/jwt-go/v4 kullanıyor
- Nginx'te gzip/brotli kapalı
- /static/ 30 gün cache ama dosya adlarında hash yok
- TLSv1.1 hâlâ açık

## Güncel ölçüm (14 Eylül, PageSpeed mobil)
Performans 31, Erişilebilirlik 93, En İyi Uygulamalar 96, SEO 92
FCP 2.9 sn, LCP 27.0 sn, TBT 420 ms, CLS 0.715, Speed Index 6.2 sn
SIRADAKİ İŞ: CLS 0.715 → 0.1 altına indirmek

## Çalışma şekli
- Her değişiklik ayrı commit, mesajlar Türkçe
- Yapısal değişikliklerde önce plan sun, onay bekle
- Emin olmadığını sor, tahmin etme
- Ölçmediğin yere "ölçülmedi" yaz
- Jet şablon hatası çalışma zamanında çıkar — değiştirdiğin sayfayı
  tarayıcıda elle aç
