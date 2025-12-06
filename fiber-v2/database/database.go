package database

import (
	"errors"
	"fmt"
	lib "lib"
	"log"
	"os"
	"strings"

	"models"

	orm "github.com/Necoo33/neormgo/v2"
)

func Database(connString string) orm.Neorm {
	orm := orm.Neorm{}

	database, err := orm.Connect(connString, "postgres")

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return database
}

type Options struct {
	Options            *models.Options
	Medias             *[]models.Medias
	HeaderButtons      *[]models.HeaderButton
	SubelerLinks       *[]models.SubeLink
	TibbiBirimlerLinks *[]models.TibbiBirimLink
	TedkiklerLinks     *[]models.TedkikLink
	NewsLinks          *[]models.NewsLink
	Notifications      *[]models.Notification
}

func (Optionss *Options) FetchOptionsForFrontendWithCache(CurrentOptions *models.FrontendOptions) (Options, error) {
	var ActiveOptions models.Options
	var ActiveMedias []models.Medias
	var TestingOptions models.Options
	var TestingMedias []models.Medias

	// check if active option set or testing option set is empty
	ActiveOptionSetOrTestingOptionSetIsEmpty := CurrentOptions.States.ActiveOptions.Oid == "" || CurrentOptions.States.TestingOptions.Oid == ""
	// check if there is no authentication and we have active option set
	NoNeedToFetchOptions := CurrentOptions.States.ActiveOptions.Oid != "" && CurrentOptions.User.Uid == ""

	if ActiveOptionSetOrTestingOptionSetIsEmpty {
		if NoNeedToFetchOptions {
			ActiveOptions = CurrentOptions.States.ActiveOptions
			TestingOptions = CurrentOptions.States.TestingOptions
			ActiveMedias = CurrentOptions.States.Medias
			TestingMedias = CurrentOptions.States.Medias
		} else {
			Columns := []string{
				"o.*",
				"m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title", "m2.file_path as favicon_path",
				"m3.file_path as default_page_media_path", "m4.file_path as light_logo_path", "m4.alt_text as light_logo_alt_text",
				"m4.title as light_logo_title", "m3.alt_text as default_page_media_alt_text", "m3.title as default_page_media_title",
			}

			opts := CurrentOptions.Database.Select(Columns)
			opts.Table("options o")
			opts.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
			opts.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
			opts.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
			opts.LeftJoin("medias m4", "o.site_light_logo_mid", "=", "m4.mid")
			opts.Where("o.option_set_is_active", "=", true)

			if CurrentOptions.States.TestingOptions == (models.Options{}) && CurrentOptions.User.Uid != "" {
				opts.Or("o.option_set_is_testing_now", "=", true)
			}

			opts.Finish()
			err := opts.Execute()

			if err != nil {
				return Options{}, err
			}

			rows, err := opts.Rows()

			if err != nil {
				return Options{}, err
			}

			for i, _ := range rows {
				if lib.Bool(rows[i]["option_set_is_testing_now"]) {
					TestingOptions = models.Options{
						Oid:                           lib.String(rows[i]["oid"]),
						OptionSetName:                 lib.String(rows[i]["option_set_name"]),
						OptionSetDescription:          lib.String(rows[i]["option_set_description"]),
						OptionSetIsActive:             lib.Bool(rows[i]["option_set_is_active"]),
						OptionSetCreatedAt:            lib.Time(rows[i]["option_set_created_at"]),
						OptionSetUpdatedAt:            lib.Time(rows[i]["option_set_updated_at"]),
						SiteName:                      lib.String(rows[i]["site_name"]),
						SiteDescription:               lib.String(rows[i]["site_description"]),
						SiteLogoMid:                   lib.Int64(rows[i]["site_logo_mid"]),
						SiteLightLogoMid:              lib.Int64(rows[i]["site_light_logo_mid"]),
						FaviconMid:                    lib.Int64(rows[i]["site_favicon_mid"]),
						DefaultPageMid:                lib.Int64(rows[i]["default_page_mid"]),
						MaintenanceMode:               lib.Bool(rows[i]["maintenance_mode"]),
						Preloader:                     lib.String(rows[i]["preloader"]),
						SMTPHost:                      lib.String(rows[i]["smtp_host"]),
						SMTPPort:                      lib.Int64(rows[i]["smtp_port"]),
						SMTPUsername:                  lib.String(rows[i]["smtp_username"]),
						SMTPPassword:                  lib.String(rows[i]["smtp_password"]),
						SMTPEncryption:                lib.String(rows[i]["smtp_encryption"]),
						FacebookUrl:                   lib.String(rows[i]["facebook_url"]),
						TwitterUrl:                    lib.String(rows[i]["twitter_url"]),
						InstagramUrl:                  lib.String(rows[i]["instagram_url"]),
						LinkedinUrl:                   lib.String(rows[i]["linkedin_url"]),
						ContactEmail:                  lib.String(rows[i]["contact_email"]),
						ContactPhone:                  lib.String(rows[i]["contact_phone"]),
						MainPageMetaTitle:             lib.String(rows[i]["main_page_meta_title"]),
						MainPageMetaDescription:       lib.String(rows[i]["main_page_meta_description"]),
						GoogleAnalytics:               lib.String(rows[i]["google_analytics"]),
						PrimaryColor:                  lib.String(rows[i]["primary_color"]),
						SecondaryColor:                lib.String(rows[i]["secondary_color"]),
						AccentColor:                   lib.String(rows[i]["accent_color"]),
						BackgroundColor:               lib.String(rows[i]["background_color"]),
						FontColor:                     lib.String(rows[i]["font_color"]),
						FontFamily:                    lib.String(rows[i]["font_family"]),
						RequireStrongPassword:         lib.Bool(rows[i]["require_strong_password"]),
						ItemsPerPage:                  lib.Int64(rows[i]["items_per_page"]),
						ShowDoctorsOnSameCity:         lib.Bool(rows[i]["show_doctors_on_same_city"]),
						ShowDoctorsOnSameCountry:      lib.Bool(rows[i]["show_doctors_on_same_country"]),
						AutoRemovePartnersWhenExpired: lib.Bool(rows[i]["auto_remove_partners_when_expired"]),
						MaxUploadSize:                 lib.Int64(rows[i]["max_upload_size"]),
						Timezone:                      lib.String(rows[i]["timezone"]),
						Language:                      lib.String(rows[i]["language"]),
						EnableTestimonials:            lib.Bool(rows[i]["enable_testimonials"]),
						EnableOurHistory:              lib.Bool(rows[i]["enable_our_history"]),
						MaximumSublinksOnAMenuItem:    lib.Int64(rows[i]["maximum_sublinks_on_a_menu_item"]),
						ShowDoctorSocialMedia:         lib.Bool(rows[0]["show_doctor_social_media"]),
						ShowDoctorAppointmentFee:      lib.Bool(rows[i]["show_doctor_appointment_fee"]),
						DefaultPageMediaPath:          lib.String(rows[i]["default_page_media_path"]),
						DefaultPageMediaAltText:       lib.String(rows[i]["default_page_media_alt_text"]),
						DefaultPageMediaTitle:         lib.String(rows[i]["default_page_media_title"]),
						RecaptchaSiteKey:              lib.String(rows[i]["google_recaptcha_site_key"]),
						RecaptchaSecretKey:            lib.String(rows[i]["google_recaptcha_secret_key"]),
						ShowAnlasmaliKurumPictures:    lib.Bool(rows[i]["show_anlasmali_kurum_pictures"]),
					}

					TestingMedias = []models.Medias{
						{
							FilePath: lib.String(rows[i]["logo_path"]),
							AltText:  lib.String(rows[i]["logo_alt_text"]),
							Title:    lib.String(rows[i]["logo_title"]),
						},
						{
							FilePath: lib.String(rows[i]["light_logo_path"]),
							AltText:  lib.String(rows[i]["light_logo_alt_text"]),
							Title:    lib.String(rows[i]["light_logo_title"]),
						},
						{
							FilePath: lib.String(rows[i]["favicon_path"]),
						},
						{
							FilePath: lib.String(rows[i]["default_page_media_path"]),
							AltText:  lib.String(rows[i]["default_page_media_alt_text"]),
							Title:    lib.String(rows[i]["default_page_media_title"]),
						},
					}
				}

				if lib.Bool(rows[i]["option_set_is_active"]) {
					ActiveOptions = models.Options{
						Oid:                           lib.String(rows[i]["oid"]),
						OptionSetName:                 lib.String(rows[i]["option_set_name"]),
						OptionSetDescription:          lib.String(rows[i]["option_set_description"]),
						OptionSetIsActive:             lib.Bool(rows[i]["option_set_is_active"]),
						OptionSetCreatedAt:            lib.Time(rows[i]["option_set_created_at"]),
						OptionSetUpdatedAt:            lib.Time(rows[i]["option_set_updated_at"]),
						SiteName:                      lib.String(rows[i]["site_name"]),
						SiteDescription:               lib.String(rows[i]["site_description"]),
						SiteLogoMid:                   lib.Int64(rows[i]["site_logo_mid"]),
						SiteLightLogoMid:              lib.Int64(rows[i]["site_light_logo_mid"]),
						FaviconMid:                    lib.Int64(rows[i]["site_favicon_mid"]),
						DefaultPageMid:                lib.Int64(rows[i]["default_page_mid"]),
						MaintenanceMode:               lib.Bool(rows[i]["maintenance_mode"]),
						Preloader:                     lib.String(rows[i]["preloader"]),
						SMTPHost:                      lib.String(rows[i]["smtp_host"]),
						SMTPPort:                      lib.Int64(rows[i]["smtp_port"]),
						SMTPUsername:                  lib.String(rows[i]["smtp_username"]),
						SMTPPassword:                  lib.String(rows[i]["smtp_password"]),
						SMTPEncryption:                lib.String(rows[i]["smtp_encryption"]),
						FacebookUrl:                   lib.String(rows[i]["facebook_url"]),
						TwitterUrl:                    lib.String(rows[i]["twitter_url"]),
						InstagramUrl:                  lib.String(rows[i]["instagram_url"]),
						LinkedinUrl:                   lib.String(rows[i]["linkedin_url"]),
						ContactEmail:                  lib.String(rows[i]["contact_email"]),
						ContactPhone:                  lib.String(rows[i]["contact_phone"]),
						MainPageMetaTitle:             lib.String(rows[i]["main_page_meta_title"]),
						MainPageMetaDescription:       lib.String(rows[i]["main_page_meta_description"]),
						GoogleAnalytics:               lib.String(rows[i]["google_analytics"]),
						PrimaryColor:                  lib.String(rows[i]["primary_color"]),
						SecondaryColor:                lib.String(rows[i]["secondary_color"]),
						AccentColor:                   lib.String(rows[i]["accent_color"]),
						BackgroundColor:               lib.String(rows[i]["background_color"]),
						FontColor:                     lib.String(rows[i]["font_color"]),
						FontFamily:                    lib.String(rows[i]["font_family"]),
						RequireStrongPassword:         lib.Bool(rows[i]["require_strong_password"]),
						ItemsPerPage:                  lib.Int64(rows[i]["items_per_page"]),
						ShowDoctorsOnSameCity:         lib.Bool(rows[i]["show_doctors_on_same_city"]),
						ShowDoctorsOnSameCountry:      lib.Bool(rows[i]["show_doctors_on_same_country"]),
						AutoRemovePartnersWhenExpired: lib.Bool(rows[i]["auto_remove_partners_when_expired"]),
						MaxUploadSize:                 lib.Int64(rows[i]["max_upload_size"]),
						Timezone:                      lib.String(rows[i]["timezone"]),
						Language:                      lib.String(rows[i]["language"]),
						EnableTestimonials:            lib.Bool(rows[i]["enable_testimonials"]),
						EnableOurHistory:              lib.Bool(rows[i]["enable_our_history"]),
						MaximumSublinksOnAMenuItem:    lib.Int64(rows[i]["maximum_sublinks_on_a_menu_item"]),
						ShowDoctorSocialMedia:         lib.Bool(rows[i]["show_doctor_social_media"]),
						ShowDoctorAppointmentFee:      lib.Bool(rows[i]["show_doctor_appointment_fee"]),
						DefaultPageMediaPath:          lib.String(rows[i]["default_page_media_path"]),
						DefaultPageMediaAltText:       lib.String(rows[i]["default_page_media_alt_text"]),
						DefaultPageMediaTitle:         lib.String(rows[i]["default_page_media_title"]),
						RecaptchaSiteKey:              lib.String(rows[i]["google_recaptcha_site_key"]),
						RecaptchaSecretKey:            lib.String(rows[i]["google_recaptcha_secret_key"]),
						ShowAnlasmaliKurumPictures:    lib.Bool(rows[i]["show_anlasmali_kurum_pictures"]),
					}

					ActiveMedias = []models.Medias{
						{
							FilePath: lib.String(rows[i]["logo_path"]),
							AltText:  lib.String(rows[i]["logo_alt_text"]),
							Title:    lib.String(rows[i]["logo_title"]),
						},
						{
							FilePath: lib.String(rows[i]["light_logo_path"]),
							AltText:  lib.String(rows[i]["light_logo_alt_text"]),
							Title:    lib.String(rows[i]["light_logo_title"]),
						},
						{
							FilePath: lib.String(rows[i]["favicon_path"]),
						},
						{
							FilePath: lib.String(rows[i]["default_page_media_path"]),
							AltText:  lib.String(rows[i]["default_page_media_alt_text"]),
							Title:    lib.String(rows[i]["default_page_media_title"]),
						},
					}
				}
			}
		}
	} else {
		ActiveOptions = CurrentOptions.States.ActiveOptions
		ActiveMedias = CurrentOptions.States.Medias
		TestingOptions = CurrentOptions.States.TestingOptions
		TestingMedias = CurrentOptions.States.Medias
	}

	Options := Options{
		HeaderButtons: &[]models.HeaderButton{},
		NewsLinks:     &[]models.NewsLink{},
	}

	if CurrentOptions.User.Uid != "" && TestingOptions != (models.Options{}) {
		Options.Options = &TestingOptions
		CurrentOptions.States.TestingOptions = TestingOptions
		Options.Medias = &TestingMedias
		CurrentOptions.States.Medias = TestingMedias
	} else {
		Options.Options = &ActiveOptions
		CurrentOptions.States.ActiveOptions = ActiveOptions
		Options.Medias = &ActiveMedias
		CurrentOptions.States.Medias = ActiveMedias
	}

	HeaderButtons := &[]models.HeaderButton{}

	if len(CurrentOptions.States.HeaderButtons) == 0 {
		SelectHeaderButtons := CurrentOptions.Database.Select([]string{"title", "url", "target", "icon", "sort_order", "parent_id", "button_type"})
		SelectHeaderButtons.Table("header_buttons")
		SelectHeaderButtons.Where("is_active", "=", true)
		SelectHeaderButtons.OrderBy("sort_order", "ASC")
		SelectHeaderButtons.Finish()
		err := SelectHeaderButtons.Execute()
		if err != nil {
			return Options, err
		}

		rows, err := SelectHeaderButtons.Rows()
		if err != nil {
			return Options, err
		}

		for _, row := range rows {
			*HeaderButtons = append(*HeaderButtons, models.HeaderButton{
				Title:      lib.String(row["title"]),
				Url:        lib.String(row["url"]),
				Target:     lib.String(row["target"]),
				Icon:       lib.String(row["icon"]),
				SortOrder:  lib.Int64(row["sort_order"]),
				ParentId:   lib.String(row["parent_id"]),
				ButtonType: lib.String(row["button_type"]),
			})
		}

		Options.HeaderButtons = HeaderButtons
		CurrentOptions.States.HeaderButtons = *HeaderButtons
	} else {
		Options.HeaderButtons = &CurrentOptions.States.HeaderButtons
	}

	if len(CurrentOptions.States.NewsLinks) == 0 {
		NewsLinks := []models.NewsLink{}
		GetNewsLinks := CurrentOptions.Database.Select([]string{"h.hid", "h.title", "h.url_name", "h.author", "h.cover_mid", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetNewsLinks.Table("haberler h")
		GetNewsLinks.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")
		GetNewsLinks.Where("h.is_published", "=", true)
		GetNewsLinks.OrderBy("h.hid", "DESC")
		GetNewsLinks.Limit(2)
		GetNewsLinks.Finish()

		err := GetNewsLinks.Execute()

		if err != nil {
			return Options, err
		}

		rows, err := GetNewsLinks.Rows()
		if err != nil {
			return Options, err
		}

		for _, row := range rows {
			NewsLinks = append(NewsLinks, models.NewsLink{
				Hid:          lib.String(row["hid"]),
				Title:        lib.String(row["title"]),
				UrlName:      lib.String(row["url_name"]),
				Author:       lib.String(row["author"]),
				CoverMid:     lib.Int64(row["cover_mid"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
			})
		}

		Options.NewsLinks = &NewsLinks
		CurrentOptions.States.NewsLinks = NewsLinks
	} else {
		Options.NewsLinks = &CurrentOptions.States.NewsLinks
	}

	if len(CurrentOptions.States.SubelerLinks) == 0 {
		GetSubeler := CurrentOptions.Database.Select([]string{"sid", "name", "url_name", "document_mids"})
		GetSubeler.Table("subeler")
		GetSubeler.Where("is_active", "=", true)
		GetSubeler.OrderBy("sid", "ASC")
		GetSubeler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
		GetSubeler.Finish()
		err := GetSubeler.Execute()
		if err != nil {
			return Options, err
		}

		rows, err := GetSubeler.Rows()
		if err != nil {
			return Options, err
		}

		SubelerLinks := []models.SubeLink{}

		DocumentMids := []any{}
		for _, row := range rows {
			SubelerLinks = append(SubelerLinks, models.SubeLink{
				Sid:       lib.String(row["sid"]),
				Name:      lib.String(row["name"]),
				UrlName:   lib.String(row["url_name"]),
				Documents: []models.Medias{},
				//DoktorLinks: []models.DoktorLink{},
			})

			ConvertDocumentMid := row["document_mids"].(string)
			if ConvertDocumentMid != "{}" {
				ReplaceDocumentMid := strings.ReplaceAll(ConvertDocumentMid, "{", "")
				ReplaceDocumentMid = strings.ReplaceAll(ReplaceDocumentMid, "}", "")
				SplitTheDocumentMids := strings.SplitSeq(ReplaceDocumentMid, ",")
				for mid := range SplitTheDocumentMids {
					DocumentMids = append(DocumentMids, mid)
				}
			}
		}

		if len(DocumentMids) > 0 {
			GetSubeDocuments := CurrentOptions.Database.Select([]string{"mid", "file_path", "data", "target_id"})
			GetSubeDocuments.Table("medias")
			GetSubeDocuments.In("WHERE", "mid", DocumentMids)
			GetSubeDocuments.Limit(int(Options.Options.MaximumSublinksOnAMenuItem - 2))
			GetSubeDocuments.Finish()
			err := GetSubeDocuments.Execute()
			if err != nil {
				return Options, err
			}

			rows, err := GetSubeDocuments.Rows()
			if err != nil {
				return Options, err
			}

			for i := range SubelerLinks {
				for _, row := range rows {
					if SubelerLinks[i].Sid == lib.String(row["target_id"]) {
						Media := models.Medias{
							Mid:      lib.Int64(row["mid"]),
							FilePath: lib.String(row["file_path"]),
							Data:     lib.String(row["data"]),
						}

						SubelerLinks[i].Documents = append(SubelerLinks[i].Documents, Media)
					}
				}
			}

		}

		Options.SubelerLinks = &SubelerLinks
		CurrentOptions.States.SubelerLinks = SubelerLinks
	} else {
		Options.SubelerLinks = &CurrentOptions.States.SubelerLinks
	}

	if len(CurrentOptions.States.TibbiBirimlerLinks) == 0 {
		GetTibbiBirimler := CurrentOptions.Database.Select([]string{"tb.tbid", "tb.name", "tb.url_name"})
		GetTibbiBirimler.Table("tibbi_birimler tb")
		GetTibbiBirimler.Where("is_active", "=", true)
		GetTibbiBirimler.OrderBy("tbid", "ASC")
		GetTibbiBirimler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
		GetTibbiBirimler.Finish()

		err := GetTibbiBirimler.Execute()

		if err != nil {
			return Options, err
		}

		rows, err := GetTibbiBirimler.Rows()

		if err != nil {
			return Options, err
		}

		TibbiBirimlerLinks := []models.TibbiBirimLink{}

		for _, row := range rows {
			TibbiBirimlerLinks = append(TibbiBirimlerLinks, models.TibbiBirimLink{
				Tbid:    lib.String(row["tbid"]),
				Name:    lib.String(row["name"]),
				UrlName: lib.String(row["url_name"]),
			})
		}

		Options.TibbiBirimlerLinks = &TibbiBirimlerLinks
		CurrentOptions.States.TibbiBirimlerLinks = TibbiBirimlerLinks
	} else {
		Options.TibbiBirimlerLinks = &CurrentOptions.States.TibbiBirimlerLinks
	}

	if len(CurrentOptions.States.TedkiklerLinks) == 0 {
		GetTedkikler := CurrentOptions.Database.Select([]string{"tid", "name", "url_name"})
		GetTedkikler.Table("tedkikler")
		GetTedkikler.Where("is_active", "=", true)
		GetTedkikler.OrderBy("tid", "ASC")
		GetTedkikler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
		GetTedkikler.Finish()
		err := GetTedkikler.Execute()
		if err != nil {
			return Options, err
		}
		rows, err := GetTedkikler.Rows()
		if err != nil {
			return Options, err
		}

		TedkiklerLinks := []models.TedkikLink{}

		for _, row := range rows {
			TedkiklerLinks = append(TedkiklerLinks, models.TedkikLink{
				Tid:     lib.String(row["tid"]),
				Name:    lib.String(row["name"]),
				UrlName: lib.String(row["url_name"]),
			})
		}

		Options.TedkiklerLinks = &TedkiklerLinks
		CurrentOptions.States.TedkiklerLinks = TedkiklerLinks
	} else {
		Options.TedkiklerLinks = &CurrentOptions.States.TedkiklerLinks
	}

	return Options, nil
}

func (options *Options) FetchOptionsForFrontend(db *orm.Neorm, otherColumns any, unwantedColumns any, user models.AuthenticatedUser) (Options, error) {
	var columns []string

	switch otherColumns.(type) {
	case []string:
		columns = otherColumns.([]string)
	default:
		if len(otherColumns.([]string)) > 0 {
			fmt.Println("otherColumns is not a []string")
			return Options{}, errors.New("otherColumns is not a []string")
		}
	}

	switch unwantedColumns.(type) {
	case []string:
		columns = unwantedColumns.([]string)
	default:
		if len(unwantedColumns.([]string)) > 0 {
			fmt.Println("unwantedColumns is not a []string")
			return Options{}, errors.New("unwantedColumns is not a []string")
		}
		return Options{}, errors.New("unwantedColumns is not a []string")
	}

	columns = append(columns, []string{
		"o.oid", "o.site_name", "o.site_description", "m.file_path as logo_path", "m.alt_text as logo_alt_text",
		"m.title as logo_title", "m2.file_path as favicon_path", "m3.file_path as default_page_media_path",
		"m4.file_path as light_logo_path", "m4.alt_text as light_logo_alt_text", "m4.title as light_logo_title",
		"m3.alt_text as default_page_media_alt_text", "m3.title as default_page_media_title",
		"o.default_page_mid", "o.maintenance_mode", "o.preloader", "o.facebook_url", "o.twitter_url", "o.instagram_url",
		"o.linkedin_url", "o.contact_email", "o.contact_phone", "o.primary_color", "o.secondary_color", "o.accent_color",
		"o.background_color", "o.font_color", "o.font_family", "o.maximum_sublinks_on_a_menu_item", "o.google_analytics",
	}...)

	if len(otherColumns.([]string)) > 0 {
		columns = append(columns, otherColumns.([]string)...)
	}

	if len(unwantedColumns.([]string)) > 0 {
		newColumns := []string{}

		for _, column := range unwantedColumns.([]string) {
			for _, c := range columns {
				if c == column {
					continue
				}

				newColumns = append(newColumns, c)
			}

		}

		columns = newColumns
	}

	opts := db.Select(columns)
	opts.Table("options o")
	opts.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
	opts.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
	opts.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
	opts.LeftJoin("medias m4", "o.site_light_logo_mid", "=", "m4.mid")
	if user.Uid != "" {
		opts.Where("o.option_set_is_testing_now", "=", true)
	} else {
		opts.Where("o.option_set_is_active", "=", true)
	}
	opts.Finish()
	err := opts.Execute()

	if err != nil {
		return Options{}, err
	}

	rows, err := opts.Rows()

	if err != nil {
		return Options{}, err
	}

	if len(rows) == 0 {
		if user.Uid != "" {
			opts = db.Select(columns)
			opts.Table("options o")
			opts.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
			opts.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
			opts.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
			opts.LeftJoin("medias m4", "o.site_light_logo_mid", "=", "m4.mid")
			opts.Where("o.option_set_is_active", "=", true)

			opts.Finish()

			err = opts.Execute()

			if err != nil {
				return Options{}, err
			}

			rows, err = opts.Rows()

			if err != nil {
				return Options{}, err
			}

			if len(rows) == 0 {
				return Options{}, errors.New("no options found for active mode")
			}
		} else {
			return Options{}, errors.New("no options found for active mode")
		}
	}

	newOptions := models.Options{
		Oid:                           lib.String(rows[0]["oid"]),
		OptionSetName:                 lib.String(rows[0]["option_set_name"]),
		OptionSetDescription:          lib.String(rows[0]["option_set_description"]),
		OptionSetIsActive:             lib.Bool(rows[0]["option_set_is_active"]),
		OptionSetCreatedAt:            lib.Time(rows[0]["option_set_created_at"]),
		OptionSetUpdatedAt:            lib.Time(rows[0]["option_set_updated_at"]),
		SiteName:                      lib.String(rows[0]["site_name"]),
		SiteDescription:               lib.String(rows[0]["site_description"]),
		SiteLogoMid:                   lib.Int64(rows[0]["site_logo_mid"]),
		SiteLightLogoMid:              lib.Int64(rows[0]["site_light_logo_mid"]),
		FaviconMid:                    lib.Int64(rows[0]["site_favicon_mid"]),
		DefaultPageMid:                lib.Int64(rows[0]["default_page_mid"]),
		MaintenanceMode:               lib.Bool(rows[0]["maintenance_mode"]),
		Preloader:                     lib.String(rows[0]["preloader"]),
		SMTPHost:                      lib.String(rows[0]["smtp_host"]),
		SMTPPort:                      lib.Int64(rows[0]["smtp_port"]),
		SMTPUsername:                  lib.String(rows[0]["smtp_username"]),
		SMTPPassword:                  lib.String(rows[0]["smtp_password"]),
		SMTPEncryption:                lib.String(rows[0]["smtp_encryption"]),
		FacebookUrl:                   lib.String(rows[0]["facebook_url"]),
		TwitterUrl:                    lib.String(rows[0]["twitter_url"]),
		InstagramUrl:                  lib.String(rows[0]["instagram_url"]),
		LinkedinUrl:                   lib.String(rows[0]["linkedin_url"]),
		ContactEmail:                  lib.String(rows[0]["contact_email"]),
		ContactPhone:                  lib.String(rows[0]["contact_phone"]),
		MainPageMetaTitle:             lib.String(rows[0]["main_page_meta_title"]),
		MainPageMetaDescription:       lib.String(rows[0]["main_page_meta_description"]),
		GoogleAnalytics:               lib.String(rows[0]["google_analytics"]),
		PrimaryColor:                  lib.String(rows[0]["primary_color"]),
		SecondaryColor:                lib.String(rows[0]["secondary_color"]),
		AccentColor:                   lib.String(rows[0]["accent_color"]),
		BackgroundColor:               lib.String(rows[0]["background_color"]),
		FontColor:                     lib.String(rows[0]["font_color"]),
		FontFamily:                    lib.String(rows[0]["font_family"]),
		RequireStrongPassword:         lib.Bool(rows[0]["require_strong_password"]),
		ItemsPerPage:                  lib.Int64(rows[0]["items_per_page"]),
		ShowDoctorsOnSameCity:         lib.Bool(rows[0]["show_doctors_on_same_city"]),
		ShowDoctorsOnSameCountry:      lib.Bool(rows[0]["show_doctors_on_same_country"]),
		AutoRemovePartnersWhenExpired: lib.Bool(rows[0]["auto_remove_partners_when_expired"]),
		MaxUploadSize:                 lib.Int64(rows[0]["max_upload_size"]),
		Timezone:                      lib.String(rows[0]["timezone"]),
		Language:                      lib.String(rows[0]["language"]),
		EnableTestimonials:            lib.Bool(rows[0]["enable_testimonials"]),
		EnableOurHistory:              lib.Bool(rows[0]["enable_our_history"]),
		MaximumSublinksOnAMenuItem:    lib.Int64(rows[0]["maximum_sublinks_on_a_menu_item"]),
		ShowDoctorSocialMedia:         lib.Bool(rows[0]["show_doctor_social_media"]),
		ShowDoctorAppointmentFee:      lib.Bool(rows[0]["show_doctor_appointment_fee"]),
		DefaultPageMediaPath:          lib.String(rows[0]["default_page_media_path"]),
		DefaultPageMediaAltText:       lib.String(rows[0]["default_page_media_alt_text"]),
		DefaultPageMediaTitle:         lib.String(rows[0]["default_page_media_title"]),
		ShowAnlasmaliKurumPictures:    lib.Bool(rows[0]["show_anlasmali_kurum_pictures"]),
	}

	Options := Options{
		Options: &newOptions,
		Medias: &[]models.Medias{
			{
				FilePath: lib.String(rows[0]["logo_path"]),
				AltText:  lib.String(rows[0]["logo_alt_text"]),
				Title:    lib.String(rows[0]["logo_title"]),
			},
			{
				FilePath: lib.String(rows[0]["light_logo_path"]),
				AltText:  lib.String(rows[0]["light_logo_alt_text"]),
				Title:    lib.String(rows[0]["light_logo_title"]),
			},
			{
				FilePath: lib.String(rows[0]["favicon_path"]),
			},
			{
				FilePath: lib.String(rows[0]["default_page_path"]),
				AltText:  lib.String(rows[0]["default_page_alt_text"]),
				Title:    lib.String(rows[0]["default_page_title"]),
			},
		},
		HeaderButtons: &[]models.HeaderButton{},
		NewsLinks:     &[]models.NewsLink{},
	}

	headerButtons := []models.HeaderButton{}

	SelectHeaderButtons := db.Select([]string{"title", "url", "target", "icon", "sort_order", "parent_id", "button_type"})
	SelectHeaderButtons.Table("header_buttons")
	SelectHeaderButtons.Where("is_active", "=", true)
	SelectHeaderButtons.OrderBy("sort_order", "ASC")
	SelectHeaderButtons.Finish()
	err = SelectHeaderButtons.Execute()
	if err != nil {
		return Options, err
	}

	rows, err = SelectHeaderButtons.Rows()
	if err != nil {
		return Options, err
	}

	for _, row := range rows {
		headerButtons = append(headerButtons, models.HeaderButton{
			Title:      lib.String(row["title"]),
			Url:        lib.String(row["url"]),
			Target:     lib.String(row["target"]),
			Icon:       lib.String(row["icon"]),
			SortOrder:  lib.Int64(row["sort_order"]),
			ParentId:   lib.String(row["parent_id"]),
			ButtonType: lib.String(row["button_type"]),
		})
	}

	Options.HeaderButtons = &headerButtons

	NewsLinks := []models.NewsLink{}
	GetNewsLinks := db.Select([]string{"h.hid", "h.title", "h.url_name", "h.author", "h.cover_mid", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
	GetNewsLinks.Table("haberler h")
	GetNewsLinks.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")
	GetNewsLinks.Where("h.is_published", "=", true)
	GetNewsLinks.OrderBy("h.hid", "DESC")
	GetNewsLinks.Limit(2)
	GetNewsLinks.Finish()

	err = GetNewsLinks.Execute()

	if err != nil {
		return Options, err
	}

	rows, err = GetNewsLinks.Rows()
	if err != nil {
		return Options, err
	}

	for _, row := range rows {
		NewsLinks = append(NewsLinks, models.NewsLink{
			Hid:          lib.String(row["hid"]),
			Title:        lib.String(row["title"]),
			UrlName:      lib.String(row["url_name"]),
			Author:       lib.String(row["author"]),
			CoverMid:     lib.Int64(row["cover_mid"]),
			CoverPath:    lib.String(row["cover_path"]),
			CoverAltText: lib.String(row["cover_alt_text"]),
			CoverTitle:   lib.String(row["cover_title"]),
		})
	}

	Options.NewsLinks = &NewsLinks

	GetSubeler := db.Select([]string{"sid", "name", "url_name", "document_mids"})
	GetSubeler.Table("subeler")
	GetSubeler.Where("is_active", "=", true)
	GetSubeler.OrderBy("sid", "ASC")
	GetSubeler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
	GetSubeler.Finish()
	err = GetSubeler.Execute()
	if err != nil {
		return Options, err
	}

	rows, err = GetSubeler.Rows()
	if err != nil {
		return Options, err
	}

	SubelerLinks := []models.SubeLink{}

	DocumentMids := []any{}
	for _, row := range rows {
		SubelerLinks = append(SubelerLinks, models.SubeLink{
			Sid:       lib.String(row["sid"]),
			Name:      lib.String(row["name"]),
			UrlName:   lib.String(row["url_name"]),
			Documents: []models.Medias{},
			//DoktorLinks: []models.DoktorLink{},
		})

		ConvertDocumentMid := row["document_mids"].(string)
		if ConvertDocumentMid != "{}" {
			ReplaceDocumentMid := strings.ReplaceAll(ConvertDocumentMid, "{", "")
			ReplaceDocumentMid = strings.ReplaceAll(ReplaceDocumentMid, "}", "")
			SplitTheDocumentMids := strings.SplitSeq(ReplaceDocumentMid, ",")
			for mid := range SplitTheDocumentMids {
				DocumentMids = append(DocumentMids, mid)
			}
		}
	}

	if len(DocumentMids) > 0 {
		GetSubeDocuments := db.Select([]string{"mid", "file_path", "data", "target_id"})
		GetSubeDocuments.Table("medias")
		GetSubeDocuments.In("WHERE", "mid", DocumentMids)
		GetSubeDocuments.Limit(int(Options.Options.MaximumSublinksOnAMenuItem - 2))
		GetSubeDocuments.Finish()
		err = GetSubeDocuments.Execute()
		if err != nil {
			return Options, err
		}

		rows, err = GetSubeDocuments.Rows()
		if err != nil {
			return Options, err
		}

		for i := range SubelerLinks {
			for _, row := range rows {
				if SubelerLinks[i].Sid == lib.String(row["target_id"]) {
					Media := models.Medias{
						Mid:      lib.Int64(row["mid"]),
						FilePath: lib.String(row["file_path"]),
						Data:     lib.String(row["data"]),
					}

					SubelerLinks[i].Documents = append(SubelerLinks[i].Documents, Media)
				}
			}
		}

	}

	/*SubeSids := []any{}
	for _, subeLink := range subelerLinks {
		SubeSids = append(SubeSids, subeLink.Sid)
	}

	GetDoktorlar := db.Select([]string{"drid", "title", "first_name", "last_name", "url_name", "sid"})
	GetDoktorlar.Table("doktorlar")
	GetDoktorlar.In("WHERE", "sid", SubeSids)
	GetDoktorlar.Finish()
	err = GetDoktorlar.Execute()

	if err != nil {
		return Options, err
	}

	rows, err = GetDoktorlar.Rows()
	if err != nil {
		return Options, err
	}

	for _, row := range rows {
		for _, subeLink := range subelerLinks {
			if subeLink.Sid == lib.String(row["sid"]) {
				subeLink.DoktorLinks = append(subeLink.DoktorLinks, models.DoktorLink{
					Drid:      lib.String(row["drid"]),
					Sid:       lib.String(row["sid"]),
					Title:     lib.String(row["title"]),
					FirstName: lib.String(row["first_name"]),
					LastName:  lib.String(row["last_name"]),
					UrlName:   lib.String(row["url_name"]),
				})
			}
		}
	}*/

	Options.SubelerLinks = &SubelerLinks

	GetTibbiBirimler := db.Select([]string{"tb.tbid", "tb.name", "tb.url_name"})
	GetTibbiBirimler.Table("tibbi_birimler tb")
	GetTibbiBirimler.Where("is_active", "=", true)
	GetTibbiBirimler.OrderBy("tbid", "ASC")
	GetTibbiBirimler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
	GetTibbiBirimler.Finish()

	err = GetTibbiBirimler.Execute()

	if err != nil {
		return Options, err
	}

	rows, err = GetTibbiBirimler.Rows()

	if err != nil {
		return Options, err
	}

	TibbiBirimlerLinks := []models.TibbiBirimLink{}

	for _, row := range rows {
		TibbiBirimlerLinks = append(TibbiBirimlerLinks, models.TibbiBirimLink{
			Tbid:    lib.String(row["tbid"]),
			Name:    lib.String(row["name"]),
			UrlName: lib.String(row["url_name"]),
		})
	}

	Options.TibbiBirimlerLinks = &TibbiBirimlerLinks

	GetTedkikler := db.Select([]string{"tid", "name", "url_name"})
	GetTedkikler.Table("tedkikler")
	GetTedkikler.Where("is_active", "=", true)
	GetTedkikler.OrderBy("tid", "ASC")
	GetTedkikler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
	GetTedkikler.Finish()
	err = GetTedkikler.Execute()
	if err != nil {
		return Options, err
	}
	rows, err = GetTedkikler.Rows()
	if err != nil {
		return Options, err
	}

	TedkiklerLinks := []models.TedkikLink{}

	for _, row := range rows {
		TedkiklerLinks = append(TedkiklerLinks, models.TedkikLink{
			Tid:     lib.String(row["tid"]),
			Name:    lib.String(row["name"]),
			UrlName: lib.String(row["url_name"]),
		})
	}

	Options.TedkiklerLinks = &TedkiklerLinks

	return Options, nil
}

func (options *Options) FetchOptionsForBackend(db *orm.Neorm, otherColumns any, unwantedColumns any) (Options, error) {
	var columns []string

	switch otherColumns.(type) {
	case []string:
		columns = otherColumns.([]string)
	default:
		if len(otherColumns.([]string)) > 0 {
			fmt.Println("otherColumns is not a []string")
			return Options{}, errors.New("otherColumns is not a []string")
		}
	}

	switch unwantedColumns.(type) {
	case []string:
		columns = unwantedColumns.([]string)
	default:
		if len(unwantedColumns.([]string)) > 0 {
			fmt.Println("unwantedColumns is not a []string")
			return Options{}, errors.New("unwantedColumns is not a []string")
		}
		return Options{}, errors.New("unwantedColumns is not a []string")
	}

	columns = append(columns, []string{
		"o.maintenance_mode", "o.max_upload_size", "o.items_per_page",
	}...)

	if len(otherColumns.([]string)) > 0 {
		columns = append(columns, otherColumns.([]string)...)
	}

	if len(unwantedColumns.([]string)) > 0 {
		newColumns := []string{}

		for _, column := range unwantedColumns.([]string) {
			for _, c := range columns {
				if c == column {
					continue
				}

				newColumns = append(newColumns, c)
			}

		}

		columns = newColumns
	}

	opts := db.Select(columns)
	opts.Table("options o")
	opts.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
	opts.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
	opts.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
	opts.Where("o.option_set_is_active", "=", true)
	opts.Finish()
	err := opts.Execute()

	if err != nil {
		return Options{}, err
	}

	rows, err := opts.Rows()

	if err != nil {
		return Options{}, err
	}

	if len(rows) == 0 {
		return Options{}, errors.New("no options found")
	}

	newOptions := models.Options{
		Oid:                           lib.String(rows[0]["oid"]),
		OptionSetName:                 lib.String(rows[0]["option_set_name"]),
		OptionSetDescription:          lib.String(rows[0]["option_set_description"]),
		OptionSetIsActive:             lib.Bool(rows[0]["option_set_is_active"]),
		OptionSetCreatedAt:            lib.Time(rows[0]["option_set_created_at"]),
		OptionSetUpdatedAt:            lib.Time(rows[0]["option_set_updated_at"]),
		SiteName:                      lib.String(rows[0]["site_name"]),
		SiteDescription:               lib.String(rows[0]["site_description"]),
		SiteLogoMid:                   lib.Int64(rows[0]["site_logo_mid"]),
		FaviconMid:                    lib.Int64(rows[0]["site_favicon_mid"]),
		DefaultPageMid:                lib.Int64(rows[0]["default_page_mid"]),
		MaintenanceMode:               lib.Bool(rows[0]["maintenance_mode"]),
		Preloader:                     lib.String(rows[0]["preloader"]),
		SMTPHost:                      lib.String(rows[0]["smtp_host"]),
		SMTPPort:                      lib.Int64(rows[0]["smtp_port"]),
		SMTPUsername:                  lib.String(rows[0]["smtp_username"]),
		SMTPPassword:                  lib.String(rows[0]["smtp_password"]),
		SMTPEncryption:                lib.String(rows[0]["smtp_encryption"]),
		FacebookUrl:                   lib.String(rows[0]["facebook_url"]),
		TwitterUrl:                    lib.String(rows[0]["twitter_url"]),
		InstagramUrl:                  lib.String(rows[0]["instagram_url"]),
		LinkedinUrl:                   lib.String(rows[0]["linkedin_url"]),
		ContactEmail:                  lib.String(rows[0]["contact_email"]),
		ContactPhone:                  lib.String(rows[0]["contact_phone"]),
		MainPageMetaTitle:             lib.String(rows[0]["main_page_meta_title"]),
		MainPageMetaDescription:       lib.String(rows[0]["main_page_meta_description"]),
		GoogleAnalytics:               lib.String(rows[0]["google_analytics"]),
		PrimaryColor:                  lib.String(rows[0]["primary_color"]),
		SecondaryColor:                lib.String(rows[0]["secondary_color"]),
		AccentColor:                   lib.String(rows[0]["accent_color"]),
		BackgroundColor:               lib.String(rows[0]["background_color"]),
		FontColor:                     lib.String(rows[0]["font_color"]),
		FontFamily:                    lib.String(rows[0]["font_family"]),
		RequireStrongPassword:         lib.Bool(rows[0]["require_strong_password"]),
		ItemsPerPage:                  lib.Int64(rows[0]["items_per_page"]),
		ShowDoctorsOnSameCity:         lib.Bool(rows[0]["show_doctors_on_same_city"]),
		ShowDoctorsOnSameCountry:      lib.Bool(rows[0]["show_doctors_on_same_country"]),
		AutoRemovePartnersWhenExpired: lib.Bool(rows[0]["auto_remove_partners_when_expired"]),
		MaxUploadSize:                 lib.Int64(rows[0]["max_upload_size"]),
		Timezone:                      lib.String(rows[0]["timezone"]),
		Language:                      lib.String(rows[0]["language"]),
		EnableTestimonials:            lib.Bool(rows[0]["enable_testimonials"]),
		EnableOurHistory:              lib.Bool(rows[0]["enable_our_history"]),
		MaximumSublinksOnAMenuItem:    lib.Int64(rows[0]["maximum_sublinks_on_a_menu_item"]),
		RecaptchaSiteKey:              lib.String(rows[0]["google_recaptcha_site_key"]),
		RecaptchaSecretKey:            lib.String(rows[0]["google_recaptcha_secret_key"]),
		ShowAnlasmaliKurumPictures:    lib.Bool(rows[0]["show_anlasmali_kurum_pictures"]),
	}

	Options := Options{
		Options: &newOptions,
		Medias: &[]models.Medias{
			{
				FilePath: lib.String(rows[0]["logo_path"]),
				AltText:  lib.String(rows[0]["logo_alt_text"]),
				Title:    lib.String(rows[0]["logo_title"]),
			},
			{
				FilePath: lib.String(rows[0]["favicon_path"]),
			},
			{
				FilePath: lib.String(rows[0]["default_page_path"]),
				AltText:  lib.String(rows[0]["default_page_alt_text"]),
				Title:    lib.String(rows[0]["default_page_title"]),
			},
		},
	}

	return Options, nil
}

func (options *Options) FetchOptionsForPanel(db *orm.Neorm, otherColumns any, unwantedColumns any, User models.AuthenticatedUser) (Options, error) {
	var columns []string

	switch otherColumns.(type) {
	case []string:
		columns = otherColumns.([]string)
	default:
		if len(otherColumns.([]string)) > 0 {
			fmt.Println("otherColumns is not a []string")
			return Options{}, errors.New("otherColumns is not a []string")
		}
	}

	switch unwantedColumns.(type) {
	case []string:
		columns = unwantedColumns.([]string)
	default:
		if len(unwantedColumns.([]string)) > 0 {
			fmt.Println("unwantedColumns is not a []string")
			return Options{}, errors.New("unwantedColumns is not a []string")
		}
		return Options{}, errors.New("unwantedColumns is not a []string")
	}

	columns = append(columns, []string{
		"o.maintenance_mode", "o.max_upload_size", "o.items_per_page",
	}...)

	if len(otherColumns.([]string)) > 0 {
		columns = append(columns, otherColumns.([]string)...)
	}

	if len(unwantedColumns.([]string)) > 0 {
		newColumns := []string{}

		for _, column := range unwantedColumns.([]string) {
			for _, c := range columns {
				if c == column {
					continue
				}

				newColumns = append(newColumns, c)
			}

		}

		columns = newColumns
	}

	opts := db.Select(columns)
	opts.Table("options o")
	opts.LeftJoin("medias m", "o.site_logo_mid", "=", "m.mid")
	opts.LeftJoin("medias m2", "o.site_favicon_mid", "=", "m2.mid")
	opts.LeftJoin("medias m3", "o.default_page_mid", "=", "m3.mid")
	opts.Where("o.option_set_is_active", "=", true)
	opts.Finish()
	err := opts.Execute()

	if err != nil {
		return Options{}, err
	}

	rows, err := opts.Rows()

	if err != nil {
		return Options{}, err
	}

	if len(rows) == 0 {
		return Options{}, errors.New("no options found")
	}

	newOptions := models.Options{
		Oid:                           lib.String(rows[0]["oid"]),
		OptionSetName:                 lib.String(rows[0]["option_set_name"]),
		OptionSetDescription:          lib.String(rows[0]["option_set_description"]),
		OptionSetIsActive:             lib.Bool(rows[0]["option_set_is_active"]),
		OptionSetCreatedAt:            lib.Time(rows[0]["option_set_created_at"]),
		OptionSetUpdatedAt:            lib.Time(rows[0]["option_set_updated_at"]),
		SiteName:                      lib.String(rows[0]["site_name"]),
		SiteDescription:               lib.String(rows[0]["site_description"]),
		SiteLogoMid:                   lib.Int64(rows[0]["site_logo_mid"]),
		FaviconMid:                    lib.Int64(rows[0]["site_favicon_mid"]),
		DefaultPageMid:                lib.Int64(rows[0]["default_page_mid"]),
		MaintenanceMode:               lib.Bool(rows[0]["maintenance_mode"]),
		Preloader:                     lib.String(rows[0]["preloader"]),
		SMTPHost:                      lib.String(rows[0]["smtp_host"]),
		SMTPPort:                      lib.Int64(rows[0]["smtp_port"]),
		SMTPUsername:                  lib.String(rows[0]["smtp_username"]),
		SMTPPassword:                  lib.String(rows[0]["smtp_password"]),
		SMTPEncryption:                lib.String(rows[0]["smtp_encryption"]),
		FacebookUrl:                   lib.String(rows[0]["facebook_url"]),
		TwitterUrl:                    lib.String(rows[0]["twitter_url"]),
		InstagramUrl:                  lib.String(rows[0]["instagram_url"]),
		LinkedinUrl:                   lib.String(rows[0]["linkedin_url"]),
		ContactEmail:                  lib.String(rows[0]["contact_email"]),
		ContactPhone:                  lib.String(rows[0]["contact_phone"]),
		MainPageMetaTitle:             lib.String(rows[0]["main_page_meta_title"]),
		MainPageMetaDescription:       lib.String(rows[0]["main_page_meta_description"]),
		GoogleAnalytics:               lib.String(rows[0]["google_analytics"]),
		PrimaryColor:                  lib.String(rows[0]["primary_color"]),
		SecondaryColor:                lib.String(rows[0]["secondary_color"]),
		AccentColor:                   lib.String(rows[0]["accent_color"]),
		BackgroundColor:               lib.String(rows[0]["background_color"]),
		FontColor:                     lib.String(rows[0]["font_color"]),
		FontFamily:                    lib.String(rows[0]["font_family"]),
		RequireStrongPassword:         lib.Bool(rows[0]["require_strong_password"]),
		ItemsPerPage:                  lib.Int64(rows[0]["items_per_page"]),
		ShowDoctorsOnSameCity:         lib.Bool(rows[0]["show_doctors_on_same_city"]),
		ShowDoctorsOnSameCountry:      lib.Bool(rows[0]["show_doctors_on_same_country"]),
		AutoRemovePartnersWhenExpired: lib.Bool(rows[0]["auto_remove_partners_when_expired"]),
		MaxUploadSize:                 lib.Int64(rows[0]["max_upload_size"]),
		Timezone:                      lib.String(rows[0]["timezone"]),
		Language:                      lib.String(rows[0]["language"]),
		EnableTestimonials:            lib.Bool(rows[0]["enable_testimonials"]),
		EnableOurHistory:              lib.Bool(rows[0]["enable_our_history"]),
		MaximumSublinksOnAMenuItem:    lib.Int64(rows[0]["maximum_sublinks_on_a_menu_item"]),
		RecaptchaSiteKey:              lib.String(rows[0]["google_recaptcha_site_key"]),
		RecaptchaSecretKey:            lib.String(rows[0]["google_recaptcha_secret_key"]),
		ShowAnlasmaliKurumPictures:    lib.Bool(rows[0]["show_anlasmali_kurum_pictures"]),
	}

	Options := Options{
		Options: &newOptions,
		Medias: &[]models.Medias{
			{
				FilePath: lib.String(rows[0]["logo_path"]),
				AltText:  lib.String(rows[0]["logo_alt_text"]),
				Title:    lib.String(rows[0]["logo_title"]),
			},
			{
				FilePath: lib.String(rows[0]["favicon_path"]),
			},
			{
				FilePath: lib.String(rows[0]["default_page_path"]),
				AltText:  lib.String(rows[0]["default_page_alt_text"]),
				Title:    lib.String(rows[0]["default_page_title"]),
			},
		},
	}

	GetNotifications := db.Select([]string{"n.nid", "n.message", "n.notification_type", "n.notification_level", "n.link", "n.is_read", "n.created_at", "n.sid"})
	GetNotifications.Table("notifications n")
	GetNotifications.LeftJoin("users u", "u.uid", "=", lib.String(User.Uid))
	//GetNotifications.LeftJoin("subeler s", "s.sid", "=", "n.sid")
	GetNotifications.Where("n.created_at", ">", User.CreatedAt)

	if User.Role == "santral" {
		GetNotifications.OpenParenthesis("AND")
		GetNotifications.AndExpr("u.sid", "=", "n.sid")
		GetNotifications.OpenParenthesis("AND")
		GetNotifications.Or("n.notification_level", "=", "santral")
		GetNotifications.Or("n.notification_level", "=", "all")
		GetNotifications.CloseParenthesis()
		GetNotifications.CloseParenthesis()
	}

	if User.Role == "ik" {
		GetNotifications.OpenParenthesis("AND")
		GetNotifications.And("n.notification_level", "=", "ik")
		GetNotifications.Or("n.notification_level", "=", "all")
		GetNotifications.CloseParenthesis()
	}

	GetNotifications.OrderBy("n.is_read", "ASC")
	GetNotifications.OrderBy("n.created_at", "DESC")
	GetNotifications.Limit(5)
	GetNotifications.Finish()

	err = GetNotifications.Execute()

	if err != nil {
		return Options, err
	}

	rows, err = GetNotifications.Rows()
	if err != nil {
		return Options, err
	}

	Notifications := []models.Notification{}

	for _, row := range rows {
		Notifications = append(Notifications, models.Notification{
			Nid:               lib.String(row["nid"]),
			Message:           lib.String(row["message"]),
			NotificationType:  lib.String(row["notification_type"]),
			NotificationLevel: lib.String(row["notification_level"]),
			Link:              lib.String(row["link"]),
			IsRead:            lib.Bool(row["is_read"]),
			Sid:               lib.String(row["sid"]),
			CreatedAt:         lib.Time(row["created_at"]),
		})
	}

	Options.Notifications = &Notifications

	return Options, nil
}

func (options *Options) InsertMedia(db *orm.Neorm, media models.Medias, optionals models.MediaOptionals) (string, error) {
	columns := []string{}
	values := []interface{}{}

	columns = append(columns, "file_name", "file_path", "file_size", "mime_type", "file_type", "target_id")
	values = append(values, media.FileName, media.FilePath, media.FileSize, media.MimeType, media.FileType, media.TargetId)

	if media.Uid != "" {
		columns = append(columns, "uid")
		values = append(values, media.Uid)
	}

	if optionals.Data != optionals.OldData {
		if optionals.Data == "" {
			columns = append(columns, "data")
			values = append(values, nil)
		} else {
			columns = append(columns, "data")
			values = append(values, optionals.Data)
		}
	}

	if optionals.AltText != "" {
		columns = append(columns, "alt_text")
		values = append(values, optionals.AltText)
	}

	if optionals.Title != "" {
		columns = append(columns, "title")
		values = append(values, optionals.Title)
	}

	if optionals.Width != 0 {
		columns = append(columns, "width")
		values = append(values, optionals.Width)
	}
	if optionals.Height != 0 {
		columns = append(columns, "height")
		values = append(values, optionals.Height)
	}

	insertMedia := db.Insert(columns, values)
	insertMedia.Table("medias")
	insertMedia.Returning("mid")
	insertMedia.Finish()
	err := insertMedia.Execute()

	if err != nil {
		return "", err
	}

	lid, err := insertMedia.LastInsertId()

	if err != nil {
		return "", err
	}

	return lid, nil
}
