package frontend

import (
	"database"
	"log"
	"models"
	"net/url"
	"strconv"
	"strings"

	"lib"

	"github.com/gofiber/fiber/v2"
)

func HomePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	// buradaki kodlar sadece server başlatıldığında çalışıyor, o sebeple buraya bir
	// mantık koyma.
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		GetTibbiBirimler := Orm.Select([]string{"tb.name", "tb.url_name", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetTibbiBirimler.Table("tibbi_birimler tb")
		GetTibbiBirimler.LeftJoin("medias m", "tb.cover_mid", "=", "m.mid")
		GetTibbiBirimler.Where("tb.is_active", "=", true)
		GetTibbiBirimler.Limit(int(Options.Options.ItemsPerPage))
		GetTibbiBirimler.Finish()
		err = GetTibbiBirimler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := GetTibbiBirimler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		TibbiBirimler := []models.TibbiBirimler{}
		for _, row := range rows {
			TibbiBirimler = append(TibbiBirimler, models.TibbiBirimler{
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
			})
		}

		GetTetkikler := Orm.Select([]string{"t.name", "t.url_name", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetTetkikler.Table("tedkikler t")
		GetTetkikler.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")
		GetTetkikler.Where("is_active", "=", true)
		GetTetkikler.Finish()
		err = GetTetkikler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err = GetTetkikler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Tetkikler := []models.Tedkikler{}
		for _, row := range rows {
			Tetkikler = append(Tetkikler, models.Tedkikler{
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
			})
		}

		Doctors := []models.DoktorForHomePage{}
		GetDoctors := Orm.Select([]string{"d.drid", "d.title", "d.url_name", "d.first_name", "d.last_name", "d.facebook_url", "d.x_url", "d.instagram_url", "d.linkedin_url", "d.personal_url", "d.calistigi_subeler_text", "b.name as brans_name", "b.url_name as brans_url_name", "s.url_name as sube_url_name", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		GetDoctors.Table("doktorlar d")
		GetDoctors.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		GetDoctors.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		GetDoctors.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		GetDoctors.Where("d.is_active", "=", true)
		GetDoctors.AppendCustom("ORDER BY CASE d.title WHEN 'Prof. Dr.' THEN 1 WHEN 'Doç. Dr.' THEN 2 WHEN 'Op. Dr.' THEN 3 WHEN 'Uzm. Dr.' THEN 4 WHEN 'Dr.' THEN 5 ELSE 6 END, d.drid ASC")
		GetDoctors.Limit(int(Options.Options.ItemsPerPage))
		GetDoctors.Finish()

		err = GetDoctors.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err = GetDoctors.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		for _, row := range rows {
			Doctors = append(Doctors, models.DoktorForHomePage{
				Drid:                 lib.String(row["drid"]),
				Title:                lib.String(row["title"]),
				FirstName:            lib.String(row["first_name"]),
				LastName:             lib.String(row["last_name"]),
				UrlName:              lib.String(row["url_name"]),
				PhotoPath:            lib.String(row["photo_path"]),
				PhotoAltText:         lib.String(row["photo_alt_text"]),
				PhotoTitle:           lib.String(row["photo_title"]),
				FacebookUrl:          lib.String(row["facebook_url"]),
				XUrl:                 lib.String(row["x_url"]),
				InstagramUrl:         lib.String(row["instagram_url"]),
				LinkedinUrl:          lib.String(row["linkedin_url"]),
				PersonalUrl:          lib.String(row["personal_url"]),
				SubeUrlName:          lib.String(row["sube_url_name"]),
				CalistigiSubelerText: lib.String(row["calistigi_subeler_text"]),
				BransName:            lib.String(row["brans_name"]),
				BransUrlName:         lib.String(row["brans_url_name"]),
			})
		}

		Testimonials := []models.Testimonials{}

		if Options.Options.EnableTestimonials {
			GetTestimonials := Orm.Select([]string{"t.tid", "t.first_name", "t.last_name", "t.occupation", "t.content", "t.rating", "t.is_active", "m.file_path as customer_picture_path", "m.alt_text as customer_picture_alt_text", "m.title as customer_picture_title"})
			GetTestimonials.Table("testimonials t")
			GetTestimonials.LeftJoin("medias m", "t.customer_picture_mid", "=", "m.mid")
			GetTestimonials.Where("is_active", "=", true)
			GetTestimonials.Limit(int(Options.Options.ItemsPerPage))
			GetTestimonials.Finish()
			err = GetTestimonials.Execute()
			if err != nil {
				log.Printf("%v\n", err)
				return c.Redirect("/giris")
			}
			rows, err = GetTestimonials.Rows()
			if err != nil {
				log.Printf("%v\n", err)
				return c.Redirect("/giris")
			}

			for _, row := range rows {
				Testimonials = append(Testimonials, models.Testimonials{
					Tid:                    lib.String(row["tid"]),
					FirstName:              lib.String(row["first_name"]),
					LastName:               lib.String(row["last_name"]),
					Occupation:             lib.String(row["occupation"]),
					Content:                lib.String(row["content"]),
					Rating:                 lib.Float64(row["rating"]),
					IsActive:               lib.Bool(row["is_active"]),
					CustomerPictureMid:     lib.Int64(row["customer_picture_mid"]),
					CustomerPicturePath:    lib.String(row["customer_picture_path"]),
					CustomerPictureAltText: lib.String(row["customer_picture_alt_text"]),
					CustomerPictureTitle:   lib.String(row["customer_picture_title"]),
				})
			}
		}

		GetAllSubeler := Orm.Select([]string{"s.sid", "s.name"})
		GetAllSubeler.Table("subeler s")
		GetAllSubeler.Where("s.is_active", "=", true)
		GetAllSubeler.Finish()
		err = GetAllSubeler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err = GetAllSubeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		AllSubeler := []models.SubeLink{}
		for _, row := range rows {
			AllSubeler = append(AllSubeler, models.SubeLink{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

                // Anasayfa icin Subeler (resimli)
                HomeSubeler := []models.SubeForFrontendPages{}
                GetHomeSubeler := Orm.Select([]string{"s.sid, s.name, s.url_name, s.city, s.district, s.phone, s.email, m.file_path as media_path, m.alt_text as media_alt_text, m.title as media_title"})
                GetHomeSubeler.Table("subeler s")
                GetHomeSubeler.LeftJoin("medias m", "s.mid", "=", "m.mid")
                GetHomeSubeler.Where("s.is_active", "=", true)
                GetHomeSubeler.OrderBy("s.name", "ASC")
                GetHomeSubeler.Finish()
                err = GetHomeSubeler.Execute()
                if err != nil {
                        log.Printf("%v\n", err)
                }
                rows, err = GetHomeSubeler.Rows()
                if err != nil {
                        log.Printf("%v\n", err)
                }
                for _, row := range rows {
                        HomeSubeler = append(HomeSubeler, models.SubeForFrontendPages{
                                Sid:          lib.String(row["sid"]),
                                Name:         lib.String(row["name"]),
                                UrlName:      lib.String(row["url_name"]),
                                City:         lib.String(row["city"]),
                                District:     lib.String(row["district"]),
                                Phone:        lib.String(row["phone"]),
                                Email:        lib.String(row["email"]),
                                MediaPath:    lib.String(row["media_path"]),
                                MediaAltText: lib.String(row["media_alt_text"]),
                                MediaTitle:   lib.String(row["media_title"]),
                        })
                }

                GetAllHomepageContents := Orm.Select([]string{"hc.content_html", "hc.content_javascript", "hc.content_css", "hc.later_than_which_content", "hc.content_type", "hc.tibbi_birim_id", "COALESCE(tb.url_name, '') as tb_url_name"})
                GetAllHomepageContents.Table("homepage_contents hc")
                GetAllHomepageContents.LeftJoin("tibbi_birimler tb", "hc.tibbi_birim_id", "=", "tb.tbid")
                GetAllHomepageContents.Where("hc.is_active", "=", true)
                GetAllHomepageContents.OrderBy("hc.sort_order", "ASC")
		GetAllHomepageContents.Finish()
		err = GetAllHomepageContents.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err = GetAllHomepageContents.Rows()

		if err != nil {
			log.Printf("%v\n", err)
		}

		AllHomepageContents := []models.HomepageContents{}

		BannerContents := []models.HomepageContents{}

		for _, row := range rows {
			if row["content_type"] == "banner_page" {
				BannerContents = append(BannerContents, models.HomepageContents{
					ContentHtml:           lib.String(row["content_html"]),
					ContentJavascript:     lib.String(row["content_javascript"]),
					ContentCss:            lib.String(row["content_css"]),
					LaterThanWhichContent: lib.Int64(row["later_than_which_content"]),
					ContentType:           lib.String(row["content_type"]),
                                        TibbiBirimId:          lib.String(row["tibbi_birim_id"]),
                                        TibbiBirimUrlName:     lib.String(row["tb_url_name"]),
				})
			} else {
				AllHomepageContents = append(AllHomepageContents, models.HomepageContents{
					ContentHtml:           lib.String(row["content_html"]),
					ContentJavascript:     lib.String(row["content_javascript"]),
					ContentCss:            lib.String(row["content_css"]),
					LaterThanWhichContent: lib.Int64(row["later_than_which_content"]),
					ContentType:           lib.String(row["content_type"]),
				})
			}
		}

		return c.Render("views/frontend/index", fiber.Map{
			"PathOnStart":         "",
			"Route":               "/",
			"Options":             Options,
			"TibbiBirimler":       TibbiBirimler,
			"Tetkikler":           Tetkikler,
			"User":                OurUser,
			"Testimonials":        Testimonials,
			"Doctors":             Doctors,
			"AllSubeler":          AllSubeler,
                        "Subeler":             HomeSubeler,
			"AllHomepageContents": AllHomepageContents,
			"BannerContents":      BannerContents,
			"Title":               Options.Options.MainPageMetaTitle,
			"Description":         Options.Options.MainPageMetaDescription,
		}, "layouts/main/main")
	}
}

func AboutUsPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		GetHakkimizdaContents := Orm.Select([]string{"content_html", "content_javascript", "content_css", "created_at", "updated_at"})
		GetHakkimizdaContents.Table("custom_contents")
		GetHakkimizdaContents.Where("content_type", "=", "about-us")
		GetHakkimizdaContents.And("is_active", "=", true)
		GetHakkimizdaContents.OrderBy("sort_order", "ASC")
		GetHakkimizdaContents.Finish()
		err := GetHakkimizdaContents.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err := GetHakkimizdaContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		HtmlContents := []string{}
		for _, row := range rows {
			HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
		}

		CssContents := []string{}
		for _, row := range rows {
			CssContents = append(CssContents, lib.String(row["content_css"]))
		}

		JavascriptContents := []string{}
		for _, row := range rows {
			JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))
		}

		MinCreatedAt := lib.Time(nil)
		MaxUpdatedAt := lib.Time(nil)
		for _, row := range rows {
			RowCreatedAt := lib.Time(row["created_at"])
			RowUpdatedAt := lib.Time(row["updated_at"])
			if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
				MinCreatedAt = RowCreatedAt
			}
			if RowUpdatedAt.After(MaxUpdatedAt) {
				MaxUpdatedAt = RowUpdatedAt
			}
		}

		return c.Render("views/frontend/about-us", fiber.Map{
			"PathOnStart":        "../",
			"Route":              "/kurumsal/hakkimizda",
			"Options":            Options,
			"User":               OurUser,
			"HtmlContents":       HtmlContents,
			"CssContents":        CssContents,
			"JavascriptContents": JavascriptContents,
			"Title":              "Hakkımızda | " + Options.Options.SiteName,
			"Description":        "Bu sayfa, " + Options.Options.SiteName + " hakkında bilgi verir.",
			"PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
			"UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
			"ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
		}, "layouts/main/main")
	}
}

func LoginPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err == nil {
			return c.Redirect("/panel")
		}

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		return c.Render("views/frontend/giris", fiber.Map{
			"PathOnStart": "",
			"Route":       "/giris",
			"Options":     Options,
			"User":        OurUser,
			"Title":       "Giriş Yap | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin admin paneli giriş sayfasıdır.",
		}, "layouts/main/main")
	}
}

func ContactPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		// Fetch şube data for contact information
		var Subeler []models.SubeForFrontendPages
		GetSubeler := Orm.Select([]string{"s.sid, s.name, s.url_name, s.city, s.district, s.google_map_iframe, s.phone, s.email, m.file_path as media_path, m.alt_text as media_alt_text, m.title as media_title, s.transportation_info"})
		GetSubeler.Table("subeler s")
		GetSubeler.LeftJoin("medias m", "s.mid", "=", "m.mid")
		GetSubeler.Where("s.is_active", "=", true)
		GetSubeler.OrderBy("s.name", "ASC")
		GetSubeler.Finish()
		err := GetSubeler.Execute()

		if err != nil {
			log.Printf("Cannot fetch subeler: %v\n", err)
		}

		rows, err := GetSubeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		for _, row := range rows {
			Subeler = append(Subeler, models.SubeForFrontendPages{
				Sid:                lib.String(row["sid"]),
				Name:               lib.String(row["name"]),
				UrlName:            lib.String(row["url_name"]),
				City:               lib.String(row["city"]),
				District:           lib.String(row["district"]),
				GoogleMapIframe:    lib.String(row["google_map_iframe"]),
				Phone:              lib.String(row["phone"]),
				Email:              lib.String(row["email"]),
				MediaPath:          lib.String(row["media_path"]),
				MediaAltText:       lib.String(row["media_alt_text"]),
				MediaTitle:         lib.String(row["media_title"]),
				TransportationInfo: lib.String(row["transportation_info"]),
			})
		}

		return c.Render("views/frontend/iletisim", fiber.Map{
			"PathOnStart": "../",
			"Route":       "/iletisim",
			"Options":     Options,
			"User":        OurUser,
			"Subeler":     Subeler,
			"Title":       "İletişim | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin iletişim sayfası olup şubelerimizin iletişim bilgilerini içerir.",
		}, "layouts/main/main")
	}
}

func MissionVisionPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		GetMissionVisionContents := Orm.Select([]string{"c.content_html", "c.content_javascript", "c.content_css", "c.created_at", "c.updated_at"})
		GetMissionVisionContents.Table("custom_contents c")
		GetMissionVisionContents.Where("c.content_type", "=", "mission-vision")
		GetMissionVisionContents.And("c.is_active", "=", true)
		GetMissionVisionContents.OrderBy("c.sort_order", "ASC")
		GetMissionVisionContents.Finish()

		err := GetMissionVisionContents.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err := GetMissionVisionContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		HtmlContents := []string{}
		for _, row := range rows {
			HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
		}

		CssContents := []string{}
		for _, row := range rows {
			CssContents = append(CssContents, lib.String(row["content_css"]))
		}

		JavascriptContents := []string{}
		for _, row := range rows {
			JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))
		}

		MinCreatedAt := lib.Time(nil)
		MaxUpdatedAt := lib.Time(nil)
		for _, row := range rows {
			RowCreatedAt := lib.Time(row["created_at"])
			RowUpdatedAt := lib.Time(row["updated_at"])
			if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
				MinCreatedAt = RowCreatedAt
			}
			if RowUpdatedAt.After(MaxUpdatedAt) {
				MaxUpdatedAt = RowUpdatedAt
			}
		}

		return c.Render("views/frontend/misyon-vizyon", fiber.Map{
			"PathOnStart":        "../",
			"Route":              "/kurumsal/misyon-vizyon",
			"Options":            Options,
			"HtmlContents":       HtmlContents,
			"CssContents":        CssContents,
			"JavascriptContents": JavascriptContents,
			"User":               OurUser,
			"Title":              "Misyon ve Vizyon | " + Options.Options.SiteName,
			"Description":        "Bu sayfa, " + Options.Options.SiteName + " sitesinin misyon ve vizyon sayfası olup, bu sayfada misyon ve vizyonumuzu bulabilirsiniz.",
			"PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
			"UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
			"ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
		}, "layouts/main/main")
	}
}

func AnlasmaliKurumlarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		GetAnlasmaliKurumlar := Orm.Select([]string{"a.akid", "a.name", "a.sid", "s.name as sube_name", "a.discount_rate", "a.is_active", "m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title"})
		GetAnlasmaliKurumlar.Table("anlasmali_kurumlar a")
		GetAnlasmaliKurumlar.InnerJoin("subeler s", "a.sid", "=", "s.sid")
		GetAnlasmaliKurumlar.LeftJoin("medias m", "a.logo_mid", "=", "m.mid")
		GetAnlasmaliKurumlar.Where("a.is_active", "=", true)
		GetAnlasmaliKurumlar.OrderBy("a.name", "ASC")
		GetAnlasmaliKurumlar.Limit(int(Options.Options.ItemsPerPage))
		GetAnlasmaliKurumlar.Finish()
		err := GetAnlasmaliKurumlar.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/")
		}
		rows, err := GetAnlasmaliKurumlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/")
		}

		AnlasmaliKurumlar := []models.AnlasmaliKurumlar{}
		for _, row := range rows {
			AnlasmaliKurumlar = append(AnlasmaliKurumlar, models.AnlasmaliKurumlar{
				Akid:         lib.String(row["akid"]),
				Name:         lib.String(row["name"]),
				Sid:          lib.String(row["sid"]),
				SubeName:     lib.String(row["sube_name"]),
				DiscountRate: lib.Float64(row["discount_rate"]),
				IsActive:     lib.Bool(row["is_active"]),
				LogoMid:      lib.Int64(row["logo_mid"]),
				LogoPath:     lib.String(row["logo_path"]),
				LogoAltText:  lib.String(row["logo_alt_text"]),
				LogoTitle:    lib.String(row["logo_title"]),
			})
		}

		return c.Render("views/frontend/anlasmali-kurumlar", fiber.Map{
			"PathOnStart":       "../",
			"Route":             "/kurumsal/anlasmali-kurumlar",
			"Options":           Options,
			"User":              OurUser,
			"AnlasmaliKurumlar": AnlasmaliKurumlar,
			"Title":             "Anlasmali Kurumlar | " + Options.Options.SiteName,
			"Description":       "Bu sayfa, " + Options.Options.SiteName + " sitesinin anlaşmalı kurumlar sayfası olup, bu sayfada Anlaşma yaptığımız kurumları bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func InsanKaynaklariPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/")
		}

		GetHrContents := Orm.Select([]string{"c.content_html", "c.content_javascript", "c.content_css", "c.created_at", "c.updated_at"})
		GetHrContents.Table("custom_contents c")
		GetHrContents.Where("c.content_type", "=", "hr")
		GetHrContents.And("c.is_active", "=", true)
		GetHrContents.OrderBy("c.sort_order", "ASC")
		GetHrContents.Finish()

		err = GetHrContents.Execute()

		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err := GetHrContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		HtmlContents := []string{}
		for _, row := range rows {
			HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
		}

		CssContents := []string{}
		for _, row := range rows {
			CssContents = append(CssContents, lib.String(row["content_css"]))
		}

		JavascriptContents := []string{}
		for _, row := range rows {
			JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))
		}

		MinCreatedAt := lib.Time(nil)
		MaxUpdatedAt := lib.Time(nil)
		for _, row := range rows {
			RowCreatedAt := lib.Time(row["created_at"])
			RowUpdatedAt := lib.Time(row["updated_at"])
			if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
				MinCreatedAt = RowCreatedAt
			}
			if RowUpdatedAt.After(MaxUpdatedAt) {
				MaxUpdatedAt = RowUpdatedAt
			}
		}

		return c.Render("views/frontend/insan-kaynaklari", fiber.Map{
			"PathOnStart":        "../",
			"Route":              "/kurumsal/insan-kaynaklari",
			"Options":            Options,
			"User":               OurUser,
			"Title":              "İnsan Kaynakları | " + Options.Options.SiteName,
			"Description":        "Bu sayfa, " + Options.Options.SiteName + " sitesinin insan kaynakları sayfası olup, bu sayfada insan kaynakları politikamızla alakalı bilgileri bulabilirsiniz.",
			"HtmlContents":       HtmlContents,
			"CssContents":        CssContents,
			"JavascriptContents": JavascriptContents,
			"PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
			"UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
			"ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
		}, "layouts/main/main")
	}
}

func HaberlerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("news options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		Page := 1
		if c.Query("page") != "" {
			ConvertQuery, err := strconv.Atoi(c.Query("page"))

			if err != nil {
				Page = 1
			} else {
				Page = ConvertQuery
			}
		}
		if Page < 1 {
			Page = 1
		}

		Category := "genel"
		if c.Query("category") != "" {
			Category = c.Query("category")
		}

		Text := ""

		if c.Query("text") != "" {
			Text = c.Query("text")
		}

		PageSize := int(Options.Options.ItemsPerPage)
		if PageSize < 1 {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		GetTotal := Orm.Select([]string{"COUNT(*) as count"})
		GetTotal.Table("haberler h")
		GetTotal.Where("h.is_published", "=", true)
		GetTotal.And("h.category", "=", Category)
		if Text != "" {
			GetTotal.OpenParenthesis("AND")
			GetTotal.Like("AND", "h.title", Text, "contains")
			GetTotal.Like("OR", "h.summary", Text, "contains")
			GetTotal.Like("OR", "h.content", Text, "contains")
			GetTotal.CloseParenthesis()
		}
		GetTotal.Finish()
		if err := GetTotal.Execute(); err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		TotalRows, err := GetTotal.Rows()
		if err != nil || len(TotalRows) != 1 {
			log.Printf("news count rows: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		Total := lib.Int64(TotalRows[0]["count"])
		TotalPages := int((Total + int64(PageSize) - 1) / int64(PageSize))
		if TotalPages > 0 && Page > TotalPages {
			Page = TotalPages
		}
		Offset := (Page - 1) * PageSize

		GetHaberler := Orm.Select([]string{"h.hid", "h.title", "h.publish_date", "h.url_name", "h.author", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetHaberler.Table("haberler h")
		GetHaberler.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")

		GetHaberler.Where("h.is_published", "=", true)
		GetHaberler.And("h.category", "=", Category)
		if Text != "" {
			GetHaberler.OpenParenthesis("AND")
			GetHaberler.Like("AND", "h.title", Text, "contains")
			GetHaberler.Like("OR", "h.summary", Text, "contains")
			GetHaberler.Like("OR", "h.content", Text, "contains")
			GetHaberler.CloseParenthesis()
		}

		GetHaberler.OrderBy("h.publish_date", "DESC")
		GetHaberler.Limit(PageSize)
		GetHaberler.Offset(Offset)
		GetHaberler.Finish()

		log.Printf("GetHaberler Query: %v", GetHaberler.Query)

		err = GetHaberler.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetHaberler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		//fmt.Printf("Rows: %v\n", rows)

		Haberler := []models.HaberlerForHaberlerPage{}
		for _, row := range rows {
			Haberler = append(Haberler, models.HaberlerForHaberlerPage{
				Hid:          lib.String(row["hid"]),
				Title:        lib.String(row["title"]),
				UrlName:      lib.String(row["url_name"]),
				Author:       lib.String(row["author"]),
				CoverMid:     lib.Int64(row["cover_mid"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
				PublishDate:  lib.Time(row["publish_date"]),
			})
		}

		GetCategoryCounts := Orm.Select([]string{"category", "COUNT(*) as count"})
		GetCategoryCounts.Table("haberler")
		GetCategoryCounts.Where("is_published", "=", true)
		GetCategoryCounts.GroupBy("category")
		GetCategoryCounts.Finish()

		err = GetCategoryCounts.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err = GetCategoryCounts.Rows()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		CategoryCounts := []models.HaberlerPageCategoryNameAndCounts{}
		for _, row := range rows {
			CategoryCounts = append(CategoryCounts, models.HaberlerPageCategoryNameAndCounts{
				Name:  lib.String(row["category"]),
				Count: lib.Int64(row["count"]),
			})
		}

		PaginationBase := "/haberler?category=" + url.QueryEscape(Category) + "&text=" + url.QueryEscape(Text) + "&page="
		return c.Render("views/frontend/haberler", fiber.Map{
			"PathOnStart":    "../",
			"Route":          "/haberler",
			"Options":        Options,
			"User":           OurUser,
			"Haberler":       Haberler,
			"CategoryCounts": CategoryCounts,
			"Category":       Category,
			"Text":           Text,
			"Total":          Total,
			"Page":           Page,
			"PreviousPage":   Page - 1,
			"NextPage":       Page + 1,
			"TotalPages":     TotalPages,
			"PaginationBase": PaginationBase,
			"Title":          "Haberler | " + Options.Options.SiteName,
			"Description":    "Bu sayfa, " + Options.Options.SiteName + " sitesinin haberler sayfası olup, bu sayfada hastanemiz hakkında yayınlanan haberleri bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func HaberPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("news detail options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		HaberUrlName := c.Params("haber")

		GetHaber := Orm.Select([]string{"h.*", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetHaber.Table("haberler h")
		GetHaber.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")
		GetHaber.Where("h.url_name", "=", HaberUrlName)
		GetHaber.And("h.is_published", "=", true)
		GetHaber.Finish()

		err = GetHaber.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetHaber.Rows()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if len(rows) == 0 {
			return FallbackPage(states, utilities)(c)
		}

		UpdateViewsCount := Orm.Update()
		UpdateViewsCount.Table("haberler")
		UpdateViewsCount.SetExpr("views_count", "views_count + 1")
		UpdateViewsCount.Where("hid", "=", lib.Int64(rows[0]["hid"]))
		UpdateViewsCount.Finish()

		log.Printf("UpdateViewsCount Query: %v", UpdateViewsCount.Query)

		err = UpdateViewsCount.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		ra, err := UpdateViewsCount.RowsAffected()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if ra == 0 {
			log.Printf("Cannot update views count on that haber: %v\n", rows[0]["url_name"])
		}

		Haber := models.Haberler{
			Title:          lib.String(rows[0]["title"]),
			UrlName:        lib.String(rows[0]["url_name"]),
			Summary:        lib.String(rows[0]["summary"]),
			Content:        lib.String(rows[0]["content"]),
			CoverMid:       lib.Int64(rows[0]["cover_mid"]),
			CoverPath:      lib.String(rows[0]["cover_path"]),
			CoverAltText:   lib.String(rows[0]["cover_alt_text"]),
			CoverTitle:     lib.String(rows[0]["cover_title"]),
			PublishDate:    lib.Time(rows[0]["publish_date"]),
			Author:         lib.String(rows[0]["author"]),
			ViewsCount:     lib.Int64(rows[0]["views_count"]),
			IsPublished:    lib.Bool(rows[0]["is_published"]),
			IsFeatured:     lib.Bool(rows[0]["is_featured"]),
			CreatedAt:      lib.Time(rows[0]["created_at"]),
			UpdatedAt:      lib.Time(rows[0]["updated_at"]),
			Category:       lib.String(rows[0]["category"]),
			Tags:           lib.StringArray(rows[0]["tags"]),
			SeoTitle:       lib.String(rows[0]["seo_title"]),
			SeoDescription: lib.String(rows[0]["seo_description"]),
			SeoKeywords:    lib.String(rows[0]["seo_keywords"]),
		}

		HaberPublishedAt := Haber.PublishDate
		if HaberPublishedAt.IsZero() {
			HaberPublishedAt = Haber.CreatedAt
		}

		return c.Render("views/frontend/haber", fiber.Map{
			"PathOnStart":   "../",
			"Route":         "/haberler/" + Haber.UrlName,
			"Options":       Options,
			"User":          OurUser,
			"Haber":         Haber,
			"Title":         Haber.SeoTitle,
			"Description":   Haber.SeoDescription,
			"PublishedDate": lib.FormatFrontendDate(HaberPublishedAt),
			"UpdatedDate":   lib.FormatFrontendDate(Haber.UpdatedAt),
			"ShowUpdated":   lib.IsDifferentDay(HaberPublishedAt, Haber.UpdatedAt),
		}, "layouts/main/main")
	}
}

func FotoGaleriPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		return c.Render("views/frontend/foto-galeri", fiber.Map{
			"PathOnStart": "",
			"Route":       "/foto-galeri",
			"Options":     Options,
			"User":        OurUser,
			"Title":       "Foto Galeri | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin foto galeri sayfası olup, bu sayfada hastanemiz hakkında yayınlanan foto galerileri bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func VideoGaleriPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		return c.Render("views/frontend/video-galeri", fiber.Map{
			"PathOnStart": "",
			"Route":       "/video-galeri",
			"Options":     Options,
			"User":        OurUser,
			"Title":       "Video Galeri | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin video galeri sayfası olup, bu sayfada hastanemiz hakkında yayınlanan video galerileri bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func UlasimPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
    return func(c *fiber.Ctx) error {
        OurUser, _ := lib.CheckAuth(c)
        Orm := utilities.Orm

        FrontendOptions := models.FrontendOptions{
            Database: Orm,
            User:     OurUser,
            States:   states,
        }

        Options := database.Options{}
        Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

        // ✅ Admin panelde "İçerik Tipi" = ulasim seçilecek
        GetUlasimContents := Orm.Select([]string{"c.content_html", "c.content_javascript", "c.content_css", "c.created_at", "c.updated_at"})
        GetUlasimContents.Table("custom_contents c")
        GetUlasimContents.Where("c.content_type", "=", "ulasim")
        GetUlasimContents.And("c.is_active", "=", true)
        GetUlasimContents.OrderBy("c.sort_order", "ASC")
        GetUlasimContents.Finish()

        err := GetUlasimContents.Execute()
        if err != nil {
            log.Printf("%v\n", err)
        }

        rows, err := GetUlasimContents.Rows()
        if err != nil {
            log.Printf("%v\n", err)
        }

        HtmlContents := []string{}
        CssContents := []string{}
        JavascriptContents := []string{}

        MinCreatedAt := lib.Time(nil)
        MaxUpdatedAt := lib.Time(nil)
        for _, row := range rows {
            HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
            CssContents = append(CssContents, lib.String(row["content_css"]))
            JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))

            RowCreatedAt := lib.Time(row["created_at"])
            RowUpdatedAt := lib.Time(row["updated_at"])
            if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
                MinCreatedAt = RowCreatedAt
            }
            if RowUpdatedAt.After(MaxUpdatedAt) {
                MaxUpdatedAt = RowUpdatedAt
            }
        }

        return c.Render("views/frontend/ulasim", fiber.Map{
            "PathOnStart":        "../",
            "Route":              "/ulasim",
            "Options":            Options,
            "User":               OurUser,
            "HtmlContents":       HtmlContents,
            "CssContents":        CssContents,
            "JavascriptContents": JavascriptContents,
            "Title":              "Ulaşım | " + Options.Options.SiteName,
            "Description":        "Bu sayfa, " + Options.Options.SiteName + " kurumunun ulaşım bilgilerini içerir.",
            "PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
            "UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
            "ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
        }, "layouts/main/main")
    }
}


func AnlasmaliKurumlarSayfasiPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)
		Orm := utilities.Orm
		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}
		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		GetContents := Orm.Select([]string{"c.content_html", "c.content_javascript", "c.content_css", "c.created_at", "c.updated_at"})
		GetContents.Table("custom_contents c")
		GetContents.Where("c.content_type", "=", "anlasmali-kurumlar")
		GetContents.And("c.is_active", "=", true)
		GetContents.OrderBy("c.sort_order", "ASC")
		GetContents.Finish()
		err := GetContents.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}
		rows, err := GetContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}
		HtmlContents := []string{}
		CssContents := []string{}
		JavascriptContents := []string{}
		MinCreatedAt := lib.Time(nil)
		MaxUpdatedAt := lib.Time(nil)
		for _, row := range rows {
			HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
			CssContents = append(CssContents, lib.String(row["content_css"]))
			JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))

			RowCreatedAt := lib.Time(row["created_at"])
			RowUpdatedAt := lib.Time(row["updated_at"])
			if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
				MinCreatedAt = RowCreatedAt
			}
			if RowUpdatedAt.After(MaxUpdatedAt) {
				MaxUpdatedAt = RowUpdatedAt
			}
		}
		return c.Render("views/frontend/anlasmali-kurumlar", fiber.Map{
			"PathOnStart":        "../",
			"Route":              "/kurumsal/anlasmali-kurumlar",
			"Options":            Options,
			"User":               OurUser,
			"HtmlContents":       HtmlContents,
			"CssContents":        CssContents,
			"JavascriptContents": JavascriptContents,
			"Title":              "Anlaşmalı Kurumlar | " + Options.Options.SiteName,
			"Description":        "Bu sayfa, " + Options.Options.SiteName + " kurumunun anlaşmalı kurumlarını içerir.",
			"PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
			"UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
			"ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
		}, "layouts/main/main")
	}
}
func OrganizasyonSemasiPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)
		Orm := utilities.Orm
		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}
		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		// Tüm şubeleri getir
		GetSubeler := Orm.Select([]string{"s.sid", "s.name", "s.document_mids"})
		GetSubeler.Table("subeler s")
		GetSubeler.Where("s.is_active", "=", true)
		GetSubeler.OrderBy("s.sid", "ASC")
		GetSubeler.Finish()
		err := GetSubeler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}
		subeRows, err := GetSubeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		type OrgDoc struct {
			SubeName string
			FileName string
			FilePath string
		}
		OrgDocs := []OrgDoc{}

		for _, sube := range subeRows {
			mids := lib.StringArray(sube["document_mids"])
			if len(mids) == 0 {
				continue
			}
			midArgs := []any{}
			for _, m := range mids {
				midArgs = append(midArgs, m)
			}
			GetDocs := Orm.Select([]string{"mid", "file_name", "file_path", "data"})
			GetDocs.Table("medias")
			GetDocs.In("WHERE", "mid", midArgs)
			GetDocs.And("data", "=", "organizasyon-semasi")
			GetDocs.Finish()
			err = GetDocs.Execute()
			if err != nil {
				log.Printf("%v\n", err)
				continue
			}
			docRows, err := GetDocs.Rows()
			if err != nil {
				log.Printf("%v\n", err)
				continue
			}
			for _, doc := range docRows {
				OrgDocs = append(OrgDocs, OrgDoc{
					SubeName: lib.String(sube["name"]),
					FileName: lib.String(doc["file_name"]),
					FilePath: lib.String(doc["file_path"]),
				})
			}
		}

		return c.Render("views/frontend/organizasyon-semasi", fiber.Map{
			"PathOnStart": "../",
			"Route":       "/kurumsal/organizasyon-semasi",
			"Options":     Options,
			"User":        OurUser,
			"OrgDocs":     OrgDocs,
			"Title":       "Organizasyon Şeması | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " kurumunun organizasyon şemasını içerir.",
		}, "layouts/main/main")
	}
}

func CookiePolicyPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
        return func(c *fiber.Ctx) error {
                OurUser, _ := lib.CheckAuth(c)

                Orm := utilities.Orm

                FrontendOptions := models.FrontendOptions{
                        Database: Orm,
                        User:     OurUser,
                        States:   states,
                }

                Options := database.Options{}
                Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

                GetCookieContents := Orm.Select([]string{"c.content_html", "c.content_javascript", "c.content_css", "c.created_at", "c.updated_at"})
                GetCookieContents.Table("custom_contents c")
                GetCookieContents.Where("c.content_type", "=", "cookie-policy")
                GetCookieContents.And("c.is_active", "=", true)
                GetCookieContents.OrderBy("c.sort_order", "ASC")
                GetCookieContents.Finish()

                err := GetCookieContents.Execute()
                if err != nil {
                        log.Printf("%v\n", err)
                }

                rows, err := GetCookieContents.Rows()
                if err != nil {
                        log.Printf("%v\n", err)
                }

                HtmlContents := []string{}
                CssContents := []string{}
                JavascriptContents := []string{}

                MinCreatedAt := lib.Time(nil)
                MaxUpdatedAt := lib.Time(nil)
                for _, row := range rows {
                        HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
                        CssContents = append(CssContents, lib.String(row["content_css"]))
                        JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))

                        RowCreatedAt := lib.Time(row["created_at"])
                        RowUpdatedAt := lib.Time(row["updated_at"])
                        if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
                                MinCreatedAt = RowCreatedAt
                        }
                        if RowUpdatedAt.After(MaxUpdatedAt) {
                                MaxUpdatedAt = RowUpdatedAt
                        }
                }

                return c.Render("views/frontend/kvkk", fiber.Map{
                        "PathOnStart":        "../",
                        "Route":              "/kurumsal/cerez-politikasi",
                        "Options":            Options,
                        "User":               OurUser,
                        "HtmlContents":       HtmlContents,
                        "CssContents":        CssContents,
                        "JavascriptContents": JavascriptContents,
                        "Title":              "Çerez Politikası | " + Options.Options.SiteName,
                        "Description":        "Bu sayfa, " + Options.Options.SiteName + " sitesinin çerez politikası sayfasıdır.",
                        "PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
                        "UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
                        "ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
                }, "layouts/main/main")
        }
}

func KvkkPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		GetKvkkContents := Orm.Select([]string{"c.content_html", "c.content_javascript", "c.content_css", "c.created_at", "c.updated_at"})
		GetKvkkContents.Table("custom_contents c")
		GetKvkkContents.Where("c.content_type", "=", "privacy-policy")
		GetKvkkContents.And("c.is_active", "=", true)
		GetKvkkContents.OrderBy("c.sort_order", "ASC")
		GetKvkkContents.Finish()
		err := GetKvkkContents.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err := GetKvkkContents.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		HtmlContents := []string{}
		for _, row := range rows {
			HtmlContents = append(HtmlContents, lib.String(row["content_html"]))
		}

		CssContents := []string{}
		for _, row := range rows {
			CssContents = append(CssContents, lib.String(row["content_css"]))
		}

		JavascriptContents := []string{}
		for _, row := range rows {
			JavascriptContents = append(JavascriptContents, lib.String(row["content_javascript"]))
		}

		MinCreatedAt := lib.Time(nil)
		MaxUpdatedAt := lib.Time(nil)
		for _, row := range rows {
			RowCreatedAt := lib.Time(row["created_at"])
			RowUpdatedAt := lib.Time(row["updated_at"])
			if !RowCreatedAt.IsZero() && (MinCreatedAt.IsZero() || RowCreatedAt.Before(MinCreatedAt)) {
				MinCreatedAt = RowCreatedAt
			}
			if RowUpdatedAt.After(MaxUpdatedAt) {
				MaxUpdatedAt = RowUpdatedAt
			}
		}

		return c.Render("views/frontend/kvkk", fiber.Map{
			"PathOnStart":        "../",
			"Route":              "/kurumsal/kvkk",
			"Options":            Options,
			"User":               OurUser,
			"HtmlContents":       HtmlContents,
			"CssContents":        CssContents,
			"JavascriptContents": JavascriptContents,
			"Title":              "KVKK | " + Options.Options.SiteName,
			"Description":        "Bu sayfa, " + Options.Options.SiteName + " sitesinin KVKK sayfası olup, bu sayfada KVKK bilgilerini bulabilirsiniz.",
			"PublishedDate":      lib.FormatFrontendDate(MinCreatedAt),
			"UpdatedDate":        lib.FormatFrontendDate(MaxUpdatedAt),
			"ShowUpdated":        lib.IsDifferentDay(MinCreatedAt, MaxUpdatedAt),
		}, "layouts/main/main")
	}
}

func SubelerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Offset := (Page - 1) * int(Options.Options.MaximumSublinksOnAMenuItem)

		GetSubeler := Orm.Select([]string{"s.sid, s.name, s.url_name, s.city, s.district, s.google_map_iframe, s.phone, s.email, m.file_path as media_path, m.alt_text as media_alt_text, m.title as media_title, s.transportation_info"})
		GetSubeler.Table("subeler s")
		GetSubeler.LeftJoin("medias m", "s.mid", "=", "m.mid")
		GetSubeler.Where("s.is_active", "=", true)
		GetSubeler.OrderBy("s.name", "ASC")
		GetSubeler.Limit(int(Options.Options.MaximumSublinksOnAMenuItem))
		GetSubeler.Offset(Offset)
		GetSubeler.Finish()
		err := GetSubeler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err := GetSubeler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		Subeler := []models.SubeForFrontendPages{}
		for _, row := range rows {
			Subeler = append(Subeler, models.SubeForFrontendPages{
				Sid:                lib.String(row["sid"]),
				Name:               lib.String(row["name"]),
				UrlName:            lib.String(row["url_name"]),
				City:               lib.String(row["city"]),
				District:           lib.String(row["district"]),
				GoogleMapIframe:    lib.String(row["google_map_iframe"]),
				Phone:              lib.String(row["phone"]),
				Email:              lib.String(row["email"]),
				MediaPath:          lib.String(row["media_path"]),
				MediaAltText:       lib.String(row["media_alt_text"]),
				MediaTitle:         lib.String(row["media_title"]),
				TransportationInfo: lib.String(row["transportation_info"]),
			})
		}

		return c.Render("views/frontend/subeler", fiber.Map{
			"PathOnStart": "",
			"Route":       "/merkezlerimiz",
			"Options":     Options,
			"User":        OurUser,
			"Subeler":     Subeler,
			"Title":       "Merkezlerimiz | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin merkezlerimiz sayfası olup, bu sayfada hastanemizin merkezleri hakkında bilgi bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func SubePage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("center detail options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		subeName := c.Params("sube")

		GetSube := Orm.Select([]string{"s.*", "m.file_path as media_path", "m.alt_text as media_alt_text", "m.title as media_title"})
		GetSube.Table("subeler s")
		GetSube.LeftJoin("medias m", "s.mid", "=", "m.mid")
		GetSube.Where("s.url_name", "=", subeName)
		GetSube.And("s.is_active", "=", true)
		GetSube.Finish()
		err = GetSube.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetSube.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if len(rows) == 0 {
			return FallbackPage(states, utilities)(c)
		}

		Sube := models.Subeler{
			Sid:                    lib.String(rows[0]["sid"]),
			Name:                   lib.String(rows[0]["name"]),
			UrlName:                lib.String(rows[0]["url_name"]),
			City:                   lib.String(rows[0]["city"]),
			District:               lib.String(rows[0]["district"]),
			GoogleMapIframe:        lib.String(rows[0]["google_map_iframe"]),
			Description:            lib.String(rows[0]["description"]),
			Address:                lib.String(rows[0]["address"]),
			PostalCode:             lib.String(rows[0]["postal_code"]),
			Fax:                    lib.String(rows[0]["fax"]),
			Website:                lib.String(rows[0]["website"]),
			Latitude:               lib.Float64(rows[0]["latitude"]),
			Longitude:              lib.Float64(rows[0]["longitude"]),
			WorkingHours:           lib.String(rows[0]["working_hours"]),
			TransportationInfoHtml: lib.String(rows[0]["transportation_info_html"]),
			AnlasmaliKurumlarHtml:  lib.String(rows[0]["anlasmali_kurumlar_html"]),
			DocumentMids:           lib.StringArray(rows[0]["document_mids"]),
			Phone:                  lib.String(rows[0]["phone"]),
			Email:                  lib.String(rows[0]["email"]),
			SubeMediaPath:          lib.String(rows[0]["media_path"]),
			SubeMediaAltText:       lib.String(rows[0]["media_alt_text"]),
			SubeMediaTitle:         lib.String(rows[0]["media_title"]),
			TransportationInfo:     lib.String(rows[0]["transportation_info"]),
		}

		SubeMids := []any{}
		if len(Sube.DocumentMids) > 0 {
			for _, mid := range Sube.DocumentMids {
				SubeMids = append(SubeMids, mid)
			}
		}

		SubeDocumentsArray := []models.Medias{}
		if len(SubeMids) > 0 {
			SubeDocuments := Orm.Select([]string{"mid", "file_path", "file_name", "file_size", "mime_type", "data"})
			SubeDocuments.Table("medias")
			SubeDocuments.In("WHERE", "mid", SubeMids)
			SubeDocuments.Finish()
			err = SubeDocuments.Execute()
			if err != nil {
				log.Printf("Cannot get sube documents: %v\n", err)
			}

			subeDocumentsRows, err := SubeDocuments.Rows()
			if err != nil {
				log.Printf("Cannot get sube documents rows: %v\n", err)
			}

			for _, row := range subeDocumentsRows {
				SubeDocumentsArray = append(SubeDocumentsArray, models.Medias{
					Mid:      lib.Int64(row["mid"]),
					FilePath: lib.String(row["file_path"]),
					FileName: lib.String(row["file_name"]),
					FileSize: lib.Int64(row["file_size"]),
					MimeType: lib.String(row["mime_type"]),
					Data:     lib.String(row["data"]),
				})
			}
		}

		GetAnlasmaliKurumCount := Orm.Count("anlasmali_kurumlar")
		GetAnlasmaliKurumCount.Where("sid", "=", Sube.Sid)
		GetAnlasmaliKurumCount.And("is_active", "=", true)
		GetAnlasmaliKurumCount.Finish()
		err = GetAnlasmaliKurumCount.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		AnlasmaliKurumCount := GetAnlasmaliKurumCount.Length()

		GetAnlasmaliKurumlar := Orm.Select([]string{"ak.akid", "ak.name", "ak.url_name", "ak.type", "ak.discount_rate", "ak.is_active", "ak.created_at", "ak.updated_at"})
		GetAnlasmaliKurumlar.Table("anlasmali_kurumlar ak")
		GetAnlasmaliKurumlar.LeftJoin("medias m", "ak.logo_mid", "=", "m.mid")
		GetAnlasmaliKurumlar.Where("ak.sid", "=", Sube.Sid)
		GetAnlasmaliKurumlar.And("ak.is_active", "=", true)
		GetAnlasmaliKurumlar.OrderBy("ak.name", "ASC")
		GetAnlasmaliKurumlar.Finish()
		err = GetAnlasmaliKurumlar.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err = GetAnlasmaliKurumlar.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		AnlasmaliKurumlar := []models.AnlasmaliKurumlar{}
		for _, row := range rows {
			AnlasmaliKurumlar = append(AnlasmaliKurumlar, models.AnlasmaliKurumlar{
				Akid:          lib.String(row["akid"]),
				Name:          lib.String(row["name"]),
				UrlName:       lib.String(row["url_name"]),
				Type:          lib.String(row["type"]),
				ContactPerson: lib.String(row["contact_person"]),
				Phone:         lib.String(row["phone"]),
				Email:         lib.String(row["email"]),
				IsActive:      lib.Bool(row["is_active"]),
				CreatedAt:     lib.Time(row["created_at"]),
				UpdatedAt:     lib.Time(row["updated_at"]),
				LogoPath:      lib.String(row["logo_path"]),
				LogoAltText:   lib.String(row["logo_alt_text"]),
				LogoTitle:     lib.String(row["logo_title"]),
			})
		}

                // Doktorlar sorgula
                GetDoktorlar := Orm.Select([]string{"d.drid", "d.title", "d.first_name", "d.last_name", "d.url_name", "b.name as brans_name", "b.url_name as brans_url_name", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
                GetDoktorlar.Table("doktorlar d")
                GetDoktorlar.LeftJoin("branslar b", "d.brid", "=", "b.brid")
                GetDoktorlar.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
                GetDoktorlar.LeftJoin("doktor_subeler ds", "d.drid", "=", "ds.drid")
                GetDoktorlar.Where("ds.sid", "=", Sube.Sid)
                GetDoktorlar.And("d.is_active", "=", true)
                GetDoktorlar.AppendCustom("ORDER BY d.last_name ASC")
                GetDoktorlar.Finish()
                err = GetDoktorlar.Execute()
                if err != nil {
                        log.Printf("Doktor sorgu hatasi: %v\n", err)
                }
                doktorRows, _ := GetDoktorlar.Rows()
                SubeDoktorlar := []models.DoktorForHomePage{}
                for _, row := range doktorRows {
                        SubeDoktorlar = append(SubeDoktorlar, models.DoktorForHomePage{
                                Drid:         lib.String(row["drid"]),
                                Title:        lib.String(row["title"]),
                                FirstName:    lib.String(row["first_name"]),
                                LastName:     lib.String(row["last_name"]),
                                UrlName:      lib.String(row["url_name"]),
                                BransName:    lib.String(row["brans_name"]),
                                BransUrlName: lib.String(row["brans_url_name"]),
                                PhotoPath:    lib.String(row["photo_path"]),
                                PhotoAltText: lib.String(row["photo_alt_text"]),
                                PhotoTitle:   lib.String(row["photo_title"]),
                        })
                }

		// Galeri sorgula
                GetGaleri := Orm.Select([]string{"sg.sgid", "sg.sid", "sg.mid", "sg.caption", "sg.sort_order", "m.file_path", "m.alt_text", "m.title"})
                GetGaleri.Table("sube_galerileri sg")
                GetGaleri.LeftJoin("medias m", "sg.mid", "=", "m.mid")
                GetGaleri.Where("sg.sid", "=", Sube.Sid)
                GetGaleri.And("sg.is_active", "=", true)
                GetGaleri.AppendCustom("ORDER BY sg.sort_order ASC, sg.sgid ASC")
                GetGaleri.Finish()
                err = GetGaleri.Execute()
                if err != nil {
                        log.Printf("Galeri sorgu hatasi: %v\n", err)
                }
                galeriRows, _ := GetGaleri.Rows()
                SubeGaleri := []models.SubeGaleri{}
                for _, row := range galeriRows {
                        SubeGaleri = append(SubeGaleri, models.SubeGaleri{
                                Sgid:      lib.String(row["sgid"]),
                                Sid:       lib.String(row["sid"]),
                                Mid:       lib.Int64(row["mid"]),
                                Caption:   lib.String(row["caption"]),
                                SortOrder: int(lib.Int64(row["sort_order"])),
                                IsActive:  lib.Bool(row["is_active"]),
                                FilePath:  lib.String(row["file_path"]),
                                AltText:   lib.String(row["alt_text"]),
                                Title:     lib.String(row["title"]),
                        })
                }
                return c.Render("views/frontend/sube", fiber.Map{
			"PathOnStart":         "../../",
			"Route":               "/merkezlerimiz/" + subeName,
			"Options":             Options,
			"User":                OurUser,
			"Sube":                Sube,
			"AnlasmaliKurumCount": AnlasmaliKurumCount,
			"AnlasmaliKurumlar":   AnlasmaliKurumlar,
			"SubeDocuments":       SubeDocumentsArray,
                        "SubeGaleri":          SubeGaleri,
                        "SubeDoktorlar":       SubeDoktorlar,
			"Title":               Sube.Name + " | " + Options.Options.SiteName,
			"Description":         "Bu sayfa, " + Options.Options.SiteName + " hastaneleri'nin " + Sube.Name + " merkezinin sayfası olup, bu sayfada " + Sube.Name + " merkezinin hakkında bilgi bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func DoktorlarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("center doctors options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		subeName := c.Params("sube")
		GetSubeName := Orm.Select([]string{"sid", "name"})
		GetSubeName.Table("subeler")
		GetSubeName.Where("url_name", "=", subeName)
		GetSubeName.And("is_active", "=", true)
		GetSubeName.Finish()
		err = GetSubeName.Execute()
		if err != nil {
			log.Printf("center doctors center: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		subeRows, err := GetSubeName.Rows()
		if err != nil {
			log.Printf("center doctors center rows: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		if len(subeRows) == 0 {
			return FallbackPage(states, utilities)(c)
		}
		SubeName := models.SubeForFrontendPages{
			Sid:  lib.String(subeRows[0]["sid"]),
			Name: lib.String(subeRows[0]["name"]),
		}

		Doctors := []models.DoktorForHomePage{}
		GetDoctors := Orm.Select([]string{"d.drid", "d.url_name", "d.title", "d.first_name", "d.last_name", "d.facebook_url", "d.x_url", "d.instagram_url", "d.linkedin_url", "d.personal_url", "b.name as brans_name", "b.url_name as brans_url_name", "s.url_name as sube_url_name", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		GetDoctors.Table("doktorlar d")
		GetDoctors.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		GetDoctors.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		GetDoctors.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		GetDoctors.Where("s.url_name", "=", subeName)
		GetDoctors.And("s.is_active", "=", true)
		GetDoctors.And("d.is_active", "=", true)
		GetDoctors.AppendCustom("ORDER BY CASE d.title WHEN 'Prof. Dr.' THEN 1 WHEN 'Doç. Dr.' THEN 2 WHEN 'Op. Dr.' THEN 3 WHEN 'Uzm. Dr.' THEN 4 WHEN 'Dr.' THEN 5 ELSE 6 END, d.drid ASC")
		GetDoctors.Finish()

		log.Printf("GetDoctors Query: %v", GetDoctors.Query)

		err = GetDoctors.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetDoctors.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		for _, row := range rows {
			Doctors = append(Doctors, models.DoktorForHomePage{
				Drid:         lib.String(row["drid"]),
				Title:        lib.String(row["title"]),
				FirstName:    lib.String(row["first_name"]),
				LastName:     lib.String(row["last_name"]),
				UrlName:      lib.String(row["url_name"]),
				PhotoPath:    lib.String(row["photo_path"]),
				PhotoAltText: lib.String(row["photo_alt_text"]),
				PhotoTitle:   lib.String(row["photo_title"]),
				FacebookUrl:  lib.String(row["facebook_url"]),
				XUrl:         lib.String(row["x_url"]),
				InstagramUrl: lib.String(row["instagram_url"]),
				LinkedinUrl:  lib.String(row["linkedin_url"]),
				PersonalUrl:  lib.String(row["personal_url"]),
				SubeUrlName:  lib.String(row["sube_url_name"]),
				BransName:    lib.String(row["brans_name"]),
				BransUrlName: lib.String(row["brans_url_name"]),
			})
		}

		return c.Render("views/frontend/doktorlar", fiber.Map{
			"PathOnStart": "../../",
			"Route":       "/merkezlerimiz/" + subeName + "/doktorlar",
			"Options":     Options,
			"User":        OurUser,
			"Doctors":     Doctors,
			"SubeName":    SubeName,
			"Title":       "Doktorlar | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin doktorlar sayfası olup, bu sayfada hastanemizin doktorları hakkında bilgi bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func TumDoktorlarPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Page := 1
		if c.Query("page") != "" {
			NewPage, err := strconv.Atoi(c.Query("page"))
			if err != nil {
				Page = 1
			} else {
				Page = NewPage
			}
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		GetCountOfDoctors := Orm.Count("doktorlar")
		GetCountOfDoctors.Where("is_active", "=", true)
		GetCountOfDoctors.Finish()
		err := GetCountOfDoctors.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/")
		}
		CountOfDoctors := GetCountOfDoctors.Length()

		ItemsPerPage := Options.Options.ItemsPerPage

		Offset := (Page - 1) * int(ItemsPerPage)

		Doctors := []models.DoktorForHomePage{}
		GetDoctors := Orm.Select([]string{"d.drid", "d.calistigi_subeler_text", "d.url_name", "d.title", "d.first_name", "d.last_name", "d.facebook_url", "d.x_url", "d.instagram_url", "d.linkedin_url", "d.personal_url", "b.name as brans_name", "b.url_name as brans_url_name", "s.name as sube_name", "s.url_name as sube_url_name", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		GetDoctors.Table("doktorlar d")
		GetDoctors.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		GetDoctors.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		GetDoctors.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		GetDoctors.Where("d.is_active", "=", true)
		GetDoctors.AppendCustom("ORDER BY CASE d.title WHEN 'Prof. Dr.' THEN 1 WHEN 'Doç. Dr.' THEN 2 WHEN 'Op. Dr.' THEN 3 WHEN 'Uzm. Dr.' THEN 4 WHEN 'Dr.' THEN 5 ELSE 6 END, d.drid ASC")
		GetDoctors.Limit(int(ItemsPerPage))
		GetDoctors.Offset(Offset)
		GetDoctors.Finish()

		err = GetDoctors.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/")
		}

		rows, err := GetDoctors.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/")
		}

		for _, row := range rows {
			Doctors = append(Doctors, models.DoktorForHomePage{
				Drid:                 lib.String(row["drid"]),
				CalistigiSubelerText: lib.String(row["calistigi_subeler_text"]),
				Title:                lib.String(row["title"]),
				FirstName:            lib.String(row["first_name"]),
				LastName:             lib.String(row["last_name"]),
				UrlName:              lib.String(row["url_name"]),
				PhotoPath:            lib.String(row["photo_path"]),
				PhotoAltText:         lib.String(row["photo_alt_text"]),
				PhotoTitle:           lib.String(row["photo_title"]),
				FacebookUrl:          lib.String(row["facebook_url"]),
				XUrl:                 lib.String(row["x_url"]),
				InstagramUrl:         lib.String(row["instagram_url"]),
				LinkedinUrl:          lib.String(row["linkedin_url"]),
				PersonalUrl:          lib.String(row["personal_url"]),
				SubeUrlName:          lib.String(row["sube_url_name"]),
				SubeName:             lib.String(row["sube_name"]),
				BransName:            lib.String(row["brans_name"]),
				BransUrlName:         lib.String(row["brans_url_name"]),
			})
		}

		TotalPages := (int(CountOfDoctors) + int(ItemsPerPage) - 1) / int(ItemsPerPage)

		return c.Render("views/frontend/tum-doktorlar", fiber.Map{
			"PathOnStart":    "../../",
			"Route":          "/doktorlarimiz",
			"Options":        Options,
			"User":           OurUser,
			"Doctors":        Doctors,
			"CountOfDoctors": CountOfDoctors,
			"Page":           Page,
			"TotalPages":     TotalPages,
			"Offset":         Offset,
			"Title":          "Doktorlarımız | " + Options.Options.SiteName,
			"Description":    "Bu sayfa, " + Options.Options.SiteName + " sitesinin doktorlar sayfası olup, bu sayfada kurumumuzun tüm doktorları hakkında bilgi bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func DoktorPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("center doctor options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		subeName := c.Params("sube")
		doktorName := c.Params("doktor")

		GetDoktor := Orm.Select([]string{"d.*", "s.name as sube_name", "b.name as brans_name", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		GetDoktor.Table("doktorlar d")
		GetDoktor.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		GetDoktor.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		GetDoktor.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		GetDoktor.Where("d.url_name", "=", doktorName)
		GetDoktor.And("d.is_active", "=", true)
		GetDoktor.And("s.url_name", "=", subeName)
		GetDoktor.And("s.is_active", "=", true)
		GetDoktor.Finish()

		err = GetDoktor.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetDoktor.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if len(rows) == 0 {
			return FallbackPage(states, utilities)(c)
		}

		Doktor := models.Doktorlar{}
		for _, row := range rows {
			Doktor = models.Doktorlar{
				Drid:                 lib.String(row["drid"]),
				UrlName:              lib.String(row["url_name"]),
				Title:                lib.String(row["title"]),
				FirstName:            lib.String(row["first_name"]),
				LastName:             lib.String(row["last_name"]),
				PhotoPath:            lib.String(row["photo_path"]),
				PhotoAltText:         lib.String(row["photo_alt_text"]),
				PhotoTitle:           lib.String(row["photo_title"]),
				SubeName:             lib.String(row["sube_name"]),
				BranchName:           lib.String(row["brans_name"]),
				FacebookUrl:          lib.String(row["facebook_url"]),
				XUrl:                 lib.String(row["x_url"]),
				InstagramUrl:         lib.String(row["instagram_url"]),
				LinkedinUrl:          lib.String(row["linkedin_url"]),
				PersonalUrl:          lib.String(row["personal_url"]),
				AppointmentFee:       lib.Float64(row["appointment_fee"]),
				Phone:                lib.String(row["phone"]),
				Email:                lib.String(row["email"]),
				Biography:            lib.String(row["biography"]),
				Education:            lib.String(row["education"]),
				Languages:            lib.String(row["languages"]),
				BirthDate:            lib.Time(row["birth_date"]),
				WorkingHours:         lib.String(row["working_hours"]),
				CalistigiSubelerText: lib.String(row["calistigi_subeler_text"]),
				DoctorInfosHtml:      lib.String(row["doctor_infos_html"]),
				DoctorInfosCss:       lib.String(row["doctor_infos_css"]),
				DoctorInfosJs:        lib.String(row["doctor_infos_js"]),
				Experiences:          []models.DoctorExperiences{},
				Expertises:           []models.DoctorExpertises{},
			}
		}

		GetDoctorExperiences := Orm.Select([]string{"de.name", "de.description", "de.start_date", "de.end_date", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetDoctorExperiences.Table("doctor_experiences de")
		GetDoctorExperiences.LeftJoin("medias m", "de.cover_mid", "=", "m.mid")
		GetDoctorExperiences.Where("drid", "=", Doktor.Drid)
		GetDoctorExperiences.Finish()
		err = GetDoctorExperiences.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err = GetDoctorExperiences.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		DoctorExperiences := []models.DoctorExperiences{}
		for _, row := range rows {
			DoctorExperiences = append(DoctorExperiences, models.DoctorExperiences{
				Name:        lib.String(row["name"]),
				Description: lib.String(row["description"]),
				StartDate:   lib.Time(row["start_date"]),
				EndDate:     lib.Time(row["end_date"]),
			})
		}

		Doktor.Experiences = DoctorExperiences

		GetDoctorExpertises := Orm.Select([]string{"u.name", "u.description", "de.certification_date", "de.certification_institution", "de.is_primary"})
		GetDoctorExpertises.Table("doctor_expertises de")
		GetDoctorExpertises.LeftJoin("uzmanliklar u", "de.uzid", "=", "u.uzid")
		GetDoctorExpertises.Where("drid", "=", Doktor.Drid)
		GetDoctorExpertises.Finish()
		err = GetDoctorExpertises.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err = GetDoctorExpertises.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		DoctorExpertises := []models.DoctorExpertises{}
		for _, row := range rows {
			DoctorExpertises = append(DoctorExpertises, models.DoctorExpertises{
				UzmanlikName:             lib.String(row["name"]),
				UzmanlikDescription:      lib.String(row["description"]),
				CertificationDate:        lib.Time(row["certification_date"]),
				CertificationInstitution: lib.String(row["certification_institution"]),
				IsPrimary:                lib.Bool(row["is_primary"]),
			})
		}

		Doktor.Expertises = DoctorExpertises

		return c.Render("views/frontend/doktor", fiber.Map{
			"PathOnStart": "../../../",
			"Route":       "/merkezlerimiz/" + subeName + "/doktorlar/" + doktorName,
			"Options":     Options,
			"User":        OurUser,
			"Doktor":      Doktor,
			"Title":       Doktor.Title + Doktor.FirstName + Doktor.LastName + " | " + Options.Options.SiteName,
			"Description": "Bu sayfa, hastanemizin doktorlarından olan " + Doktor.Title + Doktor.FirstName + Doktor.LastName + " hakkında bilgi içeren sayfadır.",
		}, "layouts/main/main")
	}
}

func DoktorPageForDoktorlarimiz(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		doktorName := c.Params("doktor")

		GetDoktor := Orm.Select([]string{"d.*", "s.name as sube_name", "s.url_name as sube_url_name", "b.name as brans_name", "b.url_name as brans_url_name", "m.file_path as photo_path", "m.alt_text as photo_alt_text", "m.title as photo_title"})
		GetDoktor.Table("doktorlar d")
		GetDoktor.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		GetDoktor.LeftJoin("subeler s", "d.sid", "=", "s.sid")
		GetDoktor.LeftJoin("branslar b", "d.brid", "=", "b.brid")
		GetDoktor.Where("d.url_name", "=", doktorName)
		GetDoktor.And("d.is_active", "=", true)
		GetDoktor.Finish()

		err := GetDoktor.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetDoktor.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		if len(rows) == 0 {
			return FallbackPage(states, utilities)(c)
		}

		Doktor := models.Doktorlar{}
		for _, row := range rows {
			Doktor = models.Doktorlar{
				Drid:                 lib.String(row["drid"]),
				Title:                lib.String(row["title"]),
				FirstName:            lib.String(row["first_name"]),
				LastName:             lib.String(row["last_name"]),
				PhotoPath:            lib.String(row["photo_path"]),
				PhotoAltText:         lib.String(row["photo_alt_text"]),
				PhotoTitle:           lib.String(row["photo_title"]),
				SubeName:             lib.String(row["sube_name"]),
				BranchName:           lib.String(row["brans_name"]),
				FacebookUrl:          lib.String(row["facebook_url"]),
				XUrl:                 lib.String(row["x_url"]),
				InstagramUrl:         lib.String(row["instagram_url"]),
				LinkedinUrl:          lib.String(row["linkedin_url"]),
				PersonalUrl:          lib.String(row["personal_url"]),
				AppointmentFee:       lib.Float64(row["appointment_fee"]),
				Phone:                lib.String(row["phone"]),
				Email:                lib.String(row["email"]),
				Biography:            lib.String(row["biography"]),
				Education:            lib.String(row["education"]),
				Languages:            lib.String(row["languages"]),
				BirthDate:            lib.Time(row["birth_date"]),
				WorkingHours:         lib.String(row["working_hours"]),
				CalistigiSubelerText: lib.String(row["calistigi_subeler_text"]),
				DoctorInfosHtml:      lib.String(row["doctor_infos_html"]),
				DoctorInfosCss:       lib.String(row["doctor_infos_css"]),
				DoctorInfosJs:        lib.String(row["doctor_infos_js"]),
				Experiences:          []models.DoctorExperiences{},
				Expertises:           []models.DoctorExpertises{},
			}
		}

		GetDoctorExperiences := Orm.Select([]string{"de.name", "de.description", "de.start_date", "de.end_date", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetDoctorExperiences.Table("doctor_experiences de")
		GetDoctorExperiences.LeftJoin("medias m", "de.cover_mid", "=", "m.mid")
		GetDoctorExperiences.Where("drid", "=", Doktor.Drid)
		GetDoctorExperiences.Finish()
		err = GetDoctorExperiences.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err = GetDoctorExperiences.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoctorExperiences := []models.DoctorExperiences{}
		for _, row := range rows {
			DoctorExperiences = append(DoctorExperiences, models.DoctorExperiences{
				Name:        lib.String(row["name"]),
				Description: lib.String(row["description"]),
				StartDate:   lib.Time(row["start_date"]),
				EndDate:     lib.Time(row["end_date"]),
			})
		}

		Doktor.Experiences = DoctorExperiences

		GetDoctorExpertises := Orm.Select([]string{"u.name", "u.description", "de.certification_date", "de.certification_institution", "de.is_primary"})
		GetDoctorExpertises.Table("doctor_expertises de")
		GetDoctorExpertises.LeftJoin("uzmanliklar u", "de.uzid", "=", "u.uzid")
		GetDoctorExpertises.Where("drid", "=", Doktor.Drid)
		GetDoctorExpertises.Finish()
		err = GetDoctorExpertises.Execute()
		if err != nil {
			log.Printf("%v\n", err)
		}

		rows, err = GetDoctorExpertises.Rows()
		if err != nil {
			log.Printf("%v\n", err)
		}

		DoctorExpertises := []models.DoctorExpertises{}
		for _, row := range rows {
			DoctorExpertises = append(DoctorExpertises, models.DoctorExpertises{
				UzmanlikName:             lib.String(row["name"]),
				UzmanlikDescription:      lib.String(row["description"]),
				CertificationDate:        lib.Time(row["certification_date"]),
				CertificationInstitution: lib.String(row["certification_institution"]),
				IsPrimary:                lib.Bool(row["is_primary"]),
			})
		}

		Doktor.Expertises = DoctorExpertises

		return c.Render("views/frontend/doktor", fiber.Map{
			"PathOnStart": "../../../",
			"Route":       "/doktorlarimiz/" + Doktor.UrlName,
			"Options":     Options,
			"User":        OurUser,
			"Doktor":      Doktor,
			"Title":       Doktor.Title + Doktor.FirstName + Doktor.LastName + " | " + Options.Options.SiteName,
			"Description": "Bu sayfa, hastanemizin doktorlarından olan " + Doktor.Title + Doktor.FirstName + Doktor.LastName + " hakkında bilgi içeren sayfadır.",
		}, "layouts/main/main")
	}
}

func RandevuPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		RandevuMerkezleri := loadRandevuMerkezleri(utilities)

		return c.Render("views/frontend/randevu", fiber.Map{
			"PathOnStart":       "",
			"Route":             "/randevu",
			"Options":           Options,
			"User":              OurUser,
			"RandevuMerkezleri": RandevuMerkezleri,
			"Title":             "Randevu Talebi | " + Options.Options.SiteName,
			"Description":       "Bu sayfa, " + Options.Options.SiteName + " merkezlerine 4 adımda randevu talebi bırakabileceğiniz sayfadır; merkezimiz sizi arayıp uygun saati birlikte belirler.",
		}, "layouts/main/main")
	}
}

func TibbiBirimlerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)
		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		GetTibbiBirimler := Orm.Select([]string{"tb.name", "tb.url_name", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetTibbiBirimler.Table("tibbi_birimler tb")
		GetTibbiBirimler.LeftJoin("medias m", "tb.cover_mid", "=", "m.mid")
		GetTibbiBirimler.Where("tb.is_active", "=", true)
		GetTibbiBirimler.Limit(int(Options.Options.ItemsPerPage))
		GetTibbiBirimler.Finish()
		err = GetTibbiBirimler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := GetTibbiBirimler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		TibbiBirimler := []models.TibbiBirimler{}
		for _, row := range rows {
			TibbiBirimler = append(TibbiBirimler, models.TibbiBirimler{
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
			})
		}

		return c.Render("views/frontend/tibbi-birimler", fiber.Map{
			"PathOnStart":   "",
			"Route":         "/tibbi-birimler",
			"Options":       Options,
			"User":          OurUser,
			"TibbiBirimler": TibbiBirimler,
			"Title":         "Tibbi Birimler | " + Options.Options.SiteName,
			"Description":   "Bu sayfa, " + Options.Options.SiteName + " sitesinin tibbi birimler sayfası olup, bu sayfada hastanemizin tibbi birimleri hakkında bilgi bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func TibbiBirimPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("medical unit detail options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		tibbiBirimName := c.Params("tibbibirim")

		GetTibbiBirim := Orm.Select([]string{"tb.name", "tb.url_name", "tb.description", "tb.html_content", "tb.javascript_content", "tb.css_content", "tb.is_active", "tb.created_at", "tb.updated_at", "tb.video_mid", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title", "m2.file_path as video_path"})
		GetTibbiBirim.Table("tibbi_birimler tb")
		GetTibbiBirim.LeftJoin("medias m", "tb.cover_mid", "=", "m.mid")
		GetTibbiBirim.LeftJoin("medias m2", "tb.video_mid", "=", "m2.mid")
		GetTibbiBirim.Where("tb.url_name", "=", tibbiBirimName)
		GetTibbiBirim.And("tb.is_active", "=", true)
		GetTibbiBirim.Finish()
		err = GetTibbiBirim.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetTibbiBirim.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		if len(rows) == 0 {
			return FallbackPage(states, utilities)(c)
		}

		TibbiBirim := models.TibbiBirimler{}
		for _, row := range rows {
			TibbiBirim = models.TibbiBirimler{
				Name:              lib.String(row["name"]),
				UrlName:           lib.String(row["url_name"]),
				CoverPath:         lib.String(row["cover_path"]),
				CoverAltText:      lib.String(row["cover_alt_text"]),
				CoverTitle:        lib.String(row["cover_title"]),
				Description:       lib.String(row["description"]),
				HtmlContent:       lib.String(row["html_content"]),
				JavascriptContent: lib.String(row["javascript_content"]),
				CssContent:        lib.String(row["css_content"]),
				IsActive:          lib.Bool(row["is_active"]),
				CreatedAt:         lib.Time(row["created_at"]),
				UpdatedAt:         lib.Time(row["updated_at"]),
				VideoMid:          lib.Int64(row["video_mid"]),
				VideoPath:         lib.String(row["video_path"]),
			}
		}

		return c.Render("views/frontend/tibbi-birim", fiber.Map{
			"PathOnStart":   "../",
			"Route":         "/tibbi-birimler/" + tibbiBirimName,
			"Options":       Options,
			"User":          OurUser,
			"TibbiBirim":    TibbiBirim,
			"Title":         TibbiBirim.Name + " | " + Options.Options.SiteName,
			"Description":   "Bu sayfa, hastanemizin tibbi birimlerinden olan " + TibbiBirim.Name + " hakkında bilgi içeren sayfadır.",
			"PublishedDate": lib.FormatFrontendDate(TibbiBirim.CreatedAt),
			"UpdatedDate":   lib.FormatFrontendDate(TibbiBirim.UpdatedAt),
			"ShowUpdated":   lib.IsDifferentDay(TibbiBirim.CreatedAt, TibbiBirim.UpdatedAt),
		}, "layouts/main/main")
	}
}

func TedkiklerPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)
		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)

		GetTetkikler := Orm.Select([]string{"t.name", "t.url_name", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetTetkikler.Table("tedkikler t")
		GetTetkikler.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")
		GetTetkikler.Where("is_active", "=", true)
		GetTetkikler.Finish()
		err := GetTetkikler.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		rows, err := GetTetkikler.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/giris")
		}

		Tetkikler := []models.Tedkikler{}
		for _, row := range rows {
			Tetkikler = append(Tetkikler, models.Tedkikler{
				Name:         lib.String(row["name"]),
				UrlName:      lib.String(row["url_name"]),
				CoverPath:    lib.String(row["cover_path"]),
				CoverAltText: lib.String(row["cover_alt_text"]),
				CoverTitle:   lib.String(row["cover_title"]),
			})
		}

		return c.Render("views/frontend/tedkikler", fiber.Map{
			"PathOnStart": "../",
			"Route":       "/tedkikler",
			"Options":     Options,
			"User":        OurUser,
			"Tetkikler":   Tetkikler,
			"Title":       "Tedkikler | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin tedkikler sayfası olup, bu sayfada hastanemiz bünyesinde yapılan tedkikler hakkında bilgi bulabilirsiniz.",
		}, "layouts/main/main")
	}
}

func TedkikPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)
		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("examination detail options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		tedkikName := c.Params("tetkik")

		GetTedkik := Orm.Select([]string{"t.name", "t.url_name", "t.description", "t.html_content", "t.javascript_content", "t.css_content", "t.created_at", "t.updated_at", "m.file_path as cover_path", "m.alt_text as cover_alt_text", "m.title as cover_title"})
		GetTedkik.Table("tedkikler t")
		GetTedkik.LeftJoin("medias m", "t.cover_mid", "=", "m.mid")
		GetTedkik.Where("t.url_name", "=", tedkikName)
		GetTedkik.And("t.is_active", "=", true)
		GetTedkik.Finish()
		err = GetTedkik.Execute()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		rows, err := GetTedkik.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		if len(rows) == 0 {
			return FallbackPage(states, utilities)(c)
		}

		Tedkik := models.Tedkikler{}
		for _, row := range rows {
			Tedkik = models.Tedkikler{
				Name:              lib.String(row["name"]),
				UrlName:           lib.String(row["url_name"]),
				Description:       lib.String(row["description"]),
				HtmlContent:       lib.String(row["html_content"]),
				JavascriptContent: lib.String(row["javascript_content"]),
				CssContent:        lib.String(row["css_content"]),
				CoverPath:         lib.String(row["cover_path"]),
				CoverAltText:      lib.String(row["cover_alt_text"]),
				CoverTitle:        lib.String(row["cover_title"]),
				CreatedAt:         lib.Time(row["created_at"]),
				UpdatedAt:         lib.Time(row["updated_at"]),
			}
		}

		return c.Render("views/frontend/tedkik", fiber.Map{
			"PathOnStart":   "../",
			"Route":         "/tetkikler/" + tedkikName,
			"Options":       Options,
			"User":          OurUser,
			"Tedkik":        Tedkik,
			"PublishedDate": lib.FormatFrontendDate(Tedkik.CreatedAt),
			"UpdatedDate":   lib.FormatFrontendDate(Tedkik.UpdatedAt),
			"ShowUpdated":   lib.IsDifferentDay(Tedkik.CreatedAt, Tedkik.UpdatedAt),
			"Title":       Tedkik.Name + " | " + Options.Options.SiteName,
			"Description": "Bu sayfa, hastanemiz bünyesinde yapılan tedkiklerden olan " + Tedkik.Name + " hakkında bilgi içeren sayfadır.",
		}, "layouts/main/main")
	}
}

func FallbackPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)
		Orm := utilities.Orm

		FrontendOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, err := Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
		if err != nil {
			log.Printf("fallback options: %v\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		c.Set("X-Robots-Tag", "noindex, nofollow")
		c.Status(fiber.StatusNotFound)
		return c.Render("views/frontend/fallback", fiber.Map{
			"PathOnStart": "/",
			"Route":       "*",
			"Options":     Options,
			"User":        OurUser,
			"Title":       "Sayfa Bulunamadı | " + Options.Options.SiteName,
			"Description": "Bu sayfa, " + Options.Options.SiteName + " sitesinin sayfa bulunamadı sayfasıdır.",
		}, "layouts/main/main")
	}
}

func SearchPage(states *models.AppState, utilities *models.Utilities) fiber.Handler {
        return func(c *fiber.Ctx) error {
                OurUser, _ := lib.CheckAuth(c)
                Orm := utilities.Orm
                FrontendOptions := models.FrontendOptions{
                        Database: Orm,
                        User:     OurUser,
                        States:   states,
                }
                Options := database.Options{}
                Options, _ = Options.FetchOptionsForFrontendWithCache(&FrontendOptions)
                q := strings.TrimSpace(c.Query("q"))

                type SearchResult struct {
                        Title   string
                        Url     string
                        Excerpt string
                        Type    string
                }
                results := []SearchResult{}

                if q != "" {
                        escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q)
                        like := "%" + escaped + "%"
                        db := Orm.Pool

                        rows, err := db.Query(`SELECT name, url_name, COALESCE(LEFT(description,150),'') FROM tibbi_birimler WHERE (name ILIKE $1 OR description ILIKE $1) AND is_active=true LIMIT 10`, like)
                        if err != nil {
                                log.Println("SearchPage tibbi_birimler query error:", err)
                        }
                        if rows != nil {
                                for rows.Next() {
                                        var t, u, e string
                                        rows.Scan(&t, &u, &e)
                                        results = append(results, SearchResult{Title: t, Url: "/tibbi-birimler/" + u, Excerpt: e, Type: "Tıbbi Birim"})
                                }
                                rows.Close()
                        }

                        rows2, err := db.Query(`SELECT title, url_name, COALESCE(LEFT(summary,150),'') FROM haberler WHERE (title ILIKE $1 OR content ILIKE $1) AND is_published=true LIMIT 10`, like)
                        if err != nil {
                                log.Println("SearchPage haberler query error:", err)
                        }
                        if rows2 != nil {
                                for rows2.Next() {
                                        var t, u, e string
                                        rows2.Scan(&t, &u, &e)
                                        results = append(results, SearchResult{Title: t, Url: "/haberler/" + u, Excerpt: e, Type: "Haber"})
                                }
                                rows2.Close()
                        }

                        rows3, err := db.Query(`SELECT CONCAT(title,' ',first_name,' ',last_name), url_name, COALESCE(LEFT(biography,150),'') FROM doktorlar WHERE (title ILIKE $1 OR first_name ILIKE $1 OR last_name ILIKE $1 OR biography ILIKE $1) AND is_active=true LIMIT 10`, like)
                        if err != nil {
                                log.Println("SearchPage doktorlar query error:", err)
                        }
                        if rows3 != nil {
                                for rows3.Next() {
                                        var t, u, e string
                                        rows3.Scan(&t, &u, &e)
                                        results = append(results, SearchResult{Title: t, Url: "/doktorlarimiz/" + u, Excerpt: e, Type: "Doktor"})
                                }
                                rows3.Close()
                        }

                        rows4, err := db.Query(`SELECT name, url_name, COALESCE(LEFT(description,150),'') FROM tedkikler WHERE (name ILIKE $1 OR description ILIKE $1) AND is_active=true LIMIT 10`, like)
                        if err != nil {
                                log.Println("SearchPage tedkikler query error:", err)
                        }
                        if rows4 != nil {
                                for rows4.Next() {
                                        var t, u, e string
                                        rows4.Scan(&t, &u, &e)
                                        results = append(results, SearchResult{Title: t, Url: "/tetkikler/" + u, Excerpt: e, Type: "Tetkik"})
                                }
                                rows4.Close()
                        }
                }

                return c.Render("views/frontend/arama", fiber.Map{
                        "PathOnStart": "",
                        "Route":       "/arama",
                        "Options":     Options,
                        "User":        OurUser,
                        "Query":       q,
                        "Results":     results,
                        "Title":       "Arama | " + Options.Options.SiteName,
                        "Description": "Site içi arama sonuçları",
                }, "layouts/main/main")
        }
}

// RobotsTxt — arama motorları için robots.txt dosyası döndürür.
// PageSpeed'in 1.235 satır hata saymasının ana sebebi bu route'un
// eksikliğiydi: fallback handler tüm HTML'i text/plain olarak döndürüyordu.
func RobotsTxt() fiber.Handler {
	return func(c *fiber.Ctx) error {
		robotsContent := `User-agent: *
Allow: /
Disallow: /panel/
Disallow: /backend/
Disallow: /giris

Sitemap: https://nivgoz.com/sitemap.xml`

		c.Set("Content-Type", "text/plain; charset=utf-8")
		c.Set("Cache-Control", "public, max-age=86400")
		return c.SendString(robotsContent)
	}
}

// SitemapXml — arama motorları için dinamik sitemap.xml oluşturur.
// Tüm frontend sayfalarını, doktorları, tıbbi birimleri, tetkikleri,
// haberleri ve şubeleri içerir.
func SitemapXml(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		Orm := utilities.Orm

		baseURL := "https://nivgoz.com"

		// Sabit sayfalar
		staticPages := []string{
			"/", "/kurumsal/hakkimizda", "/kurumsal/misyon-vizyon",
			"/iletisim", "/ulasim", "/randevu",
			"/kurumsal/anlasmali-kurumlar", "/kurumsal/organizasyon-semasi",
			"/kurumsal/kvkk", "/kurumsal/insan-kaynaklari",
			"/haberler", "/tibbi-birimler", "/tetkikler",
			"/merkezlerimiz", "/doktorlarimiz",
			"/foto-galeri", "/video-galeri",
			"/cerez-politikasi",
		}

		var sb strings.Builder
		sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
		sb.WriteString("\n")
		sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
		sb.WriteString("\n")

		// Sabit sayfalar
		for _, page := range staticPages {
			sb.WriteString("  <url>\n")
			sb.WriteString("    <loc>" + baseURL + page + "</loc>\n")
			sb.WriteString("    <changefreq>weekly</changefreq>\n")
			if page == "/" {
				sb.WriteString("    <priority>1.0</priority>\n")
			} else {
				sb.WriteString("    <priority>0.8</priority>\n")
			}
			sb.WriteString("  </url>\n")
		}

		// Tıbbi birimler
		GetTibbiBirimler := Orm.Select([]string{"url_name"})
		GetTibbiBirimler.Table("tibbi_birimler")
		GetTibbiBirimler.Where("is_active", "=", true)
		GetTibbiBirimler.Finish()
		if err := GetTibbiBirimler.Execute(); err == nil {
			if rows, err := GetTibbiBirimler.Rows(); err == nil {
				for _, row := range rows {
					urlName := lib.String(row["url_name"])
					if urlName != "" {
						sb.WriteString("  <url>\n")
						sb.WriteString("    <loc>" + baseURL + "/tibbi-birimler/" + urlName + "</loc>\n")
						sb.WriteString("    <changefreq>monthly</changefreq>\n")
						sb.WriteString("    <priority>0.7</priority>\n")
						sb.WriteString("  </url>\n")
					}
				}
			}
		}

		// Tetkikler
		GetTetkikler := Orm.Select([]string{"url_name"})
		GetTetkikler.Table("tedkikler")
		GetTetkikler.Where("is_active", "=", true)
		GetTetkikler.Finish()
		if err := GetTetkikler.Execute(); err == nil {
			if rows, err := GetTetkikler.Rows(); err == nil {
				for _, row := range rows {
					urlName := lib.String(row["url_name"])
					if urlName != "" {
						sb.WriteString("  <url>\n")
						sb.WriteString("    <loc>" + baseURL + "/tetkikler/" + urlName + "</loc>\n")
						sb.WriteString("    <changefreq>monthly</changefreq>\n")
						sb.WriteString("    <priority>0.7</priority>\n")
						sb.WriteString("  </url>\n")
					}
				}
			}
		}

		// Şubeler
		GetSubeler := Orm.Select([]string{"url_name"})
		GetSubeler.Table("subeler")
		GetSubeler.Where("is_active", "=", true)
		GetSubeler.Finish()
		if err := GetSubeler.Execute(); err == nil {
			if rows, err := GetSubeler.Rows(); err == nil {
				for _, row := range rows {
					urlName := lib.String(row["url_name"])
					if urlName != "" {
						sb.WriteString("  <url>\n")
						sb.WriteString("    <loc>" + baseURL + "/merkezlerimiz/" + urlName + "</loc>\n")
						sb.WriteString("    <changefreq>monthly</changefreq>\n")
						sb.WriteString("    <priority>0.7</priority>\n")
						sb.WriteString("  </url>\n")
					}
				}
			}
		}

		// Haberler
		GetHaberler := Orm.Select([]string{"url_name"})
		GetHaberler.Table("haberler")
		GetHaberler.Where("is_published", "=", true)
		GetHaberler.Finish()
		if err := GetHaberler.Execute(); err == nil {
			if rows, err := GetHaberler.Rows(); err == nil {
				for _, row := range rows {
					urlName := lib.String(row["url_name"])
					if urlName != "" {
						sb.WriteString("  <url>\n")
						sb.WriteString("    <loc>" + baseURL + "/haberler/" + urlName + "</loc>\n")
						sb.WriteString("    <changefreq>weekly</changefreq>\n")
						sb.WriteString("    <priority>0.6</priority>\n")
						sb.WriteString("  </url>\n")
					}
				}
			}
		}

		sb.WriteString("</urlset>")

		c.Set("Content-Type", "application/xml; charset=utf-8")
		c.Set("Cache-Control", "public, max-age=3600")
		return c.SendString(sb.String())
	}
}
