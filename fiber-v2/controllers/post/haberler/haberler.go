package haberler

import (
	"database"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func AddNews(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/haberler/haber-ekle?error=only_admins_can_add_news")
		}

		inputs := models.Haberler{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Title == "" {
			return c.Redirect("/panel/haberler/haber-ekle?error=title_required")
		}
		if inputs.Content == "" {
			return c.Redirect("/panel/haberler/haber-ekle?error=content_required")
		}

		columns := []string{"title", "content", "is_featured", "is_published"}
		values := []interface{}{inputs.Title, inputs.Content, inputs.IsFeatured, inputs.IsPublished}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.Summary != "" {
			columns = append(columns, "summary")
			values = append(values, inputs.Summary)
		}
		if inputs.Category != "" {
			columns = append(columns, "category")
			values = append(values, inputs.Category)
		}
		if inputs.Author != "" {
			columns = append(columns, "author")
			values = append(values, inputs.Author)
		}
		if !inputs.PublishDate.IsZero() {
			columns = append(columns, "publish_date")
			values = append(values, inputs.PublishDate)
		}
		if len(inputs.Tags) > 0 {
			columns = append(columns, "tags")
			values = append(values, inputs.Tags)
		}
		if inputs.SeoTitle != "" {
			columns = append(columns, "seo_title")
			values = append(values, inputs.SeoTitle)
		}
		if inputs.SeoDescription != "" {
			columns = append(columns, "seo_description")
			values = append(values, inputs.SeoDescription)
		}
		if inputs.SeoKeywords != "" {
			columns = append(columns, "seo_keywords")
			values = append(values, inputs.SeoKeywords)
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
		}

		insertHaber := Orm.Insert(columns, values)
		insertHaber.Table("haberler")
		insertHaber.Returning("hid")
		insertHaber.Finish()
		err = insertHaber.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert news: %v\n", err)
			return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
		}

		hid, err := insertHaber.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
		}

		// Handle file upload if provided
		var HaberMediaMid string = ""
		haberMediaInput, err := c.FormFile("cover_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
			}

			// File size validation (5MB max)
			if haberMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", haberMediaInput.Size)
				return c.Redirect("/panel/haberler/haber-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "haberler", hid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + haberMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", UniqueFilePath.Extension)
				return c.Redirect("/panel/haberler/haber-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/haberler/" + hid + "/" + UniqueFilePath.BaseName,
				FileSize: haberMediaInput.Size,
				MimeType: haberMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: hid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.CoverAltText,
				Title:   inputs.CoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			HaberMediaMid, err = OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *haberMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
			}

			// Update news with media ID
			if HaberMediaMid != "" {
				updateHaber := Orm.Update()
				updateHaber.Table("haberler")
				updateHaber.Set("cover_mid", HaberMediaMid)
				updateHaber.Where("hid", "=", hid)
				updateHaber.Finish()
				err = updateHaber.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update news with media: %v\n", err)
					return c.Redirect("/panel/haberler/haber-ekle?error=internal_server_error")
				}
			}
		}

		Orm.Commit()

		states.NewsLinks = []models.NewsLink{}

		return c.Redirect("/panel/haberler/" + hid)
	}
}

func EditNews(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Hid := c.Params("hid")
		Orm := utilities.Orm

		inputs := models.HaberlerEdit{}
		c.BodyParser(&inputs)
		inputs.Hid = Hid

		// Required fields validation
		if inputs.Title == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Haber başlığı zorunludur.",
			})
		}
		if inputs.Content == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Haber içeriği zorunludur.",
			})
		}

		// Email validation for author if provided
		if inputs.Author != "" && inputs.Author != inputs.OldAuthor {
			// Author field is just a string, no email validation needed
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		updateHaber := Orm.Update()
		updateHaber.Table("haberler")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Title != inputs.OldTitle && inputs.Title != "" {
			updateHaber.Set("title", inputs.Title)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateHaber.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.Summary != inputs.OldSummary {
			updateHaber.Set("summary", inputs.Summary)
			SomethingSet = true
		}

		if inputs.Content != inputs.OldContent && inputs.Content != "" {
			updateHaber.Set("content", inputs.Content)
			SomethingSet = true
		}

		if inputs.Category != inputs.OldCategory {
			updateHaber.Set("category", inputs.Category)
			SomethingSet = true
		}

		if inputs.Author != inputs.OldAuthor {
			updateHaber.Set("author", inputs.Author)
			SomethingSet = true
		}

		if inputs.PublishDate != inputs.OldPublishDate {
			updateHaber.Set("publish_date", inputs.PublishDate)
			SomethingSet = true
		}

		if inputs.IsFeatured != inputs.OldIsFeatured {
			updateHaber.Set("is_featured", inputs.IsFeatured)
			SomethingSet = true
		}

		if inputs.IsPublished != inputs.OldIsPublished {
			updateHaber.Set("is_published", inputs.IsPublished)
			SomethingSet = true
		}

		if inputs.SeoTitle != inputs.OldSeoTitle {
			updateHaber.Set("seo_title", inputs.SeoTitle)
			SomethingSet = true
		}

		if inputs.SeoDescription != inputs.OldSeoDescription {
			updateHaber.Set("seo_description", inputs.SeoDescription)
			SomethingSet = true
		}

		if inputs.SeoKeywords != inputs.OldSeoKeywords {
			updateHaber.Set("seo_keywords", inputs.SeoKeywords)
			SomethingSet = true
		}

		if SomethingSet {
			updateHaber.Set("updated_at", "NOW()")
			updateHaber.Where("hid", "=", Hid)
			updateHaber.Finish()
			err = updateHaber.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update haber: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		states.NewsLinks = []models.NewsLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Haber başarıyla güncellendi.",
		})
	}
}

func DeleteNews(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		HaberId := c.Params("hid")
		Orm := utilities.Orm

		if HaberId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Haber ID is required",
			})
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// First, get the haber to find the cover_mid
		GetHaber := Orm.Select([]string{"hid", "title", "cover_mid"})
		GetHaber.Table("haberler")
		GetHaber.Where("hid", "=", HaberId)
		GetHaber.Finish()
		err = GetHaber.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get haber: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetHaber.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			log.Printf("Haber not found: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Haber bulunamadı",
			})
		}

		coverMid := lib.Int64(rows[0]["cover_mid"])

		// Delete the haber
		DeleteHaberQuery := Orm.Delete()
		DeleteHaberQuery.Table("haberler")
		DeleteHaberQuery.Where("hid", "=", HaberId)
		DeleteHaberQuery.Finish()
		err = DeleteHaberQuery.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete haber: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// If there's a cover image, delete it and its file
		if coverMid > 0 {
			// Get media file path
			GetMedia := Orm.Select([]string{"mid", "file_path"})
			GetMedia.Table("medias")
			GetMedia.Where("mid", "=", coverMid)
			GetMedia.Finish()
			err = GetMedia.Execute()

			if err == nil {
				mediaRows, err := GetMedia.Rows()
				if err == nil && len(mediaRows) > 0 {
					filePath := lib.String(mediaRows[0]["file_path"])

					// Delete the physical file
					if filePath != "" {
						RootDir := os.Getenv("ROOT_DIRECTORY")
						if RootDir != "" {
							fullPath := RootDir + "/static/" + filePath
							err = lib.DeleteFile(fullPath)
							if err != nil {
								log.Printf("Cannot delete file %s: %v\n", fullPath, err)
							}
						}
					}

					// Delete media record
					DeleteMedia := Orm.Delete()
					DeleteMedia.Table("medias")
					DeleteMedia.Where("mid", "=", coverMid)
					DeleteMedia.Finish()
					err = DeleteMedia.Execute()

					if err != nil {
						log.Printf("Cannot delete media record: %v\n", err)
					}
				}
			}
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		states.NewsLinks = []models.NewsLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Haber başarıyla silindi",
		})
	}
}

func UpdateNewsPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		HaberId := c.Params("hid")
		haberMediaAltText := c.FormValue("haber_media_alt_text")
		haberMediaTitle := c.FormValue("haber_media_title")
		oldHaberMediaAltText := c.FormValue("old_haber_media_alt_text")
		oldHaberMediaTitle := c.FormValue("old_haber_media_title")

		Orm := utilities.Orm
		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("ROOT_DIRECTORY is empty")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		haberMediaInput, err := c.FormFile("haber_media_path")
		if err == nil {
			// File upload case
			err = Orm.Begin()
			if err != nil {
				log.Printf("Cannot begin transaction: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// File size validation (5MB max)
			if haberMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "haberler", HaberId)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + haberMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.",
				})
			}

			// Insert new media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/haberler/" + HaberId + "/" + UniqueFilePath.BaseName,
				FileSize: haberMediaInput.Size,
				MimeType: haberMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: HaberId,
			}

			optionals := models.MediaOptionals{
				AltText: haberMediaAltText,
				Title:   haberMediaTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			HaberMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Delete old media files
			entries, err := lib.ReadDirectory(estimatedPath)
			if err != nil && err != os.ErrNotExist {
				log.Printf("Cannot read directory: %v\n", err)
			}

			FilePaths := []any{}
			for _, entry := range entries {
				FilePath := filepath.Join("files/haberler/"+HaberId, entry.Name())
				FilePaths = append(FilePaths, FilePath)
			}

			if len(FilePaths) > 0 {
				DeleteMedias := Orm.Delete()
				DeleteMedias.Table("medias")
				DeleteMedias.In("WHERE", "file_path", FilePaths)
				DeleteMedias.Finish()
				err = DeleteMedias.Execute()
				if err != nil {
					log.Printf("Cannot delete old media records: %v\n", err)
				}
			}

			// Update news with new media ID
			UpdateHaber := Orm.Update()
			UpdateHaber.Table("haberler")
			UpdateHaber.Set("cover_mid", HaberMediaMid)
			UpdateHaber.Where("hid", "=", HaberId)
			UpdateHaber.Finish()
			err = UpdateHaber.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update news: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *haberMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Delete old physical files
			for _, filePath := range FilePaths {
				GetAbsolutePath := filepath.Join(RootDir, "static", lib.String(filePath))
				err := lib.DeleteFile(GetAbsolutePath)
				if err != nil {
					log.Printf("Cannot remove old file: %v\n", err)
				}
			}

			Orm.Commit()

			states.NewsLinks = []models.NewsLink{}
		} else {
			// Metadata only update case
			if haberMediaAltText != oldHaberMediaAltText || haberMediaTitle != oldHaberMediaTitle {
				// Get current media ID
				GetHaberMedia := Orm.Select([]string{"cover_mid"})
				GetHaberMedia.Table("haberler")
				GetHaberMedia.Where("hid", "=", HaberId)
				GetHaberMedia.Finish()
				err = GetHaberMedia.Execute()

				if err != nil {
					log.Printf("Cannot get news media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetHaberMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Haber bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["cover_mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if haberMediaAltText != oldHaberMediaAltText {
						if haberMediaAltText != "" {
							UpdateMedia.Set("alt_text", haberMediaAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if haberMediaTitle != oldHaberMediaTitle {
						if haberMediaTitle != "" {
							UpdateMedia.Set("title", haberMediaTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", HaberId)
					UpdateMedia.Finish()

					err = UpdateMedia.Execute()
					if err != nil {
						log.Printf("Cannot update media metadata: %v\n", err)
						return c.JSON(fiber.Map{
							"status":  500,
							"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
						})
					}
				}

				states.NewsLinks = []models.NewsLink{}
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Haber görseli başarıyla güncellendi.",
		})
	}
}

func DeleteNewsPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if OurUser.Role != "admin" {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		HaberId := c.Params("hid")
		Orm := utilities.Orm

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Get current media information
		GetHaberMedia := Orm.Select([]string{"h.cover_mid", "m.file_path"})
		GetHaberMedia.Table("haberler h")
		GetHaberMedia.LeftJoin("medias m", "h.cover_mid", "=", "m.mid")
		GetHaberMedia.Where("h.hid", "=", HaberId)
		GetHaberMedia.Finish()
		err = GetHaberMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get news media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetHaberMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Haber bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["cover_mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Haber görseli bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", HaberId)
		DeleteMedia.Finish()
		err = DeleteMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot delete media record: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteMedia.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get rows affected: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Haber görseli bulunamadı.",
			})
		}

		// Update news to remove media reference
		UpdateHaber := Orm.Update()
		UpdateHaber.Table("haberler")
		UpdateHaber.Set("cover_mid", nil)
		UpdateHaber.Where("hid", "=", HaberId)
		UpdateHaber.Finish()
		err = UpdateHaber.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update news: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete physical file
		if filePath != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			fullPath := filepath.Join(RootDir, "static", filePath)
			err = lib.DeleteFile(fullPath)
			if err != nil {
				log.Printf("Cannot delete physical file: %v\n", err)
				// Don't rollback for file deletion errors, just log
				// The database operation was successful
			}
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		states.NewsLinks = []models.NewsLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Haber görseli başarıyla silindi.",
		})
	}
}
