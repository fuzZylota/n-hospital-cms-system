package postgres

import (
	"context"
	"database/sql"
	"strconv"

	"models/data"
)

const activeSiteOptionsSQL = `SELECT
    o.oid,
    o.option_set_is_active,
    o.option_set_is_testing_now,
    o.site_name,
    o.site_description,
    o.maintenance_mode,
    o.preloader,
    o.facebook_url,
    o.twitter_url,
    o.instagram_url,
    o.linkedin_url,
    o.contact_email,
    o.contact_phone,
    o.main_page_meta_title,
    o.main_page_meta_description,
    o.google_analytics,
    o.primary_color,
    o.secondary_color,
    o.accent_color,
    o.background_color,
    o.font_color,
    o.font_family,
    o.items_per_page,
    o.enable_testimonials,
    o.maximum_sublinks_on_a_menu_item,
    o.show_doctor_social_media,
    o.show_doctor_appointment_fee,
    o.show_anlasmali_kurum_pictures,
    o.google_recaptcha_site_key,
    logo.file_path,
    logo.alt_text,
    logo.title,
    light_logo.file_path,
    light_logo.alt_text,
    light_logo.title,
    favicon.file_path,
    default_media.file_path,
    default_media.alt_text,
    default_media.title
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
LEFT JOIN medias light_logo ON o.site_light_logo_mid = light_logo.mid
LEFT JOIN medias favicon ON o.site_favicon_mid = favicon.mid
LEFT JOIN medias default_media ON o.default_page_mid = default_media.mid
WHERE o.option_set_is_active = TRUE`

const testingSiteOptionsSQL = `SELECT
    o.oid,
    o.option_set_is_active,
    o.option_set_is_testing_now,
    o.site_name,
    o.site_description,
    o.maintenance_mode,
    o.preloader,
    o.facebook_url,
    o.twitter_url,
    o.instagram_url,
    o.linkedin_url,
    o.contact_email,
    o.contact_phone,
    o.main_page_meta_title,
    o.main_page_meta_description,
    o.google_analytics,
    o.primary_color,
    o.secondary_color,
    o.accent_color,
    o.background_color,
    o.font_color,
    o.font_family,
    o.items_per_page,
    o.enable_testimonials,
    o.maximum_sublinks_on_a_menu_item,
    o.show_doctor_social_media,
    o.show_doctor_appointment_fee,
    o.show_anlasmali_kurum_pictures,
    o.google_recaptcha_site_key,
    logo.file_path,
    logo.alt_text,
    logo.title,
    light_logo.file_path,
    light_logo.alt_text,
    light_logo.title,
    favicon.file_path,
    default_media.file_path,
    default_media.alt_text,
    default_media.title
FROM options o
LEFT JOIN medias logo ON o.site_logo_mid = logo.mid
LEFT JOIN medias light_logo ON o.site_light_logo_mid = light_logo.mid
LEFT JOIN medias favicon ON o.site_favicon_mid = favicon.mid
LEFT JOIN medias default_media ON o.default_page_mid = default_media.mid
WHERE o.option_set_is_testing_now = TRUE`

const readUploadPolicySQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, max_upload_size
FROM options
WHERE option_set_is_active = TRUE`

// The legacy AddDoctor lookup uses the first active row if more than one exists.
// Keep that behavior only for the transaction-bound transitional read.
const readUploadPolicyTxSQL = readUploadPolicySQL + "\nLIMIT 1"

const readPasswordPolicySQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, require_strong_password
FROM options
WHERE option_set_is_active = TRUE`

const readMailDeliveryOptionsSQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, smtp_host, smtp_port, smtp_username, smtp_password
FROM options
WHERE option_set_is_active = TRUE`

const readCaptchaVerificationOptionsSQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, google_recaptcha_site_key, google_recaptcha_secret_key
FROM options
WHERE option_set_is_active = TRUE`

const readOptionMediaReferencesSQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, site_logo_mid, site_light_logo_mid, site_favicon_mid, default_page_mid
FROM options
WHERE option_set_is_active = TRUE`

// OptionsRepository reads option data from the supplied application-owned pool.
type OptionsRepository struct {
	db *sql.DB
}

// NewOptionsRepository borrows db. It does not open, register, or close a pool.
func NewOptionsRepository(db *sql.DB) *OptionsRepository {
	return &OptionsRepository{db: db}
}

var (
	_ data.SiteOptionsReader                = (*OptionsRepository)(nil)
	_ data.UploadPolicyReader               = (*OptionsRepository)(nil)
	_ data.PasswordPolicyReader             = (*OptionsRepository)(nil)
	_ data.MailDeliveryOptionsReader        = (*OptionsRepository)(nil)
	_ data.CaptchaVerificationOptionsReader = (*OptionsRepository)(nil)
	_ data.OptionMediaReferencesReader      = (*OptionsRepository)(nil)
)

type siteOptionsRow struct {
	id                        int64
	active                    sql.NullBool
	testing                   sql.NullBool
	siteName                  sql.NullString
	siteDescription           sql.NullString
	maintenanceMode           sql.NullBool
	preloader                 sql.NullString
	facebookURL               sql.NullString
	twitterURL                sql.NullString
	instagramURL              sql.NullString
	linkedInURL               sql.NullString
	contactEmail              sql.NullString
	contactPhone              sql.NullString
	mainPageMetaTitle         sql.NullString
	mainPageMetaDescription   sql.NullString
	googleAnalytics           sql.NullString
	primaryColor              sql.NullString
	secondaryColor            sql.NullString
	accentColor               sql.NullString
	backgroundColor           sql.NullString
	fontColor                 sql.NullString
	fontFamily                sql.NullString
	itemsPerPage              sql.NullInt64
	enableTestimonials        sql.NullBool
	maximumSublinksOnMenuItem sql.NullInt64
	showDoctorSocialMedia     sql.NullBool
	showDoctorAppointmentFee  sql.NullBool
	showPartnerPictures       sql.NullBool
	recaptchaSiteKey          sql.NullString
	siteLogoPath              sql.NullString
	siteLogoAltText           sql.NullString
	siteLogoTitle             sql.NullString
	siteLightLogoPath         sql.NullString
	siteLightLogoAltText      sql.NullString
	siteLightLogoTitle        sql.NullString
	faviconPath               sql.NullString
	defaultPageMediaPath      sql.NullString
	defaultPageMediaAltText   sql.NullString
	defaultPageMediaTitle     sql.NullString
}

// ReadSiteOptions reads exactly the requested option-set state. Missing testing
// data is not replaced with active data.
func (r *OptionsRepository) ReadSiteOptions(ctx context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
	var query string
	switch selection {
	case data.ActiveOptionSet:
		query = activeSiteOptionsSQL
	case data.TestingOptionSet:
		query = testingSiteOptionsSQL
	default:
		return data.SiteOptions{}, false, newRepositoryReadError(readSiteOptions, readInvalidSelection, nil)
	}

	var row siteOptionsRow
	found, err := r.readOne(ctx, readSiteOptions, query, func(rows *sql.Rows) error {
		return rows.Scan(
			&row.id,
			&row.active,
			&row.testing,
			&row.siteName,
			&row.siteDescription,
			&row.maintenanceMode,
			&row.preloader,
			&row.facebookURL,
			&row.twitterURL,
			&row.instagramURL,
			&row.linkedInURL,
			&row.contactEmail,
			&row.contactPhone,
			&row.mainPageMetaTitle,
			&row.mainPageMetaDescription,
			&row.googleAnalytics,
			&row.primaryColor,
			&row.secondaryColor,
			&row.accentColor,
			&row.backgroundColor,
			&row.fontColor,
			&row.fontFamily,
			&row.itemsPerPage,
			&row.enableTestimonials,
			&row.maximumSublinksOnMenuItem,
			&row.showDoctorSocialMedia,
			&row.showDoctorAppointmentFee,
			&row.showPartnerPictures,
			&row.recaptchaSiteKey,
			&row.siteLogoPath,
			&row.siteLogoAltText,
			&row.siteLogoTitle,
			&row.siteLightLogoPath,
			&row.siteLightLogoAltText,
			&row.siteLightLogoTitle,
			&row.faviconPath,
			&row.defaultPageMediaPath,
			&row.defaultPageMediaAltText,
			&row.defaultPageMediaTitle,
		)
	})
	if err != nil || !found {
		return data.SiteOptions{}, found, err
	}
	if row.id <= 0 {
		return data.SiteOptions{}, false, newRepositoryReadError(readSiteOptions, readInvalidIdentifier, nil)
	}
	if (selection == data.ActiveOptionSet && !row.active.Bool) || (selection == data.TestingOptionSet && !row.testing.Bool) {
		return data.SiteOptions{}, false, newRepositoryReadError(readSiteOptions, readSelectedRow, nil)
	}

	return data.SiteOptions{
		Set: data.OptionSetIdentity{
			ID:        strconv.FormatInt(row.id, 10),
			IsActive:  row.active.Bool,
			IsTesting: row.testing.Bool,
		},
		SiteName:                  row.siteName.String,
		SiteDescription:           row.siteDescription.String,
		MaintenanceMode:           row.maintenanceMode.Bool,
		Preloader:                 row.preloader.String,
		FacebookURL:               row.facebookURL.String,
		TwitterURL:                row.twitterURL.String,
		InstagramURL:              row.instagramURL.String,
		LinkedInURL:               row.linkedInURL.String,
		ContactEmail:              row.contactEmail.String,
		ContactPhone:              row.contactPhone.String,
		MainPageMetaTitle:         row.mainPageMetaTitle.String,
		MainPageMetaDescription:   row.mainPageMetaDescription.String,
		GoogleAnalytics:           row.googleAnalytics.String,
		PrimaryColor:              row.primaryColor.String,
		SecondaryColor:            row.secondaryColor.String,
		AccentColor:               row.accentColor.String,
		BackgroundColor:           row.backgroundColor.String,
		FontColor:                 row.fontColor.String,
		FontFamily:                row.fontFamily.String,
		ItemsPerPage:              row.itemsPerPage.Int64,
		EnableTestimonials:        row.enableTestimonials.Bool,
		MaximumSublinksOnMenuItem: row.maximumSublinksOnMenuItem.Int64,
		ShowDoctorSocialMedia:     row.showDoctorSocialMedia.Bool,
		ShowDoctorAppointmentFee:  row.showDoctorAppointmentFee.Bool,
		ShowPartnerPictures:       row.showPartnerPictures.Bool,
		RecaptchaSiteKey:          row.recaptchaSiteKey.String,
		SiteLogo: data.PublicMedia{
			Path: row.siteLogoPath.String, AltText: row.siteLogoAltText.String, Title: row.siteLogoTitle.String,
		},
		SiteLightLogo: data.PublicMedia{
			Path: row.siteLightLogoPath.String, AltText: row.siteLightLogoAltText.String, Title: row.siteLightLogoTitle.String,
		},
		Favicon: data.PublicMedia{Path: row.faviconPath.String},
		DefaultPageMedia: data.PublicMedia{
			Path: row.defaultPageMediaPath.String, AltText: row.defaultPageMediaAltText.String, Title: row.defaultPageMediaTitle.String,
		},
	}, true, nil
}

type optionIdentityRow struct {
	id      int64
	active  sql.NullBool
	testing sql.NullBool
}

func (r *OptionsRepository) ReadUploadPolicy(ctx context.Context) (data.UploadPolicy, bool, error) {
	return r.readUploadPolicy(ctx, nil)
}

// ReadUploadPolicyTx reads through the caller's active transaction, including
// when that transaction was opened by the legacy doctor workflow.
func (r *OptionsRepository) ReadUploadPolicyTx(ctx context.Context, tx *sql.Tx) (data.UploadPolicy, bool, error) {
	if tx == nil {
		return data.UploadPolicy{}, false, newRepositoryReadError(readUploadPolicy, readDependency, nil)
	}
	return r.readUploadPolicy(ctx, tx)
}

func (r *OptionsRepository) readUploadPolicy(ctx context.Context, tx *sql.Tx) (data.UploadPolicy, bool, error) {
	var identity optionIdentityRow
	var maxBytes sql.NullInt64
	scan := func(rows *sql.Rows) error {
		return rows.Scan(&identity.id, &identity.active, &identity.testing, &maxBytes)
	}
	var found bool
	var err error
	if tx != nil {
		found, err = r.readOneTx(ctx, tx, readUploadPolicy, readUploadPolicyTxSQL, scan)
	} else {
		found, err = r.readOne(ctx, readUploadPolicy, readUploadPolicySQL, scan)
	}
	if err != nil || !found {
		return data.UploadPolicy{}, found, err
	}
	// Both legacy AddDoctor and AddBranch transaction-bound upload checks accept
	// an active options row with oid=0. Keep that exception local to this read.
	if tx != nil && identity.id == 0 && identity.active.Bool {
		return data.UploadPolicy{
			Set:      data.OptionSetIdentity{ID: "0", IsActive: true, IsTesting: identity.testing.Bool},
			MaxBytes: maxBytes.Int64,
		}, true, nil
	}
	set, err := activeOptionIdentity(identity, readUploadPolicy)
	if err != nil {
		return data.UploadPolicy{}, false, err
	}
	return data.UploadPolicy{Set: set, MaxBytes: maxBytes.Int64}, true, nil
}

func (r *OptionsRepository) ReadPasswordPolicy(ctx context.Context) (data.PasswordPolicy, bool, error) {
	var identity optionIdentityRow
	var requireStrong sql.NullBool
	found, err := r.readOne(ctx, readPasswordPolicy, readPasswordPolicySQL, func(rows *sql.Rows) error {
		return rows.Scan(&identity.id, &identity.active, &identity.testing, &requireStrong)
	})
	if err != nil || !found {
		return data.PasswordPolicy{}, found, err
	}
	set, err := activeOptionIdentity(identity, readPasswordPolicy)
	if err != nil {
		return data.PasswordPolicy{}, false, err
	}
	return data.PasswordPolicy{Set: set, RequireStrong: requireStrong.Bool}, true, nil
}

func (r *OptionsRepository) ReadMailDeliveryOptions(ctx context.Context) (data.MailDeliveryOptions, bool, error) {
	var identity optionIdentityRow
	var host, username, password sql.NullString
	var port sql.NullInt64
	found, err := r.readOne(ctx, readMailDeliveryOptions, readMailDeliveryOptionsSQL, func(rows *sql.Rows) error {
		return rows.Scan(&identity.id, &identity.active, &identity.testing, &host, &port, &username, &password)
	})
	if err != nil || !found {
		return data.MailDeliveryOptions{}, found, err
	}
	set, err := activeOptionIdentity(identity, readMailDeliveryOptions)
	if err != nil {
		return data.MailDeliveryOptions{}, false, err
	}
	return data.MailDeliveryOptions{
		Set: set, Host: host.String, Port: port.Int64, Username: username.String, Password: password.String,
	}, true, nil
}

func (r *OptionsRepository) ReadCaptchaVerificationOptions(ctx context.Context) (data.CaptchaVerificationOptions, bool, error) {
	var identity optionIdentityRow
	var siteKey, secretKey sql.NullString
	found, err := r.readOne(ctx, readCaptchaVerificationOptions, readCaptchaVerificationOptionsSQL, func(rows *sql.Rows) error {
		return rows.Scan(&identity.id, &identity.active, &identity.testing, &siteKey, &secretKey)
	})
	if err != nil || !found {
		return data.CaptchaVerificationOptions{}, found, err
	}
	set, err := activeOptionIdentity(identity, readCaptchaVerificationOptions)
	if err != nil {
		return data.CaptchaVerificationOptions{}, false, err
	}
	return data.CaptchaVerificationOptions{Set: set, SiteKey: siteKey.String, SecretKey: secretKey.String}, true, nil
}

func (r *OptionsRepository) ReadOptionMediaReferences(ctx context.Context) (data.OptionMediaReferences, bool, error) {
	var identity optionIdentityRow
	var siteLogoID, siteLightLogoID, faviconID, defaultPageMediaID sql.NullInt64
	found, err := r.readOne(ctx, readOptionMediaReferences, readOptionMediaReferencesSQL, func(rows *sql.Rows) error {
		return rows.Scan(
			&identity.id, &identity.active, &identity.testing,
			&siteLogoID, &siteLightLogoID, &faviconID, &defaultPageMediaID,
		)
	})
	if err != nil || !found {
		return data.OptionMediaReferences{}, found, err
	}
	set, err := activeOptionIdentity(identity, readOptionMediaReferences)
	if err != nil {
		return data.OptionMediaReferences{}, false, err
	}
	references := data.OptionMediaReferences{Set: set}
	for source, target := range map[*sql.NullInt64]**string{
		&siteLogoID: &references.SiteLogoID, &siteLightLogoID: &references.SiteLightLogoID,
		&faviconID: &references.FaviconID, &defaultPageMediaID: &references.DefaultPageMediaID,
	} {
		if !source.Valid {
			continue
		}
		if source.Int64 <= 0 {
			return data.OptionMediaReferences{}, false, newRepositoryReadError(readOptionMediaReferences, readInvalidIdentifier, nil)
		}
		value := strconv.FormatInt(source.Int64, 10)
		*target = &value
	}
	return references, true, nil
}

func (r *OptionsRepository) readOne(ctx context.Context, operation readOperation, query string, scan func(*sql.Rows) error) (found bool, err error) {
	if r == nil || r.db == nil {
		return false, newRepositoryReadError(operation, readDependency, nil)
	}
	return readOneWithQueryer(ctx, r.db, operation, query, scan)
}

func (r *OptionsRepository) readOneTx(ctx context.Context, tx *sql.Tx, operation readOperation, query string, scan func(*sql.Rows) error) (found bool, err error) {
	if r == nil || r.db == nil || tx == nil {
		return false, newRepositoryReadError(operation, readDependency, nil)
	}
	return readOneWithQueryer(ctx, tx, operation, query, scan)
}

type optionsQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func readOneWithQueryer(ctx context.Context, queryer optionsQueryer, operation readOperation, query string, scan func(*sql.Rows) error) (found bool, err error) {
	rows, queryErr := queryer.QueryContext(ctx, query)
	if queryErr != nil {
		return false, newRepositoryReadError(operation, readQuery, queryErr)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			found = false
			err = newRepositoryReadError(operation, readRowsClose, closeErr)
		}
	}()

	if !rows.Next() {
		if rowsErr := rows.Err(); rowsErr != nil {
			return false, newRepositoryReadError(operation, readRowsClose, rowsErr)
		}
		return false, nil
	}
	if scanErr := scan(rows); scanErr != nil {
		return false, newRepositoryReadError(operation, readScan, scanErr)
	}
	if rows.Next() {
		if scanErr := scan(rows); scanErr != nil {
			return false, newRepositoryReadError(operation, readScan, scanErr)
		}
		return false, newRepositoryReadError(operation, readCardinality, nil)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return false, newRepositoryReadError(operation, readRowsClose, rowsErr)
	}
	return true, nil
}

func activeOptionIdentity(row optionIdentityRow, operation readOperation) (data.OptionSetIdentity, error) {
	if row.id <= 0 {
		return data.OptionSetIdentity{}, newRepositoryReadError(operation, readInvalidIdentifier, nil)
	}
	if !row.active.Bool {
		return data.OptionSetIdentity{}, newRepositoryReadError(operation, readSelectedRow, nil)
	}
	return data.OptionSetIdentity{
		ID: strconv.FormatInt(row.id, 10), IsActive: row.active.Bool, IsTesting: row.testing.Bool,
	}, nil
}
