-- Hospital CMS Database Schema
-- PostgreSQL Database for comprehensive hospital management system

-- =====================================================
-- SYSTEM CONFIGURATION TABLES
-- =====================================================

DROP SCHEMA public CASCADE;

-- Tekrar oluştur
CREATE SCHEMA public;

-- Yetkileri geri ver
GRANT ALL ON SCHEMA public TO necdet;  -- bağlandığın kullanıcı
GRANT ALL ON SCHEMA public TO public;

-- Comprehensive options table for complete CMS configuration
CREATE TABLE options (
    oid SERIAL PRIMARY KEY,

    option_set_name VARCHAR(255) DEFAULT 'varsayılan',
    option_set_description TEXT,
    option_set_is_active BOOLEAN DEFAULT FALSE,
    option_set_is_testing_now BOOLEAN DEFAULT FALSE,
    option_set_created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    option_set_updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    site_name VARCHAR(255) DEFAULT 'N-Hospital CMS',
    site_description TEXT,
    site_logo_mid INTEGER DEFAULT NULL,
    site_light_logo_mid INTEGER DEFAULT NULL,
    site_favicon_mid INTEGER DEFAULT NULL,
    default_page_mid INTEGER DEFAULT NULL,

    preloader TEXT DEFAULT NULL,

    maintenance_mode BOOLEAN DEFAULT FALSE,

    smtp_host VARCHAR(255),
    smtp_port INTEGER,
    smtp_username VARCHAR(255),
    smtp_password VARCHAR(255),
    smtp_encryption VARCHAR(20) DEFAULT 'tls' CHECK (smtp_encryption IN ('tls', 'ssl', 'none')),

    facebook_url VARCHAR(500) DEFAULT '#',
    twitter_url VARCHAR(500) DEFAULT '#',
    instagram_url VARCHAR(500) DEFAULT 'https://www.instagram.com/nermefraz',
    linkedin_url VARCHAR(500) DEFAULT 'https://www.linkedin.com/company/107576047',

    google_recaptcha_site_key VARCHAR(255) DEFAULT '6LeIxAcTAAAAAJcZVRqyHh71UMIEGNQ_MXjiZKhI',
    google_recaptcha_secret_key VARCHAR(255) DEFAULT '6LeIxAcTAAAAAGG-vFI1TnRWxMZNFuojJ4WifJWe',

    contact_email VARCHAR(255) DEFAULT 'info@nermefraz.com',
    contact_phone VARCHAR(255) DEFAULT '+90 501 149 56 99',

    main_page_meta_title VARCHAR(255) DEFAULT 'N-Hospital CMS, Hastaneler için En Gelişmiş CMS',
    main_page_meta_description VARCHAR(1000) DEFAULT 'N-Hospital CMS, Nermefraz Bilişim Teknolojileri, Yazılım Ve Danışmanlık tarafından tasarlanmış bir hastane yönetim sistemidir.',
    google_analytics TEXT,

    primary_color VARCHAR(25) DEFAULT 'rgba(40, 59, 106, 1)',
    secondary_color VARCHAR(25) DEFAULT 'rgba(223, 0, 36, 1)',
    accent_color VARCHAR(25) DEFAULT 'rgba(51, 193, 237, 1)',
    background_color VARCHAR(25) DEFAULT 'rgba(255, 255, 255, 1)',
    font_color VARCHAR(25) DEFAULT 'rgba(51, 51, 51, 1)',
    font_family VARCHAR(255) DEFAULT 'Poppins',

    require_strong_password BOOLEAN DEFAULT FALSE,
    items_per_page INTEGER DEFAULT 10,
    show_doctors_on_same_city BOOLEAN DEFAULT FALSE,
    show_doctors_on_same_country BOOLEAN DEFAULT FALSE,
    auto_remove_partners_when_expired BOOLEAN DEFAULT FALSE,

    enable_testimonials BOOLEAN DEFAULT TRUE,
    enable_our_history BOOLEAN DEFAULT TRUE,
    maximum_sublinks_on_a_menu_item INTEGER DEFAULT 5,
    show_doctor_social_media BOOLEAN DEFAULT FALSE,
    show_doctor_appointment_fee BOOLEAN DEFAULT FALSE,

    show_anlasmali_kurum_pictures BOOLEAN DEFAULT TRUE,

    max_upload_size INTEGER DEFAULT 5242880, -- 5MB
    timezone VARCHAR(100) DEFAULT 'UTC',
    language VARCHAR(100) DEFAULT 'tr' CHECK (language IN ('tr', 'en', 'de', 'us', 'es', 'fr', 'it', 'ja', 'ko', 'pt', 'ru', 'zh')),
    UNIQUE(option_set_name)
);

CREATE TABLE users (
    uid SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    surname VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(255) NOT NULL UNIQUE,
    role VARCHAR(255) NOT NULL CHECK (role IN ('admin', 'moderator', 'santral', 'ik')),
    sid INTEGER DEFAULT NULL,
    is_active BOOLEAN DEFAULT FALSE,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    last_login TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Media files table for managing all uploaded files
CREATE TABLE medias (
    mid SERIAL PRIMARY KEY,
    data TEXT,
    file_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_type VARCHAR(20) CHECK (file_type IN ('image', 'video', 'audio', 'document', 'other', 'site_logo', 'site_light_logo', 'site_favicon', 'default_page_picture', 'cv', 'experience_cover', 'tibbi_birim_cover', 'tibbi_birim_video', 'tedkik_cover')),
    target_id VARCHAR(255),
    alt_text TEXT,
    title TEXT,
    width INTEGER,
    height INTEGER,
    uid INTEGER, -- Reference to admin/user who uploaded
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (uid) REFERENCES users(uid) ON DELETE SET NULL
);

-- Header navigation buttons/links
CREATE TABLE header_buttons (
    hbid SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    url VARCHAR(255),
    target VARCHAR(20) DEFAULT '_self' CHECK (target IN ('_self', '_blank')),
    icon VARCHAR(50),
    sort_order INTEGER,
    is_active BOOLEAN DEFAULT TRUE,
    button_type VARCHAR(20) DEFAULT 'diger' CHECK (button_type IN ('anasayfa', 'kurumsal', 'subeler', 'branslar', 'doktorlar', 'haberler', 'tedkikler', 'tibbi_birimler', 'diger')),
    parent_id INTEGER,  -- Artık sadece integer, foreign key yok    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE testimonials (
    tid SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    occupation VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    rating DECIMAL(10,2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT TRUE,
    customer_picture_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- HOSPITAL STRUCTURE TABLES
-- =====================================================

-- Hospital branches/locations
CREATE TABLE subeler (
    sid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    description TEXT,
    address TEXT NOT NULL,
    google_map_iframe TEXT DEFAULT NULL,
    city VARCHAR(100) NOT NULL,
    district VARCHAR(100),
    postal_code VARCHAR(20),
    phone VARCHAR(20),
    fax VARCHAR(20),
    email VARCHAR(100),
    website VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    view_count INTEGER DEFAULT 0,
    transportation_info TEXT,
    working_hours TEXT DEFAULT '{"pazartesi": "08:00-18:00", "salı": "08:00-18:00", "çarşamba": "08:00-18:00", "perşembe": "08:00-18:00", "cuma": "08:00-18:00", "cumartesi": "08:00-13:00", "pazar": "closed"}', -- Store working hours as JSON
    mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    document_mids INTEGER[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    anlasmali_kurumlar_html TEXT DEFAULT NULL,
    transportation_info_html TEXT DEFAULT NULL,
    /*is_main BOOLEAN DEFAULT TRUE,*/
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Contracted institutions (insurance companies, corporate agreements)
CREATE TABLE anlasmali_kurumlar (
    akid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('sigorta', 'kurumsal', 'universite', 'devlet', 'ozel')),
    sid INTEGER REFERENCES subeler(sid) ON DELETE CASCADE ON UPDATE CASCADE,
    contact_person VARCHAR(100),
    phone VARCHAR(20),
    email VARCHAR(100),
    address TEXT,
    contract_start_date DATE,
    contract_end_date DATE,
    discount_rate DECIMAL(5,2) DEFAULT 0.00,
    payment_terms TEXT,
    notes TEXT,
    logo_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Medical specializations/expertise areas
CREATE TABLE uzmanliklar (
    uzid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Medical departments/branches
CREATE TABLE branslar (
    brid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    url_name VARCHAR(200) NOT NULL,
    description TEXT,
    short_description TEXT,
    services TEXT[], -- Array of services provided by this department
    mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    icon VARCHAR(50),
    sid INTEGER REFERENCES subeler(sid) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
    head_drid INTEGER, -- Will reference doktorlar(id)
    phone VARCHAR(20),
    email VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Doctors table
CREATE TABLE doktorlar (
    drid SERIAL PRIMARY KEY,
    title VARCHAR(20) DEFAULT 'Dr.' CHECK (title IN ('Dr.', 'Prof. Dr.', 'Doç. Dr.', 'Op. Dr.', 'Uzm. Dr.')),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    tc_kimlik VARCHAR(11),
    diploma_no VARCHAR(50),
    calistigi_subeler_text TEXT DEFAULT NULL,
    phone VARCHAR(20),
    email VARCHAR(100),
    biography TEXT,
    education TEXT,
    experience_years INTEGER DEFAULT 0,
    languages VARCHAR(200), -- Comma separated languages
    birth_date DATE,
    gender VARCHAR(10) CHECK (gender IN ('erkek', 'kadın', 'belirtmek istemiyorum')),
    photo_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    cv_file_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    brid INTEGER REFERENCES branslar(brid) ON DELETE SET NULL,
    sid INTEGER REFERENCES subeler(sid) ON DELETE SET NULL,
    room_number VARCHAR(20),
    appointment_duration INTEGER DEFAULT 30, -- Minutes
    appointment_fee DECIMAL(10,2) DEFAULT 0.00,
    online_appointment BOOLEAN DEFAULT TRUE,
    view_count INTEGER DEFAULT 0,
    working_hours TEXT, -- Store weekly schedule as JSON
    vacation_dates TEXT, -- Store vacation periods as JSON
    facebook_url VARCHAR(500),
    x_url VARCHAR(500),
    instagram_url VARCHAR(500),
    linkedin_url VARCHAR(500),
    personal_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE doctor_experiences (
    dtid SERIAL PRIMARY KEY,
    drid INTEGER REFERENCES doktorlar(drid) ON DELETE CASCADE ON UPDATE CASCADE,
    name VARCHAR(200) NOT NULL,
    start_date DATE,
    end_date DATE,
    cover_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Doctor specializations (many-to-many relationship)
CREATE TABLE doctor_expertises (
    duid SERIAL PRIMARY KEY,
    drid INTEGER REFERENCES doktorlar(drid) ON DELETE CASCADE ON UPDATE CASCADE,
    uzid INTEGER REFERENCES uzmanliklar(uzid) ON DELETE CASCADE,
    certification_date DATE,
    certification_institution VARCHAR(200),
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add foreign key constraint after doktorlar table is created
ALTER TABLE branslar ADD CONSTRAINT fk_branslar_head_doctor 
    FOREIGN KEY (head_drid) REFERENCES doktorlar(drid) ON DELETE SET NULL ON UPDATE CASCADE;

-- =====================================================
-- CONTENT MANAGEMENT TABLES
-- =====================================================

CREATE TABLE tibbi_birimler (
    tbid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    description TEXT,
    html_content TEXT,
    javascript_content TEXT,
    css_content TEXT,
    cover_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    video_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    view_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE homepage_contents (
    hcid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    content_type VARCHAR(50) CHECK (content_type IN ('generic', 'popup', 'banner_page')),
    later_than_which_content INTEGER DEFAULT 1,
    sort_order INTEGER NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    content_html TEXT,
    content_javascript TEXT,
    content_css TEXT,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE custom_contents (
    ccid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    content_type VARCHAR(50) CHECK (content_type IN ('history', 'about-us', 'privacy-policy', 'mission-vision', 'main-page', 'hr', 'popup', 'other')),
    sort_order INTEGER NOT NULL,
    url_name VARCHAR(200) UNIQUE NOT NULL,
    content_html TEXT,
    content_javascript TEXT,
    content_css TEXT,
    description TEXT,
    -- if following column matches with route of the request, then this content will be shown
    route VARCHAR(200),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- News and announcements
CREATE TABLE haberler (
    hid SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    url_name VARCHAR(255) UNIQUE NOT NULL,
    summary TEXT,
    content TEXT NOT NULL,
    cover_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    category VARCHAR(100) DEFAULT 'genel',
    tags TEXT[], -- Array of tags
    author VARCHAR(100),
    publish_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_featured BOOLEAN DEFAULT FALSE,
    is_published BOOLEAN DEFAULT FALSE,
    views_count INTEGER DEFAULT 0,
    seo_title VARCHAR(255),
    seo_description TEXT,
    seo_keywords TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tedkikler (
    tid SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    url_name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    html_content TEXT,
    javascript_content TEXT,
    css_content TEXT,
    view_count INTEGER DEFAULT 0,
    cover_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- APPOINTMENT AND PATIENT INTERACTION TABLES
-- =====================================================

CREATE TABLE randevu_talepleri (
    rrid SERIAL PRIMARY KEY,
    patient_first_name VARCHAR(100) NOT NULL,
    patient_last_name VARCHAR(100) NOT NULL,
    patient_phone VARCHAR(20) NOT NULL,
    patient_email VARCHAR(100),
    preferred_date DATE,
    preferred_time TIME,
    status VARCHAR(20) DEFAULT 'yeni' CHECK (status IN ('yeni', 'randevu-verildi', 'randevu-verilemedi', 'ulasilamadi', 'gelmedi', 'hasta-vazgecti')),
    message TEXT,
    drid INTEGER REFERENCES doktorlar(drid) ON DELETE SET NULL ON UPDATE CASCADE,
    sid INTEGER REFERENCES subeler(sid) ON DELETE SET NULL ON UPDATE CASCADE,
    last_modified_uid INTEGER REFERENCES users(uid) ON DELETE SET NULL ON UPDATE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Appointments
CREATE TABLE randevular (
    rid SERIAL PRIMARY KEY,
    patient_first_name VARCHAR(100) NOT NULL,
    patient_last_name VARCHAR(100) NOT NULL,
    patient_phone VARCHAR(20) NOT NULL,
    patient_email VARCHAR(100),
    patient_tc_kimlik VARCHAR(11),
    patient_birth_date DATE,
    patient_gender VARCHAR(10) CHECK (patient_gender IN ('erkek', 'kadın', 'belirtmek istemiyorum')),
    drid INTEGER REFERENCES doktorlar(drid) ON DELETE CASCADE ON UPDATE CASCADE,
    brid INTEGER REFERENCES branslar(brid) ON DELETE CASCADE ON UPDATE CASCADE,
    sid INTEGER REFERENCES subeler(sid) ON DELETE CASCADE ON UPDATE CASCADE,
    akid INTEGER REFERENCES anlasmali_kurumlar(akid) ON DELETE SET NULL ON UPDATE CASCADE,
    tid INTEGER REFERENCES tedkikler(tid) ON DELETE SET NULL ON UPDATE CASCADE,
    tbid INTEGER REFERENCES tibbi_birimler(tbid) ON DELETE SET NULL ON UPDATE CASCADE,
    rrid INTEGER REFERENCES randevu_talepleri(rrid) ON DELETE SET NULL ON UPDATE CASCADE,
    appointment_date DATE NOT NULL,
    appointment_time TIME NOT NULL,
    duration INTEGER DEFAULT 30, -- Minutes
    status VARCHAR(20) DEFAULT 'beklemede' CHECK (status IN ('beklemede', 'onaylandi', 'iptal', 'tamamlandi', 'gelmedi')),
    notes TEXT,
    complaint TEXT, -- Patient's complaint
    cancel_reason TEXT,
    reminder_sent BOOLEAN DEFAULT FALSE,
    confirmation_code VARCHAR(10),
    price DECIMAL(10,2) DEFAULT 0.00,
    payment_status VARCHAR(20) DEFAULT 'odenmedi' CHECK (payment_status IN ('odenmedi', 'odendi', 'kismen_odendi')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(appointment_date, appointment_time)
);

-- Contact requests from website
CREATE TABLE contact_requests (
    crid SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    department VARCHAR(100), -- Which department this is for
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('dusuk', 'normal', 'yuksek', 'acil')),
    status VARCHAR(20) DEFAULT 'yeni' CHECK (status IN ('yeni', 'okundu', 'cevaplanmis', 'kapali')),
    assigned_to VARCHAR(100), -- Staff member assigned to handle this
    response TEXT,
    response_date TIMESTAMP WITH TIME ZONE,
    ip_address INET,
    user_agent TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    is_replied BOOLEAN DEFAULT FALSE,
    source VARCHAR(50) DEFAULT 'website', -- Where the request came from
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Job applications
CREATE TABLE job_applications (
    jaid SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    birth_date DATE,
    gender VARCHAR(10) CHECK (gender IN ('erkek', 'kadın', 'belirtmek istemiyorum')),
    city VARCHAR(100),
    university VARCHAR(200),
    department VARCHAR(200),
    graduation_year INTEGER,
    experience_years INTEGER DEFAULT 0,
    position_applied VARCHAR(200),
    sid INTEGER REFERENCES subeler(sid) ON DELETE SET NULL,
    salary_expectation DECIMAL(10,2),
    available_start_date DATE,
    languages VARCHAR(200), -- Comma separated languages
    skills TEXT, -- Job-related skills
    cover_letter TEXT,
    cv_file_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    diploma_file_mid INTEGER REFERENCES medias(mid) ON DELETE SET NULL,
    work_references TEXT, -- Applicant's references
    status VARCHAR(20) DEFAULT 'yeni' CHECK (status IN ('yeni', 'inceleniyor', 'mulakat', 'kabul', 'red', 'beklemede')),
    notes TEXT, -- Internal notes from HR
    interview_date TIMESTAMP WITH TIME ZONE,
    interview_notes TEXT,
    rejection_reason TEXT,
    response_date TIMESTAMP WITH TIME ZONE,
    is_read BOOLEAN DEFAULT FALSE,
    is_replied BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE notifications (
    nid SERIAL PRIMARY KEY,
    message TEXT NOT NULL,
    notification_type VARCHAR(20) CHECK (notification_type IN ('success', 'warning', 'danger', 'info')),
    notification_level VARCHAR(20) CHECK (notification_level IN ('moderator', 'admin', 'santral', 'ik', 'all')),
    is_read BOOLEAN DEFAULT FALSE,
    sid INTEGER REFERENCES subeler(sid) ON DELETE SET NULL,
    link VARCHAR(200),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Options indexes
CREATE INDEX idx_options_set_name ON options(option_set_name);
CREATE INDEX idx_options_active ON options(option_set_is_active);

-- Medias indexes
CREATE INDEX idx_medias_type ON medias(file_type);
CREATE INDEX idx_medias_uid ON medias(uid);

-- Header buttons indexes
CREATE INDEX idx_header_buttons_active ON header_buttons(is_active);
CREATE INDEX idx_header_buttons_parent ON header_buttons(parent_id);
CREATE INDEX idx_header_buttons_order ON header_buttons(sort_order);

-- Branches indexes
CREATE INDEX idx_subeler_url_name ON subeler(url_name);
CREATE INDEX idx_subeler_active ON subeler(is_active);
/*CREATE INDEX idx_subeler_main ON subeler(is_main);*/

-- Contracted institutions indexes
CREATE INDEX idx_anlasmali_kurumlar_url_name ON anlasmali_kurumlar(url_name);
CREATE INDEX idx_anlasmali_kurumlar_type ON anlasmali_kurumlar(type);
CREATE INDEX idx_anlasmali_kurumlar_active ON anlasmali_kurumlar(is_active);

-- Specializations indexes
CREATE INDEX idx_uzmanliklar_url_name ON uzmanliklar(url_name);
CREATE INDEX idx_uzmanliklar_active ON uzmanliklar(is_active);

-- Departments indexes
CREATE INDEX idx_branslar_url_name ON branslar(url_name);
CREATE INDEX idx_branslar_sid ON branslar(sid);
CREATE INDEX idx_branslar_active ON branslar(is_active);

-- Doctors indexes
CREATE INDEX idx_doktorlar_url_name ON doktorlar(url_name);
CREATE INDEX idx_doktorlar_brid ON doktorlar(brid);
CREATE INDEX idx_doktorlar_sid ON doktorlar(sid);
CREATE INDEX idx_doktorlar_active ON doktorlar(is_active);
CREATE INDEX idx_doktorlar_appointment ON doktorlar(online_appointment);

-- News indexes
CREATE INDEX idx_haberler_url_name ON haberler(url_name);
CREATE INDEX idx_haberler_published ON haberler(is_published);
CREATE INDEX idx_haberler_featured ON haberler(is_featured);
CREATE INDEX idx_haberler_publish_date ON haberler(publish_date);
CREATE INDEX idx_haberler_category ON haberler(category);

-- Appointments indexes
CREATE INDEX idx_randevular_drid ON randevular(drid);
CREATE INDEX idx_randevular_date ON randevular(appointment_date);
CREATE INDEX idx_randevular_status ON randevular(status);
CREATE INDEX idx_randevular_patient_phone ON randevular(patient_phone);

-- Contact requests indexes
CREATE INDEX idx_contact_requests_status ON contact_requests(status);
CREATE INDEX idx_contact_requests_created ON contact_requests(created_at);

-- Job applications indexes
CREATE INDEX idx_job_applications_status ON job_applications(status);
CREATE INDEX idx_job_applications_position ON job_applications(position_applied);
CREATE INDEX idx_job_applications_created ON job_applications(created_at);

-- =====================================================
-- DATABASE TRIGGERS FOR DATA CONSISTENCY
-- =====================================================

CREATE OR REPLACE FUNCTION get_shift_for_insert(
    p_sort_order INT,
    p_hbid INT,
    p_parent_id INT DEFAULT NULL
)
RETURNS TABLE(our_hbid INT, new_sort_order INT)
LANGUAGE plpgsql
AS $$
DECLARE
    v_max_order INT;
    v_new_sort INT;
BEGIN
    -- mevcut max sort_order
    SELECT COALESCE(MAX(sort_order), 0)
    INTO v_max_order
    FROM header_buttons
    WHERE parent_id IS NOT DISTINCT FROM p_parent_id
    AND hbid != p_hbid;

    -- yeni sıra kararını ver
    IF p_sort_order > v_max_order THEN
        v_new_sort := v_max_order + 1;

        -- yeni eklenecek satırı döndür
        RETURN QUERY
        SELECT NULL::INT, v_new_sort;
        RETURN;

    ELSIF p_sort_order < 1 THEN
        v_new_sort := 1;
    ELSE
        v_new_sort := p_sort_order;
    END IF;

    -- önce kaydırılacak satırları döndür
    RETURN QUERY
    SELECT 
        hbid AS our_hbid,
        sort_order + 1 AS new_sort_order
    FROM header_buttons
    WHERE parent_id IS NOT DISTINCT FROM p_parent_id
      AND sort_order >= v_new_sort
      AND hbid != p_hbid
    ORDER BY sort_order DESC;

    -- en son yeni satırı döndür
    RETURN QUERY
    SELECT p_hbid, v_new_sort;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_delete(
    p_sort_order INT,
    p_hbid INT,
    p_parent_id INT DEFAULT NULL
)
RETURNS TABLE(our_hbid INT, new_sort_order INT, old_parent_id INT)
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT hbid AS our_hbid, sort_order - 1 AS new_sort_order, parent_id AS old_parent_id
    FROM header_buttons
    WHERE ((parent_id = p_parent_id AND p_parent_id IS NOT NULL)
           OR (parent_id IS NULL AND p_parent_id IS NULL))
      AND sort_order > p_sort_order
    ORDER BY sort_order ASC;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_update(
    p_sort_order INT,
    p_old_sort_order INT,
    p_hbid INT,
    p_parent_id INT DEFAULT NULL
)
RETURNS TABLE(our_hbid INT, its_parent_id INT, direction TEXT, new_sort_order INT)
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_order INT;
    v_max_order INT;
    v_new_order INT;
BEGIN
    -- Eski sıralama değeri parametreden gelir
    v_old_order := p_old_sort_order;

    -- Eski değer yoksa ya da geçersizse hiçbir şey yapma
    IF v_old_order IS NULL THEN
        RETURN;
    END IF;

    SELECT COUNT(*) INTO v_max_order 
    FROM header_buttons
    WHERE parent_id IS NOT DISTINCT FROM p_parent_id;

    -- İstenen yeni değeri normalize et: minimum 1, maksimum grup içi max
    v_new_order := GREATEST(1, COALESCE(p_sort_order, 1));
    
    -- Eğer istenen pozisyon max'tan büyükse, max'a sabitle
    IF v_new_order > v_max_order THEN
        v_new_order := v_max_order;
    END IF;

    -- Değişim yoksa (aynı pozisyon), hiçbir satır döndürme
    IF v_new_order = v_old_order THEN
        RETURN;
    END IF;

    -- Yukarı taşıma: yeni pozisyon eski pozisyondan küçükse
    IF v_new_order < v_old_order THEN
        -- [v_new_order, v_old_order-1] aralığındaki satırları +1 kaydır
        RETURN QUERY
        SELECT h.hbid AS our_hbid,
               h.parent_id AS its_parent_id,
               'up' AS direction,
               v_new_order AS new_sort_order
        FROM header_buttons h
        WHERE h.parent_id IS NOT DISTINCT FROM p_parent_id
          AND h.sort_order >= v_new_order
          AND h.sort_order <= v_old_order
          AND h.hbid <> p_hbid;
    -- Aşağı taşıma: yeni pozisyon eski pozisyondan büyükse
    ELSE
        -- [v_old_order+1, v_new_order] aralığındaki satırları -1 kaydır
        RETURN QUERY
        SELECT h.hbid AS our_hbid,
               h.parent_id AS its_parent_id,
               'down' AS direction,
               v_new_order AS new_sort_order
        FROM header_buttons h
        WHERE h.parent_id IS NOT DISTINCT FROM p_parent_id
          AND h.sort_order <= v_new_order
          AND h.sort_order >= v_old_order
          AND h.hbid <> p_hbid;

        -- Taşınan satırı yeni yerine koy

    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_insert_for_custom_contents(
    p_sort_order INT,
    p_ccid INT,
    p_content_type TEXT DEFAULT NULL
)
RETURNS TABLE(our_ccid INT, new_sort_order INT)
LANGUAGE plpgsql
AS $$
DECLARE
    v_max_order INT;
    v_new_sort INT;
BEGIN
    -- mevcut max sort_order
    SELECT COUNT(*)
    INTO v_max_order
    FROM custom_contents
    WHERE content_type = p_content_type
    AND ccid != p_ccid;

    -- yeni sıra kararını ver
    IF p_sort_order > v_max_order THEN
        v_new_sort := v_max_order + 1;

        -- yeni eklenecek satırı döndür
        RETURN QUERY
        SELECT NULL::INT, v_new_sort;
        RETURN;

    ELSIF p_sort_order < 1 THEN
        v_new_sort := 1;
    ELSE
        v_new_sort := p_sort_order;
    END IF;

    -- önce kaydırılacak satırları döndür
    RETURN QUERY
    SELECT 
        ccid AS our_ccid,
        sort_order + 1 AS new_sort_order
    FROM custom_contents
    WHERE content_type = p_content_type
      AND sort_order >= v_new_sort
      AND ccid != p_ccid
    ORDER BY sort_order DESC;

    -- en son yeni satırı döndür
    RETURN QUERY
    SELECT p_ccid, v_new_sort;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_delete_for_custom_contents(
    p_sort_order INT,
    p_ccid INT,
    p_content_type TEXT DEFAULT NULL
)
RETURNS TABLE(our_ccid INT, new_sort_order INT, old_content_type VARCHAR(50))
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT ccid AS our_ccid, sort_order - 1 AS new_sort_order, content_type AS old_content_type
    FROM custom_contents
    WHERE ((content_type = p_content_type AND p_content_type IS NOT NULL)
           OR (content_type IS NULL AND p_content_type IS NULL))
      AND sort_order > p_sort_order
    ORDER BY sort_order ASC;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_update_for_custom_contents(
    p_sort_order INT,
    p_old_sort_order INT,
    p_ccid INT,
    p_content_type TEXT DEFAULT NULL
)
RETURNS TABLE(our_ccid INT, its_content_type VARCHAR(50), direction TEXT, new_sort_order INT)
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_order INT;
    v_max_order INT;
    v_new_order INT;
BEGIN
    -- Eski sıralama değeri parametreden gelir
    v_old_order := p_old_sort_order;

    -- Eski değer yoksa ya da geçersizse hiçbir şey yapma
    IF v_old_order IS NULL THEN
        RETURN;
    END IF;

    SELECT COUNT(*) INTO v_max_order 
    FROM custom_contents
    WHERE content_type = p_content_type;

    -- İstenen yeni değeri normalize et: minimum 1, maksimum grup içi max
    v_new_order := GREATEST(1, COALESCE(p_sort_order, 1));
    
    -- Eğer istenen pozisyon max'tan büyükse, max'a sabitle
    IF v_new_order > v_max_order THEN
        v_new_order := v_max_order;
    END IF;

    -- Değişim yoksa (aynı pozisyon), hiçbir satır döndürme
    IF v_new_order = v_old_order THEN
        RETURN;
    END IF;

    -- Yukarı taşıma: yeni pozisyon eski pozisyondan küçükse
    IF v_new_order < v_old_order THEN
        -- [v_new_order, v_old_order-1] aralığındaki satırları +1 kaydır
        RETURN QUERY
        SELECT cc.ccid AS our_ccid,
               cc.content_type AS its_content_type,
               'up' AS direction,
               v_new_order AS new_sort_order
        FROM custom_contents cc
        WHERE cc.content_type = p_content_type
          AND cc.sort_order >= v_new_order
          AND cc.sort_order <= v_old_order
          AND cc.ccid <> p_ccid;
    -- Aşağı taşıma: yeni pozisyon eski pozisyondan büyükse
    ELSE
        -- [v_old_order+1, v_new_order] aralığındaki satırları -1 kaydır
        RETURN QUERY
        SELECT cc.ccid AS our_ccid,
               cc.content_type AS its_content_type,
               'down' AS direction,
               v_new_order AS new_sort_order
        FROM custom_contents cc
        WHERE cc.content_type = p_content_type
          AND cc.sort_order <= v_new_order
          AND cc.sort_order >= v_old_order
          AND cc.ccid <> p_ccid;

        -- Taşınan satırı yeni yerine koy

    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_insert_for_homepage_contents(
    p_sort_order INT,
    p_hcid INT,
    p_content_type TEXT DEFAULT NULL
)
RETURNS TABLE(our_hcid INT, new_sort_order INT)
LANGUAGE plpgsql
AS $$
DECLARE
    v_max_order INT;
    v_new_sort INT;
BEGIN
    -- mevcut max sort_order
    SELECT COUNT(*)
    INTO v_max_order
    FROM homepage_contents
    WHERE content_type = p_content_type
    AND hcid != p_hcid;

    -- yeni sıra kararını ver
    IF p_sort_order > v_max_order THEN
        v_new_sort := v_max_order + 1;

        -- yeni eklenecek satırı döndür
        RETURN QUERY
        SELECT NULL::INT, v_new_sort;
        RETURN;

    ELSIF p_sort_order < 1 THEN
        v_new_sort := 1;
    ELSE
        v_new_sort := p_sort_order;
    END IF;

    -- önce kaydırılacak satırları döndür
    RETURN QUERY
    SELECT 
        hcid AS our_hcid,
        sort_order + 1 AS new_sort_order
    FROM homepage_contents
    WHERE content_type = p_content_type
      AND sort_order >= v_new_sort
      AND hcid != p_hcid
    ORDER BY sort_order DESC;

    -- en son yeni satırı döndür
    RETURN QUERY
    SELECT p_hcid, v_new_sort;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_delete_for_homepage_contents(
    p_sort_order INT,
    p_hcid INT,
    p_content_type TEXT DEFAULT NULL
)
RETURNS TABLE(our_ccid INT, new_sort_order INT, old_content_type VARCHAR(50))
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN QUERY
    SELECT hcid AS our_hcid, sort_order - 1 AS new_sort_order, content_type AS old_content_type
    FROM homepage_contents
    WHERE ((content_type = p_content_type AND p_content_type IS NOT NULL)
           OR (content_type IS NULL AND p_content_type IS NULL))
      AND sort_order > p_sort_order
    ORDER BY sort_order ASC;
END;
$$;

CREATE OR REPLACE FUNCTION get_shift_for_update_for_homepage_contents(
    p_sort_order INT,
    p_old_sort_order INT,
    p_hcid INT,
    p_content_type TEXT DEFAULT NULL
)
RETURNS TABLE(our_hcid INT, its_content_type VARCHAR(50), direction TEXT, new_sort_order INT)
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_order INT;
    v_max_order INT;
    v_new_order INT;
BEGIN
    -- Eski sıralama değeri parametreden gelir
    v_old_order := p_old_sort_order;

    -- Eski değer yoksa ya da geçersizse hiçbir şey yapma
    IF v_old_order IS NULL THEN
        RETURN;
    END IF;

    SELECT COUNT(*) INTO v_max_order 
    FROM homepage_contents
    WHERE content_type = p_content_type;

    -- İstenen yeni değeri normalize et: minimum 1, maksimum grup içi max
    v_new_order := GREATEST(1, COALESCE(p_sort_order, 1));
    
    -- Eğer istenen pozisyon max'tan büyükse, max'a sabitle
    IF v_new_order > v_max_order THEN
        v_new_order := v_max_order;
    END IF;

    -- Değişim yoksa (aynı pozisyon), hiçbir satır döndürme
    IF v_new_order = v_old_order THEN
        RETURN;
    END IF;

    -- Yukarı taşıma: yeni pozisyon eski pozisyondan küçükse
    IF v_new_order < v_old_order THEN
        -- [v_new_order, v_old_order-1] aralığındaki satırları +1 kaydır
        RETURN QUERY
        SELECT hc.hcid AS our_hcid,
               hc.content_type AS its_content_type,
               'up' AS direction,
               v_new_order AS new_sort_order
        FROM homepage_contents hc
        WHERE hc.content_type = p_content_type
          AND hc.sort_order >= v_new_order
          AND hc.sort_order <= v_old_order
          AND hc.hcid <> p_hcid;
    -- Aşağı taşıma: yeni pozisyon eski pozisyondan büyükse
    ELSE
        -- [v_old_order+1, v_new_order] aralığındaki satırları -1 kaydır
        RETURN QUERY
        SELECT hc.hcid AS our_hcid,
               hc.content_type AS its_content_type,
               'down' AS direction,
               v_new_order AS new_sort_order
        FROM homepage_contents hc
        WHERE hc.content_type = p_content_type
          AND hc.sort_order <= v_new_order
          AND hc.sort_order >= v_old_order
          AND hc.hcid <> p_hcid;

        -- Taşınan satırı yeni yerine koy

    END IF;
END;
$$;

/*CREATE OR REPLACE FUNCTION set_sube_as_main(p_sube_id INT, p_action TEXT)
RETURNS TABLE(result BOOLEAN)
LANGUAGE plpgsql
AS $$
DECLARE
    action_upper TEXT;
    first_sid INT;
BEGIN
    -- Tüm kayıtları kilitle
    PERFORM sid FROM subeler FOR UPDATE;

    action_upper := UPPER(p_action);

    -- p_sube_id NULL ise (hiç ID verilmemişse)
    IF p_sube_id IS NULL THEN
        UPDATE subeler SET is_main = FALSE WHERE is_main = TRUE;

        SELECT sid INTO first_sid
        FROM subeler
        WHERE is_active = TRUE
        ORDER BY sid ASC
        LIMIT 1
        FOR UPDATE;

        IF first_sid IS NULL THEN
            RETURN QUERY SELECT FALSE;
            RETURN;
        END IF;

        UPDATE subeler
        SET is_main = TRUE, is_active = TRUE
        WHERE sid = first_sid
        RETURNING TRUE INTO result;

        RETURN NEXT;
        RETURN;
    END IF;

    -- DELETE işlemi
    IF action_upper = 'DELETE' THEN
        UPDATE subeler SET is_main = FALSE WHERE is_main = TRUE;

        SELECT sid INTO first_sid
        FROM subeler
        WHERE is_active = TRUE AND sid <> p_sube_id
        ORDER BY sid ASC
        LIMIT 1
        FOR UPDATE;

        IF first_sid IS NULL THEN
            RETURN QUERY SELECT FALSE;
            RETURN;
        END IF;

        UPDATE subeler
        SET is_main = TRUE, is_active = TRUE
        WHERE sid = first_sid
        RETURNING TRUE INTO result;

        RETURN NEXT;
        RETURN;

    -- INSERT veya UPDATE işlemi
    ELSIF action_upper IN ('INSERT', 'UPDATE') THEN
        UPDATE subeler SET is_main = FALSE;

        UPDATE subeler
        SET is_main = TRUE, is_active = TRUE
        WHERE sid = p_sube_id
        RETURNING TRUE INTO result;

        IF NOT FOUND THEN
            RETURN QUERY SELECT FALSE;
        ELSE
            RETURN NEXT;
        END IF;

        RETURN;

    ELSE
        RETURN QUERY SELECT FALSE;
        RETURN;
    END IF;

EXCEPTION
    WHEN OTHERS THEN
        RAISE EXCEPTION 'Error in set_sube_as_main(): %', SQLERRM;
END;
$$;*/



-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_option_set_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.option_set_updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Function to generate slug from name/title
CREATE OR REPLACE FUNCTION generate_slug_from_name()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.url_name IS NULL OR NEW.url_name = '' THEN
        NEW.url_name = lower(regexp_replace(
            regexp_replace(
                regexp_replace(NEW.name, '[çÇ]', 'c', 'g'),
                '[ğĞ]', 'g', 'g'
            ),
            '[ıİ]', 'i', 'g'
        ));
        NEW.url_name = regexp_replace(
            regexp_replace(
                regexp_replace(NEW.url_name, '[öÖ]', 'o', 'g'),
                '[şŞ]', 's', 'g'
            ),
            '[üÜ]', 'u', 'g'
        );
        NEW.url_name = regexp_replace(NEW.url_name, '[^a-z0-9]+', '-', 'g');
        NEW.url_name = trim(both '-' from NEW.url_name);
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Function to generate slug from title
CREATE OR REPLACE FUNCTION generate_slug_from_title()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.url_name IS NULL OR NEW.url_name = '' THEN
        NEW.url_name = lower(regexp_replace(
            regexp_replace(
                regexp_replace(NEW.title, '[çÇ]', 'c', 'g'),
                '[ğĞ]', 'g', 'g'
            ),
            '[ıİ]', 'i', 'g'
        ));
        NEW.url_name = regexp_replace(
            regexp_replace(
                regexp_replace(NEW.url_name, '[öÖ]', 'o', 'g'),
                '[şŞ]', 's', 'g'
            ),
            '[üÜ]', 'u', 'g'
        );
        NEW.url_name = regexp_replace(NEW.url_name, '[^a-z0-9]+', '-', 'g');
        NEW.url_name = trim(both '-' from NEW.url_name);
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Function to validate appointment time slots
CREATE OR REPLACE FUNCTION validate_appointment_slot()
RETURNS TRIGGER AS $$
DECLARE
    existing_count INTEGER;
    doctor_available BOOLEAN;
BEGIN
    -- Check if doctor is available at this time
    SELECT 
        CASE 
            WHEN working_hours IS NULL THEN TRUE
            ELSE (working_hours->to_char(NEW.appointment_date, 'Day'))::text != 'null'
        END INTO doctor_available
    FROM doktorlar 
    WHERE drid = NEW.drid;
    
    IF NOT doctor_available THEN
        RAISE EXCEPTION 'Doctor is not available on this day';
    END IF;
    
    -- Check for conflicting appointments
    SELECT COUNT(*) INTO existing_count
    FROM randevular 
    WHERE drid = NEW.drid 
      AND appointment_date = NEW.appointment_date 
      AND appointment_time = NEW.appointment_time
      AND status NOT IN ('iptal', 'gelmedi')
      AND rid != COALESCE(NEW.rid, 0);
    
    IF existing_count > 0 THEN
        RAISE EXCEPTION 'This time slot is already booked';
    END IF;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Function to update news view count
CREATE OR REPLACE FUNCTION increment_news_view()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE haberler 
    SET views_count = views_count + 1 
    WHERE hid = NEW.hid;
    RETURN NEW;
END;
$$ language 'plpgsql';




CREATE OR REPLACE FUNCTION set_active_option(p_oid INT)
RETURNS TABLE(result BOOLEAN) AS $$
DECLARE
    updated_rows INT;
    target_oid INT;
BEGIN
    -- Önce tüm option'ları pasif yap
    UPDATE options
    SET option_set_is_active = FALSE;

    IF p_oid IS NULL THEN
        -- id'si en küçük olan satırın oid'ini al
        SELECT oid INTO target_oid
        FROM options
        WHERE option_set_is_active = FALSE
        ORDER BY oid ASC
        LIMIT 1;

        -- En küçük id'li satırı aktif yap ve testing_now'u false yap
        UPDATE options
        SET option_set_is_active = TRUE,
            option_set_is_testing_now = FALSE
        WHERE oid = target_oid;

        GET DIAGNOSTICS updated_rows = ROW_COUNT;
    ELSE
        UPDATE options
        SET option_set_is_active = FALSE
        WHERE option_set_is_active = TRUE AND oid <> p_oid;

        -- Belirtilen option'ı aktif yap
        UPDATE options
        SET option_set_is_active = TRUE
        WHERE oid = p_oid;

        GET DIAGNOSTICS updated_rows = ROW_COUNT;
    END IF;

    -- Sonuç döndür
    RETURN QUERY SELECT (updated_rows > 0) AS result;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION set_testing_option(p_oid INT)
RETURNS TABLE(result BOOLEAN) AS $$
DECLARE
    updated_rows INT;
BEGIN
    IF p_oid IS NULL THEN
        UPDATE options
        SET option_set_is_testing_now = FALSE
        WHERE option_set_is_testing_now = TRUE;

        GET DIAGNOSTICS updated_rows = ROW_COUNT;

        -- Eğer 1 satır etkilendiyse başarılıdır
        RETURN QUERY SELECT TRUE AS result;
    ELSE
        -- Önce tüm option'ları pasif yap
        UPDATE options
        SET option_set_is_testing_now = FALSE
        WHERE option_set_is_testing_now = TRUE AND oid <> p_oid;

        -- Sonra belirtilen option'ı aktif yap
        UPDATE options
        SET option_set_is_testing_now = TRUE
        WHERE oid = p_oid;

        GET DIAGNOSTICS updated_rows = ROW_COUNT;

        -- Eğer 1 satır etkilendiyse başarılıdır
        RETURN QUERY SELECT (updated_rows > 0) AS result;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_head_drid(p_brid INT, p_head_drid INT)
RETURNS TABLE(result BOOLEAN)
LANGUAGE plpgsql
AS $$
DECLARE
    rows_updated INT;
BEGIN
    -- 1. Eski head_drid'i NULL yap
    UPDATE branslar
    SET head_drid = NULL
    WHERE head_drid = p_head_drid;

    -- 2. doktorlar tablosunu güncelle
    UPDATE doktorlar
    SET brid = p_brid
    WHERE drid = p_head_drid;

    GET DIAGNOSTICS rows_updated = ROW_COUNT;
    IF rows_updated = 0 THEN
        RETURN QUERY SELECT FALSE;
        RETURN;
    END IF;

    -- 3. branslar tablosunu güncelle
    UPDATE branslar
    SET head_drid = p_head_drid
    WHERE brid = p_brid;

    GET DIAGNOSTICS rows_updated = ROW_COUNT;
    IF rows_updated = 0 THEN
        RETURN QUERY SELECT FALSE;
        RETURN;
    END IF;

    -- Başarılı
    RETURN QUERY SELECT TRUE;

EXCEPTION
    WHEN OTHERS THEN
        RAISE EXCEPTION 'Error in update_head_drid(): %', SQLERRM;
        RETURN QUERY SELECT FALSE;
END;
$$;

CREATE OR REPLACE FUNCTION change_doctor_branch(p_brid INT, p_drid INT)
RETURNS TABLE(result BOOLEAN)
LANGUAGE plpgsql
AS $$
DECLARE
    rows_updated INT;
    sid_val INT;
BEGIN
    -- 1. doktorlar tablosunu güncelle
    UPDATE doktorlar
    SET brid = p_brid
    WHERE drid = p_drid;

    GET DIAGNOSTICS rows_updated = ROW_COUNT;
    IF rows_updated = 0 THEN
        RETURN QUERY SELECT FALSE;
        RETURN;
    END IF;

    -- 2. branşın sid değerini al
    SELECT sid INTO sid_val
    FROM branslar
    WHERE brid = p_brid
    LIMIT 1;

    IF sid_val IS NULL THEN
        RETURN QUERY SELECT TRUE;
        RETURN;
    END IF;

    -- 3. doktorlar tablosuna sid değerini ata
    UPDATE doktorlar
    SET sid = sid_val
    WHERE drid = p_drid;

    GET DIAGNOSTICS rows_updated = ROW_COUNT;
    IF rows_updated = 0 THEN
        RETURN QUERY SELECT FALSE;
        RETURN;
    END IF;

    -- Başarılı
    RETURN QUERY SELECT TRUE;

EXCEPTION
    WHEN OTHERS THEN
        RAISE EXCEPTION 'Error in change_doctor_branch(): %', SQLERRM;
        RETURN QUERY SELECT FALSE;
END;
$$;


CREATE OR REPLACE FUNCTION handle_primary_expertise(
    p_action text,
    p_duid int,
    p_drid int
)
RETURNS TABLE(result boolean)
LANGUAGE plpgsql
AS $$
DECLARE
    cnt INT;
    first_duid INT;
BEGIN
    IF p_action IN ('INSERT', 'UPDATE') THEN
        -- 1. Seçilen duid'i primary yap
        UPDATE doctor_expertises
        SET is_primary = TRUE
        WHERE duid = p_duid;

        -- 2. Aynı doktor için diğerlerini false yap
        UPDATE doctor_expertises
        SET is_primary = FALSE
        WHERE drid = p_drid AND duid <> p_duid;

        RETURN QUERY SELECT TRUE;

    ELSIF p_action = 'DELETE' THEN
        -- Doktorun hali hazırda primary kaydı var mı kontrol et
        SELECT COUNT(*) INTO cnt
        FROM doctor_expertises
        WHERE drid = p_drid AND is_primary = TRUE;

        IF cnt > 0 THEN
            RETURN QUERY SELECT TRUE;
        ELSE
            -- En küçük duid'yi al ve primary yap
            SELECT duid INTO first_duid
            FROM doctor_expertises
            WHERE drid = p_drid
            ORDER BY duid ASC
            LIMIT 1;

            IF first_duid IS NOT NULL THEN
                UPDATE doctor_expertises
                SET is_primary = TRUE
                WHERE duid = first_duid;
            END IF;

            RETURN QUERY SELECT TRUE;
        END IF;

    ELSE
        RAISE WARNING 'Geçersiz p_action: %', p_action;
        RETURN QUERY SELECT FALSE;
    END IF;

EXCEPTION
    WHEN OTHERS THEN
        RAISE WARNING 'Hata: %', SQLERRM;
        RETURN QUERY SELECT FALSE;
END;
$$;

-- Apply updated_at triggers to all tables
CREATE TRIGGER update_options_updated_at BEFORE UPDATE ON options FOR EACH ROW EXECUTE FUNCTION update_option_set_updated_at_column();
CREATE TRIGGER update_medias_updated_at BEFORE UPDATE ON medias FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_header_buttons_updated_at BEFORE UPDATE ON header_buttons FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_subeler_updated_at BEFORE UPDATE ON subeler FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_anlasmali_kurumlar_updated_at BEFORE UPDATE ON anlasmali_kurumlar FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_uzmanliklar_updated_at BEFORE UPDATE ON uzmanliklar FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_branslar_updated_at BEFORE UPDATE ON branslar FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_doktorlar_updated_at BEFORE UPDATE ON doktorlar FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_haberler_updated_at BEFORE UPDATE ON haberler FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_randevular_updated_at BEFORE UPDATE ON randevular FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_contact_requests_updated_at BEFORE UPDATE ON contact_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_job_applications_updated_at BEFORE UPDATE ON job_applications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_randevu_talepleri_updated_at BEFORE UPDATE ON randevu_talepleri FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE ON notifications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply slug generation triggers
CREATE TRIGGER generate_subeler_url_name BEFORE INSERT OR UPDATE ON subeler FOR EACH ROW EXECUTE FUNCTION generate_slug_from_name();
CREATE TRIGGER generate_anlasmali_kurumlar_url_name BEFORE INSERT OR UPDATE ON anlasmali_kurumlar FOR EACH ROW EXECUTE FUNCTION generate_slug_from_name();
CREATE TRIGGER generate_uzmanliklar_url_name BEFORE INSERT OR UPDATE ON uzmanliklar FOR EACH ROW EXECUTE FUNCTION generate_slug_from_name();
CREATE TRIGGER generate_branslar_url_name BEFORE INSERT OR UPDATE ON branslar FOR EACH ROW EXECUTE FUNCTION generate_slug_from_name();
CREATE TRIGGER generate_doktorlar_url_name BEFORE INSERT OR UPDATE ON doktorlar FOR EACH ROW EXECUTE FUNCTION generate_slug_from_name();
CREATE TRIGGER generate_haberler_url_name BEFORE INSERT OR UPDATE ON haberler FOR EACH ROW EXECUTE FUNCTION generate_slug_from_title();

-- Apply validation triggers
CREATE TRIGGER validate_appointment_slot_trigger BEFORE INSERT OR UPDATE ON randevular FOR EACH ROW EXECUTE FUNCTION validate_appointment_slot();

-- This completes the comprehensive hospital CMS database schema
