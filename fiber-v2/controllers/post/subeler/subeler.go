package subeler

import (
	"database"
	"fmt"
	lib "lib"
	"log"
	"models"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func AddSube(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.Redirect("/giris")
		}

		// Check if user is admin
		if OurUser.Role != "admin" {
			return c.Redirect("/panel/subeler/sube-ekle?error=only_admins_can_add_branches")
		}

		inputs := models.Subeler{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.Name == "" {
			return c.Redirect("/panel/subeler/sube-ekle?error=name_required")
		}
		if inputs.Address == "" {
			return c.Redirect("/panel/subeler/sube-ekle?error=address_required")
		}
		if inputs.City == "" {
			return c.Redirect("/panel/subeler/sube-ekle?error=city_required")
		}

		columns := []string{"name", "address", "city", "is_active"}
		values := []interface{}{inputs.Name, inputs.Address, inputs.City, inputs.IsActive}

		// Add optional fields if provided
		if inputs.UrlName != "" {
			columns = append(columns, "url_name")
			values = append(values, inputs.UrlName)
		}
		if inputs.Description != "" {
			columns = append(columns, "description")
			values = append(values, inputs.Description)
		}
		if inputs.District != "" {
			columns = append(columns, "district")
			values = append(values, inputs.District)
		}
		if inputs.PostalCode != "" {
			columns = append(columns, "postal_code")
			values = append(values, inputs.PostalCode)
		}
		if inputs.Phone != "" {
			columns = append(columns, "phone")
			values = append(values, inputs.Phone)
		}
		if inputs.Fax != "" {
			columns = append(columns, "fax")
			values = append(values, inputs.Fax)
		}
		if inputs.Email != "" {
			columns = append(columns, "email")
			values = append(values, inputs.Email)
		}
		if inputs.Website != "" {
			columns = append(columns, "website")
			values = append(values, inputs.Website)
		}
		if inputs.Latitude != 0 {
			columns = append(columns, "latitude")
			values = append(values, inputs.Latitude)
		}
		if inputs.Longitude != 0 {
			columns = append(columns, "longitude")
			values = append(values, inputs.Longitude)
		}
		if inputs.WorkingHours != "" {
			columns = append(columns, "working_hours")
			values = append(values, inputs.WorkingHours)
		}
		if inputs.GoogleMapIframe != "" {
			columns = append(columns, "google_map_iframe")
			values = append(values, inputs.GoogleMapIframe)
		}
		if inputs.TransportationInfo != "" {
			columns = append(columns, "transportation_info")
			values = append(values, inputs.TransportationInfo)
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
		}

		insertSube := Orm.Insert(columns, values)
		insertSube.Table("subeler")
		insertSube.Returning("sid")
		insertSube.Finish()
		err = insertSube.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert branch: %v\n", err)
			return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
		}

		sid, err := insertSube.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
		}

		if inputs.IsMain && sid != "" {
			SetMainSube := Orm.SelectFunction("set_sube_as_main", sid, "INSERT")
			SetMainSube.Finish()

			err = SetMainSube.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}

			rows, err := SetMainSube.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}

			if rows[0]["result"] == false {
				Orm.Rollback()
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}
		}

		// Handle file upload if provided
		var SubeMediaMid string = ""
		subeMediaInput, err := c.FormFile("mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}

			// File size validation (5MB max)
			if subeMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				log.Printf("File size is too large: %v\n", subeMediaInput.Size)
				return c.Redirect("/panel/subeler/sube-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "subeler", sid, "picture")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + subeMediaInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				log.Printf("Invalid file type: %v\n", UniqueFilePath.Extension)
				return c.Redirect("/panel/subeler/sube-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/subeler/" + sid + "/picture/" + UniqueFilePath.BaseName,
				FileSize: subeMediaInput.Size,
				MimeType: subeMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: sid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.SubeMediaAltText,
				Title:   inputs.SubeMediaTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			SubeMediaMid, err = OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert media: %v\n", err)
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}

			err = lib.SaveFileWithBuffering(estimatedPath, *subeMediaInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save file with buffering: %v\n", err)
				return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
			}

			// Update branch with media ID
			if SubeMediaMid != "" {
				updateSube := Orm.Update()
				updateSube.Table("subeler")
				updateSube.Set("mid", SubeMediaMid)
				updateSube.Where("sid", "=", sid)
				updateSube.Finish()
				err = updateSube.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update branch with media: %v\n", err)
					return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
				}
			}
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.Redirect("/panel/subeler/sube-ekle?error=internal_server_error")
		}

		states.SubelerLinks = []models.SubeLink{}

		return c.Redirect("/panel/subeler/" + sid)
	}
}

func EditSube(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		SubeId := c.Params("sid")
		Orm := utilities.Orm

		inputs := models.SubelerEdit{}
		c.BodyParser(&inputs)
		inputs.Sid = SubeId

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Şube adı zorunludur.",
			})
		}
		if inputs.Address == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Adres zorunludur.",
			})
		}
		if inputs.City == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Şehir zorunludur.",
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

		// URL validation
		if inputs.Website != "" && inputs.Website != inputs.OldWebsite {
			_, err := url.Parse(inputs.Website)
			if err != nil {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Geçerli bir website URL'si girin.",
				})
			}
		}

		// Coordinate validation
		if inputs.Latitude < -90 || inputs.Latitude > 90 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Enlem -90 ile 90 arasında olmalıdır.",
			})
		}
		if inputs.Longitude < -180 || inputs.Longitude > 180 {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Boylam -180 ile 180 arasında olmalıdır.",
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

		updateSube := Orm.Update()
		updateSube.Table("subeler")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName && inputs.Name != "" {
			updateSube.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateSube.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateSube.Set("description", inputs.Description)
			SomethingSet = true
		}

		if inputs.Address != inputs.OldAddress && inputs.Address != "" {
			updateSube.Set("address", inputs.Address)
			SomethingSet = true
		}

		if inputs.City != inputs.OldCity && inputs.City != "" {
			updateSube.Set("city", inputs.City)
			SomethingSet = true
		}

		if inputs.District != inputs.OldDistrict {
			updateSube.Set("district", inputs.District)
			SomethingSet = true
		}

		if inputs.PostalCode != inputs.OldPostalCode {
			updateSube.Set("postal_code", inputs.PostalCode)
			SomethingSet = true
		}

		if inputs.Phone != inputs.OldPhone {
			updateSube.Set("phone", inputs.Phone)
			SomethingSet = true
		}

		if inputs.Fax != inputs.OldFax {
			updateSube.Set("fax", inputs.Fax)
			SomethingSet = true
		}

		if inputs.Email != inputs.OldEmail {
			updateSube.Set("email", inputs.Email)
			SomethingSet = true
		}

		if inputs.Website != inputs.OldWebsite {
			updateSube.Set("website", inputs.Website)
			SomethingSet = true
		}

		if inputs.Latitude != inputs.OldLatitude {
			updateSube.Set("latitude", inputs.Latitude)
			SomethingSet = true
		}

		if inputs.Longitude != inputs.OldLongitude {
			updateSube.Set("longitude", inputs.Longitude)
			SomethingSet = true
		}

		if inputs.WorkingHours != inputs.OldWorkingHours {
			if inputs.WorkingHours == "" {
				updateSube.Set("working_hours", nil)
			} else {
				updateSube.Set("working_hours", inputs.WorkingHours)
			}

			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateSube.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		if inputs.GoogleMapIframe != inputs.OldGoogleMapIframe {
			updateSube.Set("google_map_iframe", inputs.GoogleMapIframe)
			SomethingSet = true
		}

		if inputs.TransportationInfo != inputs.OldTransportationInfo {
			updateSube.Set("transportation_info", inputs.TransportationInfo)
			SomethingSet = true
		}

		if SomethingSet {
			updateSube.Set("updated_at", "NOW()")
			updateSube.Where("sid", "=", SubeId)
			updateSube.Finish()
			err = updateSube.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update sube: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		if inputs.IsActive != inputs.OldIsActive && !inputs.IsActive {
			UnmainSube := Orm.SelectFunction("set_sube_as_main", nil, "UPDATE")
			UnmainSube.Finish()
			err = UnmainSube.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot unmain sube: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			rows, err := UnmainSube.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get rows: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if rows[0]["result"] == false {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Şube bulunamadı.",
				})
			}
		}

		Orm.Commit()

		states.SubelerLinks = []models.SubeLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Şube başarıyla güncellendi.",
		})
	}
}

func SetSubeAsMain(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		SubeId := c.Params("sid")
		if SubeId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Şube ID'si gereklidir.",
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

		UnmainSube := Orm.SelectFunction("set_sube_as_main", SubeId, "UPDATE")
		UnmainSube.Finish()
		err = UnmainSube.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot unmain sube: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := UnmainSube.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if rows[0]["result"] == false {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Şube bulunamadı.",
			})
		}

		err = Orm.Commit()
		if err != nil {
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		states.SubelerLinks = []models.SubeLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Şube başarıyla ana şube olarak ayarlandı.",
		})
	}
}

func DeleteSube(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		SubeId := c.Params("sid")
		if SubeId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Branch ID is required",
			})
		}

		Orm := utilities.Orm

		// Fetch branch media information before deletion
		GetBranchMedia := Orm.Select([]string{"mid"})
		GetBranchMedia.Table("subeler")
		GetBranchMedia.Where("sid", "=", SubeId)
		GetBranchMedia.Finish()
		err = GetBranchMedia.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		branchRows, err := GetBranchMedia.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(branchRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Branch not found",
			})
		}

		var mediaId int64
		if branchRows[0]["mid"] != nil {
			mediaId = lib.Int64(branchRows[0]["mid"])
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

		// Delete related media files if exists
		if mediaId > 0 {
			// Get media file path
			GetMediaPath := Orm.Select([]string{"file_path"})
			GetMediaPath.Table("medias")
			GetMediaPath.Where("mid", "=", mediaId)
			GetMediaPath.Finish()
			err = GetMediaPath.Execute()

			if err != nil {
				log.Printf("Cannot get media path: %v\n", err)
				Orm.Rollback()
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			mediaRows, err := GetMediaPath.Rows()

			if err != nil {
				log.Printf("Cannot get media rows: %v\n", err)
				Orm.Rollback()
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if len(mediaRows) == 0 {
				log.Printf("Media rows are empty")
				Orm.Rollback()
				return c.Status(404).JSON(fiber.Map{
					"status":  404,
					"message": "Şube görseli bulunamadı.",
				})
			}

			// Delete media record
			DeleteMedia := Orm.Delete()
			DeleteMedia.Table("medias")
			DeleteMedia.Where("mid", "=", mediaId)
			DeleteMedia.Finish()
			err = DeleteMedia.Execute()

			if err != nil {
				log.Printf("Cannot delete media record: %v\n", err)
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			filePath := lib.String(mediaRows[0]["file_path"])
			if filePath != "" {
				// Delete physical file
				fullPath := "static/" + filePath
				err = lib.DeleteFile(fullPath)
				if err != nil {
					log.Printf("Error deleting file %s: %v\n", fullPath, err)
				}
			}
		}

		// Delete the branch
		DeleteBranch := Orm.Delete()
		DeleteBranch.Table("subeler")
		DeleteBranch.Where("sid", "=", SubeId)
		DeleteBranch.Finish()
		err = DeleteBranch.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteBranch.RowsAffected()
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
				"message": "Şube bulunamadı.",
			})
		}

		UnmainSube := Orm.SelectFunction("set_sube_as_main", SubeId, "DELETE")
		UnmainSube.Finish()
		err = UnmainSube.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot unmain sube: %v\n", err)
		}

		_, err = UnmainSube.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get rows: %v\n", err)
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

		states.SubelerLinks = []models.SubeLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Branch deleted successfully",
		})
	}
}

func UpdateSubePicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		SubeId := c.Params("sid")
		subeMediaAltText := c.FormValue("sube_media_alt_text")
		subeMediaTitle := c.FormValue("sube_media_title")
		oldSubeMediaAltText := c.FormValue("old_sube_media_alt_text")
		oldSubeMediaTitle := c.FormValue("old_sube_media_title")

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

		subeMediaInput, err := c.FormFile("sube_media_path")
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
			if subeMediaInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "subeler", SubeId, "picture")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + subeMediaInput.Filename)

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
				FilePath: "files/subeler/" + SubeId + "/picture/" + UniqueFilePath.BaseName,
				FileSize: subeMediaInput.Size,
				MimeType: subeMediaInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: SubeId,
			}

			optionals := models.MediaOptionals{
				AltText: subeMediaAltText,
				Title:   subeMediaTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			SubeMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/subeler/"+SubeId+"/picture", entry.Name())
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

			// Update branch with new media ID
			UpdateSube := Orm.Update()
			UpdateSube.Table("subeler")
			UpdateSube.Set("mid", SubeMediaMid)
			UpdateSube.Where("sid", "=", SubeId)
			UpdateSube.Finish()
			err = UpdateSube.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update branch: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *subeMediaInput)
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
			if subeMediaAltText != oldSubeMediaAltText || subeMediaTitle != oldSubeMediaTitle {
				// Get current media ID
				GetSubeMedia := Orm.Select([]string{"mid"})
				GetSubeMedia.Table("subeler")
				GetSubeMedia.Where("sid", "=", SubeId)
				GetSubeMedia.Finish()
				err = GetSubeMedia.Execute()

				if err != nil {
					log.Printf("Cannot get branch media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetSubeMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Şube bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if subeMediaAltText != oldSubeMediaAltText {
						if subeMediaAltText != "" {
							UpdateMedia.Set("alt_text", subeMediaAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if subeMediaTitle != oldSubeMediaTitle {
						if subeMediaTitle != "" {
							UpdateMedia.Set("title", subeMediaTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", SubeId)
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
			"message": "Şube görseli başarıyla güncellendi.",
		})
	}
}

func DeleteSubePicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		SubeId := c.Params("sid")
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
		GetSubeMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetSubeMedia.Table("subeler s")
		GetSubeMedia.LeftJoin("medias m", "s.mid", "=", "m.mid")
		GetSubeMedia.Where("s.sid", "=", SubeId)
		GetSubeMedia.Finish()
		err = GetSubeMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get branch media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetSubeMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Şube bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Şube görseli bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", SubeId)
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
				"message": "Şube görseli bulunamadı.",
			})
		}

		// Update branch to remove media reference
		UpdateSube := Orm.Update()
		UpdateSube.Table("subeler")
		UpdateSube.Set("mid", nil)
		UpdateSube.Where("sid", "=", SubeId)
		UpdateSube.Finish()
		err = UpdateSube.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update branch: %v\n", err)
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
			"message": "Şube görseli başarıyla silindi.",
		})
	}
}

func AddSubeDocuments(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		OurUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{}, []string{})
		if err != nil {
			log.Printf("Cannot get options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		SubeId := c.Params("sid")
		if SubeId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Şube ID'si gereklidir.",
			})
		}

		RootDir := os.Getenv("ROOT_DIRECTORY")

		if RootDir == "" {
			log.Printf("ROOT_DIR is not set")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		GetMaximumValueOfMid := Orm.CustomSelectQuery("SELECT MAX(mid) as max_mid FROM medias")
		GetMaximumValueOfMid.Finish()
		err = GetMaximumValueOfMid.Execute()
		if err != nil {
			log.Printf("Cannot get maximum value of mid: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetMaximumValueOfMid.Rows()
		if err != nil {
			log.Printf("Cannot get maximum value of mid rows: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		maximumValueOfMid := lib.Int64(rows[0]["max_mid"])
		if maximumValueOfMid == 0 {
			maximumValueOfMid = 1
		}

		UploadDir := filepath.Join("static", "files", "subeler", SubeId, "documents")

		Queries := []string{}
		Columns := []string{
			"mid",
			"file_name",
			"file_path",
			"file_size",
			"mime_type",
			"file_type",
			"uid",
			"target_id",
			"data",
		}

		DocumentInfoPairs := []models.AddDocumentInfoPairs{}
		for i := 1; i <= 10; i++ {
			file, err := c.FormFile("file" + strconv.Itoa(i))
			if err != nil {
				continue
			}

			data := c.FormValue("data" + strconv.Itoa(i))

			// Check file size
			maxFileSize := GetOptions.Options.MaxUploadSize
			if file.Size > maxFileSize {
				continue
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "subeler", SubeId, "documents")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + file.Filename)
			if err != nil {
				continue
			}

			values := []any{
				maximumValueOfMid + int64(i),
				UniqueFilePath.BaseName,
				"files/subeler/" + SubeId + "/documents/" + UniqueFilePath.BaseName,
				file.Size,
				file.Header.Get("Content-Type"),
				"document",
				OurUser.Uid,
				SubeId,
			}

			if data != "" {
				values = append(values, data)
			} else {
				values = append(values, nil)
			}

			OurQuery := Orm.Insert(Columns, values)
			OurQuery.Table("medias")
			OurQuery.Finish()

			FullQuery := OurQuery.GetFullQuery()

			GetValuesPart := strings.Split(FullQuery, "VALUES")[1]
			Queries = append(Queries, GetValuesPart)

			DocumentInfoPairs = append(DocumentInfoPairs, models.AddDocumentInfoPairs{
				Mid:      maximumValueOfMid + int64(i),
				FilePath: "files/subeler/" + SubeId + "/documents/" + UniqueFilePath.BaseName,
				Data:     data,
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

		if len(Queries) > 0 {
			InsertQuery := "INSERT INTO medias (mid, file_name, file_path, file_size, mime_type, file_type, uid, target_id, data) VALUES " + strings.ReplaceAll(strings.Join(Queries, ","), ";", "") + " RETURNING mid;"

			CustomInsertQuery := Orm.CustomInsertQuery(InsertQuery)
			CustomInsertQuery.Finish()
			err = CustomInsertQuery.Execute()
			if err != nil {
				log.Printf("Cannot execute custom insert query: %v\n", err)
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ActuallyAddedMids := []string{}
			for i := 1; i <= 10; i++ {
				file, err := c.FormFile("file" + strconv.Itoa(i))

				if err != nil {
					continue
				}

				filePath := filepath.Join(RootDir, UploadDir)

				uniquePathResponse, err := lib.UniqueFilePath(filePath + "/" + file.Filename)
				if err != nil {
					continue
				}

				for _, documentInfoPair := range DocumentInfoPairs {
					if documentInfoPair.FilePath == "files/subeler/"+SubeId+"/documents/"+uniquePathResponse.BaseName {
						err = lib.SaveFileWithBufferingWithRenaming(filePath, uniquePathResponse.BaseName, *file)
						if err != nil {
							log.Printf("Cannot save file: %v\n", err)
							continue
						}

						ActuallyAddedMids = append(ActuallyAddedMids, strconv.FormatInt(documentInfoPair.Mid, 10))

						break
					}
				}
			}

			UpdateDocumentMids := Orm.Update()
			UpdateDocumentMids.Table("subeler")
			UpdateDocumentMids.SetExpr("document_mids", "COALESCE(document_mids, '{}') || ARRAY["+strings.Join(ActuallyAddedMids, ",")+"]")
			UpdateDocumentMids.Where("sid", "=", SubeId)
			UpdateDocumentMids.Finish()

			err = UpdateDocumentMids.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update document mids: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := UpdateDocumentMids.RowsAffected()
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
					"message": "Şube bulunamadı.",
				})
			}

			Orm.Commit()

			states.SubelerLinks = []models.SubeLink{}
		} else {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Dosya bilgisi gereklidir.",
			})
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Dosya başarıyla yüklendi.",
		})
	}
}

func EditSubeDocument(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		inputs := models.EditDocumentInputs{}
		err = c.BodyParser(&inputs)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Dosya bilgisi gereklidir.",
			})
		}

		Orm := utilities.Orm

		UpdateMedia := Orm.Update()
		UpdateMedia.Table("medias")
		UpdateMedia.Set("data", inputs.Data)
		UpdateMedia.Where("mid", "=", inputs.DocumentMid)
		UpdateMedia.Finish()
		err = UpdateMedia.Execute()
		if err != nil {
			log.Printf("Cannot update media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := UpdateMedia.RowsAffected()
		if err != nil {
			log.Printf("Cannot get rows affected: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Döküman bulunamadı.",
			})
		}

		states.SubelerLinks = []models.SubeLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Dosya başarıyla güncellendi.",
		})
	}
}

func DeleteSubeDocument(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		inputs := models.DeleteDocumentInputs{}
		err = c.BodyParser(&inputs)
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Dosya bilgisi gereklidir.",
			})
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{}, []string{})
		if err != nil {
			log.Printf("Cannot get options: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		SubeId := c.Params("sid")
		if SubeId == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Şube ID'si gereklidir.",
			})
		}

		// Get current media information
		GetSubeMedia := Orm.Select([]string{"m.file_path"})
		GetSubeMedia.Table("medias m")
		GetSubeMedia.Where("m.mid", "=", inputs.DocumentMid)
		GetSubeMedia.And("m.file_type", "=", "document")
		GetSubeMedia.And("m.target_id", "=", SubeId)
		GetSubeMedia.Finish()
		err = GetSubeMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get branch media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetSubeMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Döküman bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", inputs.DocumentMid)
		DeleteMedia.And("target_id", "=", SubeId)
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
				"message": "Şube görseli bulunamadı.",
			})
		}

		DeleteString := fmt.Sprintf("array_remove(document_mids, %s)", inputs.DocumentMid)

		// Update branch to remove media reference
		UpdateSube := Orm.Update()
		UpdateSube.Table("subeler")
		UpdateSube.SetExpr("document_mids", DeleteString)
		UpdateSube.Where("sid", "=", SubeId)
		UpdateSube.Finish()
		err = UpdateSube.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update branch: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete physical file
		if rows[0]["file_path"] != nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			fullPath := filepath.Join(RootDir, "static", lib.String(rows[0]["file_path"]))
			err = lib.DeleteFile(fullPath)
			if err != nil {
				log.Printf("Cannot delete physical file: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		Orm.Commit()

		states.SubelerLinks = []models.SubeLink{}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Dosya başarıyla yüklendi.",
		})
	}
}

func GetAllSubeler(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Orm := utilities.Orm
		GetAllSubeler := Orm.Select([]string{"sid", "name"})
		GetAllSubeler.Table("subeler")
		GetAllSubeler.Finish()

		err = GetAllSubeler.Execute()

		if err != nil {
			log.Printf("Cannot get all subeler: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAllSubeler.Rows()
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Şube bulunamadı.",
			})
		}

		Subeler := []models.Subeler{}
		for _, row := range rows {
			Subeler = append(Subeler, models.Subeler{
				Sid:  lib.String(row["sid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Şubeler başarıyla getirildi.",
			"data":    Subeler,
		})
	}
}

func GetSubeDoctors(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Sid := c.Params("sid")
		if Sid == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Sube ID is required",
			})
		}

		Orm := utilities.Orm

		GetSubeDoctors := Orm.Select([]string{"drid", "first_name", "last_name", "title"})
		GetSubeDoctors.Table("doktorlar")
		GetSubeDoctors.Where("sid", "=", Sid)
		GetSubeDoctors.Finish()

		err = GetSubeDoctors.Execute()

		if err != nil {
			log.Printf("Cannot get sube doctors: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		rows, err := GetSubeDoctors.Rows()
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		Doctors := []models.Doktorlar{}
		for _, row := range rows {
			Doctors = append(Doctors, models.Doktorlar{
				Drid:      lib.String(row["drid"]),
				FirstName: lib.String(row["first_name"]),
				LastName:  lib.String(row["last_name"]),
				Title:     lib.String(row["title"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Doktorlar başarıyla getirildi.",
			"data":    Doctors,
		})
	}
}

func GetBranchesThatFitsIndividualSube(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Sid := c.Params("sid")
		if Sid == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Sube ID is required",
			})
		}

		Orm := utilities.Orm

		GetBranchOfIndividualSube := Orm.Select([]string{"brid", "name"})
		GetBranchOfIndividualSube.Table("branslar")
		GetBranchOfIndividualSube.OpenParenthesis("WHERE")
		GetBranchOfIndividualSube.And("sid", "=", Sid)
		GetBranchOfIndividualSube.Or("sid", "=", nil)
		GetBranchOfIndividualSube.CloseParenthesis()
		GetBranchOfIndividualSube.And("is_active", "=", true)
		GetBranchOfIndividualSube.OrderBy("name", "ASC")
		GetBranchOfIndividualSube.Finish()

		err = GetBranchOfIndividualSube.Execute()

		if err != nil {
			log.Printf("Cannot get branch of individual sube: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		rows, err := GetBranchOfIndividualSube.Rows()
		if err != nil {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Branş bulunamadı.",
			})
		}

		Branches := []models.Branslar{}
		for _, row := range rows {
			Branches = append(Branches, models.Branslar{
				Brid: lib.String(row["brid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Branşlar başarıyla getirildi.",
			"data":    Branches,
		})
	}
}
