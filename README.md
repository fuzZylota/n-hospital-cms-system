# Nivgöz CMS

Public website ve CMS, `fiber-v2/` altında Go/Fiber, Jet ve PostgreSQL ile çalışır. Çalışma kuralları [AGENTS.md](AGENTS.md) içindedir. [CLAUDE.md](CLAUDE.md) tarihsel tasarım/operasyon kanıtlarını korur; eski production iddiaları güncel doğrulama değildir.

## Güncel başlangıç

| Amaç | Belge |
| --- | --- |
| Güncel yetki sınırları, deploy kapısı ve sonraki tek iş | [Devir](docs/ai/CURRENT_HANDOFF.md) |
| Yerel güvenlik düzeltmeleri ve açık riskler | [Güvenlik](docs/ai/SECURITY_BACKLOG.md) |
| Randevu iş kuralları, yetki ve snapshot sözleşmeleri | [Randevu](docs/ai/APPOINTMENT_BACKLOG.md) |
| Panel/public aktif UI işleri ve kabul ölçütleri | [UI/UX backlog](docs/ai/UI_UX_IMPLEMENTATION_BACKLOG.md) |
| Özgün UX/SEO araştırma ve kaynak kanıtları | [UX araştırması](docs/ai/UI_UX_RESEARCH.md), [SEO audit](docs/ai/SEO_TECHNICAL_AUDIT.md) |
| Rol/nesne kararları, uygulama dalgaları, tarihsel roadmap/ADR | [Uygulama planı](docs/ai/PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md) |
| Veri katmanı geçiş sözleşmesi ve güncel kabul sınırları | [Owned plan](docs/ai/OWNED_DATA_LAYER_MIGRATION_PLAN.md), [durum](docs/ai/NECOO_MIGRATION_STATUS.md) |
| Notification hub API/concurrency ve wiring geçmişi | [Hub raporu](docs/ai/N04B_NOTIFICATION_HUB_WIRING.md) |
| Kabul edilmiş kaynak denetimi / baseline | [Audit](docs/ai/VERIFIED_AUDIT_2026-09-17.md) |

Belge inceleme referansı `nivgoz-professional-v2`, HEAD `2d093b877d047e1096e92dfc3c97d98e949e6671`. Plan yürütme onayı değildir; tarihsel PASS/BLOCKED kayıtları bu temizlikte yeniden çalıştırılmış testler değildir.

## Kurulum ve deploy sınırı

Güncel, doğrulanmış production kurulum tarifi yoktur. Go workspace `fiber-v2/go.work` içindedir. Dependency indirme veya uygulama/DB/SMTP başlatma ayrıca yetkilendirilmelidir. `schema.sql` yıkıcı fresh-install kaynağıdır; `migrate.sh` ve `production-migrate.sh` mevcut DB için upgrade olarak çalıştırılamaz.

**Deploy KAPALI:** `97d533b` ile `DeleteUser`, `notification_receipts` tablosuna bağımlıdır. [Additive kişisel bildirim migration sözleşmesi](fiber-v2/migrations/20260929_01_personal_notification_contract.md) uygulanmadı; migration ve gerçek PostgreSQL silme/FK doğrulaması olmadan deploy yapılamaz. Bildirim temeli commitli, üreticiler bağlı değil. Gerçek DB/SMTP/tarayıcı/production UNKNOWN.

Geçici admin-only yazmalar moderator/santral iş akışını etkileyebilir. Ortak GET okundu yazımları, SMTP deadline, outbox/retry/tek-gönderim garantisi, olay silme/asenkron üretim yarışı, son admin ve retention açık. Sonraki tek kod işi İK başvuru liste/detay/yanıt/silme PII ve nesne yetkisi envanteridir; CV handler’ının admin-only erişimi bütün akışın kabulü değildir.

## Birleştirilen belgelerin yeni adresleri

2026-09-30 temizliğinde içerik kaybı olmadan taşındı. Eski adlar yalnız arşiv kökenidir; bağlantılar yaşayan bölümlere gider. Kökeni bilinmeyen başlangıç backlog açıklamalarına yazar veya tamamlanma ataması yapılmadı.

| Eski dosya | Yeni adres / gerekçe |
| --- | --- |
| `PROJECT_STATE.md` | [14 Eylül baseline/mimari ve korunacak commit tablosu](docs/ai/CURRENT_HANDOFF.md#historical-project-state) |
| `CHANGELOG_AI.md` | [ilk belge oluşturma kaydı](docs/ai/CURRENT_HANDOFF.md#historical-document-bootstrap) |
| `ROADMAP.md` | [A–H roadmap, F01–F28 ve A01 önerisi](docs/ai/PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-roadmap) |
| `ARCHITECTURE_DECISIONS.md` | [ADR001–006 ve onay durumları](docs/ai/PROFESSIONAL_V2_IMPLEMENTATION_PLAN.md#historical-architecture-decisions) |
| `ADMIN_BACKLOG.md` | [panel bulguları ve kabul ölçütleri](docs/ai/UI_UX_IMPLEMENTATION_BACKLOG.md#historical-admin-backlog) |
| `PUBLIC_SITE_BACKLOG.md` | [public bulguları ve kapsam sınırları](docs/ai/UI_UX_IMPLEMENTATION_BACKLOG.md#historical-public-backlog) |
| `ADD_RANDEVU_WORKFLOW_SNAPSHOT.md` | [AddRandevu sözleşmesi ve test sınırları](docs/ai/APPOINTMENT_BACKLOG.md#add-randevu-snapshot) |
| `EDIT_RANDEVU_WORKFLOW_SNAPSHOT.md` | [EditRandevu sözleşmesi ve test sınırları](docs/ai/APPOINTMENT_BACKLOG.md#edit-randevu-snapshot) |
| `N04A_NOTIFICATION_HUB.md` | [hub API/concurrency ve ilk test matrisi](docs/ai/N04B_NOTIFICATION_HUB_WIRING.md#historical-n04a-core) |
| `fiber-v2/yapılacaklar.md` | Tracked, sıfır bayt ve referanssız; bilgi içermediği için kaldırıldı. |
