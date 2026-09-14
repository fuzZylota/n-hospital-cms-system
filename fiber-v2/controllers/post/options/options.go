package options

import (
	"fmt"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"

	"database"

	"github.com/Necoo33/neormgo/v2"
	"github.com/gofiber/fiber/v2"
)

func AddOption(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		Orm := utilities.Orm

		// admin mi değil mi kontrol ediyoruz.
		if OurUser.Role != "admin" {
			checkIfUserIsAdmin := Orm.Count("users")
			checkIfUserIsAdmin.Where("uid", "=", OurUser.Uid)
			checkIfUserIsAdmin.And("role", "=", "admin")
			checkIfUserIsAdmin.Finish()

			err = checkIfUserIsAdmin.Execute()

			if err != nil {
				log.Printf("Cannot check if user is admin: %v\n", err)
				return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
			}

			if checkIfUserIsAdmin.Length() == 0 {
				log.Printf("User is not admin")
				return c.Redirect("/panel/secenek-ekle?error=only_admins_can_add_options")
			}

			return c.Redirect("/panel/secenek-ekle?error=only_admins_can_add_options")
		}

		inputs := models.Options{}
		c.BodyParser(&inputs)

		columns := []string{"option_set_is_active", "option_set_is_testing_now", "maintenance_mode", "require_strong_password",
			"enable_testimonials", "enable_our_history", "show_doctors_on_same_city", "show_doctors_on_same_country", "auto_remove_partners_when_expired",
			"show_doctor_social_media", "show_doctor_appointment_fee", "show_anlasmali_kurum_pictures"}
		values := []interface{}{inputs.OptionSetIsActive, inputs.OptionSetIsTestingNow, inputs.MaintenanceMode,
			inputs.RequireStrongPassword, inputs.EnableTestimonials, inputs.EnableOurHistory, inputs.ShowDoctorsOnSameCity,
			inputs.ShowDoctorsOnSameCountry, inputs.AutoRemovePartnersWhenExpired, inputs.ShowDoctorSocialMedia, inputs.ShowDoctorAppointmentFee, inputs.ShowAnlasmaliKurumPictures}

		{
			if inputs.OptionSetName != "" {
				columns = append(columns, "option_set_name")
				values = append(values, inputs.OptionSetName)
			}
			if inputs.OptionSetDescription != "" {
				columns = append(columns, "option_set_description")
				values = append(values, inputs.OptionSetDescription)
			}
			if inputs.SiteName != "" {
				columns = append(columns, "site_name")
				values = append(values, inputs.SiteName)
			}
			if inputs.SiteDescription != "" {
				columns = append(columns, "site_description")
				values = append(values, inputs.SiteDescription)
			}
			if inputs.SMTPHost != "" {
				columns = append(columns, "smtp_host")
				values = append(values, inputs.SMTPHost)
			}
			if inputs.SMTPPort != 0.0 {
				columns = append(columns, "smtp_port")
				values = append(values, inputs.SMTPPort)
			}
			if inputs.SMTPUsername != "" {
				columns = append(columns, "smtp_username")
				values = append(values, inputs.SMTPUsername)
			}
			if inputs.SMTPPassword != "" {
				columns = append(columns, "smtp_password")
				values = append(values, inputs.SMTPPassword)
			}
			if inputs.SMTPEncryption != "" {
				columns = append(columns, "smtp_encryption")
				values = append(values, inputs.SMTPEncryption)
			}
			if inputs.FacebookUrl != "" {
				columns = append(columns, "facebook_url")
				values = append(values, inputs.FacebookUrl)
			}
			if inputs.TwitterUrl != "" {
				columns = append(columns, "twitter_url")
				values = append(values, inputs.TwitterUrl)
			}
			if inputs.InstagramUrl != "" {
				columns = append(columns, "instagram_url")
				values = append(values, inputs.InstagramUrl)
			}
			if inputs.LinkedinUrl != "" {
				columns = append(columns, "linkedin_url")
				values = append(values, inputs.LinkedinUrl)
			}
			if inputs.ContactEmail != "" {
				columns = append(columns, "contact_email")
				values = append(values, inputs.ContactEmail)
			}
			if inputs.ContactPhone != "" {
				columns = append(columns, "contact_phone")
				values = append(values, inputs.ContactPhone)
			}
			if inputs.MainPageMetaTitle != "" {
				columns = append(columns, "main_page_meta_title")
				values = append(values, inputs.MainPageMetaTitle)
			}
			if inputs.MainPageMetaDescription != "" {
				columns = append(columns, "main_page_meta_description")
				values = append(values, inputs.MainPageMetaDescription)
			}
			if inputs.GoogleAnalytics != "" {
				columns = append(columns, "google_analytics")
				values = append(values, inputs.GoogleAnalytics)
			}
			if inputs.PrimaryColor != "" {
				columns = append(columns, "primary_color")
				values = append(values, inputs.PrimaryColor)
			}
			if inputs.SecondaryColor != "" {
				columns = append(columns, "secondary_color")
				values = append(values, inputs.SecondaryColor)
			}
			if inputs.AccentColor != "" {
				columns = append(columns, "accent_color")
				values = append(values, inputs.AccentColor)
			}
			if inputs.BackgroundColor != "" {
				columns = append(columns, "background_color")
				values = append(values, inputs.BackgroundColor)
			}
			if inputs.FontColor != "" {
				columns = append(columns, "font_color")
				values = append(values, inputs.FontColor)
			}
			if inputs.FontFamily != "" {
				columns = append(columns, "font_family")
				values = append(values, inputs.FontFamily)
			}
			if inputs.ItemsPerPage != 0.0 {
				columns = append(columns, "items_per_page")
				values = append(values, inputs.ItemsPerPage)
			}
			if inputs.MaximumSublinksOnAMenuItem != 0.0 {
				columns = append(columns, "maximum_sublinks_on_a_menu_item")
				values = append(values, inputs.MaximumSublinksOnAMenuItem)
			}
			if inputs.MaxUploadSize != 0.0 {
				columns = append(columns, "max_upload_size")
				values = append(values, inputs.MaxUploadSize)
			}
			if inputs.Timezone != "" {
				columns = append(columns, "timezone")
				values = append(values, inputs.Timezone)
			}
			if inputs.Language != "" {
				columns = append(columns, "language")
				values = append(values, inputs.Language)
			}
			if inputs.Preloader != "" {
				columns = append(columns, "preloader")
				values = append(values, inputs.Preloader)
			}
			if inputs.RecaptchaSiteKey != "" {
				columns = append(columns, "google_recaptcha_site_key")
				values = append(values, inputs.RecaptchaSiteKey)
			}
			if inputs.RecaptchaSecretKey != "" {
				columns = append(columns, "google_recaptcha_secret_key")
				values = append(values, inputs.RecaptchaSecretKey)
			}
		}

		if len(columns) == 0 {
			log.Printf("No inputs")
			return c.Redirect("/panel/secenek-ekle?error=no_inputs")
		}

		if len(values) == 0 {
			log.Printf("No inputs")
			return c.Redirect("/panel/secenek-ekle?error=no_inputs")
		}

		err = Orm.Begin()

		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
		}

		insertOption := Orm.Insert(columns, values)
		insertOption.Table("options")
		insertOption.Returning("oid")
		insertOption.Finish()
		err = insertOption.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert option: %v\n", err)
			return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
		}

		lid, err := insertOption.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
		}

		log.Printf("işte last insert id: %v\n", lid)

		if inputs.OptionSetIsActive {
			completeActivations := insertOption.Call("function", "set_active_option", "", lid)
			completeActivations.Finish()
			err = completeActivations.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute complete activations: %v\n", err)
				return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
			}
		}

		if inputs.OptionSetIsTestingNow {
			completeActivations := insertOption.Call("function", "set_testing_option", "", lid)
			completeActivations.Finish()
			err = completeActivations.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute complete testing activation: %v\n", err)
				return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
			}
		}

		if lid != "" {
			OurOptions := database.Options{}
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				log.Printf("Cannot fetch options for backend: %v\n", err)
				return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
			}

			GetOptions, err := OurOptions.FetchOptionsForBackend(&insertOption, []string{}, []string{})
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot fetch options for backend: %v\n", err)
				return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
			}

			var SiteLogoMid string = ""
			siteLogoInput, err := c.FormFile("site_logo_mid")
			if err == nil {
				// dosya yüklenmiş ve ulaşılabilir

				if siteLogoInput.Size > GetOptions.Options.MaxUploadSize {
					Orm.Rollback()
					log.Printf("File size is too large: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=file_size_is_too_large")
				}

				estimatedPath := filepath.Join(RootDir, "static", "files", "options", lid, "site_logo", "dark")

				UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + siteLogoInput.Filename)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get unique file path: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				switch UniqueFilePath.Extension {
				case ".jpg", ".jpeg", ".png", ".webp":
					break
				default:
					Orm.Rollback()
					log.Printf("Invalid file type: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=invalid_file_type")
				}

				media := models.Medias{
					FileName: UniqueFilePath.BaseName,
					FilePath: "files/options/" + lid + "/site_logo/dark/" + UniqueFilePath.BaseName,
					FileSize: siteLogoInput.Size,
					MimeType: siteLogoInput.Header.Get("Content-Type"),
					FileType: "site_logo",
					Uid:      OurUser.Uid,
					TargetId: lid,
				}

				optionals := models.MediaOptionals{
					AltText: inputs.SiteLogoAltText,
					Title:   inputs.SiteLogoTitle,
					Width:   0,
					Height:  0,
				}

				SiteLogoMid, err = OurOptions.InsertMedia(&insertOption, media, optionals)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot insert media: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				err = lib.SaveFileWithBufferingWithRenaming(estimatedPath, UniqueFilePath.BaseName, *siteLogoInput)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot save file with buffering: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}
			}

			var SiteLightLogoMid string = ""
			siteLightLogoInput, err := c.FormFile("site_light_logo_mid")
			if err == nil {
				// dosya yüklenmiş ve ulaşılabilir

				if siteLightLogoInput.Size > GetOptions.Options.MaxUploadSize {
					Orm.Rollback()
					log.Printf("File size is too large: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=file_size_is_too_large")
				}

				estimatedPath := filepath.Join(RootDir, "static", "files", "options", lid, "site_logo", "light")

				UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + siteLightLogoInput.Filename)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get unique file path: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				switch UniqueFilePath.Extension {
				case ".jpg", ".jpeg", ".png", ".webp":
					break
				default:
					Orm.Rollback()
					log.Printf("Invalid file type: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=invalid_file_type")
				}

				media := models.Medias{
					FileName: UniqueFilePath.BaseName,
					FilePath: "files/options/" + lid + "/site_logo/light/" + UniqueFilePath.BaseName,
					FileSize: siteLightLogoInput.Size,
					MimeType: siteLightLogoInput.Header.Get("Content-Type"),
					FileType: "site_light_logo",
					Uid:      OurUser.Uid,
					TargetId: lid,
				}

				optionals := models.MediaOptionals{
					AltText: inputs.SiteLightLogoAltText,
					Title:   inputs.SiteLightLogoTitle,
					Width:   0,
					Height:  0,
				}

				SiteLightLogoMid, err = OurOptions.InsertMedia(&insertOption, media, optionals)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot insert media: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				err = lib.SaveFileWithBufferingWithRenaming(estimatedPath, UniqueFilePath.BaseName, *siteLightLogoInput)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot save file with buffering: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}
			}

			var SiteFaviconMid string = ""
			siteFaviconInput, err := c.FormFile("site_favicon_mid")
			if err == nil {
				// dosya yüklenmiş ve ulaşılabilir
				if siteFaviconInput.Size > GetOptions.Options.MaxUploadSize {
					Orm.Rollback()
					log.Printf("File size is too large: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=file_size_is_too_large")
				}

				estimatedPath := filepath.Join(RootDir, "static", "files", "options", lid, "site_favicon")

				UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + siteFaviconInput.Filename)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get unique file path: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				switch UniqueFilePath.Extension {
				case ".jpg", ".jpeg", ".png", ".webp":
					break
				default:
					Orm.Rollback()
					log.Printf("Invalid file type: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=invalid_file_type")
				}

				media := models.Medias{
					FileName: UniqueFilePath.BaseName,
					FilePath: "files/options/" + lid + "/site_favicon/" + UniqueFilePath.BaseName,
					FileSize: siteFaviconInput.Size,
					MimeType: siteFaviconInput.Header.Get("Content-Type"),
					FileType: "site_favicon",
					Uid:      OurUser.Uid,
					TargetId: lid,
				}

				optionals := models.MediaOptionals{
					AltText: "",
					Title:   "",
					Width:   0,
					Height:  0,
				}

				SiteFaviconMid, err = OurOptions.InsertMedia(&insertOption, media, optionals)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot insert media: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				err = lib.SaveFileWithBuffering(estimatedPath, *siteFaviconInput)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot save file with buffering: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}
			}

			var DefaultPageMid string = ""
			defaultPageInput, err := c.FormFile("default_page_mid")
			if err == nil {
				// dosya yüklenmiş ve ulaşılabilir
				if defaultPageInput.Size > GetOptions.Options.MaxUploadSize {
					Orm.Rollback()
					log.Printf("File size is too large: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=file_size_is_too_large")
				}

				estimatedPath := filepath.Join(RootDir, "static", "files", "options", lid, "default_page")

				UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + defaultPageInput.Filename)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get unique file path: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				switch UniqueFilePath.Extension {
				case ".jpg", ".jpeg", ".png", ".webp":
					break
				default:
					Orm.Rollback()
					log.Printf("Invalid file type: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=invalid_file_type")
				}

				media := models.Medias{
					FileName: UniqueFilePath.BaseName,
					FilePath: "files/options/" + lid + "/default_page/" + UniqueFilePath.BaseName,
					FileSize: defaultPageInput.Size,
					MimeType: defaultPageInput.Header.Get("Content-Type"),
					FileType: "default_page_picture",
					Uid:      OurUser.Uid,
					TargetId: lid,
				}

				optionals := models.MediaOptionals{
					AltText: inputs.DefaultPageMediaAltText,
					Title:   inputs.DefaultPageMediaTitle,
					Width:   0,
					Height:  0,
				}

				DefaultPageMid, err = OurOptions.InsertMedia(&insertOption, media, optionals)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot insert media: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				err = lib.SaveFileWithBuffering(estimatedPath, *defaultPageInput)

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot save file with buffering: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}
			}

			if SiteLogoMid != "" || SiteLightLogoMid != "" || SiteFaviconMid != "" || DefaultPageMid != "" {
				updateOption := Orm.Update()
				updateOption.Table("options")

				if SiteLogoMid != "" {
					updateOption.Set("site_logo_mid", SiteLogoMid)
				}
				if SiteLightLogoMid != "" {
					updateOption.Set("site_light_logo_mid", SiteLightLogoMid)
				}
				if SiteFaviconMid != "" {
					updateOption.Set("site_favicon_mid", SiteFaviconMid)
				}
				if DefaultPageMid != "" {
					updateOption.Set("default_page_mid", DefaultPageMid)
				}

				updateOption.Where("oid", "=", lid)
				updateOption.Finish()
				err = updateOption.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update option: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				mediaUpdatesLid, err := updateOption.LastInsertId()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get last insert id: %v\n", err)
					return c.Redirect("/panel/secenek-ekle?error=internal_server_error")
				}

				if mediaUpdatesLid != "" {
					log.Printf("Media updates lid: %v\n", mediaUpdatesLid)
					return c.Redirect("/panel/secenekler/" + lid)
				}
			}
		}

		states.ActiveOptions = models.Options{}
		states.TestingOptions = models.Options{}
		states.Medias = []models.Medias{}
		Orm.Commit()

		return c.Redirect("/panel/secenekler/" + lid)
	}
}

func EditOption(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Oid := c.Params("oid")

		Orm := utilities.Orm

		// admin mi değil mi kontrol ediyoruz.
		if OurUser.Role != "admin" {
			checkIfUserIsAdmin := Orm.Count("users")
			checkIfUserIsAdmin.Where("uid", "=", OurUser.Uid)
			checkIfUserIsAdmin.And("role", "=", "admin")
			checkIfUserIsAdmin.Finish()

			err = checkIfUserIsAdmin.Execute()

			if err != nil {
				log.Printf("Cannot check if user is admin: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if checkIfUserIsAdmin.Length() == 0 {
				log.Printf("User is not admin")
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Only admins can add options",
				})
			}

			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can add options",
			})
		}

		inputs := models.OptionsEdit{}
		c.BodyParser(&inputs)
		inputs.Oid = lib.String(Oid)

		var OptionsAmount int64 = 0

		if inputs.OptionSetIsActive != inputs.OldOptionSetIsActive {
			checkIfOptionSetIsTheOnlyOptionSet := Orm.Count("options")
			checkIfOptionSetIsTheOnlyOptionSet.Finish()
			err = checkIfOptionSetIsTheOnlyOptionSet.Execute()
			if err != nil {
				log.Printf("Cannot check if option set is the only option set: %v\n", err)
			}

			OptionsAmount = checkIfOptionSetIsTheOnlyOptionSet.Length()

			if OptionsAmount == 0 {
				log.Printf("Option set is the only option set")
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "You cannot deactivate the only option set",
				})
			}
		}

		if (inputs.OptionSetIsTestingNow != inputs.OldOptionSetIsTestingNow) && OptionsAmount == 0 {
			checkIfOptionSetIsTheOnlyOptionSet := Orm.Count("options")
			checkIfOptionSetIsTheOnlyOptionSet.Finish()
			err = checkIfOptionSetIsTheOnlyOptionSet.Execute()
			if err != nil {
				log.Printf("Cannot check if option set is the only option set: %v\n", err)
			}

			OptionsAmount = checkIfOptionSetIsTheOnlyOptionSet.Length()

			if OptionsAmount == 0 {
				log.Printf("Option set is the only option set")
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "You cannot test the only option set",
				})
			}
		}

		Orm.Begin()

		updateOption := Orm.Update()
		updateOption.Table("options")

		SomethingSet := false

		{
			if inputs.OptionSetName != inputs.OldOptionSetName {
				updateOption.Set("option_set_name", inputs.OptionSetName)
				SomethingSet = true
			}

			if inputs.OptionSetDescription != inputs.OldOptionSetDescription {
				updateOption.Set("option_set_description", inputs.OptionSetDescription)
				SomethingSet = true
			}

			if inputs.SiteName != inputs.OldSiteName && inputs.SiteName != "" {
				updateOption.Set("site_name", inputs.SiteName)
				SomethingSet = true
			}

			if inputs.SiteDescription != inputs.OldSiteDescription {
				updateOption.Set("site_description", inputs.SiteDescription)
				SomethingSet = true
			}

			if inputs.MaintenanceMode != inputs.OldMaintenanceMode {
				updateOption.Set("maintenance_mode", inputs.MaintenanceMode)
				SomethingSet = true
			}

			if inputs.Preloader != inputs.OldPreloader {
				if inputs.Preloader == "" {
					updateOption.Set("preloader", nil)
				} else {
					updateOption.Set("preloader", inputs.Preloader)
				}

				SomethingSet = true
			}

			if inputs.SMTPHost != inputs.OldSMTPHost {
				updateOption.Set("smtp_host", inputs.SMTPHost)
				SomethingSet = true
			}

			if inputs.SMTPPort != inputs.OldSMTPPort {
				updateOption.Set("smtp_port", inputs.SMTPPort)
				SomethingSet = true
			}

			if inputs.SMTPUsername != inputs.OldSMTPUsername {
				updateOption.Set("smtp_username", inputs.SMTPUsername)
				SomethingSet = true
			}

			if inputs.SMTPPassword != inputs.OldSMTPPassword {
				updateOption.Set("smtp_password", inputs.SMTPPassword)
				SomethingSet = true
			}

			if inputs.SMTPEncryption != inputs.OldSMTPEncryption {
				updateOption.Set("smtp_encryption", inputs.SMTPEncryption)
				SomethingSet = true
			}

			if inputs.FacebookUrl != inputs.OldFacebookUrl {
				updateOption.Set("facebook_url", inputs.FacebookUrl)
				SomethingSet = true
			}

			if inputs.TwitterUrl != inputs.OldTwitterUrl {
				updateOption.Set("twitter_url", inputs.TwitterUrl)
				SomethingSet = true
			}

			if inputs.InstagramUrl != inputs.OldInstagramUrl {
				updateOption.Set("instagram_url", inputs.InstagramUrl)
				SomethingSet = true
			}

			if inputs.LinkedinUrl != inputs.OldLinkedinUrl {
				updateOption.Set("linkedin_url", inputs.LinkedinUrl)
				SomethingSet = true
			}

			if inputs.ContactEmail != inputs.OldContactEmail && inputs.ContactEmail != "" {
				updateOption.Set("contact_email", inputs.ContactEmail)
				SomethingSet = true
			}

			if inputs.ContactPhone != inputs.OldContactPhone && inputs.ContactPhone != "" {
				updateOption.Set("contact_phone", inputs.ContactPhone)
				SomethingSet = true
			}

			if inputs.MainPageMetaTitle != inputs.OldMainPageMetaTitle {
				updateOption.Set("main_page_meta_title", inputs.MainPageMetaTitle)
				SomethingSet = true
			}

			if inputs.MainPageMetaDescription != inputs.OldMainPageMetaDescription {
				updateOption.Set("main_page_meta_description", inputs.MainPageMetaDescription)
				SomethingSet = true
			}

			if inputs.GoogleAnalytics != inputs.OldGoogleAnalytics {
				updateOption.Set("google_analytics", inputs.GoogleAnalytics)
				SomethingSet = true
			}

			if inputs.PrimaryColor != inputs.OldPrimaryColor && inputs.PrimaryColor != "" {
				updateOption.Set("primary_color", inputs.PrimaryColor)
				SomethingSet = true
			}

			if inputs.SecondaryColor != inputs.OldSecondaryColor && inputs.SecondaryColor != "" {
				updateOption.Set("secondary_color", inputs.SecondaryColor)
				SomethingSet = true
			}

			if inputs.AccentColor != inputs.OldAccentColor && inputs.AccentColor != "" {
				updateOption.Set("accent_color", inputs.AccentColor)
				SomethingSet = true
			}

			if inputs.BackgroundColor != inputs.OldBackgroundColor && inputs.BackgroundColor != "" {
				updateOption.Set("background_color", inputs.BackgroundColor)
				SomethingSet = true
			}

			if inputs.FontColor != inputs.OldFontColor && inputs.FontColor != "" {
				updateOption.Set("font_color", inputs.FontColor)
				SomethingSet = true
			}

			if inputs.FontFamily != inputs.OldFontFamily && inputs.FontFamily != "" {
				updateOption.Set("font_family", inputs.FontFamily)
				SomethingSet = true
			}

			if inputs.RequireStrongPassword != inputs.OldRequireStrongPassword {
				updateOption.Set("require_strong_password", inputs.RequireStrongPassword)
				SomethingSet = true
			}

			if inputs.ItemsPerPage != inputs.OldItemsPerPage && inputs.ItemsPerPage != 0 {
				updateOption.Set("items_per_page", inputs.ItemsPerPage)
				SomethingSet = true
			}

			if inputs.ShowDoctorsOnSameCity != inputs.OldShowDoctorsOnSameCity {
				updateOption.Set("show_doctors_on_same_city", inputs.ShowDoctorsOnSameCity)
				SomethingSet = true
			}

			if inputs.ShowDoctorSocialMedia != inputs.OldShowDoctorSocialMedia {
				updateOption.Set("show_doctor_social_media", inputs.ShowDoctorSocialMedia)
				SomethingSet = true
			}

			if inputs.ShowDoctorAppointmentFee != inputs.OldShowDoctorAppointmentFee {
				updateOption.Set("show_doctor_appointment_fee", inputs.ShowDoctorAppointmentFee)
				SomethingSet = true
			}

			if inputs.ShowDoctorsOnSameCountry != inputs.OldShowDoctorsOnSameCountry {
				updateOption.Set("show_doctors_on_same_country", inputs.ShowDoctorsOnSameCountry)
				SomethingSet = true
			}

			if inputs.ShowAnlasmaliKurumPictures != inputs.OldShowAnlasmaliKurumPictures {
				updateOption.Set("show_anlasmali_kurum_pictures", inputs.ShowAnlasmaliKurumPictures)
				SomethingSet = true
			}

			if inputs.AutoRemovePartnersWhenExpired != inputs.OldAutoRemovePartnersWhenExpired {
				updateOption.Set("auto_remove_partners_when_expired", inputs.AutoRemovePartnersWhenExpired)
				SomethingSet = true
			}

			if inputs.EnableTestimonials != inputs.OldEnableTestimonials {
				updateOption.Set("enable_testimonials", inputs.EnableTestimonials)
				SomethingSet = true
			}

			if inputs.EnableOurHistory != inputs.OldEnableOurHistory {
				updateOption.Set("enable_our_history", inputs.EnableOurHistory)
				SomethingSet = true
			}

			if inputs.MaximumSublinksOnAMenuItem != inputs.OldMaximumSublinksOnAMenuItem && inputs.MaximumSublinksOnAMenuItem != 0 {
				updateOption.Set("maximum_sublinks_on_a_menu_item", inputs.MaximumSublinksOnAMenuItem)
				SomethingSet = true
			}

			if inputs.MaxUploadSize != inputs.OldMaxUploadSize && inputs.MaxUploadSize != 0 {
				updateOption.Set("max_upload_size", inputs.MaxUploadSize)
				SomethingSet = true
			}

			if inputs.Timezone != inputs.OldTimezone && inputs.Timezone != "" {
				updateOption.Set("timezone", inputs.Timezone)
				SomethingSet = true
			}

			if inputs.Language != inputs.OldLanguage && inputs.Language != "" {
				updateOption.Set("language", inputs.Language)
				SomethingSet = true
			}
			if inputs.RecaptchaSiteKey != inputs.OldRecaptchaSiteKey {
				updateOption.Set("google_recaptcha_site_key", inputs.RecaptchaSiteKey)
				SomethingSet = true
			}
			if inputs.RecaptchaSecretKey != inputs.OldRecaptchaSecretKey {
				updateOption.Set("google_recaptcha_secret_key", inputs.RecaptchaSecretKey)
				SomethingSet = true
			}
		}

		if SomethingSet {
			updateOption.Where("oid", "=", Oid)
			updateOption.Finish()

			fmt.Printf("işte updateOption.Query: %s\n", updateOption.Query)
			err = updateOption.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		} else {
			OneOfIsTrue := false

			if OptionsAmount > 1 && inputs.OptionSetIsActive != inputs.OldOptionSetIsActive {
				OneOfIsTrue = true
			}
			if OptionsAmount > 1 && inputs.OptionSetIsTestingNow != inputs.OldOptionSetIsTestingNow {
				OneOfIsTrue = true
			}

			if !OneOfIsTrue {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Nothing changed",
				})
			}
		}

		if OptionsAmount > 1 && inputs.OptionSetIsActive != inputs.OldOptionSetIsActive {
			var completeActivations neormgo.Neorm

			if inputs.OptionSetIsActive {
				completeActivations = updateOption.SelectFunction("set_active_option", Oid)
			} else {
				completeActivations = updateOption.SelectFunction("set_active_option", nil)
			}
			completeActivations.Finish()
			err = completeActivations.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute complete activations: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			rows, err := completeActivations.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if rows[0]["result"] == false {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "You cannot activate the only option set",
				})
			}
		}

		if (OptionsAmount > 1 && inputs.OptionSetIsTestingNow != inputs.OldOptionSetIsTestingNow) && (inputs.OptionSetIsActive == inputs.OldOptionSetIsActive) {
			var completeTestingActivations neormgo.Neorm

			if inputs.OptionSetIsTestingNow {
				completeTestingActivations = updateOption.SelectFunction("set_testing_option", Oid)
			} else {
				completeTestingActivations = updateOption.SelectFunction("set_testing_option", nil)
			}
			completeTestingActivations.Finish()
			err = completeTestingActivations.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot execute complete testing activation: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			rows, err := completeTestingActivations.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if rows[0]["result"] == false {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "You cannot test the only option set",
				})
			}

			fmt.Printf("Everything is OK!\n")
		}

		states.ActiveOptions = models.Options{}
		states.TestingOptions = models.Options{}
		states.Medias = []models.Medias{}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Option edited successfully",
		})
	}
}

func DeleteOption(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Oid := c.Params("oid")

		Orm := utilities.Orm

		// admin mi değil mi kontrol ediyoruz.
		if OurUser.Role != "admin" {
			checkIfUserIsAdmin := Orm.Count("users")
			checkIfUserIsAdmin.Where("uid", "=", OurUser.Uid)
			checkIfUserIsAdmin.And("role", "=", "admin")
			checkIfUserIsAdmin.Finish()

			err = checkIfUserIsAdmin.Execute()

			if err != nil {
				log.Printf("Cannot check if user is admin: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if checkIfUserIsAdmin.Length() == 0 {
				log.Printf("User is not admin")
				return c.JSON(fiber.Map{
					"status":  403,
					"message": "Only admins can add options",
				})
			}

			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Only admins can add options",
			})
		}

		CheckIfIsTheOnlyOptionSet := Orm.Count("options")
		CheckIfIsTheOnlyOptionSet.Finish()
		err = CheckIfIsTheOnlyOptionSet.Execute()
		if err != nil {
			log.Printf("Cannot check if option is the only option set: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if CheckIfIsTheOnlyOptionSet.Length() < 2 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "You cannot delete the only option set",
			})
		}

		CheckIfOptionSetIsActive := Orm.Count("options")
		CheckIfOptionSetIsActive.Where("oid", "=", Oid)
		CheckIfOptionSetIsActive.And("option_set_is_active", "=", true)
		CheckIfOptionSetIsActive.Finish()
		err = CheckIfOptionSetIsActive.Execute()
		if err != nil {
			log.Printf("Cannot check if option set is active: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if CheckIfOptionSetIsActive.Length() > 0 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "You cannot delete the active option set",
			})
		}

		Options := Orm.Select([]string{"o.oid", "m1.file_path as logo_path", "m2.file_path as favicon_path", "m3.file_path as default_page_path", "m4.file_path as light_logo_path"})
		Options.Table("options o")
		Options.LeftJoin("medias m1", "o.site_logo_mid", "=", "m1.mid")
		Options.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
		Options.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
		Options.LeftJoin("medias m4", "o.site_light_logo_mid", "=", "m4.mid")
		Options.Where("o.oid", "=", Oid)
		Options.Finish()

		err = Options.Execute()

		if err != nil {
			log.Printf("Cannot fetch options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, err := Options.Rows()
		if err != nil {
			log.Printf("Cannot get rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		Orm.Begin()

		if len(rows) > 0 && (rows[0]["logo_path"] != "" || rows[0]["favicon_path"] != "" || rows[0]["default_page_path"] != "" || rows[0]["light_logo_path"] != "") {
			valuesArray := []any{}

			if lib.String(rows[0]["logo_path"]) != "" {
				valuesArray = append(valuesArray, lib.String(rows[0]["logo_path"]))
			}
			if lib.String(rows[0]["favicon_path"]) != "" {
				valuesArray = append(valuesArray, lib.String(rows[0]["favicon_path"]))
			}
			if lib.String(rows[0]["default_page_path"]) != "" {
				valuesArray = append(valuesArray, lib.String(rows[0]["default_page_path"]))
			}
			if lib.String(rows[0]["light_logo_path"]) != "" {
				valuesArray = append(valuesArray, lib.String(rows[0]["light_logo_path"]))
			}

			if len(valuesArray) > 0 {
				DeleteMedia := Orm.Delete()
				DeleteMedia.Table("medias")
				DeleteMedia.In("WHERE", "file_path", valuesArray)
				DeleteMedia.Finish()

				err = DeleteMedia.Execute()

				if err != nil {
					Orm.Rollback()
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Internal server error",
					})
				}

				ra, err := DeleteMedia.RowsAffected()
				if err != nil {
					Orm.Rollback()
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Internal server error",
					})
				}

				if ra == 0 {
					Orm.Rollback()
					return c.JSON(fiber.Map{
						"status":  400,
						"message": "Media not found",
					})
				}
			}
		}

		DeleteOption := Orm.Delete()
		DeleteOption.Table("options")
		DeleteOption.Where("oid", "=", Oid)
		DeleteOption.Finish()
		err = DeleteOption.Execute()
		if err != nil {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		ra, err := DeleteOption.RowsAffected()
		if err != nil {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Option not found",
			})
		}

		if rows[0]["logo_path"] != nil && rows[0]["logo_path"] != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			Path := filepath.Join(RootDir, "static", lib.String(rows[0]["logo_path"]))

			err = lib.DeleteFile(Path)

			if err != nil {
				log.Printf("Cannot delete file: %v\n", err)
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		if rows[0]["light_logo_path"] != nil && rows[0]["light_logo_path"] != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			Path := filepath.Join(RootDir, "static", lib.String(rows[0]["light_logo_path"]))

			err = lib.DeleteFile(Path)

			if err != nil {
				log.Printf("Cannot delete file: %v\n", err)
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		if rows[0]["favicon_path"] != nil && rows[0]["favicon_path"] != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			Path := filepath.Join(RootDir, "static", lib.String(rows[0]["favicon_path"]))

			err = lib.DeleteFile(Path)

			if err != nil {
				log.Printf("Cannot delete file: %v\n", err)
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		if rows[0]["default_page_path"] != nil && rows[0]["default_page_path"] != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			Path := filepath.Join(RootDir, "static", lib.String(rows[0]["default_page_path"]))

			err = lib.DeleteFile(Path)

			if err != nil {
				log.Printf("Cannot delete file: %v\n", err)
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}
		}

		states.ActiveOptions = models.Options{}
		states.TestingOptions = models.Options{}
		states.Medias = []models.Medias{}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Option deleted successfully",
		})
	}
}

func DeleteOptionMedia(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		fmt.Printf("%s", string(c.Body()))

		inputs := models.DeleteOptionMediaInputs{}
		Oid := c.Params("oid")
		err = c.BodyParser(&inputs)

		if err != nil {
			log.Printf("Cannot parse body: %v %T\n", err, err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		Orm := *utilities.Orm

		Orm.Begin()

		GetFilePath := Orm.Select([]string{"file_path"})
		GetFilePath.Table("medias")
		GetFilePath.Where("target_id", "=", Oid)
		GetFilePath.And("file_type", "=", inputs.MediaType)
		GetFilePath.Finish()

		err = GetFilePath.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get file path: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		rows, _ := GetFilePath.Rows()

		if len(rows) == 0 {
			Orm.Rollback()
			log.Printf("Media not found: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Media not found",
			})
		}

		FilePath := lib.String(rows[0]["file_path"])

		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("file_path", "=", FilePath)
		DeleteMedia.And("target_id", "=", Oid)
		DeleteMedia.And("file_type", "=", inputs.MediaType)
		DeleteMedia.Finish()

		err = DeleteMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		ra, err := DeleteMedia.RowsAffected()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get rows affected: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			log.Printf("Cannot delete media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Media not found",
			})
		}

		RemovePictureFromOption := Orm.Update()
		RemovePictureFromOption.Table("options")

		switch inputs.MediaType {
		case "site_logo":
			RemovePictureFromOption.Set("site_logo_mid", nil)
		case "site_light_logo":
			RemovePictureFromOption.Set("site_light_logo_mid", nil)
		case "site_favicon":
			RemovePictureFromOption.Set("site_favicon_mid", nil)
		case "default_page_picture":
			RemovePictureFromOption.Set("default_page_mid", nil)
		}

		RemovePictureFromOption.Where("oid", "=", Oid)
		RemovePictureFromOption.Finish()

		err = RemovePictureFromOption.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot remove picture from option: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			log.Printf("Cannot remove picture from option: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Media not found",
			})
		}

		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			Orm.Rollback()
			log.Printf("Cannot get root directory: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		FilePath = filepath.Join(RootDir, "static", FilePath)

		err = lib.DeleteFile(FilePath)

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete file: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		states.ActiveOptions = models.Options{}
		states.TestingOptions = models.Options{}
		states.Medias = []models.Medias{}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Option deleted successfully",
		})
	}
}

func UpdateOptionMedia(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		siteLogoAltText := c.FormValue("site_logo_alt_text")
		siteLogoTitle := c.FormValue("site_logo_title")
		oldSiteLogoAltText := c.FormValue("old_site_logo_alt_text")
		oldSiteLogoTitle := c.FormValue("old_site_logo_title")
		siteLightLogoAltText := c.FormValue("site_light_logo_alt_text")
		siteLightLogoTitle := c.FormValue("site_light_logo_title")
		oldSiteLightLogoAltText := c.FormValue("old_site_light_logo_alt_text")
		oldSiteLightLogoTitle := c.FormValue("old_site_light_logo_title")
		defaultPageMediaAltText := c.FormValue("default_page_media_alt_text")
		defaultPageMediaTitle := c.FormValue("default_page_media_title")
		oldDefaultPageMediaAltText := c.FormValue("old_default_page_media_alt_text")
		oldDefaultPageMediaTitle := c.FormValue("old_default_page_media_title")

		Oid := c.Params("oid")

		OurOptions := database.Options{}
		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		Orm := *utilities.Orm

		GetOptions, err := OurOptions.FetchOptionsForBackend(&Orm, []string{"o.site_logo_mid", "o.site_light_logo_mid", "o.site_favicon_mid", "o.default_page_mid"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Internal server error",
			})
		}

		siteLogoInput, err := c.FormFile("site_logo_path")
		if err == nil {
			Orm.Begin()

			if siteLogoInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "File size is too large",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "options", Oid, "site_logo", "dark")

			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + siteLogoInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Invalid file type",
				})
			}

			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/options/" + Oid + "/site_logo/dark/" + UniqueFilePath.BaseName,
				FileSize: siteLogoInput.Size,
				MimeType: siteLogoInput.Header.Get("Content-Type"),
				FileType: "site_logo",
				Uid:      OurUser.Uid,
				TargetId: Oid,
			}

			optionals := models.MediaOptionals{
				AltText: siteLogoAltText,
				Title:   siteLogoTitle,
				Width:   0,
				Height:  0,
			}

			NewOpts := database.Options{}

			SiteLogoMid, err := NewOpts.InsertMedia(&Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if SiteLogoMid == "" {
				Orm.Rollback()
				log.Printf("cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			entries, err := lib.ReadDirectory(estimatedPath)
			if err != nil && err != os.ErrNotExist {
				log.Printf("Cannot read directory: %v, %T\n", err, err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			FilePaths := []any{}

			for _, entry := range entries {
				FilePath := filepath.Join("files/options/"+Oid+"/site_logo", entry.Name())
				fmt.Printf("FilePath: %v\n", FilePath)
				FilePaths = append(FilePaths, FilePath)
			}

			if len(FilePaths) > 0 {
				DeleteMedias := Orm.Delete()
				DeleteMedias.Table("medias")
				DeleteMedias.In("WHERE", "file_path", FilePaths)
				DeleteMedias.Finish()
				err = DeleteMedias.Execute()
				if err != nil {
					Orm.Rollback()
				}
			}

			UpdateOptions := Orm.Update()
			UpdateOptions.Table("options")
			UpdateOptions.Set("site_logo_mid", SiteLogoMid)
			UpdateOptions.Where("oid", "=", Oid)
			UpdateOptions.Finish()
			err = UpdateOptions.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			ra, err := UpdateOptions.RowsAffected()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows affected: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			err = lib.SaveFileWithBufferingWithRenaming(estimatedPath, UniqueFilePath.BaseName, *siteLogoInput)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if err != nil {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			for _, filePath := range FilePaths {
				GetAbsolutePath := filepath.Join(RootDir, "static", filePath.(string))
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove file: %v\n", err)
				}
			}

			states.ActiveOptions = models.Options{}
			states.TestingOptions = models.Options{}
			states.Medias = []models.Medias{}

			Orm.Commit()
		} else {
			if GetOptions.Options.SiteLogoMid != 0 && (siteLogoAltText != oldSiteLogoAltText || siteLogoTitle != oldSiteLogoTitle) {
				UpdateMedia := Orm.Update()
				UpdateMedia.Table("medias")

				if siteLogoAltText != oldSiteLogoAltText {
					if siteLogoAltText != "" {
						UpdateMedia.Set("alt_text", siteLogoAltText)
					} else {
						UpdateMedia.Set("alt_text", nil)
					}
				}

				if siteLogoTitle != oldSiteLogoTitle {
					if siteLogoTitle != "" {
						UpdateMedia.Set("title", siteLogoTitle)
					} else {
						UpdateMedia.Set("title", nil)
					}
				}

				UpdateMedia.Where("mid", "=", GetOptions.Options.SiteLogoMid)
				UpdateMedia.And("target_id", "=", Oid)
				UpdateMedia.Finish()

				err = UpdateMedia.Execute()

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Internal server error",
					})
				}

				states.ActiveOptions = models.Options{}
				states.TestingOptions = models.Options{}
				states.Medias = []models.Medias{}
			}
		}

		siteLightLogoInput, err := c.FormFile("site_light_logo_path")
		if err == nil {
			Orm.Begin()

			if siteLightLogoInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "File size is too large",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "options", Oid, "site_logo", "light")

			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + siteLightLogoInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Invalid file type",
				})
			}

			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/options/" + Oid + "/site_logo/light/" + UniqueFilePath.BaseName,
				FileSize: siteLightLogoInput.Size,
				MimeType: siteLightLogoInput.Header.Get("Content-Type"),
				FileType: "site_light_logo",
				Uid:      OurUser.Uid,
				TargetId: Oid,
			}

			optionals := models.MediaOptionals{
				AltText: siteLightLogoAltText,
				Title:   siteLightLogoTitle,
				Width:   0,
				Height:  0,
			}

			NewOpts := database.Options{}

			SiteLightLogoMid, err := NewOpts.InsertMedia(&Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if SiteLightLogoMid == "" {
				Orm.Rollback()
				log.Printf("cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			entries, err := lib.ReadDirectory(estimatedPath)
			if err != nil && err != os.ErrNotExist {
				log.Printf("Cannot read directory: %v, %T\n", err, err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			FilePaths := []any{}

			for _, entry := range entries {
				FilePath := filepath.Join("files/options/"+Oid+"/site_logo/light", entry.Name())
				fmt.Printf("FilePath: %v\n", FilePath)
				FilePaths = append(FilePaths, FilePath)
			}

			if len(FilePaths) > 0 {
				DeleteMedias := Orm.Delete()
				DeleteMedias.Table("medias")
				DeleteMedias.In("WHERE", "file_path", FilePaths)
				DeleteMedias.Finish()
				err = DeleteMedias.Execute()
				if err != nil {
					Orm.Rollback()
				}
			}

			UpdateOptions := Orm.Update()
			UpdateOptions.Table("options")
			UpdateOptions.Set("site_light_logo_mid", SiteLightLogoMid)
			UpdateOptions.Where("oid", "=", Oid)
			UpdateOptions.Finish()
			err = UpdateOptions.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			ra, err := UpdateOptions.RowsAffected()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows affected: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			err = lib.SaveFileWithBufferingWithRenaming(estimatedPath, UniqueFilePath.BaseName, *siteLightLogoInput)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if err != nil {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			for _, filePath := range FilePaths {
				GetAbsolutePath := filepath.Join(RootDir, "static", lib.String(filePath))
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove file: %v\n", err)
				}
			}

			Orm.Commit()

			states.ActiveOptions = models.Options{}
			states.TestingOptions = models.Options{}
			states.Medias = []models.Medias{}
		} else {
			if GetOptions.Options.SiteLightLogoMid != 0 && (siteLightLogoAltText != oldSiteLightLogoAltText || siteLightLogoTitle != oldSiteLightLogoTitle) {
				UpdateMedia := Orm.Update()
				UpdateMedia.Table("medias")

				if siteLightLogoAltText != oldSiteLightLogoAltText {
					if siteLightLogoAltText != "" {
						UpdateMedia.Set("alt_text", siteLightLogoAltText)
					} else {
						UpdateMedia.Set("alt_text", nil)
					}
				}

				if siteLightLogoTitle != oldSiteLightLogoTitle {
					if siteLightLogoTitle != "" {
						UpdateMedia.Set("title", siteLightLogoTitle)
					} else {
						UpdateMedia.Set("title", nil)
					}
				}

				UpdateMedia.Where("mid", "=", GetOptions.Options.SiteLightLogoMid)
				UpdateMedia.And("target_id", "=", Oid)
				UpdateMedia.Finish()

				err = UpdateMedia.Execute()

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Internal server error",
					})
				}

				states.ActiveOptions = models.Options{}
				states.TestingOptions = models.Options{}
				states.Medias = []models.Medias{}
			}
		}

		siteFaviconInput, err := c.FormFile("site_favicon_path")
		if err == nil {
			Orm.Begin()

			if siteFaviconInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "File size is too large",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "options", Oid, "site_favicon")

			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + siteFaviconInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Invalid file type",
				})
			}

			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/options/" + Oid + "/site_favicon/" + UniqueFilePath.BaseName,
				FileSize: siteFaviconInput.Size,
				MimeType: siteFaviconInput.Header.Get("Content-Type"),
				FileType: "site_favicon",
				Uid:      OurUser.Uid,
				TargetId: Oid,
			}

			optionals := models.MediaOptionals{
				AltText: "",
				Title:   "",
				Width:   0,
				Height:  0,
			}

			NewOpts := database.Options{}

			SiteFaviconMid, err := NewOpts.InsertMedia(&Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if SiteFaviconMid == "" {
				Orm.Rollback()
				log.Printf("Site favicon mid is not empty: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			entries, err := lib.ReadDirectory(estimatedPath)
			if err != nil && err != os.ErrNotExist {
				log.Printf("Cannot read directory: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			FilePaths := []any{}

			for _, entry := range entries {
				FilePath := filepath.Join("files/options/"+Oid+"/site_favicon", entry.Name())
				FilePaths = append(FilePaths, FilePath)
			}

			if len(FilePaths) > 0 {
				DeleteMedias := Orm.Delete()
				DeleteMedias.Table("medias")
				DeleteMedias.In("WHERE", "file_path", FilePaths)
				DeleteMedias.Finish()
				err = DeleteMedias.Execute()
				if err != nil {
					log.Printf("Cannot delete media: %v\n", err)
					Orm.Rollback()
				}
			}

			UpdateOptions := Orm.Update()
			UpdateOptions.Table("options")
			UpdateOptions.Set("site_favicon_mid", SiteFaviconMid)
			UpdateOptions.Where("oid", "=", Oid)
			UpdateOptions.Finish()
			err = UpdateOptions.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			ra, err := UpdateOptions.RowsAffected()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows affected: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			err = lib.SaveFileWithBufferingWithRenaming(estimatedPath, UniqueFilePath.BaseName, *siteFaviconInput)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			for _, filePath := range FilePaths {
				GetAbsolutePath := filepath.Join(RootDir, "static", filePath.(string))
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove file: %v\n", err)
				}
			}

			Orm.Commit()

			states.ActiveOptions = models.Options{}
			states.TestingOptions = models.Options{}
			states.Medias = []models.Medias{}
		}

		defaultPageInput, err := c.FormFile("default_page_media_path")
		if err == nil {
			Orm.Begin()

			if defaultPageInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "File size is too large",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "options", Oid, "default_page")

			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + defaultPageInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				log.Printf("Cannot read directory: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Invalid file type",
				})
			}

			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/options/" + Oid + "/default_page/" + UniqueFilePath.BaseName,
				FileSize: defaultPageInput.Size,
				MimeType: defaultPageInput.Header.Get("Content-Type"),
				FileType: "default_page_picture",
				TargetId: Oid,
				Uid:      OurUser.Uid,
			}

			optionals := models.MediaOptionals{
				AltText: defaultPageMediaAltText,
				Title:   defaultPageMediaTitle,
				Width:   0,
				Height:  0,
			}

			NewOpts := database.Options{}

			DefaultPageMid, err := NewOpts.InsertMedia(&Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if DefaultPageMid == "" {
				Orm.Rollback()
				log.Printf("Default page mid is not empty: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			entries, err := lib.ReadDirectory(estimatedPath)
			if err != nil && err != os.ErrNotExist {
				log.Printf("Cannot read directory: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			FilePaths := []any{}

			for _, entry := range entries {
				FilePath := filepath.Join("files/options/"+Oid+"/default_page", entry.Name())
				FilePaths = append(FilePaths, FilePath)
			}

			if len(FilePaths) > 0 {
				DeleteMedias := Orm.Delete()
				DeleteMedias.Table("medias")
				DeleteMedias.In("WHERE", "file_path", FilePaths)
				DeleteMedias.Finish()
				err = DeleteMedias.Execute()
				if err != nil {
					Orm.Rollback()
				}
			}

			UpdateOptions := Orm.Update()
			UpdateOptions.Table("options")
			UpdateOptions.Set("default_page_mid", DefaultPageMid)
			UpdateOptions.Where("oid", "=", Oid)
			UpdateOptions.Finish()
			err = UpdateOptions.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			ra, err := UpdateOptions.RowsAffected()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows affected: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				log.Printf("Cannot update option: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			err = lib.SaveFileWithBufferingWithRenaming(estimatedPath, UniqueFilePath.BaseName, *defaultPageInput)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			if err != nil {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Internal server error",
				})
			}

			for _, filePath := range FilePaths {
				GetAbsolutePath := filepath.Join(RootDir, "static", filePath.(string))
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove file: %v\n", err)
				}
			}

			Orm.Commit()

			states.ActiveOptions = models.Options{}
			states.TestingOptions = models.Options{}
			states.Medias = []models.Medias{}
		} else {
			if GetOptions.Options.DefaultPageMid != 0 && (defaultPageMediaAltText != oldDefaultPageMediaAltText || defaultPageMediaTitle != oldDefaultPageMediaTitle) {
				UpdateMedia := Orm.Update()
				UpdateMedia.Table("medias")

				if defaultPageMediaAltText != oldDefaultPageMediaAltText {
					if defaultPageMediaAltText != "" {
						UpdateMedia.Set("alt_text", defaultPageMediaAltText)
					} else {
						UpdateMedia.Set("alt_text", nil)
					}
				}

				if defaultPageMediaTitle != oldDefaultPageMediaTitle {
					if defaultPageMediaTitle != "" {
						UpdateMedia.Set("title", defaultPageMediaTitle)
					} else {
						UpdateMedia.Set("title", nil)
					}
				}

				UpdateMedia.Where("mid", "=", GetOptions.Options.DefaultPageMid)
				UpdateMedia.And("target_id", "=", Oid)
				UpdateMedia.Finish()

				err = UpdateMedia.Execute()

				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Internal server error",
					})
				}

				states.ActiveOptions = models.Options{}
				states.TestingOptions = models.Options{}
				states.Medias = []models.Medias{}
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Option medias updated successfully",
		})
	}
}

