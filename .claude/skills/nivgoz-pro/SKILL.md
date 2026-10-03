---
name: nivgoz-pro
description: Nivgöz "profesyonel v3" yol haritasındaki bir görevi (P0–P7) düşük maliyetle uygulamak için çalışma protokolü. Kullanıcı "sıradaki görev", "P2-3'ü yap", "yol haritasında ilerle" dediğinde kullan.
---

# Nivgöz Pro — görev protokolü

Kaynak: `docs/ai/PRO_V3_ROADMAP.md` (tek yol haritası). Bu protokol GSD /
spec-kit tarzı "spec → plan → uygula → doğrula" döngüsünün, bu repo için
sadeleştirilmiş hâlidir.

## Döngü (her görev için)
1. **Seç** — Yol haritasında durumu `[ ]` olan ilk görevi al (kullanıcı başka
   bir ID söylemediyse). Görevin "Kabul" maddeleri spesifikasyondur.
2. **Bağlam (dar)** — Yalnız görevin "Dosyalar" satırındaki dosyaları oku.
   `docs/ai/` altındaki büyük belgeleri bütün okuma; `grep` ile ilgili bölümü
   çek. Geniş tarama gerekiyorsa tek bir Explore alt ajanına ver.
3. **Uygula** — UI görevinde önce `nivgoz-design` skill'ini yükle (plan →
   eleştiri → kod). Mekanik/tekrarlı işleri (ör. 60 panel sayfasını aynı
   Tabler kalıbına çevirmek) görev başına taze bir alt ajana (`model:
   sonnet`) ver; ana oturum yalnız planı ve incelemeyi tutar
   (superpowers "subagent-driven development" deseni). Mevcut kodun deyimine uy. Jet kuralları (CLAUDE.md) geçerli.
   Yeni CSS/JS: `static/css|js/frontend|panel/`. `static/assets/` tema
   dosyalarına dokunma; yeni kütüphane `static/vendor/<ad>@<sürüm>/` altına
   sabit sürümle konur (CDN yok).
4. **Doğrula** — en az:
   - `cd fiber-v2/main && go build ./...` ve değişen paketin `go test`'i
   - `node --test fiber-v2/test/*.test.cjs` (JS değiştiyse)
   - Şablon değiştiyse: yerel geçici Postgres + uygulama + Playwright ile
     sayfayı aç, 390px ve 1280px ekran görüntüsü, klavye ile odak turu,
     axe-core taraması (hedef: kritik/ciddi ihlal 0). Çalıştırılamayan
     kontrolü "ölçülmedi/çalıştırılmadı" diye raporla.
   - UI görevi → `design-reviewer` alt ajanı; SEO etkisi → `seo-auditor`;
     güvenlik/yetki değişikliği → yerleşik `/security-review`.
     Engelleyici/yüksek bulgu kalmadan commit yok.
5. **Kaydet** — Görev başına tek commit, Türkçe mesaj (`ui:`, `sec:`, `seo:`,
   `perf:`, `panel:`, `docs:` öneki). Yol haritasında `[x]` + commit kısa
   hash'i. Push: `git push -u origin nivgoz-professional-v2`.
6. **Rapor** — 3–5 satır: ne değişti, nasıl doğrulandı, açık kalan.

## Değişmez kalite çıtası
- WCAG 2.2 AA; gövde ≥16px (panelde ≥15px kabul), dokunma hedefi ≥44px,
  görünür odak, `outline: none` yok, `prefers-reduced-motion` desteklenir.
- Marka: lacivert `#0A2241`, kırmızı `#df0024`, logo değişmez. Renkler
  yalnız token (`--nv-*`) üzerinden kullanılır.
- Her ajax: görsel geri bildirim, `status` ile başarı (200 okuma / 201 yazma),
  sunucu hatasında "Server Hatası: Lütfen daha sonra tekrar deneyin."
- Güvenlik: yetki her endpoint'te sunucuda; secret hiçbir yere yazılmaz.

## Durdur ve sor
- Production DB'ye yazma, deploy, migration çalıştırma, yeni Go bağımlılığı.
- Rol/iş kuralı kararı (kim neyi görür/onaylar).
- Yol haritasında "ONAY" etiketi taşıyan görev.

## Maliyet kuralları
- Dosyayı bir kez oku; düzenledikten sonra yeniden okuma.
- Büyük belgeleri özetleme amacıyla okuma; yalnız ilgili satırları çek.
- Bağımsız işleri paralel araç çağrılarında yap; uzun raporlar yazma.
