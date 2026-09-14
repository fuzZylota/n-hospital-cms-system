package anlasmali_kurumlar

import (
	"database"
	
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gofiber/fiber/v2"
)

func AddAnlasmaliKurum(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			log.Printf("Cannot check auth: %v\n", err)
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			log.Printf("User is not admin")
			return c.Redirect("/panel/anlasmali-kurumlar?error=only_admins_can_add_contract_partners")
		}

		inputs := models.AnlasmaliKurumlar{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Name == "" {
			log.Printf("Name is required")
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=name_required")
		}
		if inputs.Type == "" {
			log.Printf("Type is required")
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=type_required")
		}

		if inputs.Sid == "" {
			log.Printf("Sid is required")
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=sid_required")
		}

		columns := []string{"name", "type", "sid", "is_active"}
		values := []interface{}{inputs.Name, inputs.Type, inputs.Sid, inputs.IsActive}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.ContactPerson != "" {
			columns = append(columns, "contact_person")
			values = append(values, inputs.ContactPerson)
		}
		if inputs.Phone != "" {
			columns = append(columns, "phone")
			values = append(values, inputs.Phone)
		}
		if inputs.Email != "" {
			columns = append(columns, "email")
			values = append(values, inputs.Email)
		}
		if inputs.Address != "" {
			columns = append(columns, "address")
			values = append(values, inputs.Address)
		}
		if !inputs.ContractStartDate.IsZero() {
			columns = append(columns, "contract_start_date")
			values = append(values, inputs.ContractStartDate)
		}
		if !inputs.ContractEndDate.IsZero() {
			columns = append(columns, "contract_end_date")
			values = append(values, inputs.ContractEndDate)
		}
		if inputs.DiscountRate > 0 {
			columns = append(columns, "discount_rate")
			values = append(values, inputs.DiscountRate)
		}
		if inputs.PaymentTerms != "" {
			columns = append(columns, "payment_terms")
			values = append(values, inputs.PaymentTerms)
		}
		if inputs.Notes != "" {
			columns = append(columns, "notes")
			values = append(values, inputs.Notes)
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
		}

		insertAnlasmaliKurum := Orm.Insert(columns, values)
		insertAnlasmaliKurum.Table("anlasmali_kurumlar")
		insertAnlasmaliKurum.Returning("akid")
		insertAnlasmaliKurum.Finish()
		err = insertAnlasmaliKurum.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert contract partner: %v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
		}

		akid, err := insertAnlasmaliKurum.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
		}

		// Handle file upload if provided
		var AnlasmaliKurumMediaMid string = ""
		anlasmaliKurumMediaInput, err := c.FormFile("logo_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
			}

			// File size validation (5MB max)
			if anlasmaliKurumMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", anlasmaliKurumMediaInput.Size)
				return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "anlasmali-kurumlar", akid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + anlasmaliKurumMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", UniqueFilePath.Extension)
				return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/anlasmali-kurumlar/" + akid + "/" + UniqueFilePath.BaseName,
				FileSize: anlasmaliKurumMediaInput.Size,
				MimeType: anlasmaliKurumMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: akid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.LogoAltText,
				Title:   inputs.LogoTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			AnlasmaliKurumMediaMid, err = OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *anlasmaliKurumMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
			}

			// Update contract partner with media ID
			if AnlasmaliKurumMediaMid != "" {
				updateAnlasmaliKurum := Orm.Update()
				updateAnlasmaliKurum.Table("anlasmali_kurumlar")
				updateAnlasmaliKurum.Set("logo_mid", AnlasmaliKurumMediaMid)
				updateAnlasmaliKurum.Where("akid", "=", akid)
				updateAnlasmaliKurum.Finish()
				err = updateAnlasmaliKurum.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update contract partner with media: %v\n", err)
					return c.Redirect("/panel/anlasmali-kurumlar/anlasmali-kurumlar-ekle?error=internal_server_error")
				}
			}
		}

		Orm.Commit()

		return c.Redirect("/panel/anlasmali-kurumlar/" + akid)
	}
}

func EditAnlasmaliKurum(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Akid := c.Params("akid")

		inputs := models.AnlasmaliKurumlarEdit{}
		c.BodyParser(&inputs)
		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Kurum adı zorunludur.",
			})
		}
		if inputs.Type == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Kurum tipi zorunludur.",
			})
		}

		// Email validation
		if inputs.Email != "" && inputs.Email != inputs.OldEmail {
			emailRegex := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
			if !emailRegex.MatchString(inputs.Email) {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Geçerli bir e-posta adresi girin.",
				})
			}
		}

		// Discount rate validation
		if inputs.DiscountRate < 0 || inputs.DiscountRate > 100 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "İndirim oranı 0 ile 100 arasında olmalıdır.",
			})
		}

		Orm := utilities.Orm

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		updateAnlasmaliKurum := Orm.Update()
		updateAnlasmaliKurum.Table("anlasmali_kurumlar")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateAnlasmaliKurum.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateAnlasmaliKurum.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.Sid != inputs.OldSid {
			updateAnlasmaliKurum.Set("sid", inputs.Sid)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateAnlasmaliKurum.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.Type != inputs.OldType && inputs.Type != "" {
			updateAnlasmaliKurum.Set("type", inputs.Type)
			SomethingSet = true
		}

		if inputs.ContactPerson != inputs.OldContactPerson {
			updateAnlasmaliKurum.Set("contact_person", inputs.ContactPerson)
			SomethingSet = true
		}

		if inputs.Phone != inputs.OldPhone {
			updateAnlasmaliKurum.Set("phone", inputs.Phone)
			SomethingSet = true
		}

		if inputs.Email != inputs.OldEmail {
			updateAnlasmaliKurum.Set("email", inputs.Email)
			SomethingSet = true
		}

		if inputs.Address != inputs.OldAddress {
			updateAnlasmaliKurum.Set("address", inputs.Address)
			SomethingSet = true
		}

		if inputs.ContractStartDate != inputs.OldContractStartDate {
			updateAnlasmaliKurum.Set("contract_start_date", inputs.ContractStartDate)
			SomethingSet = true
		}

		if inputs.ContractEndDate != inputs.OldContractEndDate {
			updateAnlasmaliKurum.Set("contract_end_date", inputs.ContractEndDate)
			SomethingSet = true
		}

		if inputs.DiscountRate != inputs.OldDiscountRate {
			updateAnlasmaliKurum.Set("discount_rate", inputs.DiscountRate)
			SomethingSet = true
		}

		if inputs.PaymentTerms != inputs.OldPaymentTerms {
			updateAnlasmaliKurum.Set("payment_terms", inputs.PaymentTerms)
			SomethingSet = true
		}

		if inputs.Notes != inputs.OldNotes {
			updateAnlasmaliKurum.Set("notes", inputs.Notes)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateAnlasmaliKurum.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		if SomethingSet {
			updateAnlasmaliKurum.Set("updated_at", "NOW()")
			updateAnlasmaliKurum.Where("akid", "=", Akid)
			updateAnlasmaliKurum.Finish()
			err = updateAnlasmaliKurum.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update anlasmali kurum: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Anlaşmalı kurum başarıyla güncellendi.",
		})
	}
}

func DeleteAnlasmaliKurum(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		AnlasmaliKurumId := c.Params("akid")
		if AnlasmaliKurumId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Contract partner ID is required",
			})
		}

		Orm := utilities.Orm

		// Fetch contract partner media information before deletion
		GetAnlasmaliKurumMedia := Orm.Select([]string{"logo_mid"})
		GetAnlasmaliKurumMedia.Table("anlasmali_kurumlar")
		GetAnlasmaliKurumMedia.Where("akid", "=", AnlasmaliKurumId)
		GetAnlasmaliKurumMedia.Finish()
		err = GetAnlasmaliKurumMedia.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		anlasmaliKurumRows, err := GetAnlasmaliKurumMedia.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(anlasmaliKurumRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Anlaşmalı kurum bulunamadı.",
			})
		}

		var mediaId int64
		if anlasmaliKurumRows[0]["logo_mid"] != nil {
			mediaId = lib.Int64(anlasmaliKurumRows[0]["logo_mid"])
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete the anlasmali kurum
		DeleteAnlasmaliKurum := Orm.Delete()
		DeleteAnlasmaliKurum.Table("anlasmali_kurumlar")
		DeleteAnlasmaliKurum.Where("akid", "=", AnlasmaliKurumId)
		DeleteAnlasmaliKurum.Finish()
		err = DeleteAnlasmaliKurum.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteAnlasmaliKurum.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Anlaşmalı kurum bulunamadı.",
			})
		}

		// Delete related media files if exists
		if mediaId > 0 {
			// Get media file path
			GetMediaPath := Orm.Select([]string{"file_path"})
			GetMediaPath.Table("medias")
			GetMediaPath.Where("mid", "=", mediaId)
			GetMediaPath.Finish()
			err = GetMediaPath.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			mediaRows, err := GetMediaPath.Rows()

			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if len(mediaRows) == 0 {
				Orm.Rollback()
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Anlaşmalı kurum logosu bulunamadı.",
				})
			}

			// Delete media record
			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.Where("mid", "=", mediaId)
			DeleteMedia.Finish()
			err = DeleteMedia.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := DeleteMedia.RowsAffected()
			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if ra == 0 {
				Orm.Rollback()
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Anlaşmalı kurum logosu bulunamadı.",
				})
			}

			filePath := lib.String(mediaRows[0]["file_path"])

			if filePath != "" {
				// Delete physical file
				RootDir := os.Getenv("ROOT_DIRECTORY")

				if RootDir == "" {
					Orm.Rollback()
					log.Printf("ROOT_DIRECTORY not set")
					return c.Status(500).JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				fullPath := filepath.Join(RootDir, "static", filePath)
				err = lib.DeleteFile(fullPath)
				if err != nil {
					log.Printf("Error deleting file %s: %v\n", fullPath, err)
				}
			}

		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Anlaşmalı kurum başarıyla silindi.",
		})
	}
}

func UpdateAnlasmaliKurumPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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



		AnlasmaliKurumId := c.Params("akid")
		anlasmaliKurumMediaAltText := c.FormValue("logo_alt_text")
		anlasmaliKurumMediaTitle := c.FormValue("logo_title")
		oldAnlasmaliKurumMediaAltText := c.FormValue("old_logo_alt_text")
		oldAnlasmaliKurumMediaTitle := c.FormValue("old_logo_title")

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
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		anlasmaliKurumMediaInput, err := c.FormFile("logo_path")
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
			if anlasmaliKurumMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "anlasmali-kurumlar", AnlasmaliKurumId)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + anlasmaliKurumMediaInput.Filename)

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
				FilePath: "files/anlasmali-kurumlar/" + AnlasmaliKurumId + "/" + UniqueFilePath.BaseName,
				FileSize: anlasmaliKurumMediaInput.Size,
				MimeType: anlasmaliKurumMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: AnlasmaliKurumId,
			}

			optionals := models.MediaOptionals{
				AltText: anlasmaliKurumMediaAltText,
				Title:   anlasmaliKurumMediaTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			AnlasmaliKurumMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/anlasmali-kurumlar/"+AnlasmaliKurumId, entry.Name())
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

			// Update contract partner with new media ID
			UpdateAnlasmaliKurum := Orm.Update()
			UpdateAnlasmaliKurum.Table("anlasmali_kurumlar")
			UpdateAnlasmaliKurum.Set("logo_mid", AnlasmaliKurumMediaMid)
			UpdateAnlasmaliKurum.Where("akid", "=", AnlasmaliKurumId)
			UpdateAnlasmaliKurum.Finish()
			err = UpdateAnlasmaliKurum.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update contract partner: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *anlasmaliKurumMediaInput)
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
		} else {
			// Metadata only update case
			if anlasmaliKurumMediaAltText != oldAnlasmaliKurumMediaAltText || anlasmaliKurumMediaTitle != oldAnlasmaliKurumMediaTitle {
				// Get current media ID
				GetAnlasmaliKurumMedia := Orm.Select([]string{"logo_mid"})
				GetAnlasmaliKurumMedia.Table("anlasmali_kurumlar")
				GetAnlasmaliKurumMedia.Where("akid", "=", AnlasmaliKurumId)
				GetAnlasmaliKurumMedia.Finish()
				err = GetAnlasmaliKurumMedia.Execute()

				if err != nil {
					log.Printf("Cannot get contract partner media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetAnlasmaliKurumMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Anlaşmalı kurum bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["logo_mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if anlasmaliKurumMediaAltText != oldAnlasmaliKurumMediaAltText {
						if anlasmaliKurumMediaAltText != "" {
							UpdateMedia.Set("alt_text", anlasmaliKurumMediaAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if anlasmaliKurumMediaTitle != oldAnlasmaliKurumMediaTitle {
						if anlasmaliKurumMediaTitle != "" {
							UpdateMedia.Set("title", anlasmaliKurumMediaTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", AnlasmaliKurumId)
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
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Anlaşmalı kurum logosu başarıyla güncellendi.",
		})
	}
}

func DeleteAnlasmaliKurumPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		AnlasmaliKurumId := c.Params("akid")
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
		GetAnlasmaliKurumMedia := Orm.Select([]string{"ak.logo_mid", "m.file_path"})
		GetAnlasmaliKurumMedia.Table("anlasmali_kurumlar ak")
		GetAnlasmaliKurumMedia.LeftJoin("medias m", "ak.logo_mid", "=", "m.mid")
		GetAnlasmaliKurumMedia.Where("ak.akid", "=", AnlasmaliKurumId)
		GetAnlasmaliKurumMedia.Finish()
		err = GetAnlasmaliKurumMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get contract partner media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAnlasmaliKurumMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Anlaşmalı kurum bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["logo_mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Anlaşmalı kurum logosu bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", AnlasmaliKurumId)
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
				"message": "Anlaşmalı kurum logosu bulunamadı.",
			})
		}

		// Update contract partner to remove media reference
		UpdateAnlasmaliKurum := Orm.Update()
		UpdateAnlasmaliKurum.Table("anlasmali_kurumlar")
		UpdateAnlasmaliKurum.Set("logo_mid", nil)
		UpdateAnlasmaliKurum.Where("akid", "=", AnlasmaliKurumId)
		UpdateAnlasmaliKurum.Finish()
		err = UpdateAnlasmaliKurum.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update contract partner: %v\n", err)
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

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Anlaşmalı kurum logosu başarıyla silindi.",
		})
	}
}

func GetAllAnlasmaliKurumlar(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Orm := utilities.Orm
		GetAllAnlasmaliKurumlar := Orm.Select([]string{"akid", "name"})
		GetAllAnlasmaliKurumlar.Table("anlasmali_kurumlar")
		GetAllAnlasmaliKurumlar.Finish()

		err = GetAllAnlasmaliKurumlar.Execute()

		if err != nil {
			log.Printf("Cannot get all anlasmali kurumlar: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAllAnlasmaliKurumlar.Rows()
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Anlaşmalı kurum bulunamadı.",
			})
		}

		AnlasmaliKurumlar := []models.AnlasmaliKurumlar{}
		for _, row := range rows {
			AnlasmaliKurumlar = append(AnlasmaliKurumlar, models.AnlasmaliKurumlar{
				Akid: lib.String(row["akid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Anlaşmalı kurumlar başarıyla getirildi.",
			"data":    AnlasmaliKurumlar,
		})
	}
}

func GetAnlasmaliKurumlarWithPagination(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, _ := lib.CheckAuth(c)

		Orm := utilities.Orm

		CurrentOptions := models.FrontendOptions{
			Database: Orm,
			User:     OurUser,
			States:   states,
		}

		Options := database.Options{}
		Options, _ = Options.FetchOptionsForFrontendWithCache(&CurrentOptions)

		inputs := models.PaginateAnlasmaliKurumlarInputs{}
		if err := c.BodyParser(&inputs); err != nil {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid inputs",
			})
		}

		if inputs.Offset < 0 {
			inputs.Offset = 0
		}

		columns := []string{"a.akid", "a.name", "a.discount_rate", "a.logo_mid", "s.name as sube_name", "s.sid as sube_sid"}

		if Options.Options.ShowAnlasmaliKurumPictures {
			columns = append(columns, "m.file_path as logo_path", "m.alt_text as logo_alt_text", "m.title as logo_title")
		}

		GetAnlasmaliKurumlar := Orm.Select(columns)
		GetAnlasmaliKurumlar.Table("anlasmali_kurumlar a")
		GetAnlasmaliKurumlar.InnerJoin("subeler s", "a.sid", "=", "s.sid")

		if Options.Options.ShowAnlasmaliKurumPictures {
			GetAnlasmaliKurumlar.LeftJoin("medias m", "a.logo_mid", "=", "m.mid")
		}

		GetAnlasmaliKurumlar.Where("a.is_active", "=", true)
		GetAnlasmaliKurumlar.OrderBy("a.name", "ASC")
		GetAnlasmaliKurumlar.Offset(int(inputs.Offset))
		GetAnlasmaliKurumlar.Limit(int(Options.Options.ItemsPerPage))
		GetAnlasmaliKurumlar.Finish()

		err := GetAnlasmaliKurumlar.Execute()
		if err != nil {
			log.Printf("Cannot get anlasmali kurumlar with pagination: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAnlasmaliKurumlar.Rows()
		if err != nil {
			log.Printf("Cannot get anlasmali kurumlar with pagination: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		AnlasmaliKurumlar := []models.AnlasmaliKurumlar{}
		for _, row := range rows {
			AnlasmaliKurumlar = append(AnlasmaliKurumlar, models.AnlasmaliKurumlar{
				Akid:         lib.String(row["akid"]),
				Name:         lib.String(row["name"]),
				DiscountRate: lib.Float64(row["discount_rate"]),
				LogoMid:      lib.Int64(row["logo_mid"]),
				LogoPath:     lib.String(row["logo_path"]),
				LogoAltText:  lib.String(row["logo_alt_text"]),
				LogoTitle:    lib.String(row["logo_title"]),
				SubeName:     lib.String(row["sube_name"]),
				Sid:          lib.String(row["sube_sid"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Anlaşmalı kurumlar başarıyla getirildi.",
			"data":    AnlasmaliKurumlar,
		})
	}
}
