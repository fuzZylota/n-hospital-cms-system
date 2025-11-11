package models

import (
	"time"

	wsb "github.com/Necoo33/fiber-ws-broadcaster"
	orm "github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/contrib/websocket"
)

// type that represents a websocket connection
type WebsocketConnection struct {
	Uid        string          `form:"uid" json:"uid"`
	InsertForm string          `form:"insert_form" json:"insert_form"`
	Connection *websocket.Conn `form:"connection" json:"connection"`
	// esas websocket rabıtası
}

// type that represents individual messages to sent websocket
type WebsocketMessage struct {
	Uid string `form:"uid" json:"uid"`
	//InsertForm  string `form:"insert_form" json:"insert_form"`
	Message     string `form:"message" json:"message"`
	RequestLink string `form:"request_link" json:"request_link"`
}

// type that represents all the changeable app states
type AppState struct {
	Broadcaster        *wsb.Broadcaster
	Connections        []WebsocketConnection
	ActiveOptions      Options
	TestingOptions     Options
	Medias             []Medias
	HeaderButtons      []HeaderButton
	SubelerLinks       []SubeLink
	TibbiBirimlerLinks []TibbiBirimLink
	TedkiklerLinks     []TedkikLink
	NewsLinks          []NewsLink
	Notifications      []Notification
}

// type that represents all the unchanging utilites.
type Utilities struct {
	Orm *orm.Neorm
	// Limiter
}

// type that represents the necessary information for getting the frontend options
type FrontendOptions struct {
	Database *orm.Neorm
	User     AuthenticatedUser
	States   *AppState
}

// Options Related Structs

type SubeLink struct {
	Sid          string
	Name         string
	UrlName      string
	DocumentMids []string
	Documents    []Medias
	//DoktorLinks []DoktorLink
}

type TibbiBirimLink struct {
	Tbid    string
	Name    string
	UrlName string
}

type TedkikLink struct {
	Tid     string
	Name    string
	UrlName string
}

type NewsLink struct {
	Hid          string
	Title        string
	UrlName      string
	Author       string
	CoverMid     int64
	CoverPath    string
	CoverAltText string
	CoverTitle   string
}

type DoktorForHomePage struct {
	Drid         string
	Sid          string
	SubeUrlName  string
	Brid         string
	BransName    string
	BransUrlName string
	PhotoMid     int64
	PhotoPath    string
	PhotoAltText string
	PhotoTitle   string
	Title        string
	FirstName    string
	LastName     string
	UrlName      string
	FacebookUrl  string
	XUrl         string
	InstagramUrl string
	LinkedinUrl  string
	PersonalUrl  string
}

type SubeForFrontendPages struct {
	Sid                string `form:"sid" json:"sid"`
	Name               string `form:"name" json:"name"`
	UrlName            string `form:"url_name" json:"url_name"`
	City               string `form:"city" json:"city"`
	District           string `form:"district" json:"district"`
	GoogleMapIframe    string `form:"google_map_iframe" json:"google_map_iframe"`
	TransportationInfo string `form:"transportation_info" json:"transportation_info"`
	Phone              string `form:"phone" json:"phone"`
	Email              string `form:"email" json:"email"`
	IsMain             bool   `form:"is_main" json:"is_main"`
	MediaPath          string `form:"media_path" json:"media_path"`
	MediaAltText       string `form:"media_alt_text" json:"media_alt_text"`
	MediaTitle         string `form:"media_title" json:"media_title"`
}

type HaberlerForHaberlerPage struct {
	Hid          string
	Title        string
	PublishDate  time.Time
	UrlName      string
	Author       string
	CoverMid     int64
	CoverPath    string
	CoverAltText string
	CoverTitle   string
}

type HaberlerPageCategoryNameAndCounts struct {
	Name  string
	Count int64
}

type BranchCount struct {
	Count      int64
	BranchName string
}

type DoctorCount struct {
	Count      int64
	DoctorName string
}

type Options struct {
	Oid                           string    `form:"oid" json:"oid"`
	OptionSetName                 string    `form:"option_set_name" json:"option_set_name"`
	OptionSetDescription          string    `form:"option_set_description" json:"option_set_description"`
	OptionSetIsActive             bool      `form:"option_set_is_active" json:"option_set_is_active"`
	OptionSetIsTestingNow         bool      `form:"option_set_is_testing_now" json:"option_set_is_testing_now"`
	OptionSetCreatedAt            time.Time `form:"option_set_created_at" json:"option_set_created_at"`
	OptionSetUpdatedAt            time.Time `form:"option_set_updated_at" json:"option_set_updated_at"`
	SiteName                      string    `form:"site_name" json:"site_name"`
	SiteDescription               string    `form:"site_description" json:"site_description"`
	SiteLogoMid                   int64     `form:"site_logo_mid" json:"site_logo_mid"`
	SiteLightLogoMid              int64     `form:"site_light_logo_mid" json:"site_light_logo_mid"`
	FaviconMid                    int64     `form:"favicon_mid" json:"favicon_mid"`
	DefaultPageMid                int64     `form:"default_page_mid" json:"default_page_mid"`
	MaintenanceMode               bool      `form:"maintenance_mode" json:"maintenance_mode"`
	Preloader                     string    `form:"preloader" json:"preloader"`
	SMTPHost                      string    `form:"smtp_host" json:"smtp_host"`
	SMTPPort                      int64     `form:"smtp_port" json:"smtp_port"`
	SMTPUsername                  string    `form:"smtp_username" json:"smtp_username"`
	SMTPPassword                  string    `form:"smtp_password" json:"smtp_password"`
	SMTPEncryption                string    `form:"smtp_encryption" json:"smtp_encryption"`
	FacebookUrl                   string    `form:"facebook_url" json:"facebook_url"`
	TwitterUrl                    string    `form:"twitter_url" json:"twitter_url"`
	InstagramUrl                  string    `form:"instagram_url" json:"instagram_url"`
	LinkedinUrl                   string    `form:"linkedin_url" json:"linkedin_url"`
	ContactEmail                  string    `form:"contact_email" json:"contact_email"`
	ContactPhone                  string    `form:"contact_phone" json:"contact_phone"`
	MainPageMetaTitle             string    `form:"main_page_meta_title" json:"main_page_meta_title"`
	MainPageMetaDescription       string    `form:"main_page_meta_description" json:"main_page_meta_description"`
	GoogleAnalytics               string    `form:"google_analytics" json:"google_analytics"`
	PrimaryColor                  string    `form:"primary_color" json:"primary_color"`
	SecondaryColor                string    `form:"secondary_color" json:"secondary_color"`
	AccentColor                   string    `form:"accent_color" json:"accent_color"`
	BackgroundColor               string    `form:"background_color" json:"background_color"`
	FontColor                     string    `form:"font_color" json:"font_color"`
	FontFamily                    string    `form:"font_family" json:"font_family"`
	RequireStrongPassword         bool      `form:"require_strong_password" json:"require_strong_password"`
	ItemsPerPage                  int64     `form:"items_per_page" json:"items_per_page"`
	ShowDoctorsOnSameCity         bool      `form:"show_doctors_on_same_city" json:"show_doctors_on_same_city"`
	ShowDoctorsOnSameCountry      bool      `form:"show_doctors_on_same_country" json:"show_doctors_on_same_country"`
	AutoRemovePartnersWhenExpired bool      `form:"auto_remove_partners_when_expired" json:"auto_remove_partners_when_expired"`
	EnableTestimonials            bool      `form:"enable_testimonials" json:"enable_testimonials"`
	EnableOurHistory              bool      `form:"enable_our_history" json:"enable_our_history"`
	MaximumSublinksOnAMenuItem    int64     `form:"maximum_sublinks_on_a_menu_item" json:"maximum_sublinks_on_a_menu_item"`
	MaxUploadSize                 int64     `form:"max_upload_size" json:"max_upload_size"`
	ShowDoctorSocialMedia         bool      `form:"show_doctor_social_media" json:"show_doctor_social_media"`
	ShowDoctorAppointmentFee      bool      `form:"show_doctor_appointment_fee" json:"show_doctor_appointment_fee"`
	ShowAnlasmaliKurumPictures    bool      `form:"show_anlasmali_kurum_pictures" json:"show_anlasmali_kurum_pictures"`
	Timezone                      string    `form:"timezone" json:"timezone"`
	Language                      string    `form:"language" json:"language"`
	IsActive                      bool      `form:"is_active" json:"is_active"`
	SiteLogoPath                  string    `form:"site_logo_path" json:"site_logo_path"`
	SiteLogoAltText               string    `form:"site_logo_alt_text" json:"site_logo_alt_text"`
	SiteLogoTitle                 string    `form:"site_logo_title" json:"site_logo_title"`
	SiteLightLogoPath             string    `form:"site_light_logo_path" json:"site_light_logo_path"`
	SiteLightLogoAltText          string    `form:"site_light_logo_alt_text" json:"site_light_logo_alt_text"`
	SiteLightLogoTitle            string    `form:"site_light_logo_title" json:"site_light_logo_title"`
	SiteFaviconPath               string    `form:"site_favicon_path" json:"site_favicon_path"`
	DefaultPageMediaPath          string    `form:"default_page_media_path" json:"default_page_media_path"`
	DefaultPageMediaAltText       string    `form:"default_page_alt_text" json:"default_page_alt_text"`
	DefaultPageMediaTitle         string    `form:"default_page_title" json:"default_page_title"`
	RecaptchaSiteKey              string    `form:"recaptcha_site_key" json:"recaptcha_site_key"`
	RecaptchaSecretKey            string    `form:"recaptcha_secret_key" json:"recaptcha_secret_key"`
}

type OptionsEdit struct {
	Oid                              string    `form:"oid" json:"oid"`
	OptionSetName                    string    `form:"option_set_name" json:"option_set_name"`
	OldOptionSetName                 string    `form:"old_option_set_name" json:"old_option_set_name"`
	OptionSetDescription             string    `form:"option_set_description" json:"option_set_description"`
	OldOptionSetDescription          string    `form:"old_option_set_description" json:"old_option_set_description"`
	OptionSetIsActive                bool      `form:"option_set_is_active" json:"option_set_is_active"`
	OldOptionSetIsActive             bool      `form:"old_option_set_is_active" json:"old_option_set_is_active"`
	OptionSetIsTestingNow            bool      `form:"option_set_is_testing_now" json:"option_set_is_testing_now"`
	OldOptionSetIsTestingNow         bool      `form:"old_option_set_is_testing_now" json:"old_option_set_is_testing_now"`
	OptionSetCreatedAt               time.Time `form:"option_set_created_at" json:"option_set_created_at"`
	OldOptionSetCreatedAt            time.Time `form:"old_option_set_created_at" json:"old_option_set_created_at"`
	OptionSetUpdatedAt               time.Time `form:"option_set_updated_at" json:"option_set_updated_at"`
	OldOptionSetUpdatedAt            time.Time `form:"old_option_set_updated_at" json:"old_option_set_updated_at"`
	SiteName                         string    `form:"site_name" json:"site_name"`
	OldSiteName                      string    `form:"old_site_name" json:"old_site_name"`
	SiteDescription                  string    `form:"site_description" json:"site_description"`
	OldSiteDescription               string    `form:"old_site_description" json:"old_site_description"`
	SiteLogoMid                      int64     `form:"site_logo_mid" json:"site_logo_mid"`
	OldSiteLogoMid                   int64     `form:"old_site_logo_mid" json:"old_site_logo_mid"`
	SiteLightLogoMid                 int64     `form:"site_light_logo_mid" json:"site_light_logo_mid"`
	OldSiteLightLogoMid              int64     `form:"old_site_light_logo_mid" json:"old_site_light_logo_mid"`
	FaviconMid                       int64     `form:"favicon_mid" json:"favicon_mid"`
	OldFaviconMid                    int64     `form:"old_favicon_mid" json:"old_favicon_mid"`
	DefaultPageMid                   int64     `form:"default_page_mid" json:"default_page_mid"`
	OldDefaultPageMid                int64     `form:"old_default_page_mid" json:"old_default_page_mid"`
	MaintenanceMode                  bool      `form:"maintenance_mode" json:"maintenance_mode"`
	OldMaintenanceMode               bool      `form:"old_maintenance_mode" json:"old_maintenance_mode"`
	Preloader                        string    `form:"preloader" json:"preloader"`
	OldPreloader                     string    `form:"old_preloader" json:"old_preloader"`
	SMTPHost                         string    `form:"smtp_host" json:"smtp_host"`
	OldSMTPHost                      string    `form:"old_smtp_host" json:"old_smtp_host"`
	SMTPPort                         int64     `form:"smtp_port" json:"smtp_port"`
	OldSMTPPort                      int64     `form:"old_smtp_port" json:"old_smtp_port"`
	SMTPUsername                     string    `form:"smtp_username" json:"smtp_username"`
	OldSMTPUsername                  string    `form:"old_smtp_username" json:"old_smtp_username"`
	SMTPPassword                     string    `form:"smtp_password" json:"smtp_password"`
	OldSMTPPassword                  string    `form:"old_smtp_password" json:"old_smtp_password"`
	SMTPEncryption                   string    `form:"smtp_encryption" json:"smtp_encryption"`
	OldSMTPEncryption                string    `form:"old_smtp_encryption" json:"old_smtp_encryption"`
	FacebookUrl                      string    `form:"facebook_url" json:"facebook_url"`
	OldFacebookUrl                   string    `form:"old_facebook_url" json:"old_facebook_url"`
	TwitterUrl                       string    `form:"twitter_url" json:"twitter_url"`
	OldTwitterUrl                    string    `form:"old_twitter_url" json:"old_twitter_url"`
	InstagramUrl                     string    `form:"instagram_url" json:"instagram_url"`
	OldInstagramUrl                  string    `form:"old_instagram_url" json:"old_instagram_url"`
	LinkedinUrl                      string    `form:"linkedin_url" json:"linkedin_url"`
	OldLinkedinUrl                   string    `form:"old_linkedin_url" json:"old_linkedin_url"`
	ContactEmail                     string    `form:"contact_email" json:"contact_email"`
	OldContactEmail                  string    `form:"old_contact_email" json:"old_contact_email"`
	ContactPhone                     string    `form:"contact_phone" json:"contact_phone"`
	OldContactPhone                  string    `form:"old_contact_phone" json:"old_contact_phone"`
	MainPageMetaTitle                string    `form:"main_page_meta_title" json:"main_page_meta_title"`
	OldMainPageMetaTitle             string    `form:"old_main_page_meta_title" json:"old_main_page_meta_title"`
	MainPageMetaDescription          string    `form:"main_page_meta_description" json:"main_page_meta_description"`
	OldMainPageMetaDescription       string    `form:"old_main_page_meta_description" json:"old_main_page_meta_description"`
	GoogleAnalytics                  string    `form:"google_analytics" json:"google_analytics"`
	OldGoogleAnalytics               string    `form:"old_google_analytics" json:"old_google_analytics"`
	PrimaryColor                     string    `form:"primary_color" json:"primary_color"`
	OldPrimaryColor                  string    `form:"old_primary_color" json:"old_primary_color"`
	SecondaryColor                   string    `form:"secondary_color" json:"secondary_color"`
	OldSecondaryColor                string    `form:"old_secondary_color" json:"old_secondary_color"`
	AccentColor                      string    `form:"accent_color" json:"accent_color"`
	OldAccentColor                   string    `form:"old_accent_color" json:"old_accent_color"`
	BackgroundColor                  string    `form:"background_color" json:"background_color"`
	OldBackgroundColor               string    `form:"old_background_color" json:"old_background_color"`
	FontColor                        string    `form:"font_color" json:"font_color"`
	OldFontColor                     string    `form:"old_font_color" json:"old_font_color"`
	FontFamily                       string    `form:"font_family" json:"font_family"`
	OldFontFamily                    string    `form:"old_font_family" json:"old_font_family"`
	RequireStrongPassword            bool      `form:"require_strong_password" json:"require_strong_password"`
	OldRequireStrongPassword         bool      `form:"old_require_strong_password" json:"old_require_strong_password"`
	ItemsPerPage                     int64     `form:"items_per_page" json:"items_per_page"`
	OldItemsPerPage                  int64     `form:"old_items_per_page" json:"old_items_per_page"`
	ShowDoctorsOnSameCity            bool      `form:"show_doctors_on_same_city" json:"show_doctors_on_same_city"`
	OldShowDoctorsOnSameCity         bool      `form:"old_show_doctors_on_same_city" json:"old_show_doctors_on_same_city"`
	ShowDoctorsOnSameCountry         bool      `form:"show_doctors_on_same_country" json:"show_doctors_on_same_country"`
	OldShowDoctorsOnSameCountry      bool      `form:"old_show_doctors_on_same_country" json:"old_show_doctors_on_same_country"`
	AutoRemovePartnersWhenExpired    bool      `form:"auto_remove_partners_when_expired" json:"auto_remove_partners_when_expired"`
	OldAutoRemovePartnersWhenExpired bool      `form:"old_auto_remove_partners_when_expired" json:"old_auto_remove_partners_when_expired"`
	EnableTestimonials               bool      `form:"enable_testimonials" json:"enable_testimonials"`
	OldEnableTestimonials            bool      `form:"old_enable_testimonials" json:"old_enable_testimonials"`
	EnableOurHistory                 bool      `form:"enable_our_history" json:"enable_our_history"`
	OldEnableOurHistory              bool      `form:"old_enable_our_history" json:"old_enable_our_history"`
	MaximumSublinksOnAMenuItem       int64     `form:"maximum_sublinks_on_a_menu_item" json:"maximum_sublinks_on_a_menu_item"`
	OldMaximumSublinksOnAMenuItem    int64     `form:"old_maximum_sublinks_on_a_menu_item" json:"old_maximum_sublinks_on_a_menu_item"`
	MaxUploadSize                    int64     `form:"max_upload_size" json:"max_upload_size"`
	OldMaxUploadSize                 int64     `form:"old_max_upload_size" json:"old_max_upload_size"`
	ShowDoctorSocialMedia            bool      `form:"show_doctor_social_media" json:"show_doctor_social_media"`
	OldShowDoctorSocialMedia         bool      `form:"old_show_doctor_social_media" json:"old_show_doctor_social_media"`
	ShowDoctorAppointmentFee         bool      `form:"show_doctor_appointment_fee" json:"show_doctor_appointment_fee"`
	OldShowDoctorAppointmentFee      bool      `form:"old_show_doctor_appointment_fee" json:"old_show_doctor_appointment_fee"`
	ShowAnlasmaliKurumPictures       bool      `form:"show_anlasmali_kurum_pictures" json:"show_anlasmali_kurum_pictures"`
	OldShowAnlasmaliKurumPictures    bool      `form:"old_show_anlasmali_kurum_pictures" json:"old_show_anlasmali_kurum_pictures"`
	Timezone                         string    `form:"timezone" json:"timezone"`
	OldTimezone                      string    `form:"old_timezone" json:"old_timezone"`
	Language                         string    `form:"language" json:"language"`
	OldLanguage                      string    `form:"old_language" json:"old_language"`
	IsActive                         bool      `form:"is_active" json:"is_active"`
	SiteLogoPath                     string    `form:"site_logo_path" json:"site_logo_path"`
	OldSiteLogoPath                  string    `form:"old_site_logo_path" json:"old_site_logo_path"`
	SiteLogoAltText                  string    `form:"site_logo_alt_text" json:"site_logo_alt_text"`
	OldSiteLogoAltText               string    `form:"old_site_logo_alt_text" json:"old_site_logo_alt_text"`
	SiteLogoTitle                    string    `form:"site_logo_title" json:"site_logo_title"`
	OldSiteLogoTitle                 string    `form:"old_site_logo_title" json:"old_site_logo_title"`
	SiteFaviconPath                  string    `form:"site_favicon_path" json:"site_favicon_path"`
	OldSiteFaviconPath               string    `form:"old_site_favicon_path" json:"old_site_favicon_path"`
	DefaultPageMediaPath             string    `form:"default_page_media_path" json:"default_page_media_path"`
	OldDefaultPageMediaPath          string    `form:"old_default_page_media_path" json:"old_default_page_media_path"`
	DefaultPageMediaAltText          string    `form:"default_page_alt_text" json:"default_page_alt_text"`
	OldDefaultPageMediaAltText       string    `form:"old_default_page_alt_text" json:"old_default_page_alt_text"`
	DefaultPageMediaTitle            string    `form:"default_page_title" json:"default_page_title"`
	OldDefaultPageMediaTitle         string    `form:"old_default_page_title" json:"old_default_page_title"`
	RecaptchaSiteKey                 string    `form:"recaptcha_site_key" json:"recaptcha_site_key"`
	OldRecaptchaSiteKey              string    `form:"old_recaptcha_site_key" json:"old_recaptcha_site_key"`
	RecaptchaSecretKey               string    `form:"recaptcha_secret_key" json:"recaptcha_secret_key"`
	OldRecaptchaSecretKey            string    `form:"old_recaptcha_secret_key" json:"old_recaptcha_secret_key"`
}

type Medias struct {
	Mid       int64
	FileName  string
	Data      string
	FilePath  string
	FileSize  int64
	MimeType  string
	FileType  string
	AltText   string
	Title     string
	Width     int64
	Height    int64
	Uid       string
	TargetId  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MediaOptionals struct {
	AltText string
	Title   string
	Width   int64
	Height  int64
	Data    string
	OldData string
}

type AuthInputs struct {
	EmailOrPhone string `form:"email_or_phone" json:"email_or_phone"`
	Password     string `form:"password" json:"password"`
	Remember     bool   `form:"remember" json:"remember"`
}

type AuthenticatedUser struct {
	Uid       string
	Email     string
	Phone     string
	Name      string
	Surname   string
	Role      string
	Timezone  string
	IsActive  bool
	LastLogin time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	Remember  bool
}

type Users struct {
	Uid             string    `form:"uid" json:"uid"`
	Email           string    `form:"email" json:"email"`
	Password        string    `form:"password" json:"password"`
	PasswordConfirm string    `form:"password_confirm" json:"password_confirm"`
	Phone           string    `form:"phone" json:"phone"`
	Name            string    `form:"name" json:"name"`
	Surname         string    `form:"surname" json:"surname"`
	Role            string    `form:"role" json:"role"`
	Timezone        string    `form:"timezone" json:"timezone"`
	IsActive        bool      `form:"is_active" json:"is_active"`
	LastLogin       time.Time `form:"last_login" json:"last_login"`
	CreatedAt       time.Time `form:"created_at" json:"created_at"`
	UpdatedAt       time.Time `form:"updated_at" json:"updated_at"`
}

type UsersEdit struct {
	Uid             string    `form:"uid" json:"uid"`
	Email           string    `form:"email" json:"email"`
	OldEmail        string    `form:"old_email" json:"old_email"`
	Password        string    `form:"password" json:"password"`
	PasswordConfirm string    `form:"password_confirm" json:"password_confirm"`
	OldPassword     string    `form:"old_password" json:"old_password"`
	Phone           string    `form:"phone" json:"phone"`
	OldPhone        string    `form:"old_phone" json:"old_phone"`
	Name            string    `form:"name" json:"name"`
	OldName         string    `form:"old_name" json:"old_name"`
	Surname         string    `form:"surname" json:"surname"`
	OldSurname      string    `form:"old_surname" json:"old_surname"`
	Role            string    `form:"role" json:"role"`
	OldRole         string    `form:"old_role" json:"old_role"`
	IsActive        bool      `form:"is_active" json:"is_active"`
	OldIsActive     bool      `form:"old_is_active" json:"old_is_active"`
	Timezone        string    `form:"timezone" json:"timezone"`
	OldTimezone     string    `form:"old_timezone" json:"old_timezone"`
	LastLogin       time.Time `form:"last_login" json:"last_login"`
	OldLastLogin    time.Time `form:"old_last_login" json:"old_last_login"`
	CreatedAt       time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt    time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt       time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt    time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type HeaderButton struct {
	Hbid       string    `form:"hbid" json:"hbid"`
	Title      string    `form:"title" json:"title"`
	Url        string    `form:"url" json:"url"`
	Target     string    `form:"target" json:"target"`
	Icon       string    `form:"icon" json:"icon"`
	SortOrder  int64     `form:"sort_order" json:"sort_order"`
	IsActive   bool      `form:"is_active" json:"is_active"`
	ButtonType string    `form:"button_type" json:"button_type"`
	ParentId   string    `form:"parent_id" json:"parent_id"`
	CreatedAt  time.Time `form:"created_at" json:"created_at"`
	UpdatedAt  time.Time `form:"updated_at" json:"updated_at"`
}

type HeaderButtonEdit struct {
	Hbid          string    `form:"hbid" json:"hbid"`
	Title         string    `form:"title" json:"title"`
	OldTitle      string    `form:"old_title" json:"old_title"`
	Url           string    `form:"url" json:"url"`
	OldUrl        string    `form:"old_url" json:"old_url"`
	Target        string    `form:"target" json:"target"`
	OldTarget     string    `form:"old_target" json:"old_target"`
	Icon          string    `form:"icon" json:"icon"`
	OldIcon       string    `form:"old_icon" json:"old_icon"`
	SortOrder     int64     `form:"sort_order" json:"sort_order"`
	OldSortOrder  int64     `form:"old_sort_order" json:"old_sort_order"`
	IsActive      bool      `form:"is_active" json:"is_active"`
	OldIsActive   bool      `form:"old_is_active" json:"old_is_active"`
	ButtonType    string    `form:"button_type" json:"button_type"`
	OldButtonType string    `form:"old_button_type" json:"old_button_type"`
	ParentId      string    `form:"parent_id" json:"parent_id"`
	OldParentId   string    `form:"old_parent_id" json:"old_parent_id"`
	CreatedAt     time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt  time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt     time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt  time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Testimonials struct {
	Tid                    string    `form:"tid" json:"tid"`
	FirstName              string    `form:"first_name" json:"first_name"`
	LastName               string    `form:"last_name" json:"last_name"`
	Occupation             string    `form:"occupation" json:"occupation"`
	Content                string    `form:"content" json:"content"`
	Rating                 float64   `form:"rating" json:"rating"`
	IsActive               bool      `form:"is_active" json:"is_active"`
	CustomerPictureMid     int64     `form:"customer_picture_mid" json:"customer_picture_mid"`
	CreatedAt              time.Time `form:"created_at" json:"created_at"`
	UpdatedAt              time.Time `form:"updated_at" json:"updated_at"`
	CustomerPicturePath    string    `form:"customer_picture_path" json:"customer_picture_path"`
	CustomerPictureAltText string    `form:"customer_picture_alt_text" json:"customer_picture_alt_text"`
	CustomerPictureTitle   string    `form:"customer_picture_title" json:"customer_picture_title"`
}

type TestimonialsEdit struct {
	Tid                   string    `form:"tid" json:"tid"`
	FirstName             string    `form:"first_name" json:"first_name"`
	OldFirstName          string    `form:"old_first_name" json:"old_first_name"`
	LastName              string    `form:"last_name" json:"last_name"`
	OldLastName           string    `form:"old_last_name" json:"old_last_name"`
	Occupation            string    `form:"occupation" json:"occupation"`
	OldOccupation         string    `form:"old_occupation" json:"old_occupation"`
	Content               string    `form:"content" json:"content"`
	OldContent            string    `form:"old_content" json:"old_content"`
	Rating                float64   `form:"rating" json:"rating"`
	OldRating             float64   `form:"old_rating" json:"old_rating"`
	IsActive              bool      `form:"is_active" json:"is_active"`
	OldIsActive           bool      `form:"old_is_active" json:"old_is_active"`
	CustomerPictureMid    int64     `form:"customer_picture_mid" json:"customer_picture_mid"`
	OldCustomerPictureMid int64     `form:"old_customer_picture_mid" json:"old_customer_picture_mid"`
	CreatedAt             time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt          time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt             time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt          time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Subeler struct {
	Sid                string    `form:"sid" json:"sid"`
	Name               string    `form:"name" json:"name"`
	UrlName            string    `form:"url_name" json:"url_name"`
	Description        string    `form:"description" json:"description"`
	Address            string    `form:"address" json:"address"`
	GoogleMapIframe    string    `form:"google_map_iframe" json:"google_map_iframe"`
	TransportationInfo string    `form:"transportation_info" json:"transportation_info"`
	City               string    `form:"city" json:"city"`
	District           string    `form:"district" json:"district"`
	PostalCode         string    `form:"postal_code" json:"postal_code"`
	Phone              string    `form:"phone" json:"phone"`
	Fax                string    `form:"fax" json:"fax"`
	Email              string    `form:"email" json:"email"`
	Website            string    `form:"website" json:"website"`
	Latitude           float64   `form:"latitude" json:"latitude"`
	Longitude          float64   `form:"longitude" json:"longitude"`
	WorkingHours       string    `form:"working_hours" json:"working_hours"`
	Mid                int64     `form:"mid" json:"mid"`
	DocumentMids       []string  `form:"document_mids" json:"document_mids"`
	SubeDocuments      []Medias  `form:"sube_documents" json:"sube_documents"`
	SubeMediaPath      string    `form:"sube_media_path" json:"sube_media_path"`
	SubeMediaAltText   string    `form:"sube_media_alt_text" json:"sube_media_alt_text"`
	SubeMediaTitle     string    `form:"sube_media_title" json:"sube_media_title"`
	IsMain             bool      `form:"is_main" json:"is_main"`
	IsActive           bool      `form:"is_active" json:"is_active"`
	CreatedAt          time.Time `form:"created_at" json:"created_at"`
	UpdatedAt          time.Time `form:"updated_at" json:"updated_at"`
}

type SubelerEdit struct {
	Sid                   string    `form:"sid" json:"sid"`
	Name                  string    `form:"name" json:"name"`
	OldName               string    `form:"old_name" json:"old_name"`
	UrlName               string    `form:"url_name" json:"url_name"`
	OldUrlName            string    `form:"old_url_name" json:"old_url_name"`
	Description           string    `form:"description" json:"description"`
	OldDescription        string    `form:"old_description" json:"old_description"`
	Address               string    `form:"address" json:"address"`
	OldAddress            string    `form:"old_address" json:"old_address"`
	GoogleMapIframe       string    `form:"google_map_iframe" json:"google_map_iframe"`
	OldGoogleMapIframe    string    `form:"old_google_map_iframe" json:"old_google_map_iframe"`
	TransportationInfo    string    `form:"transportation_info" json:"transportation_info"`
	OldTransportationInfo string    `form:"old_transportation_info" json:"old_transportation_info"`
	City                  string    `form:"city" json:"city"`
	OldCity               string    `form:"old_city" json:"old_city"`
	District              string    `form:"district" json:"district"`
	OldDistrict           string    `form:"old_district" json:"old_district"`
	PostalCode            string    `form:"postal_code" json:"postal_code"`
	OldPostalCode         string    `form:"old_postal_code" json:"old_postal_code"`
	Phone                 string    `form:"phone" json:"phone"`
	OldPhone              string    `form:"old_phone" json:"old_phone"`
	Fax                   string    `form:"fax" json:"fax"`
	OldFax                string    `form:"old_fax" json:"old_fax"`
	Email                 string    `form:"email" json:"email"`
	OldEmail              string    `form:"old_email" json:"old_email"`
	Website               string    `form:"website" json:"website"`
	OldWebsite            string    `form:"old_website" json:"old_website"`
	Latitude              float64   `form:"latitude" json:"latitude"`
	OldLatitude           float64   `form:"old_latitude" json:"old_latitude"`
	Longitude             float64   `form:"longitude" json:"longitude"`
	OldLongitude          float64   `form:"old_longitude" json:"old_longitude"`
	WorkingHours          string    `form:"working_hours" json:"working_hours"`
	OldWorkingHours       string    `form:"old_working_hours" json:"old_working_hours"`
	Mid                   int64     `form:"mid" json:"mid"`
	OldMid                int64     `form:"old_mid" json:"old_mid"`
	IsMain                bool      `form:"is_main" json:"is_main"`
	OldIsMain             bool      `form:"old_is_main" json:"old_is_main"`
	IsActive              bool      `form:"is_active" json:"is_active"`
	OldIsActive           bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt             time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt          time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt             time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt          time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type AnlasmaliKurumlar struct {
	Akid              string    `form:"akid" json:"akid"`
	Name              string    `form:"name" json:"name"`
	UrlName           string    `form:"url_name" json:"url_name"`
	Description       string    `form:"description" json:"description"`
	Type              string    `form:"type" json:"type"`
	Sid               string    `form:"sid" json:"sid"`
	SubeName          string    `form:"sube_name" json:"sube_name"`
	ContactPerson     string    `form:"contact_person" json:"contact_person"`
	Phone             string    `form:"phone" json:"phone"`
	Email             string    `form:"email" json:"email"`
	Address           string    `form:"address" json:"address"`
	ContractStartDate time.Time `form:"contract_start_date" json:"contract_start_date"`
	ContractEndDate   time.Time `form:"contract_end_date" json:"contract_end_date"`
	DiscountRate      float64   `form:"discount_rate" json:"discount_rate"`
	PaymentTerms      string    `form:"payment_terms" json:"payment_terms"`
	Notes             string    `form:"notes" json:"notes"`
	LogoMid           int64     `form:"logo_mid" json:"logo_mid"`
	LogoAltText       string    `form:"logo_alt_text" json:"logo_alt_text"`
	LogoTitle         string    `form:"logo_title" json:"logo_title"`
	LogoPath          string    `form:"logo_path" json:"logo_path"`
	IsActive          bool      `form:"is_active" json:"is_active"`
	CreatedAt         time.Time `form:"created_at" json:"created_at"`
	UpdatedAt         time.Time `form:"updated_at" json:"updated_at"`
}

type AnlasmaliKurumlarEdit struct {
	Akid                 string    `form:"akid" json:"akid"`
	Name                 string    `form:"name" json:"name"`
	OldName              string    `form:"old_name" json:"old_name"`
	UrlName              string    `form:"url_name" json:"url_name"`
	OldUrlName           string    `form:"old_url_name" json:"old_url_name"`
	Description          string    `form:"description" json:"description"`
	OldDescription       string    `form:"old_description" json:"old_description"`
	Type                 string    `form:"type" json:"type"`
	OldType              string    `form:"old_type" json:"old_type"`
	Sid                  string    `form:"sid" json:"sid"`
	OldSid               string    `form:"old_sid" json:"old_sid"`
	ContactPerson        string    `form:"contact_person" json:"contact_person"`
	OldContactPerson     string    `form:"old_contact_person" json:"old_contact_person"`
	Phone                string    `form:"phone" json:"phone"`
	OldPhone             string    `form:"old_phone" json:"old_phone"`
	Email                string    `form:"email" json:"email"`
	OldEmail             string    `form:"old_email" json:"old_email"`
	Address              string    `form:"address" json:"address"`
	OldAddress           string    `form:"old_address" json:"old_address"`
	ContractStartDate    time.Time `form:"contract_start_date" json:"contract_start_date"`
	OldContractStartDate time.Time `form:"old_contract_start_date" json:"old_contract_start_date"`
	ContractEndDate      time.Time `form:"contract_end_date" json:"contract_end_date"`
	OldContractEndDate   time.Time `form:"old_contract_end_date" json:"old_contract_end_date"`
	DiscountRate         float64   `form:"discount_rate" json:"discount_rate"`
	OldDiscountRate      float64   `form:"old_discount_rate" json:"old_discount_rate"`
	PaymentTerms         string    `form:"payment_terms" json:"payment_terms"`
	OldPaymentTerms      string    `form:"old_payment_terms" json:"old_payment_terms"`
	Notes                string    `form:"notes" json:"notes"`
	OldNotes             string    `form:"old_notes" json:"old_notes"`
	LogoMid              int64     `form:"logo_mid" json:"logo_mid"`
	OldLogoMid           int64     `form:"old_logo_mid" json:"old_logo_mid"`
	IsActive             bool      `form:"is_active" json:"is_active"`
	OldIsActive          bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt            time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt         time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt            time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt         time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Uzmanliklar struct {
	Uzid        string    `form:"uzid" json:"uzid"`
	Name        string    `form:"name" json:"name"`
	UrlName     string    `form:"url_name" json:"url_name"`
	Description string    `form:"description" json:"description"`
	Icon        string    `form:"icon" json:"icon"`
	IsActive    bool      `form:"is_active" json:"is_active"`
	CreatedAt   time.Time `form:"created_at" json:"created_at"`
	UpdatedAt   time.Time `form:"updated_at" json:"updated_at"`
}

type UzmanliklarEdit struct {
	Uzid           string    `form:"uzid" json:"uzid"`
	Name           string    `form:"name" json:"name"`
	OldName        string    `form:"old_name" json:"old_name"`
	UrlName        string    `form:"url_name" json:"url_name"`
	OldUrlName     string    `form:"old_url_name" json:"old_url_name"`
	Description    string    `form:"description" json:"description"`
	OldDescription string    `form:"old_description" json:"old_description"`
	Icon           string    `form:"icon" json:"icon"`
	OldIcon        string    `form:"old_icon" json:"old_icon"`
	IsActive       bool      `form:"is_active" json:"is_active"`
	OldIsActive    bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt      time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt   time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt      time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt   time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Branslar struct {
	Brid              string    `form:"brid" json:"brid"`
	Name              string    `form:"name" json:"name"`
	UrlName           string    `form:"url_name" json:"url_name"`
	Description       string    `form:"description" json:"description"`
	ShortDescription  string    `form:"short_description" json:"short_description"`
	Services          []string  `form:"services" json:"services"`
	Mid               int64     `form:"mid" json:"mid"`
	BransMediaPath    string    `form:"brans_media_path" json:"brans_media_path"`
	BransMediaAltText string    `form:"brans_media_alt_text" json:"brans_media_alt_text"`
	BransMediaTitle   string    `form:"brans_media_title" json:"brans_media_title"`
	Icon              string    `form:"icon" json:"icon"`
	Sid               string    `form:"sid" json:"sid"`
	HeadDrid          string    `form:"head_drid" json:"head_drid"`
	Phone             string    `form:"phone" json:"phone"`
	Email             string    `form:"email" json:"email"`
	IsActive          bool      `form:"is_active" json:"is_active"`
	CreatedAt         time.Time `form:"created_at" json:"created_at"`
	UpdatedAt         time.Time `form:"updated_at" json:"updated_at"`
	// Head doctor info
	HeadDoctorName         string `form:"head_doctor_name" json:"head_doctor_name"`
	HeadDoctorTitle        string `form:"head_doctor_title" json:"head_doctor_title"`
	HeadDoctorPhone        string `form:"head_doctor_phone" json:"head_doctor_phone"`
	HeadDoctorPhotoPath    string `form:"head_doctor_photo_path" json:"head_doctor_photo_path"`
	HeadDoctorPhotoAltText string `form:"head_doctor_photo_alt_text" json:"head_doctor_photo_alt_text"`
	// Branch info
	BranchName string `form:"branch_name" json:"branch_name"`
	BranchCity string `form:"branch_city" json:"branch_city"`
}

type BranslarEdit struct {
	Brid                string    `form:"brid" json:"brid"`
	Name                string    `form:"name" json:"name"`
	OldName             string    `form:"old_name" json:"old_name"`
	UrlName             string    `form:"url_name" json:"url_name"`
	OldUrlName          string    `form:"old_url_name" json:"old_url_name"`
	Description         string    `form:"description" json:"description"`
	OldDescription      string    `form:"old_description" json:"old_description"`
	ShortDescription    string    `form:"short_description" json:"short_description"`
	OldShortDescription string    `form:"old_short_description" json:"old_short_description"`
	Services            []string  `form:"services" json:"services"`
	OldServices         []string  `form:"old_services" json:"old_services"`
	Mid                 int64     `form:"mid" json:"mid"`
	OldMid              int64     `form:"old_mid" json:"old_mid"`
	Icon                string    `form:"icon" json:"icon"`
	OldIcon             string    `form:"old_icon" json:"old_icon"`
	Sid                 string    `form:"sid" json:"sid"`
	OldSid              string    `form:"old_sid" json:"old_sid"`
	HeadDrid            string    `form:"head_drid" json:"head_drid"`
	OldHeadDrid         string    `form:"old_head_drid" json:"old_head_drid"`
	Phone               string    `form:"phone" json:"phone"`
	OldPhone            string    `form:"old_phone" json:"old_phone"`
	Email               string    `form:"email" json:"email"`
	OldEmail            string    `form:"old_email" json:"old_email"`
	IsActive            bool      `form:"is_active" json:"is_active"`
	OldIsActive         bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt           time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt        time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt           time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt        time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Doktorlar struct {
	Drid                string    `form:"drid" json:"drid"`
	Title               string    `form:"title" json:"title"`
	FirstName           string    `form:"first_name" json:"first_name"`
	LastName            string    `form:"last_name" json:"last_name"`
	UrlName             string    `form:"url_name" json:"url_name"`
	TcKimlik            string    `form:"tc_kimlik" json:"tc_kimlik"`
	DiplomaNo           string    `form:"diploma_no" json:"diploma_no"`
	Phone               string    `form:"phone" json:"phone"`
	Email               string    `form:"email" json:"email"`
	Biography           string    `form:"biography" json:"biography"`
	Education           string    `form:"education" json:"education"`
	ExperienceYears     int64     `form:"experience_years" json:"experience_years"`
	Languages           string    `form:"languages" json:"languages"`
	BirthDate           time.Time `form:"birth_date" json:"birth_date"`
	Gender              string    `form:"gender" json:"gender"`
	PhotoMid            int64     `form:"photo_mid" json:"photo_mid"`
	CvFileMid           int64     `form:"cv_file_mid" json:"cv_file_mid"`
	PhotoPath           string    `form:"photo_path" json:"photo_path"`
	PhotoAltText        string    `form:"photo_alt_text" json:"photo_alt_text"`
	PhotoTitle          string    `form:"photo_title" json:"photo_title"`
	CvFilePath          string    `form:"cv_file_path" json:"cv_file_path"`
	Brid                string    `form:"brid" json:"brid"`
	Sid                 string    `form:"sid" json:"sid"`
	HeadDrid            string    `form:"head_drid" json:"head_drid"`
	RoomNumber          string    `form:"room_number" json:"room_number"`
	AppointmentDuration int64     `form:"appointment_duration" json:"appointment_duration"`
	AppointmentFee      float64   `form:"appointment_fee" json:"appointment_fee"`
	OnlineAppointment   bool      `form:"online_appointment" json:"online_appointment"`
	WorkingHours        string    `form:"working_hours" json:"working_hours"`
	VacationDates       string    `form:"vacation_dates" json:"vacation_dates"`
	FacebookUrl         string    `form:"facebook_url" json:"facebook_url"`
	LinkedinUrl         string    `form:"linkedin_url" json:"linkedin_url"`
	InstagramUrl        string    `form:"instagram_url" json:"instagram_url"`
	XUrl                string    `form:"x_url" json:"x_url"`
	PersonalUrl         string    `form:"personal_url" json:"personal_url"`
	IsActive            bool      `form:"is_active" json:"is_active"`
	CreatedAt           time.Time `form:"created_at" json:"created_at"`
	UpdatedAt           time.Time `form:"updated_at" json:"updated_at"`
	// fields that'll use with that struct:
	Experiences []DoctorExperiences `form:"doctor_experiences" json:"doctor_experiences"`
	Expertises  []DoctorExpertises  `form:"doctor_expertises" json:"doctor_expertises"`
	BranchName  string              `form:"branch_name" json:"branch_name"`
	SubeName    string              `form:"sube_name" json:"sube_name"`
	SubeCity    string              `form:"sube_city" json:"sube_city"`
}

type DoktorlarEdit struct {
	Drid                   string    `form:"drid" json:"drid"`
	Title                  string    `form:"title" json:"title"`
	OldTitle               string    `form:"old_title" json:"old_title"`
	FirstName              string    `form:"first_name" json:"first_name"`
	OldFirstName           string    `form:"old_first_name" json:"old_first_name"`
	LastName               string    `form:"last_name" json:"last_name"`
	OldLastName            string    `form:"old_last_name" json:"old_last_name"`
	UrlName                string    `form:"url_name" json:"url_name"`
	OldUrlName             string    `form:"old_url_name" json:"old_url_name"`
	TcKimlik               string    `form:"tc_kimlik" json:"tc_kimlik"`
	OldTcKimlik            string    `form:"old_tc_kimlik" json:"old_tc_kimlik"`
	DiplomaNo              string    `form:"diploma_no" json:"diploma_no"`
	OldDiplomaNo           string    `form:"old_diploma_no" json:"old_diploma_no"`
	Phone                  string    `form:"phone" json:"phone"`
	OldPhone               string    `form:"old_phone" json:"old_phone"`
	Email                  string    `form:"email" json:"email"`
	OldEmail               string    `form:"old_email" json:"old_email"`
	Biography              string    `form:"biography" json:"biography"`
	OldBiography           string    `form:"old_biography" json:"old_biography"`
	Education              string    `form:"education" json:"education"`
	OldEducation           string    `form:"old_education" json:"old_education"`
	ExperienceYears        int64     `form:"experience_years" json:"experience_years"`
	OldExperienceYears     int64     `form:"old_experience_years" json:"old_experience_years"`
	Languages              string    `form:"languages" json:"languages"`
	OldLanguages           string    `form:"old_languages" json:"old_languages"`
	BirthDate              time.Time `form:"birth_date" json:"birth_date"`
	OldBirthDate           time.Time `form:"old_birth_date" json:"old_birth_date"`
	Gender                 string    `form:"gender" json:"gender"`
	OldGender              string    `form:"old_gender" json:"old_gender"`
	PhotoMid               int64     `form:"photo_mid" json:"photo_mid"`
	OldPhotoMid            int64     `form:"old_photo_mid" json:"old_photo_mid"`
	CvFileMid              int64     `form:"cv_file_mid" json:"cv_file_mid"`
	OldCvFileMid           int64     `form:"old_cv_file_mid" json:"old_cv_file_mid"`
	Brid                   string    `form:"brid" json:"brid"`
	OldBrid                string    `form:"old_brid" json:"old_brid"`
	Sid                    string    `form:"sid" json:"sid"`
	OldSid                 string    `form:"old_sid" json:"old_sid"`
	HeadDrid               string    `form:"head_drid" json:"head_drid"`
	OldHeadDrid            string    `form:"old_head_drid" json:"old_head_drid"`
	RoomNumber             string    `form:"room_number" json:"room_number"`
	OldRoomNumber          string    `form:"old_room_number" json:"old_room_number"`
	AppointmentDuration    int64     `form:"appointment_duration" json:"appointment_duration"`
	OldAppointmentDuration int64     `form:"old_appointment_duration" json:"old_appointment_duration"`
	AppointmentFee         float64   `form:"appointment_fee" json:"appointment_fee"`
	OldAppointmentFee      float64   `form:"old_appointment_fee" json:"old_appointment_fee"`
	OnlineAppointment      bool      `form:"online_appointment" json:"online_appointment"`
	OldOnlineAppointment   bool      `form:"old_online_appointment" json:"old_online_appointment"`
	WorkingHours           string    `form:"working_hours" json:"working_hours"`
	OldWorkingHours        string    `form:"old_working_hours" json:"old_working_hours"`
	VacationDates          string    `form:"vacation_dates" json:"vacation_dates"`
	OldVacationDates       string    `form:"old_vacation_dates" json:"old_vacation_dates"`
	FacebookUrl            string    `form:"facebook_url" json:"facebook_url"`
	OldFacebookUrl         string    `form:"old_facebook_url" json:"old_facebook_url"`
	LinkedinUrl            string    `form:"linkedin_url" json:"linkedin_url"`
	OldLinkedinUrl         string    `form:"old_linkedin_url" json:"old_linkedin_url"`
	InstagramUrl           string    `form:"instagram_url" json:"instagram_url"`
	OldInstagramUrl        string    `form:"old_instagram_url" json:"old_instagram_url"`
	XUrl                   string    `form:"x_url" json:"x_url"`
	OldXUrl                string    `form:"old_x_url" json:"old_x_url"`
	PersonalUrl            string    `form:"personal_url" json:"personal_url"`
	OldPersonalUrl         string    `form:"old_personal_url" json:"old_personal_url"`
	IsActive               bool      `form:"is_active" json:"is_active"`
	OldIsActive            bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt              time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt           time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt              time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt           time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type DoctorExperiences struct {
	Dtid         string    `form:"dtid" json:"dtid"`
	Drid         string    `form:"drid" json:"drid"`
	Name         string    `form:"name" json:"name"`
	StartDate    time.Time `form:"start_date" json:"start_date"`
	EndDate      time.Time `form:"end_date" json:"end_date"`
	CoverMid     int64     `form:"cover_mid" json:"cover_mid"`
	Description  string    `form:"description" json:"description"`
	IsActive     bool      `form:"is_active" json:"is_active"`
	CreatedAt    time.Time `form:"created_at" json:"created_at"`
	UpdatedAt    time.Time `form:"updated_at" json:"updated_at"`
	CoverPath    string    `form:"cover_path" json:"cover_path"`
	CoverAltText string    `form:"cover_alt_text" json:"cover_alt_text"`
	CoverTitle   string    `form:"cover_title" json:"cover_title"`
}

type DoctorExperiencesEdit struct {
	Dtid                      string    `form:"dtid" json:"dtid"`
	Drid                      string    `form:"drid" json:"drid"`
	OldDrid                   string    `form:"old_drid" json:"old_drid"`
	Name                      string    `form:"name" json:"name"`
	OldName                   string    `form:"old_name" json:"old_name"`
	StartDate                 time.Time `form:"start_date" json:"start_date"`
	OldStartDate              time.Time `form:"old_start_date" json:"old_start_date"`
	EndDate                   time.Time `form:"end_date" json:"end_date"`
	OldEndDate                time.Time `form:"old_end_date" json:"old_end_date"`
	CoverMid                  int64     `form:"cover_mid" json:"cover_mid"`
	OldCoverMid               int64     `form:"old_cover_mid" json:"old_cover_mid"`
	Description               string    `form:"description" json:"description"`
	OldDescription            string    `form:"old_description" json:"old_description"`
	IsActive                  bool      `form:"is_active" json:"is_active"`
	OldIsActive               bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt                 time.Time `form:"created_at" json:"created_at"`
	UpdatedAt                 time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt              time.Time `form:"old_updated_at" json:"old_updated_at"`
	ExperienceCoverAltText    string    `form:"experience_cover_alt_text" json:"experience_cover_alt_text"`
	OldExperienceCoverAltText string    `form:"old_experience_cover_alt_text" json:"old_experience_cover_alt_text"`
	ExperienceCoverTitle      string    `form:"experience_cover_title" json:"experience_cover_title"`
	OldExperienceCoverTitle   string    `form:"old_experience_cover_title" json:"old_experience_cover_title"`
}

type DoctorExpertises struct {
	Duid                     string    `form:"duid" json:"duid"`
	Drid                     string    `form:"drid" json:"drid"`
	Uzid                     string    `form:"uzid" json:"uzid"`
	UzmanlikName             string    `form:"uzmanlik_name" json:"uzmanlik_name"`
	UzmanlikDescription      string    `form:"uzmanlik_description" json:"uzmanlik_description"`
	CertificationDate        time.Time `form:"certification_date" json:"certification_date"`
	CertificationInstitution string    `form:"certification_institution" json:"certification_institution"`
	IsPrimary                bool      `form:"is_primary" json:"is_primary"`
	CreatedAt                time.Time `form:"created_at" json:"created_at"`
	UpdatedAt                time.Time `form:"updated_at" json:"updated_at"`
}

type DoctorExpertisesEdit struct {
	Duid                        string    `form:"duid" json:"duid"`
	Drid                        string    `form:"drid" json:"drid"`
	OldDrid                     string    `form:"old_drid" json:"old_drid"`
	Uzid                        string    `form:"uzid" json:"uzid"`
	OldUzid                     string    `form:"old_uzid" json:"old_uzid"`
	CertificationDate           time.Time `form:"certification_date" json:"certification_date"`
	OldCertificationDate        time.Time `form:"old_certification_date" json:"old_certification_date"`
	CertificationInstitution    string    `form:"certification_institution" json:"certification_institution"`
	OldCertificationInstitution string    `form:"old_certification_institution" json:"old_certification_institution"`
	IsPrimary                   bool      `form:"is_primary" json:"is_primary"`
	OldIsPrimary                bool      `form:"old_is_primary" json:"old_is_primary"`
	CreatedAt                   time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt                time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt                   time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt                time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type TibbiBirimler struct {
	Tbid         string    `form:"tbid" json:"tbid"`
	Name         string    `form:"name" json:"name"`
	UrlName      string    `form:"url_name" json:"url_name"`
	Description  string    `form:"description" json:"description"`
	IsActive     bool      `form:"is_active" json:"is_active"`
	CreatedAt    time.Time `form:"created_at" json:"created_at"`
	UpdatedAt    time.Time `form:"updated_at" json:"updated_at"`
	VideoMid     int64     `form:"video_mid" json:"video_mid"`
	VideoPath    string    `form:"video_path" json:"video_path"`
	CoverPath    string    `form:"cover_path" json:"cover_path"`
	CoverAltText string    `form:"cover_alt_text" json:"cover_alt_text"`
	CoverTitle   string    `form:"cover_title" json:"cover_title"`
	CoverMid     int64     `form:"cover_mid" json:"cover_mid"`
}

type TibbiBirimlerEdit struct {
	Tbid            string    `form:"tbid" json:"tbid"`
	Name            string    `form:"name" json:"name"`
	OldName         string    `form:"old_name" json:"old_name"`
	UrlName         string    `form:"url_name" json:"url_name"`
	OldUrlName      string    `form:"old_url_name" json:"old_url_name"`
	Description     string    `form:"description" json:"description"`
	OldDescription  string    `form:"old_description" json:"old_description"`
	IsActive        bool      `form:"is_active" json:"is_active"`
	OldIsActive     bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt       time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt    time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt       time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt    time.Time `form:"old_updated_at" json:"old_updated_at"`
	VideoMid        int64     `form:"video_mid" json:"video_mid"`
	OldVideoMid     int64     `form:"old_video_mid" json:"old_video_mid"`
	VideoPath       string    `form:"video_path" json:"video_path"`
	OldVideoPath    string    `form:"old_video_path" json:"old_video_path"`
	CoverPath       string    `form:"cover_path" json:"cover_path"`
	OldCoverPath    string    `form:"old_cover_path" json:"old_cover_path"`
	CoverAltText    string    `form:"cover_alt_text" json:"cover_alt_text"`
	OldCoverAltText string    `form:"old_cover_alt_text" json:"old_cover_alt_text"`
	CoverTitle      string    `form:"cover_title" json:"cover_title"`
	OldCoverTitle   string    `form:"old_cover_title" json:"old_cover_title"`
	CoverMid        int64     `form:"cover_mid" json:"cover_mid"`
	OldCoverMid     int64     `form:"old_cover_mid" json:"old_cover_mid"`
}

type HomepageContents struct {
	Hcid                  string    `form:"hcid" json:"hcid"`
	Name                  string    `form:"name" json:"name"`
	ContentType           string    `form:"content_type" json:"content_type"`
	SortOrder             int64     `form:"sort_order" json:"sort_order"`
	UrlName               string    `form:"url_name" json:"url_name"`
	ContentHtml           string    `form:"content_html" json:"content_html"`
	ContentJavascript     string    `form:"content_javascript" json:"content_javascript"`
	LaterThanWhichContent int64     `form:"later_than_which_content" json:"later_than_which_content"`
	ContentCss            string    `form:"content_css" json:"content_css"`
	Description           string    `form:"description" json:"description"`
	IsActive              bool      `form:"is_active" json:"is_active"`
	CreatedAt             time.Time `form:"created_at" json:"created_at"`
	UpdatedAt             time.Time `form:"updated_at" json:"updated_at"`
}

type HomepageContentsEdit struct {
	Hcid                     string    `form:"hcid" json:"hcid"`
	Name                     string    `form:"name" json:"name"`
	OldName                  string    `form:"old_name" json:"old_name"`
	ContentType              string    `form:"content_type" json:"content_type"`
	OldContentType           string    `form:"old_content_type" json:"old_content_type"`
	SortOrder                int64     `form:"sort_order" json:"sort_order"`
	OldSortOrder             int64     `form:"old_sort_order" json:"old_sort_order"`
	UrlName                  string    `form:"url_name" json:"url_name"`
	OldUrlName               string    `form:"old_url_name" json:"old_url_name"`
	ContentHtml              string    `form:"content_html" json:"content_html"`
	OldContentHtml           string    `form:"old_content_html" json:"old_content_html"`
	ContentJavascript        string    `form:"content_javascript" json:"content_javascript"`
	OldContentJavascript     string    `form:"old_content_javascript" json:"old_content_javascript"`
	ContentCss               string    `form:"content_css" json:"content_css"`
	OldContentCss            string    `form:"old_content_css" json:"old_content_css"`
	LaterThanWhichContent    int64     `form:"later_than_which_content" json:"later_than_which_content"`
	OldLaterThanWhichContent int64     `form:"old_later_than_which_content" json:"old_later_than_which_content"`
	Description              string    `form:"description" json:"description"`
	OldDescription           string    `form:"old_description" json:"old_description"`
	IsActive                 bool      `form:"is_active" json:"is_active"`
	OldIsActive              bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt                time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt             time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt                time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt             time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type CustomContents struct {
	Ccid              string    `form:"ccid" json:"ccid"`
	Name              string    `form:"name" json:"name"`
	ContentType       string    `form:"content_type" json:"content_type"`
	SortOrder         int64     `form:"sort_order" json:"sort_order"`
	UrlName           string    `form:"url_name" json:"url_name"`
	ContentHtml       string    `form:"content_html" json:"content_html"`
	ContentJavascript string    `form:"content_javascript" json:"content_javascript"`
	ContentCss        string    `form:"content_css" json:"content_css"`
	Description       string    `form:"description" json:"description"`
	Route             string    `form:"route" json:"route"`
	IsActive          bool      `form:"is_active" json:"is_active"`
	CreatedAt         time.Time `form:"created_at" json:"created_at"`
	UpdatedAt         time.Time `form:"updated_at" json:"updated_at"`
}

type CustomContentsEdit struct {
	Ccid                 string    `form:"ccid" json:"ccid"`
	Name                 string    `form:"name" json:"name"`
	OldName              string    `form:"old_name" json:"old_name"`
	ContentType          string    `form:"content_type" json:"content_type"`
	OldContentType       string    `form:"old_content_type" json:"old_content_type"`
	SortOrder            int64     `form:"sort_order" json:"sort_order"`
	OldSortOrder         int64     `form:"old_sort_order" json:"old_sort_order"`
	UrlName              string    `form:"url_name" json:"url_name"`
	OldUrlName           string    `form:"old_url_name" json:"old_url_name"`
	ContentHtml          string    `form:"content_html" json:"content_html"`
	OldContentHtml       string    `form:"old_content_html" json:"old_content_html"`
	ContentJavascript    string    `form:"content_javascript" json:"content_javascript"`
	OldContentJavascript string    `form:"old_content_javascript" json:"old_content_javascript"`
	ContentCss           string    `form:"content_css" json:"content_css"`
	OldContentCss        string    `form:"old_content_css" json:"old_content_css"`
	Description          string    `form:"description" json:"description"`
	OldDescription       string    `form:"old_description" json:"old_description"`
	Route                string    `form:"route" json:"route"`
	OldRoute             string    `form:"old_route" json:"old_route"`
	IsActive             bool      `form:"is_active" json:"is_active"`
	OldIsActive          bool      `form:"old_is_active" json:"old_is_active"`
	CreatedAt            time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt         time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt            time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt         time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Haberler struct {
	Hid            string    `form:"hid" json:"hid"`
	Title          string    `form:"title" json:"title"`
	UrlName        string    `form:"url_name" json:"url_name"`
	Summary        string    `form:"summary" json:"summary"`
	Content        string    `form:"content" json:"content"`
	CoverMid       int64     `form:"cover_mid" json:"cover_mid"`
	CoverPath      string    `form:"cover_path" json:"cover_path"`
	CoverAltText   string    `form:"cover_alt_text" json:"cover_alt_text"`
	CoverTitle     string    `form:"cover_title" json:"cover_title"`
	Category       string    `form:"category" json:"category"`
	Tags           []string  `form:"tags" json:"tags"`
	Author         string    `form:"author" json:"author"`
	PublishDate    time.Time `form:"publish_date" json:"publish_date"`
	IsFeatured     bool      `form:"is_featured" json:"is_featured"`
	IsPublished    bool      `form:"is_published" json:"is_published"`
	ViewsCount     int64     `form:"views_count" json:"views_count"`
	SeoTitle       string    `form:"seo_title" json:"seo_title"`
	SeoDescription string    `form:"seo_description" json:"seo_description"`
	SeoKeywords    string    `form:"seo_keywords" json:"seo_keywords"`
	CreatedAt      time.Time `form:"created_at" json:"created_at"`
	UpdatedAt      time.Time `form:"updated_at" json:"updated_at"`
}

type HaberlerEdit struct {
	Hid               string    `form:"hid" json:"hid"`
	Title             string    `form:"title" json:"title"`
	OldTitle          string    `form:"old_title" json:"old_title"`
	UrlName           string    `form:"url_name" json:"url_name"`
	OldUrlName        string    `form:"old_url_name" json:"old_url_name"`
	Summary           string    `form:"summary" json:"summary"`
	OldSummary        string    `form:"old_summary" json:"old_summary"`
	Content           string    `form:"content" json:"content"`
	OldContent        string    `form:"old_content" json:"old_content"`
	CoverMid          int64     `form:"cover_mid" json:"cover_mid"`
	OldCoverMid       int64     `form:"old_cover_mid" json:"old_cover_mid"`
	CoverAltText      string    `form:"cover_alt_text" json:"cover_alt_text"`
	OldCoverAltText   string    `form:"old_cover_alt_text" json:"old_cover_alt_text"`
	CoverTitle        string    `form:"cover_title" json:"cover_title"`
	OldCoverTitle     string    `form:"old_cover_title" json:"old_cover_title"`
	Category          string    `form:"category" json:"category"`
	OldCategory       string    `form:"old_category" json:"old_category"`
	Tags              []string  `form:"tags" json:"tags"`
	OldTags           []string  `form:"old_tags" json:"old_tags"`
	Author            string    `form:"author" json:"author"`
	OldAuthor         string    `form:"old_author" json:"old_author"`
	PublishDate       time.Time `form:"publish_date" json:"publish_date"`
	OldPublishDate    time.Time `form:"old_publish_date" json:"old_publish_date"`
	IsFeatured        bool      `form:"is_featured" json:"is_featured"`
	OldIsFeatured     bool      `form:"old_is_featured" json:"old_is_featured"`
	IsPublished       bool      `form:"is_published" json:"is_published"`
	OldIsPublished    bool      `form:"old_is_published" json:"old_is_published"`
	ViewsCount        int64     `form:"views_count" json:"views_count"`
	OldViewsCount     int64     `form:"old_views_count" json:"old_views_count"`
	SeoTitle          string    `form:"seo_title" json:"seo_title"`
	OldSeoTitle       string    `form:"old_seo_title" json:"old_seo_title"`
	SeoDescription    string    `form:"seo_description" json:"seo_description"`
	OldSeoDescription string    `form:"old_seo_description" json:"old_seo_description"`
	SeoKeywords       string    `form:"seo_keywords" json:"seo_keywords"`
	OldSeoKeywords    string    `form:"old_seo_keywords" json:"old_seo_keywords"`
	CreatedAt         time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt      time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt         time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt      time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Tedkikler struct {
	Tid          string    `form:"tid" json:"tid"`
	Name         string    `form:"name" json:"name"`
	UrlName      string    `form:"url_name" json:"url_name"`
	Description  string    `form:"description" json:"description"`
	IsActive     bool      `form:"is_active" json:"is_active"`
	CoverMid     int64     `form:"cover_mid" json:"cover_mid"`
	CoverPath    string    `form:"cover_path" json:"cover_path"`
	CoverAltText string    `form:"cover_alt_text" json:"cover_alt_text"`
	CoverTitle   string    `form:"cover_title" json:"cover_title"`
	CreatedAt    time.Time `form:"created_at" json:"created_at"`
	UpdatedAt    time.Time `form:"updated_at" json:"updated_at"`
}

type TedkiklerEdit struct {
	Tid             string    `form:"tid" json:"tid"`
	Name            string    `form:"name" json:"name"`
	OldName         string    `form:"old_name" json:"old_name"`
	UrlName         string    `form:"url_name" json:"url_name"`
	OldUrlName      string    `form:"old_url_name" json:"old_url_name"`
	Description     string    `form:"description" json:"description"`
	OldDescription  string    `form:"old_description" json:"old_description"`
	IsActive        bool      `form:"is_active" json:"is_active"`
	OldIsActive     bool      `form:"old_is_active" json:"old_is_active"`
	CoverMid        int64     `form:"cover_mid" json:"cover_mid"`
	OldCoverMid     int64     `form:"old_cover_mid" json:"old_cover_mid"`
	CoverAltText    string    `form:"cover_alt_text" json:"cover_alt_text"`
	OldCoverAltText string    `form:"old_cover_alt_text" json:"old_cover_alt_text"`
	CoverTitle      string    `form:"cover_title" json:"cover_title"`
	OldCoverTitle   string    `form:"old_cover_title" json:"old_cover_title"`
	CreatedAt       time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt    time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt       time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt    time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Randevular struct {
	Rid              string    `form:"rid" json:"rid"`
	PatientFirstName string    `form:"patient_first_name" json:"patient_first_name"`
	PatientLastName  string    `form:"patient_last_name" json:"patient_last_name"`
	PatientPhone     string    `form:"patient_phone" json:"patient_phone"`
	PatientEmail     string    `form:"patient_email" json:"patient_email"`
	PatientTcKimlik  string    `form:"patient_tc_kimlik" json:"patient_tc_kimlik"`
	PatientBirthDate time.Time `form:"patient_birth_date" json:"patient_birth_date"`
	PatientGender    string    `form:"patient_gender" json:"patient_gender"`
	Drid             string    `form:"drid" json:"drid"`
	Brid             string    `form:"brid" json:"brid"`
	Sid              string    `form:"sid" json:"sid"`
	Akid             string    `form:"akid" json:"akid"`
	Tid              string    `form:"tid" json:"tid"`
	Tbid             string    `form:"tbid" json:"tbid"`
	Rrid             string    `form:"rrid" json:"rrid"`
	AppointmentDate  time.Time `form:"appointment_date" json:"appointment_date"`
	AppointmentTime  time.Time `form:"appointment_time" json:"appointment_time"`
	Duration         int64     `form:"duration" json:"duration"`
	Status           string    `form:"status" json:"status"`
	Notes            string    `form:"notes" json:"notes"`
	Complaint        string    `form:"complaint" json:"complaint"`
	CancelReason     string    `form:"cancel_reason" json:"cancel_reason"`
	ReminderSent     bool      `form:"reminder_sent" json:"reminder_sent"`
	ConfirmationCode string    `form:"confirmation_code" json:"confirmation_code"`
	Price            float64   `form:"price" json:"price"`
	PaymentStatus    string    `form:"payment_status" json:"payment_status"`
	CreatedAt        time.Time `form:"created_at" json:"created_at"`
	UpdatedAt        time.Time `form:"updated_at" json:"updated_at"`
	// Additional fields for display
	DoctorName string `json:"doctor_name"`
	SubeName   string `json:"sube_name"`
	SubeCity   string `json:"sube_city"`
	BranchName string `json:"branch_name"`
}

type RandevularEdit struct {
	Rid                 string    `form:"rid" json:"rid"`
	PatientFirstName    string    `form:"patient_first_name" json:"patient_first_name"`
	OldPatientFirstName string    `form:"old_patient_first_name" json:"old_patient_first_name"`
	PatientLastName     string    `form:"patient_last_name" json:"patient_last_name"`
	OldPatientLastName  string    `form:"old_patient_last_name" json:"old_patient_last_name"`
	PatientPhone        string    `form:"patient_phone" json:"patient_phone"`
	OldPatientPhone     string    `form:"old_patient_phone" json:"old_patient_phone"`
	PatientEmail        string    `form:"patient_email" json:"patient_email"`
	OldPatientEmail     string    `form:"old_patient_email" json:"old_patient_email"`
	PatientTcKimlik     string    `form:"patient_tc_kimlik" json:"patient_tc_kimlik"`
	OldPatientTcKimlik  string    `form:"old_patient_tc_kimlik" json:"old_patient_tc_kimlik"`
	PatientBirthDate    time.Time `form:"patient_birth_date" json:"patient_birth_date"`
	OldPatientBirthDate time.Time `form:"old_patient_birth_date" json:"old_patient_birth_date"`
	PatientGender       string    `form:"patient_gender" json:"patient_gender"`
	OldPatientGender    string    `form:"old_patient_gender" json:"old_patient_gender"`
	Drid                string    `form:"drid" json:"drid"`
	OldDrid             string    `form:"old_drid" json:"old_drid"`
	Brid                string    `form:"brid" json:"brid"`
	OldBrid             string    `form:"old_brid" json:"old_brid"`
	Sid                 string    `form:"sid" json:"sid"`
	OldSid              string    `form:"old_sid" json:"old_sid"`
	Akid                string    `form:"akid" json:"akid"`
	OldAkid             string    `form:"old_akid" json:"old_akid"`
	Tid                 string    `form:"tid" json:"tid"`
	OldTid              string    `form:"old_tid" json:"old_tid"`
	Tbid                string    `form:"tbid" json:"tbid"`
	OldTbid             string    `form:"old_tbid" json:"old_tbid"`
	AppointmentDate     time.Time `form:"appointment_date" json:"appointment_date"`
	OldAppointmentDate  time.Time `form:"old_appointment_date" json:"old_appointment_date"`
	AppointmentTime     time.Time `form:"appointment_time" json:"appointment_time"`
	OldAppointmentTime  time.Time `form:"old_appointment_time" json:"old_appointment_time"`
	Duration            int64     `form:"duration" json:"duration"`
	OldDuration         int64     `form:"old_duration" json:"old_duration"`
	Status              string    `form:"status" json:"status"`
	OldStatus           string    `form:"old_status" json:"old_status"`
	Notes               string    `form:"notes" json:"notes"`
	OldNotes            string    `form:"old_notes" json:"old_notes"`
	Complaint           string    `form:"complaint" json:"complaint"`
	OldComplaint        string    `form:"old_complaint" json:"old_complaint"`
	CancelReason        string    `form:"cancel_reason" json:"cancel_reason"`
	OldCancelReason     string    `form:"old_cancel_reason" json:"old_cancel_reason"`
	ReminderSent        bool      `form:"reminder_sent" json:"reminder_sent"`
	OldReminderSent     bool      `form:"old_reminder_sent" json:"old_reminder_sent"`
	ConfirmationCode    string    `form:"confirmation_code" json:"confirmation_code"`
	OldConfirmationCode string    `form:"old_confirmation_code" json:"old_confirmation_code"`
	Price               float64   `form:"price" json:"price"`
	OldPrice            float64   `form:"old_price" json:"old_price"`
	PaymentStatus       string    `form:"payment_status" json:"payment_status"`
	OldPaymentStatus    string    `form:"old_payment_status" json:"old_payment_status"`
	CreatedAt           time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt        time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt           time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt        time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type RandevuRequests struct {
	Rrid             string    `form:"rrid" json:"rrid"`
	PatientFirstName string    `form:"patient_first_name" json:"patient_first_name"`
	PatientLastName  string    `form:"patient_last_name" json:"patient_last_name"`
	PatientPhone     string    `form:"patient_phone" json:"patient_phone"`
	PatientEmail     string    `form:"patient_email" json:"patient_email"`
	PreferredDate    time.Time `form:"preferred_date" json:"preferred_date"`
	PreferredTime    time.Time `form:"preferred_time" json:"preferred_time"`
	Message          string    `form:"message" json:"message"`
	Drid             string    `form:"drid" json:"drid"`
	Sid              string    `form:"sid" json:"sid"`
	RecaptchaToken   string    `form:"recaptcha_token" json:"recaptcha_token"`
	CreatedAt        time.Time `form:"created_at" json:"created_at"`
	UpdatedAt        time.Time `form:"updated_at" json:"updated_at"`
}

type ContactRequests struct {
	Crid           string    `form:"crid" json:"crid"`
	FirstName      string    `form:"first_name" json:"first_name"`
	LastName       string    `form:"last_name" json:"last_name"`
	Email          string    `form:"email" json:"email"`
	Phone          string    `form:"phone" json:"phone"`
	Subject        string    `form:"subject" json:"subject"`
	Message        string    `form:"message" json:"message"`
	Department     string    `form:"department" json:"department"`
	Priority       string    `form:"priority" json:"priority"`
	Status         string    `form:"status" json:"status"`
	AssignedTo     string    `form:"assigned_to" json:"assigned_to"`
	Response       string    `form:"response" json:"response"`
	ResponseDate   time.Time `form:"response_date" json:"response_date"`
	IpAddress      string    `form:"ip_address" json:"ip_address"`
	UserAgent      string    `form:"user_agent" json:"user_agent"`
	IsRead         bool      `form:"is_read" json:"is_read"`
	IsReplied      bool      `form:"is_replied" json:"is_replied"`
	Source         string    `form:"source" json:"source"`
	RecaptchaToken string    `form:"recaptcha_token" json:"recaptcha_token"`
	CreatedAt      time.Time `form:"created_at" json:"created_at"`
	UpdatedAt      time.Time `form:"updated_at" json:"updated_at"`
}

type ContactRequestsEdit struct {
	Crid            string    `form:"crid" json:"crid"`
	FirstName       string    `form:"first_name" json:"first_name"`
	OldFirstName    string    `form:"old_first_name" json:"old_first_name"`
	LastName        string    `form:"last_name" json:"last_name"`
	OldLastName     string    `form:"old_last_name" json:"old_last_name"`
	Email           string    `form:"email" json:"email"`
	OldEmail        string    `form:"old_email" json:"old_email"`
	Phone           string    `form:"phone" json:"phone"`
	OldPhone        string    `form:"old_phone" json:"old_phone"`
	Subject         string    `form:"subject" json:"subject"`
	OldSubject      string    `form:"old_subject" json:"old_subject"`
	Message         string    `form:"message" json:"message"`
	OldMessage      string    `form:"old_message" json:"old_message"`
	Department      string    `form:"department" json:"department"`
	OldDepartment   string    `form:"old_department" json:"old_department"`
	Priority        string    `form:"priority" json:"priority"`
	OldPriority     string    `form:"old_priority" json:"old_priority"`
	Status          string    `form:"status" json:"status"`
	OldStatus       string    `form:"old_status" json:"old_status"`
	AssignedTo      string    `form:"assigned_to" json:"assigned_to"`
	OldAssignedTo   string    `form:"old_assigned_to" json:"old_assigned_to"`
	Response        string    `form:"response" json:"response"`
	OldResponse     string    `form:"old_response" json:"old_response"`
	ResponseDate    time.Time `form:"response_date" json:"response_date"`
	OldResponseDate time.Time `form:"old_response_date" json:"old_response_date"`
	IpAddress       string    `form:"ip_address" json:"ip_address"`
	OldIpAddress    string    `form:"old_ip_address" json:"old_ip_address"`
	UserAgent       string    `form:"user_agent" json:"user_agent"`
	OldUserAgent    string    `form:"old_user_agent" json:"old_user_agent"`
	Source          string    `form:"source" json:"source"`
	OldSource       string    `form:"old_source" json:"old_source"`
	CreatedAt       time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt    time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt       time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt    time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type JobApplications struct {
	Jaid               string    `form:"jaid" json:"jaid"`
	FirstName          string    `form:"first_name" json:"first_name"`
	LastName           string    `form:"last_name" json:"last_name"`
	Email              string    `form:"email" json:"email"`
	Phone              string    `form:"phone" json:"phone"`
	TcKimlik           string    `form:"tc_kimlik" json:"tc_kimlik"`
	BirthDate          time.Time `form:"birth_date" json:"birth_date"`
	Gender             string    `form:"gender" json:"gender"`
	Address            string    `form:"address" json:"address"`
	City               string    `form:"city" json:"city"`
	EducationLevel     string    `form:"education_level" json:"education_level"`
	University         string    `form:"university" json:"university"`
	Department         string    `form:"department" json:"department"`
	GraduationYear     int64     `form:"graduation_year" json:"graduation_year"`
	ExperienceYears    int64     `form:"experience_years" json:"experience_years"`
	PositionApplied    string    `form:"position_applied" json:"position_applied"`
	DepartmentApplied  string    `form:"department_applied" json:"department_applied"`
	Sid                int64     `form:"sid" json:"sid"`
	SalaryExpectation  float64   `form:"salary_expectation" json:"salary_expectation"`
	AvailableStartDate time.Time `form:"available_start_date" json:"available_start_date"`
	Languages          string    `form:"languages" json:"languages"`
	Skills             string    `form:"skills" json:"skills"`
	CoverLetter        string    `form:"cover_letter" json:"cover_letter"`
	CvFileMid          int64     `form:"cv_file_mid" json:"cv_file_mid"`
	CvFilePath         string    `form:"cv_file_path" json:"cv_file_path"`
	DiplomaFileMid     int64     `form:"diploma_file_mid" json:"diploma_file_mid"`
	DiplomaFilePath    string    `form:"diploma_file_path" json:"diploma_file_path"`
	WorkReferences     string    `form:"work_references" json:"work_references"`
	Status             string    `form:"status" json:"status"`
	Notes              string    `form:"notes" json:"notes"`
	InterviewDate      time.Time `form:"interview_date" json:"interview_date"`
	InterviewNotes     string    `form:"interview_notes" json:"interview_notes"`
	RejectionReason    string    `form:"rejection_reason" json:"rejection_reason"`
	IsRead             bool      `form:"is_read" json:"is_read"`
	RecaptchaToken     string    `form:"recaptcha_token" json:"recaptcha_token"`
	CreatedAt          time.Time `form:"created_at" json:"created_at"`
	UpdatedAt          time.Time `form:"updated_at" json:"updated_at"`
}

type JobApplicationsEdit struct {
	Jaid                  string    `form:"jaid" json:"jaid"`
	FirstName             string    `form:"first_name" json:"first_name"`
	OldFirstName          string    `form:"old_first_name" json:"old_first_name"`
	LastName              string    `form:"last_name" json:"last_name"`
	OldLastName           string    `form:"old_last_name" json:"old_last_name"`
	Email                 string    `form:"email" json:"email"`
	OldEmail              string    `form:"old_email" json:"old_email"`
	Phone                 string    `form:"phone" json:"phone"`
	OldPhone              string    `form:"old_phone" json:"old_phone"`
	TcKimlik              string    `form:"tc_kimlik" json:"tc_kimlik"`
	OldTcKimlik           string    `form:"old_tc_kimlik" json:"old_tc_kimlik"`
	BirthDate             time.Time `form:"birth_date" json:"birth_date"`
	OldBirthDate          time.Time `form:"old_birth_date" json:"old_birth_date"`
	Gender                string    `form:"gender" json:"gender"`
	OldGender             string    `form:"old_gender" json:"old_gender"`
	Address               string    `form:"address" json:"address"`
	OldAddress            string    `form:"old_address" json:"old_address"`
	City                  string    `form:"city" json:"city"`
	OldCity               string    `form:"old_city" json:"old_city"`
	EducationLevel        string    `form:"education_level" json:"education_level"`
	OldEducationLevel     string    `form:"old_education_level" json:"old_education_level"`
	University            string    `form:"university" json:"university"`
	OldUniversity         string    `form:"old_university" json:"old_university"`
	Department            string    `form:"department" json:"department"`
	OldDepartment         string    `form:"old_department" json:"old_department"`
	GraduationYear        int64     `form:"graduation_year" json:"graduation_year"`
	OldGraduationYear     int64     `form:"old_graduation_year" json:"old_graduation_year"`
	ExperienceYears       int64     `form:"experience_years" json:"experience_years"`
	OldExperienceYears    int64     `form:"old_experience_years" json:"old_experience_years"`
	PositionApplied       string    `form:"position_applied" json:"position_applied"`
	OldPositionApplied    string    `form:"old_position_applied" json:"old_position_applied"`
	DepartmentApplied     string    `form:"department_applied" json:"department_applied"`
	OldDepartmentApplied  string    `form:"old_department_applied" json:"old_department_applied"`
	SalaryExpectation     float64   `form:"salary_expectation" json:"salary_expectation"`
	OldSalaryExpectation  float64   `form:"old_salary_expectation" json:"old_salary_expectation"`
	AvailableStartDate    time.Time `form:"available_start_date" json:"available_start_date"`
	OldAvailableStartDate time.Time `form:"old_available_start_date" json:"old_available_start_date"`
	Languages             string    `form:"languages" json:"languages"`
	OldLanguages          string    `form:"old_languages" json:"old_languages"`
	Skills                string    `form:"skills" json:"skills"`
	OldSkills             string    `form:"old_skills" json:"old_skills"`
	CoverLetter           string    `form:"cover_letter" json:"cover_letter"`
	OldCoverLetter        string    `form:"old_cover_letter" json:"old_cover_letter"`
	CvFileMid             int64     `form:"cv_file_mid" json:"cv_file_mid"`
	OldCvFileMid          int64     `form:"old_cv_file_mid" json:"old_cv_file_mid"`
	DiplomaFileMid        int64     `form:"diploma_file_mid" json:"diploma_file_mid"`
	OldDiplomaFileMid     int64     `form:"old_diploma_file_mid" json:"old_diploma_file_mid"`
	WorkReferences        string    `form:"work_references" json:"work_references"`
	OldWorkReferences     string    `form:"old_work_references" json:"old_work_references"`
	Status                string    `form:"status" json:"status"`
	OldStatus             string    `form:"old_status" json:"old_status"`
	Notes                 string    `form:"notes" json:"notes"`
	OldNotes              string    `form:"old_notes" json:"old_notes"`
	InterviewDate         time.Time `form:"interview_date" json:"interview_date"`
	OldInterviewDate      time.Time `form:"old_interview_date" json:"old_interview_date"`
	InterviewNotes        string    `form:"interview_notes" json:"interview_notes"`
	OldInterviewNotes     string    `form:"old_interview_notes" json:"old_interview_notes"`
	RejectionReason       string    `form:"rejection_reason" json:"rejection_reason"`
	OldRejectionReason    string    `form:"old_rejection_reason" json:"old_rejection_reason"`
	CreatedAt             time.Time `form:"created_at" json:"created_at"`
	OldCreatedAt          time.Time `form:"old_created_at" json:"old_created_at"`
	UpdatedAt             time.Time `form:"updated_at" json:"updated_at"`
	OldUpdatedAt          time.Time `form:"old_updated_at" json:"old_updated_at"`
}

type Notification struct {
	Nid               string    `form:"nid" json:"nid"`
	Message           string    `form:"message" json:"message"`
	NotificationType  string    `form:"notification_type" json:"notification_type"`
	NotificationLevel string    `form:"notification_level" json:"notification_level"`
	Link              string    `form:"link" json:"link"`
	IsRead            bool      `form:"is_read" json:"is_read"`
	CreatedAt         time.Time `form:"created_at" json:"created_at"`
	UpdatedAt         time.Time `form:"updated_at" json:"updated_at"`
}

type DeleteOptionMediaInputs struct {
	MediaType string `form:"media_type" json:"media_type"`
}

type BanUserInputs struct {
	IsActive bool `form:"is_active" json:"is_active"`
}

type ChangeDoctorBranchInputs struct {
	Brid string `form:"brid" json:"brid"`
}

type ChangeOrderInputs struct {
	ParentId     string `form:"parent_id" json:"parent_id"`
	OldParentId  string `form:"old_parent_id" json:"old_parent_id"`
	NewSortOrder int64  `form:"new_sort_order" json:"new_sort_order"`
	OldSortOrder int64  `form:"old_sort_order" json:"old_sort_order"`
}

type AddExpertiseToADoctorInputs struct {
	Uzid                     string `json:"uzid"`
	CertificationDate        string `json:"certification_date"`
	CertificationInstitution string `json:"certification_institution"`
	IsPrimary                bool   `json:"is_primary"`
}

type DeleteFileInputs struct {
	FileName string `form:"file_name" json:"file_name"`
}

type DeleteDocumentInputs struct {
	DocumentMid string `form:"document_mid" json:"document_mid"`
}

type EditDocumentInputs struct {
	DocumentMid string `form:"document_mid" json:"document_mid"`
	Data        string `form:"data" json:"data"`
}

type AddDocumentInfoPairs struct {
	Mid      int64  `form:"mid" json:"mid"`
	FilePath string `form:"file_path" json:"file_path"`
	Data     string `form:"data" json:"data"`
}

type File struct {
	Name string `json:"name" form:"name"`
	Url  string `json:"url" form:"url"`
	Size int64  `json:"size" form:"size"`
	Type string `json:"type" form:"type"`
	Icon string `json:"icon" form:"icon"`
}

type EmailInfos struct {
	Host        string
	Port        int64
	Username    string
	Password    string
	From        string
	To          []string
	Subject     string
	Body        string
	PlainText   string
	Attachments []string
}

type RespondEmailInputs struct {
	Email       string   `form:"email" json:"email"`
	Subject     string   `form:"subject" json:"subject"`
	Body        string   `form:"body" json:"body"`
	PlainText   string   `form:"plain_text" json:"plain_text"`
	Attachments []string `form:"attachments" json:"attachments"`
}

type PanelStatistics struct {
	// Users statistics
	TotalUsers     int `json:"total_users"`
	ActiveUsers    int `json:"active_users"`
	InactiveUsers  int `json:"inactive_users"`
	AdminUsers     int `json:"admin_users"`
	ModeratorUsers int `json:"moderator_users"`

	// Branches (Şubeler) statistics
	TotalBranches    int `json:"total_branches"`
	ActiveBranches   int `json:"active_branches"`
	InactiveBranches int `json:"inactive_branches"`
	MainBranch       int `json:"main_branch"`

	// Doctors (Doktorlar) statistics
	TotalDoctors             int            `json:"total_doctors"`
	ActiveDoctors            int            `json:"active_doctors"`
	InactiveDoctors          int            `json:"inactive_doctors"`
	OnlineAppointmentDoctors int            `json:"online_appointment_doctors"`
	DoctorsByTitle           map[string]int `json:"doctors_by_title"`

	// Departments (Branslar) statistics
	TotalDepartments    int `json:"total_departments"`
	ActiveDepartments   int `json:"active_departments"`
	InactiveDepartments int `json:"inactive_departments"`
	DepartmentsWithHead int `json:"departments_with_head"`

	// Appointments (Randevular) statistics
	TotalAppointments     int `json:"total_appointments"`
	PendingAppointments   int `json:"pending_appointments"`
	ConfirmedAppointments int `json:"confirmed_appointments"`
	CompletedAppointments int `json:"completed_appointments"`
	CancelledAppointments int `json:"cancelled_appointments"`
	TodayAppointments     int `json:"today_appointments"`
	WeekAppointments      int `json:"week_appointments"`
	MonthAppointments     int `json:"month_appointments"`

	// Appointment Requests (Randevu Talepleri) statistics
	TotalAppointmentRequests int `json:"total_appointment_requests"`
	TodayAppointmentRequests int `json:"today_appointment_requests"`
	WeekAppointmentRequests  int `json:"week_appointment_requests"`
	MonthAppointmentRequests int `json:"month_appointment_requests"`

	// Contact Requests (İletişim İstekleri) statistics
	TotalContactRequests   int `json:"total_contact_requests"`
	UnreadContactRequests  int `json:"unread_contact_requests"`
	RepliedContactRequests int `json:"replied_contact_requests"`

	// Job Applications (İş Başvuruları) statistics
	TotalJobApplications  int `json:"total_job_applications"`
	ReadJobApplications   int `json:"read_job_applications"`
	UnreadJobApplications int `json:"unread_job_applications"`

	// News (Haberler) statistics
	TotalNews      int `json:"total_news"`
	PublishedNews  int `json:"published_news"`
	DraftNews      int `json:"draft_news"`
	FeaturedNews   int `json:"featured_news"`
	TotalNewsViews int `json:"total_news_views"`

	// Contracted Institutions (Anlaşmalı Kurumlar) statistics
	TotalContractedInstitutions    int `json:"total_contracted_institutions"`
	ActiveContractedInstitutions   int `json:"active_contracted_institutions"`
	InactiveContractedInstitutions int `json:"inactive_contracted_institutions"`
	InsuranceInstitutions          int `json:"insurance_institutions"`
	CorporateInstitutions          int `json:"corporate_institutions"`
	ExpiringSoonContracts          int `json:"expiring_soon_contracts"` // Contracts expiring in next 30 days

	// Medical Units (Tıbbi Birimler) statistics
	TotalMedicalUnits    int `json:"total_medical_units"`
	ActiveMedicalUnits   int `json:"active_medical_units"`
	InactiveMedicalUnits int `json:"inactive_medical_units"`

	// Tests (Tedkikler) statistics
	TotalTests    int `json:"total_tests"`
	ActiveTests   int `json:"active_tests"`
	InactiveTests int `json:"inactive_tests"`

	// Specializations (Uzmanlıklar) statistics
	TotalSpecializations    int `json:"total_specializations"`
	ActiveSpecializations   int `json:"active_specializations"`
	InactiveSpecializations int `json:"inactive_specializations"`

	// Doctor Experiences statistics
	TotalDoctorExperiences    int `json:"total_doctor_experiences"`
	ActiveDoctorExperiences   int `json:"active_doctor_experiences"`
	InactiveDoctorExperiences int `json:"inactive_doctor_experiences"`

	// Doctor Expertises statistics
	TotalDoctorExpertises   int `json:"total_doctor_expertises"`
	PrimaryDoctorExpertises int `json:"primary_doctor_expertises"`

	// Header Buttons statistics
	TotalHeaderButtons    int `json:"total_header_buttons"`
	ActiveHeaderButtons   int `json:"active_header_buttons"`
	InactiveHeaderButtons int `json:"inactive_header_buttons"`
	ParentHeaderButtons   int `json:"parent_header_buttons"`
	ChildHeaderButtons    int `json:"child_header_buttons"`

	// Testimonials statistics
	TotalTestimonials    int `json:"total_testimonials"`
	ActiveTestimonials   int `json:"active_testimonials"`
	InactiveTestimonials int `json:"inactive_testimonials"`

	// Media Files (Medyalar) statistics
	TotalMediaFiles int   `json:"total_media_files"`
	ImageFiles      int   `json:"image_files"`
	VideoFiles      int   `json:"video_files"`
	DocumentFiles   int   `json:"document_files"`
	TotalMediaSize  int64 `json:"total_media_size"` // in bytes

	// Homepage Contents statistics
	TotalHomepageContents    int `json:"total_homepage_contents"`
	ActiveHomepageContents   int `json:"active_homepage_contents"`
	InactiveHomepageContents int `json:"inactive_homepage_contents"`
	PopupHomepageContents    int `json:"popup_homepage_contents"`

	// Custom Contents statistics
	TotalCustomContents    int `json:"total_custom_contents"`
	ActiveCustomContents   int `json:"active_custom_contents"`
	InactiveCustomContents int `json:"inactive_custom_contents"`
	MainPageCustomContents int `json:"main_page_custom_contents"`
	HistoryCustomContents  int `json:"history_custom_contents"`
	AboutUsCustomContents  int `json:"about_us_custom_contents"`

	// Options statistics
	TotalOptionSets    int `json:"total_option_sets"`
	ActiveOptionSets   int `json:"active_option_sets"`
	InactiveOptionSets int `json:"inactive_option_sets"`
	TestingOptionSets  int `json:"testing_option_sets"`

	// Notifications statistics
	TotalNotifications  int `json:"total_notifications"`
	UnreadNotifications int `json:"unread_notifications"`
	ReadNotifications   int `json:"read_notifications"`

	// Financial statistics
	TotalRevenue            float64 `json:"total_revenue"`
	MonthRevenue            float64 `json:"month_revenue"`
	TodayRevenue            float64 `json:"today_revenue"`
	AverageAppointmentPrice float64 `json:"average_appointment_price"`
}

type RecaptchaResponse struct {
	Success     bool      `json:"success"`
	ChallengeTs time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

type PaginateAnlasmaliKurumlarInputs struct {
	Offset int64 `form:"offset" json:"offset"`
}
