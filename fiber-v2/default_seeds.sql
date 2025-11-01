-- Default Seeds for Hospital CMS Database
-- This file contains all the minimum required data for the CMS to function properly

-- =====================================================
-- DEFAULT OPTIONS CONFIGURATION
-- =====================================================

-- Insert default options configuration
INSERT INTO options (
    option_set_name,
    option_set_description,
    option_set_is_active,
    site_name,
    site_description,
    maintenance_mode,
    preloader,
    smtp_host,
    smtp_port,
    smtp_username,
    smtp_password,
    smtp_encryption,
    facebook_url,
    twitter_url,
    instagram_url,
    linkedin_url,
    main_page_meta_title,
    main_page_meta_description,
    google_analytics,
    primary_color,
    secondary_color,
    accent_color,
    background_color,
    font_color,
    font_family,
    require_strong_password,
    items_per_page,
    max_upload_size,
    timezone
) VALUES (
    'varsayılan',
    'Varsayılan hastane CMS ayarları',
    TRUE,
    'N-Hospital CMS',
    'Modern tıbbın tüm imkanları ile 7/24 hizmet veren güvenilir sağlık kuruluşu. Uzman doktor kadromuz ile sağlığınız için buradayız.',
    FALSE,
    '<style>.preloader-overlay{position:fixed;inset:0;background:var(--theme-primary-color);display:flex;align-items:center;justify-content:center;z-index:999999;opacity:1;visibility:visible;transition:opacity .5s ease,visibility .5s ease}.preloader-overlay.preloader-hidden{opacity:0;visibility:hidden}.preloader-card{width:280px;height:280px;border-radius:24px;background:linear-gradient(180deg,rgba(255,255,255,0.06),var(--theme-secondary-color));border:1px solid rgba(255,255,255,0.1);backdrop-filter:blur(14px);-webkit-backdrop-filter:blur(14px);box-shadow:0 30px 120px rgba(0,0,0,.45);position:relative;overflow:hidden;display:flex;align-items:center;justify-content:center}/* Spinning ring + logo combo */.preloader-ring{position:absolute;width:200px;height:200px;border-radius:50%;background:conic-gradient(from 0deg,var(--theme-primary-color),var(--theme-secondary-color),var(--theme-accent-color),var(--theme-primary-color));-webkit-mask:radial-gradient(farthest-side,transparent 62%,#000 63%);mask:radial-gradient(farthest-side,transparent 62%,#000 63%);animation:spin 1.4s linear infinite;filter:blur(.5px) saturate(1.2)}.preloader-glow{position:absolute;width:200px;height:200px;border-radius:50%;background:var(--theme-primary-color);filter:blur(8px);animation:pulse 1.8s ease-in-out infinite}.preloader-logo{position:relative;width:120px;height:120px;border-radius:16px;background:color-mix(in srgb,var(--theme-primary-color),white 10%);border:1px solid rgba(255,255,255,0.08);display:grid;place-items:center;overflow:hidden;box-shadow:inset 0 0 0 1px rgba(255,255,255,0.04),0 10px 40px rgba(0,0,0,.45)}.preloader-logo img{max-width:80px;max-height:80px;object-fit:contain;filter:drop-shadow(0 4px 14px var(--theme-primary-color));animation:float 2.4s ease-in-out infinite}/* Progress bar */.preloader-progress{position:absolute;bottom:18px;left:18px;right:18px;height:6px;border-radius:999px;background:rgba(255,255,255,0.08);overflow:hidden}.preloader-progress>span{display:block;height:100%;width:0%;background:linear-gradient(90deg,var(--theme-primary-color),var(--theme-secondary-color),var(--theme-accent-color));border-radius:inherit;box-shadow:0 0 24px var(--theme-primary-color);transition:width .3s ease}@keyframes spin{to{transform:rotate(360deg)}}@keyframes pulse{0%,100%{transform:scale(1);opacity:.7}50%{transform:scale(1.06);opacity:1}}@keyframes float{0%,100%{transform:translateY(0)}50%{transform:translateY(-6px)}}</style><div class="preloader-overlay" id="preloader"><div class="preloader-card"><div class="preloader-ring"></div><div class="preloader-glow"></div><div class="preloader-logo"><img src="files/defaults/logo/n-hospital-logo.png" alt="N-Hospital, Hastaneler için yüksek hız ve özelleştirme için geliştirilmiş bir yönetim sistemi" title="N-Hospital Logosu" /></div><div class="preloader-progress"><span id="preloaderBar"></span></div></div></div><script>(function(){var preloader=document.getElementById("preloader");var bar=document.getElementById("preloaderBar");var progress=0;var step=7+Math.random()*8;function normalizeImagePath(){var img=preloader.querySelector("img");if(img){var currentPath=window.location.pathname;var pathSegments=currentPath.split("/").filter(function(segment){return segment.length>0});var pathDepth=pathSegments.length;if(pathDepth>0){var currentSrc=img.getAttribute("src");img.setAttribute("src","../".repeat(pathDepth)+currentSrc)}}}function tick(){progress=Math.min(100,progress+step);bar.style.width=progress+"%";if(progress<100){step=Math.max(2,step*(0.92+Math.random()*0.1));requestAnimationFrame(tick)}else{setTimeout(hide,250)}}function hide(){preloader.classList.add("preloader-hidden");setTimeout(function(){if(preloader&&preloader.parentNode)preloader.parentNode.removeChild(preloader)},1000)}normalizeImagePath();setTimeout(hide,6000);requestAnimationFrame(tick)})();</script>',
    'smtp.gmail.com',
    587,
    '',
    '',
    'tls',
    'https://facebook.com/saglikhastanesi',
    'https://twitter.com/saglikhastanesi',
    'https://www.instagram.com/nermefraz',
    'https://www.linkedin.com/company/107576047',
    'N-Hospital CMS, Hastaneler için En Gelişmiş CMS',
    'N-Hospital CMS, Nermefraz Bilişim Teknolojileri, Yazılım Ve Danışmanlık tarafından tasarlanmış bir hastane yönetim sistemidir.',
    '',
    'rgba(10, 34, 65, 1)',
    'rgba(51, 193, 237, 1)',
    'rgba(223, 0, 36, 1)',
    'rgba(255, 255, 255, 1)',
    'rgba(51, 51, 51, 1)',
    'Poppins',
    FALSE,
    10,
    5242880,
    'UTC'
);

-- =====================================================
-- 1. ADMIN USER
-- =====================================================

-- Insert 1 admin user (password: admin123 - should be changed)
INSERT INTO users (
    name,
    surname,
    password,
    email,
    phone,
    role,
    is_active,
    timezone
) VALUES (
    'Admin',
    'Kullanıcı',
    'sNPRDmbPCLFiaK3eFlpk50hDMRPFAQtRqoVxtyuMasK7OOdM', -- admin123
    'admin@nhospital.com',
    '+90 555 000 0001',
    'admin',
    true,
    'Europe/Istanbul'
);

-- =====================================================
-- 2. HEADER BUTTONS - COMPREHENSIVE NAVIGATION
-- =====================================================

-- Main menu items
INSERT INTO header_buttons (title, url, sort_order, is_active, icon, button_type) VALUES
('Anasayfa', '/', 1, true, 'home', 'anasayfa'),
('Kurumsal', '#', 2, true, 'building', 'kurumsal'),
('Şubelerimiz', '/subelerimiz', 3, true, 'map-marker-alt', 'subeler'),
('Tıbbi Birimler', '/tibbi-birimler', 4, true, 'hospital', 'tibbi_birimler'),
('Tetkikler', '/tetkikler', 5, true, 'microscope', 'tedkikler'),
('İletişim', '/iletisim', 6, true, 'phone', 'diger');

-- Kurumsal sub-menu
INSERT INTO header_buttons (title, url, sort_order, is_active, button_type, parent_id) VALUES
('Hakkımızda', '/kurumsal/hakkimizda', 1, true, 'kurumsal', 2),
('Misyon - Vizyon', '/kurumsal/misyon-vizyon', 2, true, 'kurumsal', 2),
('Anlaşmalı Kurumlar', '/kurumsal/anlasmali-kurumlar', 3, true, 'kurumsal', 2),
('İnsan Kaynakları', '/kurumsal/insan-kaynaklari', 4, true, 'kurumsal', 2),
('Haberler', '/haberler', 5, true, 'kurumsal', 2),
('KVKK', '/kurumsal/kvkk', 6, true, 'kurumsal', 2);

-- =====================================================
-- 3. TESTIMONIALS
-- =====================================================

INSERT INTO testimonials (
    first_name,
    last_name,
    occupation,
    content,
    rating,
    is_active
) VALUES
('Ahmet', 'Yılmaz', 'Mühendis', 'Hastanenizde aldığım hizmet gerçekten mükemmeldi. Doktorlar çok ilgili ve profesyonel. Kesinlikle tavsiye ediyorum.', 5.0, true),
('Fatma', 'Kaya', 'Öğretmen', 'Çocuğumun tedavisi için geldiğimizde çok memnun kaldık. Hem doktorlar hem de hemşireler çok sabırlı ve anlayışlıydı.', 4.8, true),
('Mehmet', 'Demir', 'İş İnsanı', 'Modern cihazlar ve uzman kadro sayesinde kısa sürede sağlığıma kavuştum. Teşekkürler.', 4.9, true);

-- =====================================================
-- 4. BRANCHES - 3 DEFAULT BRANCHES
-- =====================================================

INSERT INTO subeler (
    name, 
    url_name, 
    description, 
    address, 
    city, 
    district, 
    phone, 
    email, 
    working_hours, 
    is_main, 
    is_active
) VALUES
('İstanbul Merkez Hastane', 'istanbul-merkez', 'Ana hastane binası, tüm bölümlerimizin bulunduğu merkez lokasyonumuz.', 'Merkez Mah. Sağlık Cad. No:1', 'İstanbul', 'Beyoğlu', '+90 212 555 0000', 'istanbul@nhospital.com', '{"pazartesi": "08:00-18:00", "salı": "08:00-18:00", "çarşamba": "08:00-18:00", "perşembe": "08:00-18:00", "cuma": "08:00-18:00", "cumartesi": "08:00-13:00", "pazar": "closed"}', true, true),
('Ankara Şubesi', 'ankara-subesi', 'Başkent Ankara''daki modern şubemiz, kapsamlı sağlık hizmetleri sunmaktadır.', 'Çankaya Mah. Sağlık Sok. No:25', 'Ankara', 'Çankaya', '+90 312 555 0000', 'ankara@nhospital.com', '{"pazartesi": "08:30-17:30", "salı": "08:30-17:30", "çarşamba": "08:30-17:30", "perşembe": "08:30-17:30", "cuma": "08:30-17:30", "cumartesi": "09:00-13:00", "pazar": "closed"}', false, true),
('İzmir Şubesi', 'izmir-subesi', 'Ege bölgesindeki şubemiz, deneyimli doktor kadrosu ile hizmet vermektedir.', 'Konak Mah. Hastane Cad. No:15', 'İzmir', 'Konak', '+90 232 555 0000', 'izmir@nhospital.com', '{"pazartesi": "09:00-17:00", "salı": "09:00-17:00", "çarşamba": "09:00-17:00", "perşembe": "09:00-17:00", "cuma": "09:00-17:00", "cumartesi": "09:00-12:00", "pazar": "closed"}', false, true);

-- =====================================================
-- 5. CONTRACTED INSTITUTIONS - 8 INSTITUTIONS
-- =====================================================

INSERT INTO anlasmali_kurumlar (
    name, 
    url_name, 
    type, 
    sid,
    contact_person, 
    phone, 
    email, 
    discount_rate, 
    is_active
) VALUES
('Allianz Sigorta', 'allianz-sigorta', 'sigorta', 1, 'Mehmet Yılmaz', '+90 212 555 0001', 'saglik@allianz.com.tr', 20.00, true),
('Axa Sigorta', 'axa-sigorta', 'sigorta', 1, 'Ayşe Demir', '+90 212 555 0002', 'saglik@axa.com.tr', 15.00, true),
('Nermefraz Bilişim', 'nermefraz-bilisim', 'kurumsal', 1, 'Necdet Bey', '+90 532 123 45 67', 'info@nermefraz.com', 25.00, true),
('Türk Telekom', 'turk-telekom', 'kurumsal', 2, 'Fatma Özkan', '+90 212 555 0003', 'saglik@turktelekom.com.tr', 10.00, true),
('İstanbul Üniversitesi', 'istanbul-universitesi', 'universite', 2, 'Prof. Dr. Ali Veli', '+90 212 555 0004', 'saglik@istanbul.edu.tr', 30.00, true),
('Aksigorta', 'aksigorta', 'sigorta', 3, 'Hasan Çelik', '+90 212 555 0005', 'saglik@aksigorta.com.tr', 18.00, true),
('Yapı Kredi Bankası', 'yapi-kredi-bankasi', 'kurumsal', 3, 'Zeynep Arslan', '+90 212 555 0006', 'saglik@yapikredi.com.tr', 12.00, true);

-- =====================================================
-- 6. SPECIALIZATIONS - REQUESTED SPECIALIZATIONS
-- =====================================================

INSERT INTO uzmanliklar (name, url_name, description, icon, is_active) VALUES
('Katarakt', 'katarakt', 'Göz merceği bulanıklığı tedavisi uzmanlığı', 'eye', true),
('Bel Fıtığı', 'bel-fitigi', 'Omurga ve bel fıtığı tedavisi uzmanlığı', 'spine', true),
('Parkinson Hastalığı', 'parkinson-hastaligi', 'Parkinson hastalığı tedavisi uzmanlığı', 'brain', true),
('Miyopi', 'miyopi', 'Miyopi ve görme bozuklukları tedavisi uzmanlığı', 'glasses', true),
('Böbrek Nakli', 'bobrek-nakli', 'Böbrek nakli cerrahisi uzmanlığı', 'kidney', true),
('Karaciğer Nakli', 'karaciger-nakli', 'Karaciğer nakli cerrahisi uzmanlığı', 'liver', true);

-- =====================================================
-- 7. DEPARTMENTS - 4 REQUESTED DEPARTMENTS
-- =====================================================

INSERT INTO branslar (
    name, 
    url_name, 
    description, 
    short_description, 
    services, 
    sid, 
    phone, 
    email, 
    is_active
) VALUES
('Ortopedi', 'ortopedi', 'Kemik, eklem ve kas hastalıklarının teşhis ve tedavisi yapılan bölüm', 'Ortopedi ve Travmatoloji', ARRAY['Röntgen', 'MR', 'Artroskopi', 'Protez Cerrahisi', 'Kırık Tedavisi'], 1, '+90 212 555 0100', 'ortopedi@nhospital.com', true),
('Kardiyoloji', 'kardiyoloji', 'Kalp ve damar hastalıklarının teşhis ve tedavisi yapılan bölüm', 'Kalp Sağlığı Merkezi', ARRAY['EKG', 'Ekokardiyografi', 'Stres Testi', 'Anjiyografi', 'Holter'], 1, '+90 212 555 0101', 'kardiyoloji@nhospital.com', true),
('Fizik Tedavi Ve Rehabilitasyon', 'fizik-tedavi-rehabilitasyon', 'Fizik tedavi ve rehabilitasyon hizmetlerinin verildiği bölüm', 'Fizik Tedavi Merkezi', ARRAY['Fizik Tedavi', 'Egzersiz Tedavisi', 'Elektroterapi', 'Hidroterapi', 'Manuel Terapi'], 1, '+90 212 555 0102', 'fizyoterapi@nhospital.com', true),
('Oftalmoloji', 'oftalmoloji', 'Göz hastalıklarının teşhis ve tedavisi yapılan bölüm', 'Göz Sağlığı Merkezi', ARRAY['Göz Muayenesi', 'Katarakt Cerrahisi', 'Retina Tedavisi', 'Lazer Tedavisi', 'Lens İmplantı'], 1, '+90 212 555 0103', 'goz@nhospital.com', true);

-- =====================================================
-- 8. DOCTORS - 2 DOCTORS PER DEPARTMENT (8 TOTAL)
-- =====================================================

INSERT INTO doktorlar (
    drid,
    title, 
    first_name, 
    last_name, 
    url_name, 
    phone, 
    email, 
    biography, 
    education, 
    experience_years, 
    languages, 
    birth_date, 
    gender, 
    brid, 
    sid, 
    room_number, 
    appointment_duration, 
    appointment_fee, 
    online_appointment, 
    working_hours, 
    personal_url,
    is_active
) VALUES
-- Ortopedi Doctors
(1, 'Prof. Dr.', 'Ahmet', 'Özkan', 'prof-dr-ahmet-ozkan', '+90 212 555 0200', 'ahmet.ozkan@nhospital.com', '20 yıllık deneyime sahip ortopedi uzmanı. Kemik ve eklem cerrahisinde uzmanlaşmıştır.', 'İstanbul Üniversitesi Tıp Fakültesi, Ortopedi Uzmanlığı', 20, 'Türkçe, İngilizce', '1975-03-10', 'erkek', 1, 1, '101', 30, 600.00, true, '{"pazartesi": "09:00-17:00", "salı": "09:00-17:00", "çarşamba": "09:00-17:00", "perşembe": "09:00-17:00", "cuma": "09:00-17:00", "cumartesi": "09:00-12:00", "pazar": "closed"}', 'ahmetozkan.com', true),
(2, 'Doç. Dr.', 'Zeynep', 'Kaya', 'doc-dr-zeynep-kaya', '+90 212 555 0201', 'zeynep.kaya@nhospital.com', '12 yıllık deneyime sahip ortopedi uzmanı. Spor yaralanmaları ve artroskopi konularında uzmanlaşmıştır.', 'Hacettepe Üniversitesi Tıp Fakültesi, Ortopedi Uzmanlığı', 12, 'Türkçe, İngilizce', '1983-07-15', 'kadın', 1, 1, '102', 30, 500.00, true, '{"pazartesi": "08:00-16:00", "salı": "08:00-16:00", "çarşamba": "08:00-16:00", "perşembe": "08:00-16:00", "cuma": "08:00-16:00", "cumartesi": "08:00-12:00", "pazar": "closed"}', 'zeynep-kaya.com', true),

-- Kardiyoloji Doctors
(3, 'Prof. Dr.', 'Mehmet', 'Yılmaz', 'prof-dr-mehmet-yilmaz', '+90 212 555 0202', 'mehmet.yilmaz@nhospital.com', '25 yıllık deneyime sahip kardiyoloji uzmanı. Kalp hastalıklarının teşhis ve tedavisinde uzmanlaşmıştır.', 'İstanbul Üniversitesi Tıp Fakültesi, Kardiyoloji Uzmanlığı', 25, 'Türkçe, İngilizce', '1970-05-15', 'erkek', 2, 1, '201', 30, 650.00, true, '{"pazartesi": "09:00-17:00", "salı": "09:00-17:00", "çarşamba": "09:00-17:00", "perşembe": "09:00-17:00", "cuma": "09:00-17:00", "cumartesi": "09:00-12:00", "pazar": "closed"}', 'mehmet-yilmaz.com', true),
(4, 'Op. Dr.', 'Ayşe', 'Demir', 'op-dr-ayse-demir', '+90 212 555 0203', 'ayse.demir@nhospital.com', '15 yıllık deneyime sahip kardiyoloji uzmanı. Girişimsel kardiyoloji konularında uzmanlaşmıştır.', 'Ankara Üniversitesi Tıp Fakültesi, Kardiyoloji Uzmanlığı', 15, 'Türkçe, İngilizce', '1978-08-22', 'kadın', 2, 1, '202', 30, 550.00, true, '{"pazartesi": "08:00-16:00", "salı": "08:00-16:00", "çarşamba": "08:00-16:00", "perşembe": "08:00-16:00", "cuma": "08:00-16:00", "cumartesi": "08:00-12:00", "pazar": "closed"}', 'ayse-demir.com', true),

-- Fizik Tedavi Doctors
(5, 'Uzm. Dr.', 'Fatma', 'Çelik', 'uzm-dr-fatma-celik', '+90 212 555 0204', 'fatma.celik@nhospital.com', '10 yıllık deneyime sahip fizik tedavi uzmanı. Rehabilitasyon ve egzersiz tedavilerinde uzmanlaşmıştır.', 'Gazi Üniversitesi Tıp Fakültesi, Fizik Tedavi Uzmanlığı', 10, 'Türkçe, İngilizce', '1985-11-30', 'kadın', 3, 1, '301', 45, 400.00, true, '{"pazartesi": "08:00-17:00", "salı": "08:00-17:00", "çarşamba": "08:00-17:00", "perşembe": "08:00-17:00", "cuma": "08:00-17:00", "cumartesi": "08:00-12:00", "pazar": "closed"}', 'fatma-celik.com', true),
(6, 'Dr.', 'Hasan', 'Arslan', 'dr-hasan-arslan', '+90 212 555 0205', 'hasan.arslan@nhospital.com', '8 yıllık deneyime sahip fizik tedavi uzmanı. Spor fizyoterapisi konularında uzmanlaşmıştır.', 'Marmara Üniversitesi Tıp Fakültesi, Fizik Tedavi Uzmanlığı', 8, 'Türkçe', '1987-01-20', 'erkek', 3, 1, '302', 45, 350.00, true, '{"pazartesi": "09:00-17:00", "salı": "09:00-17:00", "çarşamba": "09:00-17:00", "perşembe": "09:00-17:00", "cuma": "09:00-17:00", "cumartesi": "09:00-12:00", "pazar": "closed"}', 'hasan-arslan.com', true),

-- Oftalmoloji Doctors
(7, 'Prof. Dr.', 'Elif', 'Şahin', 'prof-dr-elif-sahin', '+90 212 555 0206', 'elif.sahin@nhospital.com', '18 yıllık deneyime sahip göz hastalıkları uzmanı. Katarakt ve retina cerrahisinde uzmanlaşmıştır.', 'İstanbul Üniversitesi Cerrahpaşa Tıp Fakültesi, Oftalmoloji Uzmanlığı', 18, 'Türkçe, İngilizce, Almanca', '1977-09-05', 'kadın', 4, 1, '401', 30, 700.00, true, '{"pazartesi": "08:00-16:00", "salı": "08:00-16:00", "çarşamba": "08:00-16:00", "perşembe": "08:00-16:00", "cuma": "08:00-16:00", "cumartesi": "08:00-12:00", "pazar": "closed"}', 'elif-sahin.com', true),
(8, 'Doç. Dr.', 'Oğuz', 'Kılıç', 'doc-dr-oguz-kilic', '+90 212 555 0207', 'oguz.kilic@nhospital.com', '14 yıllık deneyime sahip göz hastalıkları uzmanı. Lazer tedavileri ve miyopi cerrahisinde uzmanlaşmıştır.', 'Hacettepe Üniversitesi Tıp Fakültesi, Oftalmoloji Uzmanlığı', 14, 'Türkçe, İngilizce', '1981-12-18', 'erkek', 4, 1, '402', 30, 600.00, true, '{"pazartesi": "09:00-17:00", "salı": "09:00-17:00", "çarşamba": "09:00-17:00", "perşembe": "09:00-17:00", "cuma": "09:00-17:00", "cumartesi": "09:00-12:00", "pazar": "closed"}', 'oguz-kilic.com', true);

-- =====================================================
-- 9. DOCTOR EXPERIENCES - SOME DOCTORS HAVE EXPERIENCES
-- =====================================================

INSERT INTO doctor_experiences (
    drid, 
    name, 
    start_date, 
    end_date, 
    description, 
    is_active
) VALUES
-- Prof. Dr. Ahmet Özkan experiences
(1, 'İstanbul Üniversitesi Tıp Fakültesi Araştırma Görevlisi', '2000-01-01', '2005-12-31', 'Ortopedi bölümünde araştırma görevlisi olarak çalıştım.', true),
(1, 'Acıbadem Hastanesi Ortopedi Uzmanı', '2006-01-01', '2015-12-31', '10 yıl boyunca Acıbadem Hastanesi''nde ortopedi uzmanı olarak görev yaptım.', true),

-- Prof. Dr. Mehmet Yılmaz experiences  
(3, 'Türkiye Kalp Vakfı Araştırmacı', '1998-01-01', '2003-12-31', 'Kalp hastalıkları araştırmaları üzerine çalıştım.', true),
(3, 'Amerikan Kalp Derneği Misafir Araştırmacı', '2010-06-01', '2011-06-30', 'Amerika''da kalp cerrahisi teknikleri üzerine araştırma yaptım.', true),

-- Prof. Dr. Elif Şahin experiences
(7, 'Almanya Göz Hastalıkları Enstitüsü Fellowship', '2008-01-01', '2009-12-31', 'Almanya''da katarakt cerrahisi konusunda uzmanlaştım.', true);

-- =====================================================
-- 10. DOCTOR SPECIALIZATIONS - EACH DOCTOR LINKED TO SPECIALIZATIONS
-- =====================================================

INSERT INTO doctor_expertises (
    drid, 
    uzid, 
    certification_date, 
    certification_institution, 
    is_primary
) VALUES
-- Ortopedi doctors linked to Bel Fıtığı
(1, 2, '2005-06-15', 'İstanbul Üniversitesi Tıp Fakültesi', true),
(2, 2, '2011-09-20', 'Hacettepe Üniversitesi Tıp Fakültesi', true),

-- Kardiyoloji doctors - no direct match with given specializations, linking to closest
(3, 5, '1998-07-10', 'İstanbul Üniversitesi Tıp Fakültesi', true), -- Böbrek Nakli (closest to cardiac surgery)
(4, 6, '2008-05-25', 'Ankara Üniversitesi Tıp Fakültesi', true), -- Karaciğer Nakli (closest to cardiac surgery)

-- Fizik Tedavi doctors linked to Parkinson
(5, 3, '2013-04-15', 'Gazi Üniversitesi Tıp Fakültesi', true),
(6, 3, '2015-08-10', 'Marmara Üniversitesi Tıp Fakültesi', true),

-- Oftalmoloji doctors linked to Katarakt and Miyopi
(7, 1, '2007-03-20', 'İstanbul Üniversitesi Cerrahpaşa Tıp Fakültesi', true), -- Katarakt
(8, 4, '2009-11-15', 'Hacettepe Üniversitesi Tıp Fakültesi', true); -- Miyopi

-- =====================================================
-- 11. EXAMINATIONS - TETKIK TYPES
-- =====================================================

INSERT INTO tedkikler (
    name, 
    url_name, 
    description, 
    is_active
) VALUES
('MR Görüntüleme', 'mr-goruntuleme', 'Manyetik rezonans görüntüleme ile detaylı vücut taraması', true),
('Bilgisayarlı Tomografi (BT)', 'bilgisayarli-tomografi', 'BT cihazı ile kesitsel görüntüleme', true),
('Ultrasonografi', 'ultrasonografi', 'Ses dalgaları ile iç organ görüntüleme', true),
('Röntgen', 'rontgen', 'X-ışını ile kemik ve akciğer görüntüleme', true),
('EKG', 'ekg', 'Kalp elektriksel aktivitesinin ölçümü', true),
('Kan Tahlili', 'kan-tahlili', 'Kapsamlı kan analizi ve biyokimya testleri', true),
('İdrar Tahlili', 'idrar-tahlili', 'İdrar analizi ve mikrobiyoloji testleri', true),
('Endoskopi', 'endoskopi', 'Sindirim sistemi içsel görüntüleme', true),
('Kolonoskopi', 'kolonoskopi', 'Kalın bağırsak içsel görüntüleme', true),
('Mamografi', 'mamografi', 'Meme kanseri tarama görüntülemesi', true);

-- =====================================================
-- 12. TIBBI BIRIMLER - TIBBI BIRIMLER
-- =====================================================

INSERT INTO tibbi_birimler (
    name, 
    url_name, 
    description, 
    is_active
) VALUES

('Ortopedi', 'ortopedi', 'Ortopedi ve Travmatoloji bölümü', true),
('Kardiyoloji', 'kardiyoloji', 'Kalp ve damar hastalıkları bölümü', true),
('Fizik Tedavi Ve Rehabilitasyon', 'fizik-tedavi-rehabilitasyon', 'Fizik tedavi ve rehabilitasyon bölümü', true),
('Oftalmoloji', 'oftalmoloji', 'Göz hastalıkları bölümü', true);



-- =====================================================
-- 12. CUSTOM CONTENTS - ABOUT-US, PRIVACY-POLICY, MISSION-VISION
-- =====================================================

INSERT INTO custom_contents (
    name, 
    content_type, 
    sort_order, 
    url_name, 
    content_html, 
    description, 
    is_active
) VALUES
('Hakkımızda', 'about-us', 1, 'hakkimizda', 
'<div class="service-details__content"><div class="service-details__inner"><div class="service-details__thumbnail wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><img src="../files/defaults/logo/n-hospital-logo.png" alt="Hakkımızda Resmi" title="Hakkımızda Resmi"></div><div class="service-details__content__box wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__title">Hakkımızda</h3><p class="service-details__text">N-Hospital, 2025 yılında, hastaneler için yüksek hız ve özelleştirme için geliştirilmiş bir yönetim sistemidir. Nermefraz Bilişim Teknolojileri, Yazılım Ve Danışmanlık tarafından tasarlanmış olan sistem, Hastaneler ve Hastane Zincirleri için Yüksek performans ve özelleştirme için geliştirilmiş bir yönetim sistemidir.</p></div></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__sub-title">N-Hospital''ın Amacı</h3><p class="service-details__text">N-Hospital, hastaneler için yüksek hız ve özelleştirme için geliştirilmiş bir yönetim sistemidir. Hastaneler ve Hastane Zincirleri için Yüksek performans ve özelleştirme için geliştirilmiş bir yönetim sistemidir.</p></div><div class="service-details__info"><div class="row gutter-y-30"><div class="col-xl-6 col-lg-12 col-md-6 wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><ul class="list-unstyled service-details__info__list"><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Yüksek performans</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Özelleştirme</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Yönetim</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Hız</li></ul></div><div class="col-xl-6 col-lg-12 col-md-6 wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="100ms"><img src="../files/defaults/other/hospital-reception-1.jpg" alt="Hakkımızda Resmi" class="service-details__info__image"></div></div></div><p class="service-details__text-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms">Her sektörden firma için, teknik borcunu azaltmak ve onlara kullanışlı, performanslı, en yüksek kaliteli dijital ürün ve hizmetler sunan Nermefraz Teknoloji, özel yazılım, teknik danışmanlık gibi uzmanlık alanlarında büyükten küçüğe her boyutta işletme''ye uygun, kaliteli hizmet anlayışıyla çalışmalarını sürdürmektedir.</p><div class="service-details__faq"><h3 class="service-details__faq__title service-details__sub-title">Sık Sorulan Sorular</h3><div class="faq-accordion mediox-accordion" data-grp-name="mediox-accordion"><div class="accordion active wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><div class="accordion-title"><h4>N-Hospital Neleri Özelleştirme İmkanı Sunuyor?<span class="accordion-title__icon"></span></h4></div><div class="accordion-content"><div class="inner"><p>N-Hospital, sistemin şu özellikleri için %100 özelleştirme imkanları sunuyor:</p><ul><li>Site Logosu</li><li>Site Renkleri</li><li>Site İsmi</li><li>Site Açıklaması</li><li>E-Posta Ayarları</li><li>İletişim Bilgileri</li></ul><p>Ve Diğer Birçok özelliği dilediğiniz gibi özelleştirebilirsiniz.</p></div></div></div><div class="accordion wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="50ms"><div class="accordion-title"><h4>N-Hospital''ın Hangi Platformlarda Kullanılabilir?<span class="accordion-title__icon"></span></h4></div><div class="accordion-content"><div class="inner"><p>Bu Sistem''in kurulumu için, içinde shell komutu çalıştırabileceğiniz herhangi bir sunucu yeterlidir.</p></div></div></div><div class="accordion wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="100ms"><div class="accordion-title"><h4>N-Hospital''ın Kurulum Süresi Nedir?<span class="accordion-title__icon"></span></h4></div><div class="accordion-content"><div class="inner"><p>N-Hospital''ın kurulum süresi, genelde 2 ile 5 iş günüdür. Özel durumlarda bu süre uzayabilir.</p></div></div></div></div></div>', 
'Hastanemizin tarihçesi ve hizmet anlayışı hakkında genel bilgiler', true),

('Gizlilik Politikası', 'privacy-policy', 1, 'gizlilik-politikasi', 
'<div class="service-details__content"><div class="service-details__inner"><div class="service-details__thumbnail wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><img src="../files/defaults/logo/n-hospital-logo.png" alt="" title=""></div><div class="service-details__content__box wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__title">KVKK Aydınlatma Metni</h3><p class="service-details__text">6698 sayılı Kişisel Verilerin Korunması Kanunu ("KVKK") uyarınca, veri sorumlusu sıfatıyla N‑Hospital olarak; hizmetlerimiz kapsamında işlediğimiz kişisel verileriniz hakkında sizleri bilgilendirmek isteriz.</p></div></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__sub-title">Veri Sorumlusu</h3><p class="service-details__text">KVKK kapsamında veri sorumlusu N‑Hospital''dır. İletişim: Telefon <a href="tel:905011495699">905011495699</a>, E‑posta: info@nhospital.com.tr.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="50ms"><h3 class="service-details__sub-title">Kişisel Verilerin İşlenme Amaçları</h3><p class="service-details__text">Kimlik, iletişim, hasta işlem, finans ve işlem güvenliği verileriniz; sağlık hizmetlerinin sunulması, randevu süreçlerinin yürütülmesi, hasta memnuniyeti ve iletişim faaliyetleri, faturalama ve ödeme işlemleri, hukuki yükümlülüklerin yerine getirilmesi, bilgi güvenliği ve denetim süreçlerinin yürütülmesi amaçlarıyla işlenmektedir.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="100ms"><h3 class="service-details__sub-title">İşlenen Kişisel Veri Kategorileri</h3><p class="service-details__text">Kimlik (ad‑soyad, TCKN vb.), iletişim (telefon, e‑posta, adres), hasta işlem ve sağlık verileri, finans (fatura, ödeme bilgileri), işlem güvenliği (log kayıtları), hukuki işlem ve talep/şikayet bilgileri.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="150ms"><h3 class="service-details__sub-title">Toplama Yöntemi ve Hukuki Sebep</h3><p class="service-details__text">Kişisel verileriniz, web sitemiz, çağrı merkezimiz, çevrimiçi formlar, fiziksel başvuru ve sağlık hizmeti süreçlerinde tamamen veya kısmen otomatik yollarla; KVKK m.5/2 (a, c, ç, e, f) ve m.6/3 kapsamında sağlık hizmetlerinin sunulması, açık rıza, sözleşmenin kurulması ve ifası, hukuki yükümlülüklerin yerine getirilmesi ve meşru menfaat hukuki sebeplerine dayanılarak elde edilmektedir.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="200ms"><h3 class="service-details__sub-title">Kişisel Verilerin Aktarılması</h3><p class="service-details__text">Verileriniz, yasal zorunluluklar ve hizmetin ifası için; yetkili kamu kurum ve kuruluşları, anlaşmalı sağlık kurumları, tedarikçiler (laboratuvar, bilişim, çağrı merkezi), banka ve ödeme kuruluşları, hukuk ve mali müşavirlik hizmeti aldığımız taraflarla KVKK''ya uygun şekilde ve gerekli güvenlik önlemleri alınarak paylaşılabilecektir.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="250ms"><h3 class="service-details__sub-title">Saklama Süreleri ve İmha</h3><p class="service-details__text">Kişisel verileriniz, ilgili mevzuatta öngörülen veya işleme amaçları için gerekli azami süre boyunca saklanır; sürenin sona ermesiyle KVKK ve ilgili yönetmeliklere uygun olarak silinir, yok edilir veya anonim hale getirilir.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="300ms"><h3 class="service-details__sub-title">KVKK Madde 11 Kapsamındaki Haklarınız</h3><p class="service-details__text">Verilerinizin işlenip işlenmediğini öğrenme, işlenmişse bilgi talep etme, amaca uygun kullanımı öğrenme, yurt içinde/yurt dışında aktarıldığı üçüncü kişileri bilme, eksik/yanlış işlenmişse düzeltilmesini isteme, silinmesini/yok edilmesini isteme, yapılan işlemlerin aktarım yapılan üçüncü kişilere bildirilmesini isteme, itiraz ve zarar halinde tazmin talep etme haklarına sahipsiniz.</p></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="350ms"><h3 class="service-details__sub-title">Başvuru Yöntemi</h3><p class="service-details__text">Haklarınıza ilişkin taleplerinizi KVKK Başvuru Formu ile ya da kimliğinizi teyit edecek belgelerle birlikte; e‑posta (info@nhospital.com.tr), posta veya bizzat başvuru yöntemleriyle veri sorumlusuna iletebilirsiniz. Başvurularınız, KVKK''da öngörülen süreler içinde sonuçlandırılacaktır.</p></div></div>', 
'KVKK uyumlu gizlilik politikası metni', true),

('Misyon - Vizyon', 'mission-vision', 1, 'misyon-vizyon', 
'<div class="service-details__content"><div class="service-details__inner"><div class="service-details__thumbnail wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><img src="../files/defaults/logo/n-hospital-logo.png" alt="Misyon ve Vizyon Resmi" title="Misyon ve Vizyon Resmi"></div><div class="service-details__content__box wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__title">Misyon ve Vizyon</h3><p class="service-details__text">N-Hospital, 2025 yılında, hastaneler için yüksek hız ve özelleştirme için geliştirilmiş bir yönetim sistemidir. Nermefraz Bilişim Teknolojileri, Yazılım Ve Danışmanlık tarafından tasarlanmış olan sistem, Hastaneler ve Hastane Zincirleri için Yüksek performans ve özelleştirme için geliştirilmiş bir yönetim sistemidir.</p></div></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__sub-title">Misyon</h3><p class="service-details__text">N-Hospital, hastaneler için yüksek hız ve özelleştirme için geliştirilmiş bir yönetim sistemidir. Hastaneler ve Hastane Zincirleri için Yüksek performans ve özelleştirme için geliştirilmiş bir yönetim sistemidir.</p></div><div class="service-details__info"><div class="row gutter-y-30"><div class="col-xl-6 col-lg-12 col-md-6 wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><ul class="list-unstyled service-details__info__list"><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Yüksek performans</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Özelleştirme</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Yönetim</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Hız</li></ul></div><div class="col-xl-6 col-lg-12 col-md-6 wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="100ms"><img src="../files/defaults/other/hospital-reception-1.jpg" alt="Misyon ve Vizyon Resmi" class="service-details__info__image"></div></div></div><p class="service-details__text-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms">Her sektörden firma için, teknik borcunu azaltmak ve onlara kullanışlı, performanslı, en yüksek kaliteli dijital ürün ve hizmetler sunan Nermefraz Teknoloji, özel yazılım, teknik danışmanlık gibi uzmanlık alanlarında büyükten küçüğe her boyutta işletme''ye uygun, kaliteli hizmet anlayışıyla çalışmalarını sürdürmektedir.</p><div class="service-details__faq"><h3 class="service-details__faq__title service-details__sub-title">Sık Sorulan Sorular</h3><div class="faq-accordion mediox-accordion" data-grp-name="mediox-accordion"><div class="accordion active wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><div class="accordion-title"><h4>N-Hospital Neleri Özelleştirme İmkanı Sunuyor?<span class="accordion-title__icon"></span></h4></div><div class="accordion-content"><div class="inner"><p>N-Hospital, sistemin şu özellikleri için %100 özelleştirme imkanları sunuyor:</p><ul><li>Site Logosu</li><li>Site Renkleri</li><li>Site İsmi</li><li>Site Açıklaması</li><li>E-Posta Ayarları</li><li>İletişim Bilgileri</li></ul><p>Ve Diğer Birçok özelliği dilediğiniz gibi özelleştirebilirsiniz.</p></div></div></div><div class="accordion wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="50ms"><div class="accordion-title"><h4>N-Hospital''ın Hangi Platformlarda Kullanılabilir?<span class="accordion-title__icon"></span></h4></div><div class="accordion-content"><div class="inner"><p>Bu Sistem''in kurulumu için, içinde shell komutu çalıştırabileceğiniz herhangi bir sunucu yeterlidir.</p></div></div></div><div class="accordion wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="100ms"><div class="accordion-title"><h4>N-Hospital''ın Kurulum Süresi Nedir?<span class="accordion-title__icon"></span></h4></div><div class="accordion-content"><div class="inner"><p>N-Hospital''ın kurulum süresi, genelde 2 ile 5 iş günüdür. Özel durumlarda bu süre uzayabilir.</p></div></div></div></div></div></div>', 
'Hastanemizin misyon ve vizyon açıklaması', true),

('İnsan Kaynakları', 'hr', 1, 'insan-kaynaklari', 
'<div class="service-details__inner"><div class="service-details__content__box wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__title">İnsan Kaynakları</h3><p class="service-details__text">Hastanemizde kaliteli sağlık hizmetleri sunmak için yetenekli ve deneyimli profesyonellerle çalışıyoruz. İnsan kaynakları departmanımız, hastanemizin değerlerine uygun, hasta odaklı ve sürekli gelişim anlayışına sahip çalışanlar bulmak için çalışmaktadır.</p></div></div><div class="service-details__inner-two wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><h3 class="service-details__sub-title">Kariyer Fırsatları</h3><p class="service-details__text">Hastanemizde doktor, hemşire, teknisyen, idari personel ve diğer sağlık profesyonelleri için çeşitli kariyer fırsatları sunuyoruz. Sürekli eğitim ve gelişim programları ile çalışanlarımızın mesleki gelişimlerini destekliyoruz.</p></div><div class="service-details__info"><div class="row gutter-y-30"><div class="col-xl-6 col-lg-12 col-md-6 wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="00ms"><ul class="list-unstyled service-details__info__list"><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Rekabetçi maaş paketleri</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Sağlık sigortası</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Sürekli eğitim imkanları</li><li><span class="service-details__list__icon"><i class="icon-check"></i></span>Kariyer gelişim desteği</li></ul></div><div class="col-xl-6 col-lg-12 col-md-6 wow fadeInUp" data-wow-duration="1500ms" data-wow-delay="100ms"><img src="../files/defaults/other/hospital-reception-1.jpg" alt="N-Hospital Hastane Resepsiyonu ismi." class="service-details__info__image"></div></div></div>', 
'Hastanemizdeki insan kaynakları hakkında genel bilgiler', true);



-- Homepage welcome popup (HTML and CSS in separate columns; CSS has no <style> tag)
INSERT INTO homepage_contents (
    name,
    content_type,
    sort_order,
    url_name,
    content_html,
    content_css,
    description,
    is_active
) 

VALUES (
    'Welcome Popup',
    'popup',
    1,
    'welcome-popup',
    '<div class="popup-header"><img class="popup-header-image" src="/files/defaults/logo/n-hospital-logo.png" alt="popup header" /></div><div class="popup-body"><div class="popup-content-item"><div class="popup-content-label">Hoş Geldiniz</div><div class="popup-content-value">N‑Hospital CMS’e hoş geldiniz. Bu panel; hastane içeriğinizi, randevu süreçlerinizi, haber ve sayfalarınızı tek bir yerden yönetebilmeniz için tasarlanmış modern bir içerik ve operasyon yönetim sistemidir.</div></div><div class="popup-content-item"><div class="popup-content-label">Amacı</div><div class="popup-content-value">Şubeler, doktorlar, branşlar, tetkikler ve içerikler gibi kurumsal varlıkları merkezi olarak düzenlemenizi; randevu ve başvuru akışlarını görünür, ölçülebilir ve yönetilebilir hale getirmenizi sağlar.</div></div><div class="popup-content-item"><div class="popup-content-label">Kimler İçin?</div><div class="popup-content-value">Hastane yöneticileri ve moderatörler, içerik/editör ekipleri, İK ve çağrı merkezi çalışanları için optimize edilmiştir. Rolünüze göre yetkilendirilmiş, güvenli ve verimli bir deneyim sunar.</div></div><div class="popup-content-item"><div class="popup-content-label">İpuçları</div><div class="popup-content-value">Sol menüden ilgili modüllere ulaşabilir, üst bar bildirimlerinden anlık iş akışlarını takip edebilirsiniz. Yardım gerekli olduğunda sistem yöneticinizle iletişime geçebilirsiniz.</div></div></div>',
    '.popup-header{padding:0;text-align:center;position:relative;overflow:hidden;height:200px}.popup-header::before{content:'''';position:absolute;inset:0;background:transparent}@keyframes popup-header-glow{0%,100%{transform:translate(0,0)}50%{transform:translate(20px,20px)}}.popup-header-image{width:100%;height:100%;object-fit:contain;display:flex;align-items:center;place-content:center}@keyframes popup-icon-bounce{0%{transform:scale(0);opacity:0}50%{transform:scale(1.2)}100%{transform:scale(1);opacity:1}}.popup-title{display:none}.popup-subtitle{display:none}@keyframes popup-slide-up{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}.popup-body{padding:32px 24px;max-height:60vh;overflow-y:auto}.popup-body::-webkit-scrollbar{width:8px}.popup-body::-webkit-scrollbar-track{background:#f1f1f1;border-radius:4px}.popup-body::-webkit-scrollbar-thumb{background:#667eea;border-radius:4px}.popup-body::-webkit-scrollbar-thumb:hover{background:#764ba2}.popup-content-item{margin-bottom:20px;animation:popup-fade-in .5s ease-out backwards}.popup-content-item:nth-child(1){animation-delay:.6s}.popup-content-item:nth-child(2){animation-delay:.7s}.popup-content-item:nth-child(3){animation-delay:.8s}.popup-content-item:nth-child(4){animation-delay:.9s}@keyframes popup-fade-in{from{opacity:0;transform:translateX(-20px)}to{opacity:1;transform:translateX(0)}}.popup-content-label{font-size:22px;font-weight:700;color:#252525;text-transform:none;letter-spacing:0;margin:0 0 10px 0;text-align:center}.popup-content-value{font-size:16px;color:#333;line-height:1.6}.popup-footer{padding:20px 24px;background:#f8f9fa;display:flex;gap:12px;justify-content:flex-end;border-top:1px solid #e9ecef}.popup-btn{padding:12px 24px;border-radius:8px;font-size:14px;font-weight:600;cursor:pointer;transition:all .3s ease;border:none;outline:none}.popup-btn-secondary{background:#e9ecef;color:#495057}.popup-btn-secondary:hover{background:#dee2e6;transform:translateY(-2px)}.popup-btn-primary{background:linear-gradient(135deg,#667eea 0%,#764ba2 100%);color:#fff}.popup-btn-primary:hover{transform:translateY(-2px);box-shadow:0 8px 20px rgba(102,126,234,.4)}@media (max-width:600px){.popup-body{padding:24px 16px}.popup-footer{padding:16px;flex-direction:column}.popup-btn{width:100%}}',
    'Ana sayfa için karşılama popup içeriği',
    true
),

('Banner Page 1', 'banner_page', 1, 'banner-page-1', 
'<div class="main-slider-three__item"><div class="main-slider-three__bg" style="background-image: url(files/defaults/banner/slider-1.webp);"></div><div class="main-slider-three__container container"><div class="banner-row row"><div class="col-xxl-10 col-xl-9 mx-auto"><div class="main-slider-three__content"><div class="main-slider-three__top"><p class="main-slider-three__sub-title">KALICI ETKİ</p></div><h2 class="main-slider-three__title"><span class="main-slider-three__title__inner">Kalıcı <br> lens <span class="main-slider-three__title__shape">cerrahisi</span></span></h2><div class="main-slider-three__button-group"><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--1"><a href="/tibbi-birimler/trikofal-lens-cerrahisi" class="mediox-btn"><span>İncele</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--2"><a href="#randevu" class="mediox-btn"><span>Randevu Al</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div></div></div></div></div></div></div>',
NULL,
'Banner sayfası için banner içeriği',
true
),

('Banner Page 2', 'banner_page', 2, 'banner-page-2', 
'<div class="main-slider-three__item"><div class="main-slider-three__bg" style="background-image: url(files/defaults/banner/slider-2.webp);"></div><div class="main-slider-three__container container"><div class="banner-row row"><div class="col-xxl-10 col-xl-9 mx-auto"><div class="main-slider-three__content"><div class="main-slider-three__top"><p class="main-slider-three__sub-title">Retina''nız Emin Ellerde</p></div><h2 class="main-slider-three__title"><span class="main-slider-three__title__inner">Retina <br><span class="main-slider-three__title__shape">Cerrahisi</span></span></h2><div class="main-slider-three__button-group"><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--1"><a href="/tibbi-birimler/retina-tedavisi" class="mediox-btn"><span>İncele</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--2"><a href="#randevu" class="mediox-btn"><span>Randevu Al</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div></div></div></div></div></div></div>',
NULL,
'Banner sayfası için banner içeriği',
true
),

('Banner Page 3', 'banner_page', 3, 'banner-page-3', 
'<div class="main-slider-three__item"><div class="main-slider-three__bg" style="background-image: url(files/defaults/banner/slider-3.webp);"></div><div class="main-slider-three__container container"><div class="banner-row row"><div class="col-xxl-10 col-xl-9 mx-auto"><div class="main-slider-three__content"><div class="main-slider-three__top"><p class="main-slider-three__sub-title">Gözünüzdeki Sorunlarımızı Çözüyoruz</p></div><h2 class="main-slider-three__title"><span class="main-slider-three__title__inner">Glokom <br> <span class="main-slider-three__title__shape">Tedavisi</span></span></h2><div class="main-slider-three__button-group"><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--1"><a href="/tibbi-birimler/glokom-tedavisi" class="mediox-btn"><span>İncele</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--2"><a href="#randevu" class="mediox-btn"><span>Randevu Al</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div></div></div></div></div></div></div>',
NULL,
'Banner sayfası için banner içeriği',
true
),

('Banner Page 4', 'banner_page', 4, 'banner-page-4', 
'<div class="main-slider-three__item"><div class="main-slider-three__bg" style="background-image: url(files/defaults/banner/slider-4.webp);"></div><div class="main-slider-three__container container"><div class="banner-row row"><div class="col-xxl-10 col-xl-9 mx-auto"><div class="main-slider-three__content"><div class="main-slider-three__top"><p class="main-slider-three__sub-title">Çocuklarımız İçin En İyi Bakım</p></div><h2 class="main-slider-three__title"><span class="main-slider-three__title__inner">Şaşılık <br> <span class="main-slider-three__title__shape">Tedavisi</span></span></h2><div class="main-slider-three__button-group"><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--1"><a href="/tibbi-birimler/sasilik-tedavisi" class="mediox-btn"><span>İncele</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div><div class="main-slider-three__button-group__inner"><div class="main-slider-three__button main-slider-three__button--2"><a href="#randevu" class="mediox-btn"><span>Randevu Al</span><span class="mediox-btn__icon"><i class="icon-up-right-arrow"></i></span></a></div></div></div></div></div></div></div></div>',
NULL,
'Banner sayfası için banner içeriği',
true
);

-- =====================================================
-- ADDITIONAL SEEDS FOR PROPER FUNCTIONALITY
-- =====================================================

-- ========================
-- Fix sequences after seed
-- ========================
SELECT setval('doktorlar_drid_seq', (SELECT COALESCE(MAX(drid), 0) FROM doktorlar) + 1);
SELECT setval('options_oid_seq', (SELECT COALESCE(MAX(oid), 0) FROM options) + 1);
SELECT setval('users_uid_seq', (SELECT COALESCE(MAX(uid), 0) FROM users) + 1);
SELECT setval('medias_mid_seq', (SELECT COALESCE(MAX(mid), 0) FROM medias) + 1);
SELECT setval('header_buttons_hbid_seq', (SELECT COALESCE(MAX(hbid), 0) FROM header_buttons) + 1);
SELECT setval('testimonials_tid_seq', (SELECT COALESCE(MAX(tid), 0) FROM testimonials) + 1);
SELECT setval('subeler_sid_seq', (SELECT COALESCE(MAX(sid), 0) FROM subeler) + 1);
SELECT setval('anlasmali_kurumlar_akid_seq', (SELECT COALESCE(MAX(akid), 0) FROM anlasmali_kurumlar) + 1);
SELECT setval('uzmanliklar_uzid_seq', (SELECT COALESCE(MAX(uzid), 0) FROM uzmanliklar) + 1);
SELECT setval('branslar_brid_seq', (SELECT COALESCE(MAX(brid), 0) FROM branslar) + 1);
SELECT setval('doctor_experiences_dtid_seq', (SELECT COALESCE(MAX(dtid), 0) FROM doctor_experiences) + 1);
SELECT setval('doctor_expertises_duid_seq', (SELECT COALESCE(MAX(duid), 0) FROM doctor_expertises) + 1);
SELECT setval('tibbi_birimler_tbid_seq', (SELECT COALESCE(MAX(tbid), 0) FROM tibbi_birimler) + 1);
SELECT setval('homepage_contents_hcid_seq', (SELECT COALESCE(MAX(hcid), 0) FROM homepage_contents) + 1);
SELECT setval('custom_contents_ccid_seq', (SELECT COALESCE(MAX(ccid), 0) FROM custom_contents) + 1);
SELECT setval('haberler_hid_seq', (SELECT COALESCE(MAX(hid), 0) FROM haberler) + 1);
SELECT setval('tedkikler_tid_seq', (SELECT COALESCE(MAX(tid), 0) FROM tedkikler) + 1);
SELECT setval('randevu_talepleri_rrid_seq', (SELECT COALESCE(MAX(rrid), 0) FROM randevu_talepleri) + 1);
SELECT setval('randevular_rid_seq', (SELECT COALESCE(MAX(rid), 0) FROM randevular) + 1);
SELECT setval('contact_requests_crid_seq', (SELECT COALESCE(MAX(crid), 0) FROM contact_requests) + 1);
SELECT setval('job_applications_jaid_seq', (SELECT COALESCE(MAX(jaid), 0) FROM job_applications) + 1);
