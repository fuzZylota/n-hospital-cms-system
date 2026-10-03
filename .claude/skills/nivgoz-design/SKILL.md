---
name: nivgoz-design
description: Nivgöz public site veya panel için yeni arayüz tasarlarken/yeniden yaparken kullanılacak tasarım brifi ve token sistemi. Bileşen, sayfa, header/footer, hero, form veya panel ekranı yazmadan ÖNCE yükle.
---

# Nivgöz tasarım brifi

Kaynaklar: Anthropic `frontend-design` skill'inin yöntemi (önce plan, sonra
plana karşı eleştiri, sonra kod), UI UX Pro Max'in "Healthcare / Medical
Clinic" kural seti. Bu projeye uyarlanmış, kısa sürüm.

## Konu, kitle, birincil iş
- Konu: Adana (Barajyolu, Çukurova) ve Mersin'de 3 merkezli göz sağlığı kurumu.
- Kitle: göz sorunu yaşayan, çoğu ileri yaşta, telefondan gelen ziyaretçi;
  ekran okuyucu ve tarayıcı zoom'u normal kullanım.
- Birincil iş: **güven vermek ve randevu talebini zahmetsiz aldırmak.**
  İkincil: doğru merkeze/doktora yönlendirmek, telefonla aratmak.

## Görsel yön: "Klinik netlik"
Sakin, aydınlık, yüksek kontrastlı; göz muayenesindeki okuma tablosunun
netliğinden ilham. Cesaret tek yerde: **büyük, net tipografi**. Geri kalan
her şey sade ve disiplinli.

### Token'lar (`static/css/frontend/tokens.css`)
| Token | Değer | Rol |
| --- | --- | --- |
| `--nv-navy-900` | `#0A2241` | Marka, başlık, birincil metin |
| `--nv-red-600` | `#DF0024` | Yalnız birincil CTA ve kritik vurgu; metin rengi olarak küçük boyutta kullanma |
| `--nv-ink` | `#14233A` | Gövde metni (beyaz üstünde ≥ 14:1) |
| `--nv-paper` | `#FFFFFF` | Zemin |
| `--nv-mist` | `#EEF4FB` | Bölüm ayırıcı zemin |
| `--nv-focus` | `#FFBF00` + 3px navy dış çizgi | Odak halkası, her zeminde görünür |
Ton skalaları (50–900) bu tabanlardan türetilir; her metin/zemin çifti
için kontrast oranı token dosyasında yorum olarak yazılır.

### Tipografi
- Tek aile: **Atkinson Hyperlegible Next** (400/700/800), self-host woff2,
  `font-display: swap`, Latin Extended (Türkçe).
- Gövde 18px (mobil 17px), satır yüksekliği 1.6, satır uzunluğu ≤ 70 karakter.
- Başlıklar `clamp()` ile akışkan; ölçek 1.25 (major third).
- Yasak: başlık üstünde BÜYÜK HARF etiket, başlıkta tek kelimeyi renkli/italik
  yapmak, 16px altı metin, gri-üstü-gri metin.

### Yerleşim ve bileşen ilkeleri
- Mobil önce; tek sütun okuma akışı, masaüstünde 12 kolon.
- Birincil CTA "Randevu Talebi Oluştur" her sayfada aynı ad ve yerde; ikincil
  CTA "Hemen Ara" (tel: bağlantısı). Buton min 48px yükseklik.
- Kartlar yalnız gerçek koleksiyonlar için (doktor, merkez). Her şeyi karta
  bölme; tek gölge düzeyi, hiyerarşiye göre radius.
- Otomatik kayan slider yok. Hareket yalnız kullanıcı eylemine yanıt olarak;
  `prefers-reduced-motion` altında kapalı.
- Formlar: görünür etiket (placeholder etiket değildir), alan altında hata
  metni, adım göstergesi, gönderimde net onay ve talep numarası.
- Görseller gerçek merkez/doktor fotoğrafı; stok "gülümseyen hasta" yok.

### Panel (Tabler)
Tabler'ın varsayılanları korunur; yalnız `--tblr-primary` = navy, tehlike =
marka kırmızısı. Yoğun tablo ekranlarında 15px taban kabul edilir.

## Süreç (her yeni ekran)
1. **Plan** — 5–8 satır: amaç, ASCII tel kafes, kullanılan token/bileşenler.
2. **Eleştiri** — Plan "herhangi bir sağlık sitesi için de aynı mı olurdu?"
   Evet ise bir öğeyi bu kitleye özgü hâle getir, ne değiştiğini yaz.
3. **Kod** — Jet kuralları (CLAUDE.md). Yeni CSS sayfa adıyla aynı dosyada.
4. **İnceleme** — `design-reviewer` alt ajanını çağır; engelleyici ve yüksek
   öncelikli bulguları düzeltmeden commit etme.

## Yazım
Hasta dili, etken çatı, kısa cümle. Buton ne yapacağını söyler ("Talebi
Gönder", "Gönder" değil). Hata ne olduğunu ve nasıl düzeleceğini söyler;
özür dilemez. Boş durum bir sonraki adımı gösterir.
