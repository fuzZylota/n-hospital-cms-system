package doctors

import (
	"database"
	"fmt"
	lib "lib"
	"log"
	"models"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AddDoctor(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Redirect("/giris")
		}

		if ourUser.Role != "admin" {
			return c.Redirect("/panel")
		}

		inputs := models.Doktorlar{}
		c.BodyParser(&inputs)

		// Required fields validation
		if inputs.FirstName == "" {
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=first_name_required")
		}
		if inputs.LastName == "" {
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=last_name_required")
		}
		if inputs.UrlName == "" {
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=url_name_required")
		}

		Orm := utilities.Orm

		// Check if UrlName already exists
		CheckIfUrlNameExists := Orm.Count("doktorlar")
		CheckIfUrlNameExists.Where("url_name", "=", inputs.UrlName)
		CheckIfUrlNameExists.Finish()
		err = CheckIfUrlNameExists.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
		}

		if CheckIfUrlNameExists.Length() > 0 {
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=url_name_already_exists")
		}

		// Prepare columns and values for insertion
		columns := []string{"first_name", "last_name", "url_name", "is_active", "created_at", "updated_at"}
		values := []interface{}{inputs.FirstName, inputs.LastName, inputs.UrlName, inputs.IsActive, time.Now(), time.Now()}

		// Add optional fields if provided
		if inputs.Title != "" {
			columns = append(columns, "title")
			values = append(values, inputs.Title)
		}
		if inputs.TcKimlik != "" {
			columns = append(columns, "tc_kimlik")
			values = append(values, inputs.TcKimlik)
		}
		if inputs.DiplomaNo != "" {
			columns = append(columns, "diploma_no")
			values = append(values, inputs.DiplomaNo)
		}
		if inputs.Phone != "" {
			columns = append(columns, "phone")
			values = append(values, inputs.Phone)
		}
		if inputs.Email != "" {
			columns = append(columns, "email")
			values = append(values, inputs.Email)
		}
		if inputs.Biography != "" {
			columns = append(columns, "biography")
			values = append(values, inputs.Biography)
		}
		if inputs.Education != "" {
			columns = append(columns, "education")
			values = append(values, inputs.Education)
		}
		if inputs.ExperienceYears > 0 {
			columns = append(columns, "experience_years")
			values = append(values, inputs.ExperienceYears)
		}
		if inputs.Languages != "" {
			columns = append(columns, "languages")
			values = append(values, inputs.Languages)
		}
		if !inputs.BirthDate.IsZero() {
			columns = append(columns, "birth_date")
			values = append(values, inputs.BirthDate)
		}
		if inputs.Gender != "" {
			columns = append(columns, "gender")
			values = append(values, inputs.Gender)
		}
		if inputs.RoomNumber != "" {
			columns = append(columns, "room_number")
			values = append(values, inputs.RoomNumber)
		}
		if inputs.AppointmentDuration > 0 {
			columns = append(columns, "appointment_duration")
			values = append(values, inputs.AppointmentDuration)
		}
		if inputs.AppointmentFee > 0 {
			columns = append(columns, "appointment_fee")
			values = append(values, inputs.AppointmentFee)
		}
		if inputs.WorkingHours != "" {
			columns = append(columns, "working_hours")
			values = append(values, inputs.WorkingHours)
		}
		if inputs.VacationDates != "" {
			columns = append(columns, "vacation_dates")
			values = append(values, inputs.VacationDates)
		}
		if inputs.FacebookUrl != "" {
			columns = append(columns, "facebook_url")
			values = append(values, inputs.FacebookUrl)
		}
		if inputs.LinkedinUrl != "" {
			columns = append(columns, "linkedin_url")
			values = append(values, inputs.LinkedinUrl)
		}
		if inputs.InstagramUrl != "" {
			columns = append(columns, "instagram_url")
			values = append(values, inputs.InstagramUrl)
		}
		if inputs.XUrl != "" {
			columns = append(columns, "x_url")
			values = append(values, inputs.XUrl)
		}
		if inputs.PersonalUrl != "" {
			columns = append(columns, "personal_url")
			values = append(values, inputs.PersonalUrl)
		}
		if inputs.OnlineAppointment {
			columns = append(columns, "online_appointment")
			values = append(values, inputs.OnlineAppointment)
		}
		if inputs.Brid == "" && inputs.Sid != "" {
			columns = append(columns, "sid")
			values = append(values, inputs.Sid)
		}
		if inputs.CalistigiSubelerText != "" {
			columns = append(columns, "calistigi_subeler_text")
			values = append(values, inputs.CalistigiSubelerText)
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
		}

		// Insert doctor
		insertDoctor := Orm.Insert(columns, values)
		insertDoctor.Table("doktorlar")
		insertDoctor.Returning("drid")
		insertDoctor.Finish()

		err = insertDoctor.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot insert doctor: %v\n", err)
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
		}

		drid, err := insertDoctor.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get last insert id: %v\n", err)
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
		}

		// Get options for file size validation
		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})

		if err != nil {
			log.Printf("Cannot fetch options for backend: %v\n", err)
			Orm.Rollback()
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
		}

		// Handle photo upload if present
		photoInput, err := c.FormFile("photo_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// File size validation
			if photoInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.Status(400).JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu çok büyük.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "doktorlar", drid, "photo")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + photoInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				return c.Status(400).JSON(fiber.Map{
					"status":  400,
					"message": "Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.",
				})
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/doktorlar/" + drid + "/photo/" + UniqueFilePath.BaseName,
				FileSize: photoInput.Size,
				MimeType: photoInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      ourUser.Uid,
				TargetId: drid,
			}

			optionals := models.MediaOptionals{
				AltText: c.FormValue("photo_alt_text"),
				Title:   c.FormValue("photo_title"),
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			PhotoMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert photo media: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Update doctor with photo media ID
			updateDoctor := Orm.Update()
			updateDoctor.Table("doktorlar")
			updateDoctor.Set("photo_mid", PhotoMediaMid)
			updateDoctor.Where("drid", "=", drid)
			updateDoctor.Finish()
			err = updateDoctor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor with photo: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save photo file
			err = lib.SaveFileWithBuffering(estimatedPath, *photoInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save photo file: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		// Handle CV upload if present
		cvInput, err := c.FormFile("cv_file_mid")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			// File size validation
			if cvInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=file_size_is_too_large")
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "doktorlar", drid, "cv")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + cvInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			switch UniqueFilePath.Extension {
			case ".pdf", ".doc", ".docx":
				break
			default:
				Orm.Rollback()
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=invalid_file_type")
			}

			// Insert media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/doktorlar/" + drid + "/cv/" + UniqueFilePath.BaseName,
				FileSize: cvInput.Size,
				MimeType: cvInput.Header.Get("Content-Type"),
				FileType: "document",
				Uid:      ourUser.Uid,
				TargetId: drid,
			}

			optionals := models.MediaOptionals{
				AltText: c.FormValue("cv_alt_text"),
				Title:   c.FormValue("cv_title"),
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			CvMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert CV media: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			// Update doctor with CV media ID
			updateDoctor := Orm.Update()
			updateDoctor.Table("doktorlar")
			updateDoctor.Set("cv_file_mid", CvMediaMid)
			updateDoctor.Where("drid", "=", drid)
			updateDoctor.Finish()
			err = updateDoctor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor with CV: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			// Save CV file
			err = lib.SaveFileWithBuffering(estimatedPath, *cvInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save CV file: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}
		}

		// Handle branch assignment if provided
		if inputs.Brid != "" {
			// Call change_doctor_branch function
			ChangeDoctorBranch := Orm.SelectFunction("change_doctor_branch", inputs.Brid, drid)
			err = ChangeDoctorBranch.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot change doctor branch: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			rows, err := ChangeDoctorBranch.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get change doctor branch rows: %v\n", err)
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			if len(rows) == 0 {
				Orm.Rollback()
				log.Printf("Cannot change doctor branch: no rows returned\n")
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}

			if rows[0]["result"] == "false" {
				Orm.Rollback()
				log.Printf("Cannot change doctor branch: function returned false\n")
				return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
			}
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot commit transaction: %v\n", err)
			return c.Redirect("/panel/doktorlar/doktor-ekle?error=internal_server_error")
		}

		return c.Redirect("/panel/doktorlar/" + drid)
	}
}

func EditDoctor(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
		Orm := utilities.Orm

		inputs := models.DoktorlarEdit{}
		c.BodyParser(&inputs)
		inputs.Drid = Drid

		// Required fields validation
		if inputs.FirstName == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Doktor adı zorunludur.",
			})
		}
		if inputs.LastName == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Doktor soyadı zorunludur.",
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

		// Phone validation
		if inputs.Phone != "" && inputs.Phone != inputs.OldPhone {
			phoneRegex := regexp.MustCompile(`^[\+]?[0-9\s\-\(\)]{10,}$`)
			if !phoneRegex.MatchString(inputs.Phone) {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Geçerli bir telefon numarası girin.",
				})
			}
		}

		// TC Kimlik validation
		if inputs.TcKimlik != "" && inputs.TcKimlik != inputs.OldTcKimlik {
			tcRegex := regexp.MustCompile(`^[0-9]{11}$`)
			if !tcRegex.MatchString(inputs.TcKimlik) {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "TC Kimlik No 11 haneli olmalıdır.",
				})
			}
		}

		err = Orm.Begin()
		if err != nil {
			log.Printf("Cannot begin transaction: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		updateDoktor := Orm.Update()
		updateDoktor.Table("doktorlar")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Title != inputs.OldTitle {
			updateDoktor.Set("title", inputs.Title)
			SomethingSet = true
		}

		if inputs.FirstName != inputs.OldFirstName && inputs.FirstName != "" {
			updateDoktor.Set("first_name", inputs.FirstName)
			SomethingSet = true
		}

		if inputs.LastName != inputs.OldLastName && inputs.LastName != "" {
			updateDoktor.Set("last_name", inputs.LastName)
			SomethingSet = true
		}

		if inputs.UrlName != inputs.OldUrlName {
			updateDoktor.Set("url_name", inputs.UrlName)
			SomethingSet = true
		}

		if inputs.TcKimlik != inputs.OldTcKimlik {
			updateDoktor.Set("tc_kimlik", inputs.TcKimlik)
			SomethingSet = true
		}

		if inputs.DiplomaNo != inputs.OldDiplomaNo {
			updateDoktor.Set("diploma_no", inputs.DiplomaNo)
			SomethingSet = true
		}

		if inputs.Phone != inputs.OldPhone {
			updateDoktor.Set("phone", inputs.Phone)
			SomethingSet = true
		}

		if inputs.Email != inputs.OldEmail {
			updateDoktor.Set("email", inputs.Email)
			SomethingSet = true
		}

		if inputs.Biography != inputs.OldBiography {
			updateDoktor.Set("biography", inputs.Biography)
			SomethingSet = true
		}

		if inputs.Education != inputs.OldEducation {
			updateDoktor.Set("education", inputs.Education)
			SomethingSet = true
		}

		if inputs.ExperienceYears != inputs.OldExperienceYears {
			updateDoktor.Set("experience_years", inputs.ExperienceYears)
			SomethingSet = true
		}

		if inputs.Languages != inputs.OldLanguages {
			updateDoktor.Set("languages", inputs.Languages)
			SomethingSet = true
		}

		if inputs.BirthDate != inputs.OldBirthDate {
			updateDoktor.Set("birth_date", inputs.BirthDate)
			SomethingSet = true
		}

		if inputs.Gender != inputs.OldGender {
			updateDoktor.Set("gender", inputs.Gender)
			SomethingSet = true
		}

		if inputs.RoomNumber != inputs.OldRoomNumber {
			updateDoktor.Set("room_number", inputs.RoomNumber)
			SomethingSet = true
		}

		if inputs.AppointmentDuration != inputs.OldAppointmentDuration {
			updateDoktor.Set("appointment_duration", inputs.AppointmentDuration)
			SomethingSet = true
		}

		if inputs.AppointmentFee != inputs.OldAppointmentFee {
			updateDoktor.Set("appointment_fee", inputs.AppointmentFee)
			SomethingSet = true
		}

		if inputs.CalistigiSubelerText != inputs.OldCalistigiSubelerText {
			updateDoktor.Set("calistigi_subeler_text", inputs.CalistigiSubelerText)
			SomethingSet = true
		}

		if inputs.WorkingHours != inputs.OldWorkingHours {
			if inputs.WorkingHours == "" {
				updateDoktor.Set("working_hours", nil)
			} else {
				updateDoktor.Set("working_hours", inputs.WorkingHours)
			}
			SomethingSet = true
		}

		if inputs.VacationDates != inputs.OldVacationDates {
			if inputs.VacationDates == "" {
				updateDoktor.Set("vacation_dates", nil)
			} else {
				updateDoktor.Set("vacation_dates", inputs.VacationDates)
			}

			SomethingSet = true
		}

		if inputs.FacebookUrl != inputs.OldFacebookUrl {
			updateDoktor.Set("facebook_url", inputs.FacebookUrl)
			SomethingSet = true
		}

		if inputs.LinkedinUrl != inputs.OldLinkedinUrl {
			updateDoktor.Set("linkedin_url", inputs.LinkedinUrl)
			SomethingSet = true
		}

		if inputs.InstagramUrl != inputs.OldInstagramUrl {
			updateDoktor.Set("instagram_url", inputs.InstagramUrl)
			SomethingSet = true
		}

		if inputs.XUrl != inputs.OldXUrl {
			updateDoktor.Set("x_url", inputs.XUrl)
			SomethingSet = true
		}

		if inputs.PersonalUrl != inputs.OldPersonalUrl {
			updateDoktor.Set("personal_url", inputs.PersonalUrl)
			SomethingSet = true
		}

		if inputs.OnlineAppointment != inputs.OldOnlineAppointment {
			updateDoktor.Set("online_appointment", inputs.OnlineAppointment)
			SomethingSet = true
		}

		if inputs.IsActive != inputs.OldIsActive {
			updateDoktor.Set("is_active", inputs.IsActive)
			SomethingSet = true
		}

		if inputs.Sid != inputs.OldSid {
			if inputs.Sid == "" {
				updateDoktor.Set("sid", nil)
			} else {
				updateDoktor.Set("sid", inputs.Sid)
			}

			SomethingSet = true
		}

		if SomethingSet {
			updateDoktor.Set("updated_at", "NOW()")
			updateDoktor.Where("drid", "=", Drid)
			updateDoktor.Finish()

			fmt.Printf("query string: %s\n", updateDoktor.Query)

			err = updateDoktor.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doktor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}
		}

		fmt.Printf("inputs.Brid: %s\n", inputs.Brid)
		fmt.Printf("inputs.OldBrid: %s\n", inputs.OldBrid)

		// Handle branch change if provided
		if inputs.Brid != inputs.OldBrid {
			if inputs.Brid != "" {
				// Call change_doctor_branch function
				ChangeDoctorBranch := Orm.SelectFunction("change_doctor_branch", inputs.Brid, Drid)
				err = ChangeDoctorBranch.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot change doctor branch: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := ChangeDoctorBranch.Rows()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get change doctor branch rows: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				if len(rows) == 0 {
					Orm.Rollback()
					log.Printf("Cannot change doctor branch: no rows returned\n")
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				if rows[0]["result"] == "false" {
					Orm.Rollback()
					log.Printf("Cannot change doctor branch: function returned false\n")
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}
			} else {
				UpdateDoctor := Orm.Update()
				UpdateDoctor.Table("doktorlar")
				UpdateDoctor.Set("brid", nil)
				UpdateDoctor.Where("drid", "=", Drid)
				UpdateDoctor.Finish()
				err = UpdateDoctor.Execute()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot update doctor branch: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				ra, err := UpdateDoctor.RowsAffected()
				if err != nil {
					Orm.Rollback()
					log.Printf("Cannot get update doctor branch rows: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				if ra == 0 {
					Orm.Rollback()
					log.Printf("Cannot update doctor branch: no rows returned\n")
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Doktor başarıyla güncellendi.",
		})
	}
}

func DeleteDoctor(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		DoctorId := c.Params("did")
		if DoctorId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor ID is required",
			})
		}

		Orm := utilities.Orm

		// Fetch doctor media information before deletion (photo and CV)
		GetDoctorMedia := Orm.Select([]string{"photo_mid", "cv_file_mid"})
		GetDoctorMedia.Table("doktorlar")
		GetDoctorMedia.Where("drid", "=", DoctorId)
		GetDoctorMedia.Finish()
		err = GetDoctorMedia.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		doctorRows, err := GetDoctorMedia.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(doctorRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Doctor not found",
			})
		}

		var photoMediaId, cvMediaId int64
		if doctorRows[0]["photo_mid"] != nil {
			photoMediaId = lib.Int64(doctorRows[0]["photo_mid"])
		}
		if doctorRows[0]["cv_file_mid"] != nil {
			cvMediaId = lib.Int64(doctorRows[0]["cv_file_mid"])
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

		// Delete related media files if they exist
		mediaFilesToDelete := []any{}

		// Handle photo media
		if photoMediaId > 0 {
			GetPhotoPath := Orm.Select([]string{"file_path"})
			GetPhotoPath.Table("medias")
			GetPhotoPath.Where("mid", "=", photoMediaId)
			GetPhotoPath.Finish()
			err = GetPhotoPath.Execute()

			if err != nil {
				log.Printf("Cannot get photo media path: %v\n", err)
				Orm.Rollback()
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			photoRows, err := GetPhotoPath.Rows()
			if err == nil && len(photoRows) > 0 {
				photoPath := lib.String(photoRows[0]["file_path"])
				if photoPath != "" {
					mediaFilesToDelete = append(mediaFilesToDelete, photoPath)
				}
			}
		}

		// Handle CV media
		if cvMediaId > 0 {
			GetCVPath := Orm.Select([]string{"file_path"})
			GetCVPath.Table("medias")
			GetCVPath.Where("mid", "=", cvMediaId)
			GetCVPath.Finish()
			err = GetCVPath.Execute()

			if err != nil {
				log.Printf("Cannot get CV media path: %v\n", err)
				Orm.Rollback()
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			cvRows, err := GetCVPath.Rows()
			if err == nil && len(cvRows) > 0 {
				cvPath := lib.String(cvRows[0]["file_path"])
				if cvPath != "" {
					mediaFilesToDelete = append(mediaFilesToDelete, cvPath)
				}
			}
		}

		// Remove doctor as head from any branches
		UpdateBranches := Orm.Update()
		UpdateBranches.Table("branslar")
		UpdateBranches.Set("head_drid", nil)
		UpdateBranches.Where("head_drid", "=", DoctorId)
		UpdateBranches.Finish()
		err = UpdateBranches.Execute()

		if err != nil {
			log.Printf("Cannot update branches: %v\n", err)
			Orm.Rollback()
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		GetAllExperienceImages := Orm.Select([]string{"m.file_path"})
		GetAllExperienceImages.Table("doctor_experiences de")
		GetAllExperienceImages.InnerJoin("doktorlar d", "d.drid", "=", "de.drid")
		GetAllExperienceImages.LeftJoin("medias m", "de.cover_mid", "=", "m.mid")
		GetAllExperienceImages.Where("d.drid", "=", DoctorId)
		GetAllExperienceImages.Finish()

		err = GetAllExperienceImages.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetAllExperienceImages.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		for _, row := range rows {
			filePath := lib.String(row["file_path"])
			if filePath != "" {
				mediaFilesToDelete = append(mediaFilesToDelete, filePath)
			}
		}

		if len(mediaFilesToDelete) > 0 {
			DeleteAllMedias := Orm.Delete()
			DeleteAllMedias.Table("medias")
			DeleteAllMedias.In("WHERE", "file_path", mediaFilesToDelete)
			DeleteAllMedias.Finish()
			err = DeleteAllMedias.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := DeleteAllMedias.RowsAffected()
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
					"message": "Deneyim fotoğrafı bulunamadı.",
				})
			}
		}

		// Delete the doctor
		DeleteDoctor := Orm.Delete()
		DeleteDoctor.Table("doktorlar")
		DeleteDoctor.Where("drid", "=", DoctorId)
		DeleteDoctor.Finish()
		err = DeleteDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteDoctor.RowsAffected()
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
				"message": "Doktor bulunamadı.",
			})
		}

		// Delete physical files after successful transaction
		for _, filePath := range mediaFilesToDelete {
			fullPath := "static/" + lib.String(filePath)
			err = lib.DeleteFile(fullPath)
			if err != nil {
				Orm.Rollback()
				log.Printf("Error deleting file %s: %v\n", fullPath, err)
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
			"message": "Doctor deleted successfully",
		})
	}
}

func UpdateDoctorPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
		photoAltText := c.FormValue("photo_alt_text")
		photoTitle := c.FormValue("photo_title")
		oldPhotoAltText := c.FormValue("old_photo_alt_text")
		oldPhotoTitle := c.FormValue("old_photo_title")

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

		photoInput, err := c.FormFile("photo_mid")
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
			if photoInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "doktorlar", Drid, "photo")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + photoInput.Filename)

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
				FilePath: "files/doktorlar/" + Drid + "/photo/" + UniqueFilePath.BaseName,
				FileSize: photoInput.Size,
				MimeType: photoInput.Header.Get("Content-Type"),
				FileType: "image",
				Uid:      OurUser.Uid,
				TargetId: Drid,
			}

			optionals := models.MediaOptionals{
				AltText: photoAltText,
				Title:   photoTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			PhotoMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/doktorlar/"+Drid+"/photo", entry.Name())
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

			// Update doctor with new media ID
			UpdateDoktor := Orm.Update()
			UpdateDoktor.Table("doktorlar")
			UpdateDoktor.Set("photo_mid", PhotoMediaMid)
			UpdateDoktor.Where("drid", "=", Drid)
			UpdateDoktor.Finish()
			err = UpdateDoktor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *photoInput)
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
			if photoAltText != oldPhotoAltText || photoTitle != oldPhotoTitle {
				// Get current media ID
				GetDoktorMedia := Orm.Select([]string{"photo_mid"})
				GetDoktorMedia.Table("doktorlar")
				GetDoktorMedia.Where("drid", "=", Drid)
				GetDoktorMedia.Finish()
				err = GetDoktorMedia.Execute()

				if err != nil {
					log.Printf("Cannot get doctor media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetDoktorMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Doktor bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["photo_mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if photoAltText != oldPhotoAltText {
						if photoAltText != "" {
							UpdateMedia.Set("alt_text", photoAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if photoTitle != oldPhotoTitle {
						if photoTitle != "" {
							UpdateMedia.Set("title", photoTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", Drid)
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
			"message": "Doktor fotoğrafı başarıyla güncellendi.",
		})
	}
}

func DeleteDoctorPicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
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
		GetDoktorMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetDoktorMedia.Table("doktorlar d")
		GetDoktorMedia.LeftJoin("medias m", "d.photo_mid", "=", "m.mid")
		GetDoktorMedia.Where("d.drid", "=", Drid)
		GetDoktorMedia.Finish()
		err = GetDoktorMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get doctor media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetDoktorMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor fotoğrafı bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", Drid)
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
				"message": "Doktor fotoğrafı bulunamadı.",
			})
		}

		// Update doctor to remove media reference
		UpdateDoktor := Orm.Update()
		UpdateDoktor.Table("doktorlar")
		UpdateDoktor.Set("photo_mid", nil)
		UpdateDoktor.Where("drid", "=", Drid)
		UpdateDoktor.Finish()
		err = UpdateDoktor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update doctor: %v\n", err)
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
			"message": "Doktor fotoğrafı başarıyla silindi.",
		})
	}
}

func UpdateDoctorCv(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")

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

		cvInput, err := c.FormFile("cv_file_mid")
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
			if cvInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "doktorlar", Drid, "cv")
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + cvInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			switch UniqueFilePath.Extension {
			case ".pdf", ".doc", ".docx":
				break
			default:
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Geçersiz dosya türü. PDF, DOC veya DOCX dosyası yükleyin.",
				})
			}

			// Insert new media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/doktorlar/" + Drid + "/" + "cv" + "/" + UniqueFilePath.BaseName,
				FileSize: cvInput.Size,
				MimeType: cvInput.Header.Get("Content-Type"),
				FileType: "cv",
				Uid:      OurUser.Uid,
				TargetId: Drid,
			}

			optionals := models.MediaOptionals{
				AltText: "",
				Title:   "",
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			CvMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/doktorlar/"+Drid+"/cv", entry.Name())
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

			// Update doctor with new media ID
			UpdateDoktor := Orm.Update()
			UpdateDoktor.Table("doktorlar")
			UpdateDoktor.Set("cv_file_mid", CvMediaMid)
			UpdateDoktor.Where("drid", "=", Drid)
			UpdateDoktor.Finish()
			err = UpdateDoktor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *cvInput)
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
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Doktor CV başarıyla güncellendi.",
		})
	}
}

func DeleteDoctorCv(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
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
		GetDoktorMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetDoktorMedia.Table("doktorlar d")
		GetDoktorMedia.LeftJoin("medias m", "d.cv_file_mid", "=", "m.mid")
		GetDoktorMedia.Where("d.drid", "=", Drid)
		GetDoktorMedia.Finish()
		err = GetDoktorMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get doctor media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetDoktorMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor CV bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", Drid)
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
				"message": "Doktor CV bulunamadı.",
			})
		}

		// Update doctor to remove media reference
		UpdateDoktor := Orm.Update()
		UpdateDoktor.Table("doktorlar")
		UpdateDoktor.Set("cv_file_mid", nil)
		UpdateDoktor.Where("drid", "=", Drid)
		UpdateDoktor.Finish()
		err = UpdateDoktor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update doctor: %v\n", err)
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
			"message": "Doktor CV başarıyla silindi.",
		})
	}
}

func GetExpertiseForDoctors(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Orm := utilities.Orm

		Did := c.Params("did")

		GetExpertises := Orm.Select([]string{"uzid", "name"})
		GetExpertises.Table("uzmanliklar")
		GetExpertises.Where("drid", "=", Did)
		GetExpertises.Finish()
		err = GetExpertises.Execute()

		if err != nil {
			log.Printf("Cannot get expertises: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetExpertises.Rows()
		if err != nil {
			log.Printf("Cannot get expertises: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ExpertisesArray := []models.Uzmanliklar{}
		for _, row := range rows {
			ExpertisesArray = append(ExpertisesArray, models.Uzmanliklar{
				Uzid: lib.String(row["uzid"]),
				Name: lib.String(row["name"]),
			})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Expertise fetched successfully",
			"data":    ExpertisesArray,
		})
	}
}

func AddExpertiseToADoctor(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		DoctorId := c.Params("did")
		if DoctorId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor ID is required",
			})
		}

		var inputs = models.AddExpertiseToADoctorInputs{}

		if err := c.BodyParser(&inputs); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		if inputs.Uzid == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Uzmanlık alanı zorunludur",
			})
		}

		Orm := utilities.Orm

		// Check if doctor exists
		CheckDoctor := Orm.Select([]string{"drid"})
		CheckDoctor.Table("doktorlar")
		CheckDoctor.Where("drid", "=", DoctorId)
		CheckDoctor.Finish()
		err = CheckDoctor.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		doctorRows, err := CheckDoctor.Rows()
		if err != nil || len(doctorRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		// Check if expertise exists
		CheckExpertise := Orm.Select([]string{"uzid"})
		CheckExpertise.Table("uzmanliklar")
		CheckExpertise.Where("uzid", "=", inputs.Uzid)
		CheckExpertise.Finish()
		err = CheckExpertise.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		expertiseRows, err := CheckExpertise.Rows()
		if err != nil || len(expertiseRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Uzmanlık alanı bulunamadı.",
			})
		}

		// Check if doctor already has this expertise
		CheckExisting := Orm.Select([]string{"duid"})
		CheckExisting.Table("doctor_expertises")
		CheckExisting.Where("drid", "=", DoctorId)
		CheckExisting.And("uzid", "=", inputs.Uzid)
		CheckExisting.Finish()
		err = CheckExisting.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		existingRows, err := CheckExisting.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(existingRows) > 0 {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Bu doktor zaten bu uzmanlık alanına sahip.",
			})
		}

		// Parse certification date
		var certificationDate *time.Time
		if inputs.CertificationDate != "" {
			if parsedDate, err := time.Parse("2006-01-02", inputs.CertificationDate); err == nil {
				certificationDate = &parsedDate
			}
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

		// Insert doctor expertise
		columns := []string{"drid", "uzid", "is_primary"}
		values := []interface{}{DoctorId, inputs.Uzid, false}

		if certificationDate != nil {
			columns = append(columns, "certification_date")
			values = append(values, *certificationDate)
		}
		if inputs.CertificationInstitution != "" {
			columns = append(columns, "certification_institution")
			values = append(values, inputs.CertificationInstitution)
		}

		InsertExpertise := Orm.Insert(columns, values)
		InsertExpertise.Table("doctor_expertises")
		InsertExpertise.Returning("duid")
		InsertExpertise.Finish()

		fmt.Printf("InsertExpertise Query: %v\n", InsertExpertise.Query)

		err = InsertExpertise.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		Lid, err := InsertExpertise.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		fmt.Printf("lid: %v\n", Lid)

		if Lid == "" {
			Orm.Rollback()
			log.Printf("Cannot get last insert id\n")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if inputs.IsPrimary {
			SetExpertiseAsPrimary := Orm.SelectFunction("handle_primary_expertise", "INSERT", Lid, DoctorId)
			err = SetExpertiseAsPrimary.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			rows, err := SetExpertiseAsPrimary.Rows()
			if err != nil {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if rows[0]["result"] == false {
				Orm.Rollback()
				log.Printf("%v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
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
			"message": "Uzmanlık alanı başarıyla eklendi.",
		})
	}
}

func RemoveExpertiseFromADoctor(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		DoctorId := c.Params("did")
		if DoctorId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor ID is required",
			})
		}

		var inputs struct {
			Duid string `json:"duid"`
		}

		if err := c.BodyParser(&inputs); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		if inputs.Duid == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor expertise ID is required",
			})
		}

		Orm := utilities.Orm

		// Check if doctor expertise exists and belongs to the doctor
		CheckExpertise := Orm.Select([]string{"duid", "drid"})
		CheckExpertise.Table("doctor_expertises")
		CheckExpertise.Where("duid", "=", inputs.Duid)
		CheckExpertise.And("drid", "=", DoctorId)
		CheckExpertise.Finish()
		err = CheckExpertise.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		expertiseRows, err := CheckExpertise.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(expertiseRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Uzmanlık alanı bulunamadı.",
			})
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

		// Delete doctor expertise
		DeleteExpertise := Orm.Delete()
		DeleteExpertise.Table("doctor_expertises")
		DeleteExpertise.Where("duid", "=", inputs.Duid)
		DeleteExpertise.And("drid", "=", DoctorId)
		DeleteExpertise.Finish()
		err = DeleteExpertise.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteExpertise.RowsAffected()
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
				"message": "Uzmanlık alanı bulunamadı.",
			})
		}

		SetExpertiseAsPrimary := Orm.SelectFunction("handle_primary_expertise", "DELETE", nil, DoctorId)
		err = SetExpertiseAsPrimary.Execute()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := SetExpertiseAsPrimary.Rows()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if rows[0]["result"] == false {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
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
			"message": "Uzmanlık alanı başarıyla silindi.",
		})
	}
}

func MoveDoctorToABranch(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Printf("%s", string(c.Body()))

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Doctor to a branch moved successfully",
		})
	}
}

func AddDoctorExperience(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		DoctorId := c.Params("did")
		if DoctorId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor ID is required",
			})
		}

		var inputs = models.DoctorExperiences{}

		if err := c.BodyParser(&inputs); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		fmt.Printf("inputs: %+v\n", inputs)

		if inputs.Name == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Deneyim adı zorunludur",
			})
		}

		Orm := utilities.Orm

		GetOptions := database.Options{}
		GetOptions, err = GetOptions.FetchOptionsForBackend(Orm, []string{"o.max_upload_size"}, []string{})
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Check if doctor exists
		CheckDoctor := Orm.Select([]string{"drid"})
		CheckDoctor.Table("doktorlar")
		CheckDoctor.Where("drid", "=", DoctorId)
		CheckDoctor.Finish()
		err = CheckDoctor.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		doctorRows, err := CheckDoctor.Rows()
		if err != nil || len(doctorRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
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

		// Insert doctor experience
		columns := []string{"drid", "name", "is_active", "created_at", "updated_at"}
		values := []interface{}{DoctorId, inputs.Name, true, time.Now(), time.Now()}

		if !inputs.StartDate.IsZero() {
			columns = append(columns, "start_date")
			values = append(values, inputs.StartDate)
		}
		if !inputs.EndDate.IsZero() {
			columns = append(columns, "end_date")
			values = append(values, inputs.EndDate)
		}
		if inputs.Description != "" {
			columns = append(columns, "description")
			values = append(values, inputs.Description)
		}

		InsertExperience := Orm.Insert(columns, values)
		InsertExperience.Table("doctor_experiences")
		InsertExperience.Returning("dtid")
		InsertExperience.Finish()
		err = InsertExperience.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		lid, err := InsertExperience.LastInsertId()
		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if lid == "" {
			Orm.Rollback()
			log.Printf("Cannot get last insert id\n")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		photoInput, err := c.FormFile("experience_cover")
		if err == nil {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				Orm.Rollback()
				log.Printf("ROOT_DIRECTORY not set")
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// File size validation
			if photoInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.Status(400).JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu çok büyük.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "doktorlar", DoctorId, "experience_covers", lid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + photoInput.Filename)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot get unique file path: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			switch UniqueFilePath.Extension {
			case ".jpg", ".jpeg", ".png", ".webp":
				break
			default:
				Orm.Rollback()
				return c.Status(400).JSON(fiber.Map{
					"status":  400,
					"message": "Geçersiz dosya türü. PNG, JPG veya WEBP dosyası yükleyin.",
				})
			}

			// Insert experience cover media record
			media := models.Medias{
				FileName: UniqueFilePath.BaseName,
				FilePath: "files/doktorlar/" + DoctorId + "/experience_covers/" + lid + "/" + UniqueFilePath.BaseName,
				FileSize: photoInput.Size,
				MimeType: photoInput.Header.Get("Content-Type"),
				FileType: "experience_cover",
				Uid:      ourUser.Uid,
				TargetId: lid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.CoverAltText,
				Title:   inputs.CoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			PhotoMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot insert experience cover media: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Update doctor with experience cover media ID
			updateDoctor := Orm.Update()
			updateDoctor.Table("doctor_experiences")
			updateDoctor.Set("cover_mid", PhotoMediaMid)
			updateDoctor.Where("dtid", "=", lid)
			updateDoctor.Finish()
			err = updateDoctor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor with experience cover: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save experience cover file
			err = lib.SaveFileWithBuffering(estimatedPath, *photoInput)
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot save experience cover file: %v\n", err)
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
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
			"message": "Deneyim başarıyla eklendi.",
		})
	}
}

func EditDoctorExperience(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Orm := utilities.Orm

		fmt.Printf("req body: %s\n", string(c.Body()))

		inputs := models.DoctorExperiencesEdit{}
		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("error: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		// Required fields validation
		if inputs.Name == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Deneyim adı zorunludur.",
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

		updateExperience := Orm.Update()
		updateExperience.Table("doctor_experiences")

		SomethingSet := false

		// Check and update each field if changed
		if inputs.Name != inputs.OldName {
			updateExperience.Set("name", inputs.Name)
			SomethingSet = true
		}

		if inputs.StartDate != inputs.OldStartDate {
			updateExperience.Set("start_date", inputs.StartDate)
			SomethingSet = true
		}

		if inputs.EndDate != inputs.OldEndDate {
			updateExperience.Set("end_date", inputs.EndDate)
			SomethingSet = true
		}

		if inputs.Description != inputs.OldDescription {
			updateExperience.Set("description", inputs.Description)
			SomethingSet = true
		}

		if SomethingSet {
			updateExperience.Where("dtid", "=", inputs.Dtid)
			updateExperience.Finish()

			fmt.Printf("updateExperience Query: %s\n", updateExperience.Query)

			err = updateExperience.Execute()

			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor experience: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			ra, err := updateExperience.RowsAffected()
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
					"message": "Deneyim bulunamadı.",
				})
			}
		}

		Orm.Commit()

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Deneyim başarıyla güncellendi.",
		})
	}
}

func DeleteDoctorExperience(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		DoctorId := c.Params("did")
		if DoctorId == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor ID is required",
			})
		}

		inputs := models.DoctorExperiences{}
		if err := c.BodyParser(&inputs); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		if inputs.Dtid == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Doctor experience ID is required",
			})
		}

		Orm := utilities.Orm

		// Fetch doctor media information before deletion (photo and CV)
		GetDoctorMedia := Orm.Select([]string{"cover_mid"})
		GetDoctorMedia.Table("doctor_experiences")
		GetDoctorMedia.Where("dtid", "=", inputs.Dtid)
		GetDoctorMedia.Finish()
		err = GetDoctorMedia.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		doctorRows, err := GetDoctorMedia.Rows()
		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(doctorRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Doctor not found",
			})
		}

		var photoMediaId int64
		if doctorRows[0]["cover_mid"] != nil {
			photoMediaId = lib.Int64(doctorRows[0]["cover_mid"])
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

		// Delete related media files if they exist
		mediaFilesToDelete := []string{}

		// Handle photo media
		if photoMediaId > 0 {
			GetPhotoPath := Orm.Select([]string{"file_path"})
			GetPhotoPath.Table("medias")
			GetPhotoPath.Where("mid", "=", photoMediaId)
			GetPhotoPath.Finish()
			err = GetPhotoPath.Execute()

			if err != nil {
				log.Printf("Cannot get photo media path: %v\n", err)
				Orm.Rollback()
				return c.Status(500).JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			photoRows, err := GetPhotoPath.Rows()
			if err == nil && len(photoRows) > 0 {
				photoPath := lib.String(photoRows[0]["file_path"])
				if photoPath != "" {
					mediaFilesToDelete = append(mediaFilesToDelete, photoPath)
				}

				// Delete photo media record
				DeletePhotoMedia := Orm.Delete()
				DeletePhotoMedia.Table("medias")
				DeletePhotoMedia.Where("mid", "=", photoMediaId)
				DeletePhotoMedia.Finish()
				err = DeletePhotoMedia.Execute()

				if err != nil {
					log.Printf("Cannot delete photo media record: %v\n", err)
					Orm.Rollback()
					return c.Status(500).JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}
			}
		}

		// Delete the doctor
		DeleteDoctor := Orm.Delete()
		DeleteDoctor.Table("doctor_experiences")
		DeleteDoctor.Where("dtid", "=", inputs.Dtid)
		DeleteDoctor.Finish()
		err = DeleteDoctor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteDoctor.RowsAffected()
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
				"message": "Deneyim bulunamadı.",
			})
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

		// Delete physical files after successful transaction
		for _, filePath := range mediaFilesToDelete {
			fullPath := "static/" + filePath
			err = lib.DeleteFile(fullPath)
			if err != nil {
				log.Printf("Error deleting file %s: %v\n", fullPath, err)
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Deneyim başarıyla silindi.",
		})
	}
}

func UpdateDoctorExperiencePicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Drid := c.Params("did")
		Dtid := c.FormValue("dtid")
		experienceCoverAltText := c.FormValue("experience_cover_alt_text")
		experienceCoverTitle := c.FormValue("experience_cover_title")
		oldExperienceCoverAltText := c.FormValue("old_experience_cover_alt_text")
		oldExperienceCoverTitle := c.FormValue("old_experience_cover_title")

		inputs := models.DoctorExperiencesEdit{}
		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("error: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		fmt.Printf("body:: %s\n", string(c.Body()))

		fmt.Printf("Dtid: %s\n", Dtid)
		fmt.Printf("experienceCoverAltText: %s\n", experienceCoverAltText)
		fmt.Printf("experienceCoverTitle: %s\n", experienceCoverTitle)
		fmt.Printf("oldExperienceCoverAltText: %s\n", oldExperienceCoverAltText)
		fmt.Printf("oldExperienceCoverTitle: %s\n", oldExperienceCoverTitle)

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

		photoInput, err := c.FormFile("cover_mid")
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
			if photoInput.Size > GetOptions.Options.MaxUploadSize {
				Orm.Rollback()
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Dosya boyutu 5MB'dan büyük olamaz.",
				})
			}

			estimatedPath := filepath.Join(RootDir, "static", "files", "doktorlar", Drid, "experience_covers", Dtid)
			UniqueFilePath, err := lib.UniqueFilePath(estimatedPath + "/" + photoInput.Filename)

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
				FilePath: "files/doktorlar/" + Drid + "/experience_covers/" + Dtid + "/" + UniqueFilePath.BaseName,
				FileSize: photoInput.Size,
				MimeType: photoInput.Header.Get("Content-Type"),
				FileType: "experience_cover",
				Uid:      OurUser.Uid,
				TargetId: Dtid,
			}

			optionals := models.MediaOptionals{
				AltText: inputs.ExperienceCoverAltText,
				Title:   inputs.ExperienceCoverTitle,
				Width:   0,
				Height:  0,
			}

			OurOptions := database.Options{}
			PhotoMediaMid, err := OurOptions.InsertMedia(Orm, media, optionals)

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
				FilePath := filepath.Join("files/doktorlar/"+Drid+"/experience_covers/"+Dtid, entry.Name())
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

			// Update doctor with new media ID
			UpdateDoktor := Orm.Update()
			UpdateDoktor.Table("doctor_experiences")
			UpdateDoktor.Set("cover_mid", PhotoMediaMid)
			UpdateDoktor.Where("dtid", "=", inputs.Dtid)
			UpdateDoktor.Finish()
			err = UpdateDoktor.Execute()
			if err != nil {
				Orm.Rollback()
				log.Printf("Cannot update doctor: %v\n", err)
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			// Save new file
			err = lib.SaveFileWithBuffering(estimatedPath, *photoInput)
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
			if inputs.ExperienceCoverAltText != inputs.OldExperienceCoverAltText || inputs.ExperienceCoverTitle != inputs.OldExperienceCoverTitle {
				// Get current media ID
				GetDoktorMedia := Orm.Select([]string{"cover_mid"})
				GetDoktorMedia.Table("doctor_experiences")
				GetDoktorMedia.Where("dtid", "=", inputs.Dtid)
				GetDoktorMedia.Finish()
				err = GetDoktorMedia.Execute()

				if err != nil {
					log.Printf("Cannot get doctor media: %v\n", err)
					return c.JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				rows, err := GetDoktorMedia.Rows()
				if err != nil || len(rows) == 0 {
					return c.JSON(fiber.Map{
						"status":  404,
						"message": "Doktor bulunamadı.",
					})
				}

				mediaId := lib.Int64(rows[0]["cover_mid"])
				if mediaId > 0 {
					UpdateMedia := Orm.Update()
					UpdateMedia.Table("medias")

					if inputs.ExperienceCoverAltText != inputs.OldExperienceCoverAltText {
						if inputs.ExperienceCoverAltText != "" {
							UpdateMedia.Set("alt_text", experienceCoverAltText)
						} else {
							UpdateMedia.Set("alt_text", nil)
						}
					}

					if inputs.ExperienceCoverTitle != inputs.OldExperienceCoverTitle {
						if inputs.ExperienceCoverTitle != "" {
							UpdateMedia.Set("title", experienceCoverTitle)
						} else {
							UpdateMedia.Set("title", nil)
						}
					}

					UpdateMedia.Where("mid", "=", mediaId)
					UpdateMedia.And("target_id", "=", inputs.Dtid)
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
			"message": "Doktor fotoğrafı başarıyla güncellendi.",
		})
	}
}

func DeleteDoctorExperiencePicture(states *models.AppState, utilities *models.Utilities) fiber.Handler {
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

		Dtid := c.FormValue("dtid")
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
		GetDoktorMedia := Orm.Select([]string{"m.mid", "m.file_path"})
		GetDoktorMedia.Table("doktorlar d")
		GetDoktorMedia.LeftJoin("medias m", "d.cover_mid", "=", "m.mid")
		GetDoktorMedia.Where("d.dtid", "=", Dtid)
		GetDoktorMedia.Finish()
		err = GetDoktorMedia.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot get doctor media: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rows, err := GetDoktorMedia.Rows()
		if err != nil || len(rows) == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor bulunamadı.",
			})
		}

		mediaId := lib.Int64(rows[0]["mid"])
		filePath := lib.String(rows[0]["file_path"])

		if mediaId == 0 {
			Orm.Rollback()
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Doktor fotoğrafı bulunamadı.",
			})
		}

		// Delete media record from database
		DeleteMedia := Orm.Delete()
		DeleteMedia.Table("medias")
		DeleteMedia.Where("mid", "=", mediaId)
		DeleteMedia.And("target_id", "=", Dtid)
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
				"message": "Doktor fotoğrafı bulunamadı.",
			})
		}

		// Update doctor to remove media reference
		UpdateDoktor := Orm.Update()
		UpdateDoktor.Table("doctor_experiences")
		UpdateDoktor.Set("cover_mid", nil)
		UpdateDoktor.Where("dtid", "=", Dtid)
		UpdateDoktor.Finish()
		err = UpdateDoktor.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("Cannot update doctor: %v\n", err)
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
			"message": "Doktor fotoğrafı başarıyla silindi.",
		})
	}
}
