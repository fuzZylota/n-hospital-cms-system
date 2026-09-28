package models

import "time"

// OptionsRead is the settings-page view model. Secret fields are deliberately absent.
type OptionsRead struct {
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
}
