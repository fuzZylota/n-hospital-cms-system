package randevular

import (
	"database/sql"
	"fmt"
	lib "lib"
	"log"
	"models"
	"models/notify"
	"os"
	"path/filepath"
	"post/appointmentrequestsnapshot"
	"post/appointmentworkflowsnapshot"
	"post/notificationevent"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AddRandevuRequest(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		inputs := models.RandevuRequests{}
		if err := c.BodyParser(&inputs); err != nil {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Geçersiz istek gövdesi",
			})
		}

		// Required validation
		if inputs.PatientFirstName == "" || inputs.PatientLastName == "" || inputs.PatientPhone == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Ad, soyad ve telefon zorunludur.",
			})
		}

		Orm := utilities.Orm

		appointmentSnapshot, err := appointmentrequestsnapshot.Read(c.UserContext(), utilities.AppointmentRequestWorkflowSnapshotReader)

		if err != nil {
			log.Printf("operation=AddRandevuRequest stage=options_read")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		CheckIfItsRepeating := Orm.Count("randevu_talepleri")
		CheckIfItsRepeating.Where("patient_phone", "=", inputs.PatientPhone)
		CheckIfItsRepeating.AndExpr("created_at", ">=", "NOW() - INTERVAL '24 hours'")
		CheckIfItsRepeating.Finish()

		err = CheckIfItsRepeating.Execute()
		if err != nil {
			log.Printf("operation=AddRandevuRequest stage=duplicate_check")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if CheckIfItsRepeating.Length() > 0 {
			return c.JSON(fiber.Map{
				"status":  429,
				"message": "Bekleyen bir randevu talebiniz bulunmaktadır. Sizinle en kısa sürede iletişime geçilecektir.",
			})
		}

		if appointmentSnapshot.RecaptchaSiteKey != "" && appointmentSnapshot.RecaptchaSecretKey != "" {
			if !lib.VerifyRecaptcha(inputs.RecaptchaToken, appointmentSnapshot.RecaptchaSecretKey) {
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "reCAPTCHA doğrulama hatası. Lütfen tekrar deneyin.",
				})
			}
		}

		columns := []string{"patient_first_name", "patient_last_name", "patient_phone"}
		values := []interface{}{inputs.PatientFirstName, inputs.PatientLastName, inputs.PatientPhone}

		// Optional fields
		if inputs.PatientEmail != "" {
			columns = append(columns, "patient_email")
			values = append(values, inputs.PatientEmail)
		}
		if !inputs.PreferredDate.IsZero() {
			columns = append(columns, "preferred_date")
			values = append(values, inputs.PreferredDate)
		}
		if inputs.Sid != "" {
			columns = append(columns, "sid")
			values = append(values, inputs.Sid)
		}

		if inputs.Message != "" {
			columns = append(columns, "message")
			values = append(values, inputs.Message)
		}

		insertReq := Orm.Insert(columns, values)
		insertReq.Table("randevu_talepleri")
		insertReq.Returning("rrid")
		insertReq.Finish()
		if err := insertReq.Execute(); err != nil {
			log.Printf("operation=AddRandevuRequest stage=record_insert")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		rrid, err := insertReq.LastInsertId()
		if err != nil {
			log.Printf("operation=AddRandevuRequest stage=insert_id")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if appointmentSnapshot.SMTPHost != "" && appointmentSnapshot.SMTPPort != 0 && appointmentSnapshot.SMTPUsername != "" && appointmentSnapshot.SMTPPassword != "" && inputs.PatientEmail != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				log.Printf("operation=AddRandevuRequest stage=message_build")
			}

			GetLogo := ""

			if appointmentSnapshot.SiteLogoPath != "" {
				GetLogo = filepath.Join(RootDir, "static", appointmentSnapshot.SiteLogoPath)
			} else {
				GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
			}

			LogoName := filepath.Base(GetLogo)

			Html := ""
			// Create professional HTML email template for appointment request
			{
				dateString := fmt.Sprintf("%d.%d.%d", inputs.PreferredDate.Day(), int(inputs.PreferredDate.Month()), inputs.PreferredDate.Year())

				Html = `
				<!DOCTYPE html>
				<html lang="tr">
				<head>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<title>Randevu Talebiniz Alındı</title>
					<style>
						body {
							margin: 0;
							padding: 0;
							font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
							background-color: #f4f7fa;
							color: #333333;
						}
						.email-container {
							max-width: 600px;
							margin: 40px auto;
							background-color: #ffffff;
							border-radius: 12px;
							overflow: hidden;
							box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
						}
						.header {
							padding: 40px 30px;
							text-align: center;
							color: #ffffff;
						}
						.header img {
							max-width: 180px;
							height: auto;
							margin-bottom: 20px;
							filter: brightness(0) invert(1);
						}
						.header h1 {
							margin: 0;
							font-size: 28px;
							font-weight: 600;
							letter-spacing: -0.5px;
							color: #252525 !important;
						}
						.header .subtitle {
							margin-top: 10px;
							font-size: 16px;
							opacity: 0.95;
							font-weight: 400;
							color: #252525 !important;
						}
						.content {
							padding: 40px 30px;
						}
						.greeting {
							font-size: 18px;
							color: #283b6a;
							margin-bottom: 20px;
							font-weight: 600;
						}
						.message {
							font-size: 16px;
							line-height: 1.8;
							color: #555555;
							margin-bottom: 25px;
						}
						.info-box {
							background-color: #f8f9fc;
							border-left: 4px solid ` + appointmentSnapshot.AccentColor + `;
							padding: 20px 25px;
							margin: 25px 0;
							border-radius: 6px;
						}
						.info-box h3 {
							margin: 0 0 15px 0;
							color: #283b6a;
							font-size: 18px;
							font-weight: 600;
						}
						.info-row {
							display: flex;
							justify-content: space-between;
							padding: 10px 0;
							border-bottom: 1px solid #e1e8ed;
							flex-wrap: wrap;
						}
						.info-row:last-child {
							border-bottom: none;
						}
						.info-label {
							font-weight: 600;
							color: #283b6a;
							margin-right: 15px;
						}
						.info-value {
							color: #555555;
							text-align: right;
							flex: 1;
						}
						.status-badge {
							display: inline-block;
							background-color: #4CAF50;
							color: #ffffff;
							padding: 10px 20px;
							border-radius: 25px;
							font-size: 14px;
							font-weight: 600;
							margin: 20px 0;
						}
						.appointment-highlight {
							background: linear-gradient(135deg, #e3f2fd 0%, #f3e5f5 100%);
							border-radius: 12px;
							padding: 25px;
							margin: 25px 0;
							text-align: center;
						}
						.appointment-highlight h3 {
							color: #283b6a;
							margin: 0 0 15px 0;
							font-size: 20px;
						}
						.appointment-date {
							font-size: 24px;
							font-weight: 700;
							color: ` + appointmentSnapshot.PrimaryColor + `;
							margin: 10px 0;
						}
						.appointment-time {
							font-size: 20px;
							font-weight: 600;
							color: ` + appointmentSnapshot.SecondaryColor + `;
							margin: 10px 0;
						}
						.next-steps {
							background-color: #fff9e6;
							border-radius: 8px;
							padding: 20px;
							margin: 25px 0;
						}
						.next-steps h3 {
							color: #283b6a;
							margin-top: 0;
							font-size: 18px;
						}
						.next-steps ul {
							margin: 10px 0;
							padding-left: 20px;
						}
						.next-steps li {
							color: #555555;
							margin: 8px 0;
							line-height: 1.6;
						}
						.important-note {
							background-color: #fff3cd;
							border-left: 4px solid #ffc107;
							border-radius: 6px;
							padding: 15px 20px;
							margin: 20px 0;
						}
						.important-note strong {
							color: #856404;
							display: block;
							margin-bottom: 5px;
						}
						.important-note p {
							margin: 0;
							color: #856404;
							font-size: 14px;
						}
						.contact-info {
							background-color: #f0f4ff;
							border-radius: 8px;
							padding: 20px;
							margin: 25px 0;
							text-align: center;
						}
						.contact-info h3 {
							color: #283b6a;
							margin-top: 0;
							font-size: 18px;
						}
						.contact-info p {
							margin: 5px 0;
							color: #555555;
						}
						.contact-info a {
							color: ` + appointmentSnapshot.PrimaryColor + `;
							text-decoration: none;
							font-weight: 600;
						}
						.footer {
							background-color: #f8f9fc;
							padding: 30px;
							text-align: center;
							font-size: 13px;
							color: #888888;
							border-top: 1px solid #e1e8ed;
						}
						.footer p {
							margin: 5px 0;
						}
						.footer a {
							color: ` + appointmentSnapshot.PrimaryColor + `;
							text-decoration: none;
						}
						.social-links {
							margin: 20px 0 10px;
						}
						.social-links a {
							display: inline-block;
							margin: 0 8px;
							color: #888888;
							text-decoration: none;
							font-size: 12px;
						}
						@media only screen and (max-width: 600px) {
							.email-container {
								margin: 0;
								border-radius: 0;
							}
							.header, .content, .footer {
								padding: 25px 20px;
							}
							.header h1 {
								font-size: 24px;
							}
							.info-row {
								flex-direction: column;
							}
							.info-value {
								text-align: left;
								margin-top: 5px;
							}
							.appointment-date {
								font-size: 20px;
							}
							.appointment-time {
								font-size: 18px;
							}
						}
					</style>
				</head>
				<body>
					<div class="email-container">
						<div class="header">
							<img src="cid:` + LogoName + `" alt="` + appointmentSnapshot.SiteName + `" />
							<h1>Randevu Talebiniz Alındı</h1>
							<div class="subtitle">Sağlığınız bizim önceliğimiz</div>
						</div>
						
						<div class="content">
							<div class="greeting">
								Sayın ` + inputs.PatientFirstName + ` ` + inputs.PatientLastName + `,
							</div>
							
							<div class="message">
								<strong>` + appointmentSnapshot.SiteName + `</strong>'yi tercih ettiğiniz için teşekkür ederiz.
								Randevu talebiniz başarıyla sistemimize kaydedilmiştir ve en kısa sürede değerlendirilerek 
								size uygun bir randevu tarih ve saati belirlenecektir.
							</div>
							
							<div class="status-badge">
								✓ Talep Alındı
							</div>`

				if !inputs.PreferredDate.IsZero() || !inputs.PreferredTime.IsZero() {
					Html += `
							
							<div class="appointment-highlight">
								<h3>📅 Tercih Ettiğiniz Randevu Bilgileri</h3>`

					if !inputs.PreferredDate.IsZero() {
						Html += `
								<div class="appointment-date">` + dateString + `</div>`
					}

					Html += `
								<p style="margin-top: 15px; color: #666; font-size: 14px;">
									Tercih ettiğiniz tarih ve saat müsaitlik durumuna göre değerlendirilecektir.
								</p>
							</div>`
				}

				Html += `
							
							<div class="info-box">
								<h3>📋 İletişim Bilgileriniz</h3>
								<div class="info-row">
									<span class="info-label">Ad Soyad:</span>
									<span class="info-value">` + inputs.PatientFirstName + ` ` + inputs.PatientLastName + `</span>
								</div>
								<div class="info-row">
									<span class="info-label">Telefon:</span>
									<span class="info-value">` + inputs.PatientPhone + `</span>
								</div>
								<div class="info-row">
									<span class="info-label">E-posta:</span>
									<span class="info-value">` + inputs.PatientEmail + `</span>
								</div>
							</div>
							
							<div class="next-steps">
								<h3>📝 Sonraki Adımlar</h3>
								<ul>
									<li>Randevu talebiniz hastanemiz tarafından en kısa sürede değerlendirilecektir</li>
									<li>Uygun randevu tarihi ve saati belirlendikten sonra <strong>SMS ve e-posta</strong> ile bilgilendirileceksiniz</li>
									<li>Randevu onayı alındıktan sonra belirtilen tarih ve saatte hastanemize gelmeniz yeterlidir</li>
									<li>Randevunuza gelirken kimlik belgesi ve varsa sağlık kartınızı yanınızda bulundurmanız gerekmektedir</li>
								</ul>
							</div>
							
							<div class="important-note">
								<strong>⚠️ Önemli Hatırlatma</strong>
								<p>Randevunuza gelemeyeceğiniz durumlarda lütfen en az <strong>1 gün öncesinde</strong> hastanemizi bilgilendiriniz. 
								Bu sayede diğer hastalarımıza hizmet verebilme imkanı bulabiliriz.</p>
							</div>
							
							<div class="contact-info">
								<h3>📞 Bize Ulaşın</h3>
								<p>Randevunuzla ilgili sorularınız için:</p>
								<p><strong>Telefon:</strong> <a href="tel:` + appointmentSnapshot.ContactPhone + `">` + appointmentSnapshot.ContactPhone + `</a></p>
								<p><strong>E-posta:</strong> <a href="mailto:` + appointmentSnapshot.ContactEmail + `">` + appointmentSnapshot.ContactEmail + `</a></p>
							</div>
							
							<div class="message" style="margin-top: 30px; font-size: 15px; color: #666;">
								Bu e-posta otomatik olarak oluşturulmuştur. Lütfen bu e-postaya cevap vermeyiniz. 
								Sorularınız için yukarıdaki iletişim bilgilerini kullanabilirsiniz.
							</div>
						</div>
						
						<div class="footer">
							<p><strong>` + appointmentSnapshot.SiteName + `</strong></p>
							<p>` + appointmentSnapshot.SiteDescription + `</p>
							
							<div class="social-links">`

				if appointmentSnapshot.FacebookURL != "" && appointmentSnapshot.FacebookURL != "#" {
					Html += `
								<a href="` + appointmentSnapshot.FacebookURL + `">Facebook</a> |`
				}
				if appointmentSnapshot.TwitterURL != "" && appointmentSnapshot.TwitterURL != "#" {
					Html += `
								<a href="` + appointmentSnapshot.TwitterURL + `">Twitter</a> |`
				}
				if appointmentSnapshot.InstagramURL != "" && appointmentSnapshot.InstagramURL != "#" {
					Html += `
								<a href="` + appointmentSnapshot.InstagramURL + `">Instagram</a> |`
				}
				if appointmentSnapshot.LinkedInURL != "" && appointmentSnapshot.LinkedInURL != "#" {
					Html += `
								<a href="` + appointmentSnapshot.LinkedInURL + `">LinkedIn</a>`
				}

				Html += `
							</div>
							
							<p style="margin-top: 20px;">© 2025 ` + appointmentSnapshot.SiteName + `. Tüm hakları saklıdır.</p>
							<p style="font-size: 11px; color: #999;">
								Bu e-postayı almak istemiyorsanız, lütfen <a href="mailto:` + appointmentSnapshot.ContactEmail + `">bizimle iletişime geçin</a>.
							</p>
						</div>
					</div>
				</body>
				</html>
				`
			}

			CreateEmailInfos := models.EmailInfos{
				From:        appointmentSnapshot.SiteName,
				To:          []string{inputs.PatientEmail},
				Username:    appointmentSnapshot.SMTPUsername,
				Password:    appointmentSnapshot.SMTPPassword,
				Host:        appointmentSnapshot.SMTPHost,
				Port:        lib.Int64(appointmentSnapshot.SMTPPort),
				Subject:     "Randevu Talebiniz Alındı - " + appointmentSnapshot.SiteName,
				PlainText:   "Sayın " + inputs.PatientFirstName + " " + inputs.PatientLastName + ", randevu talebiniz başarıyla alınmıştır. En kısa sürede sizinle iletişime geçeceğiz.",
				Body:        Html,
				Attachments: []string{GetLogo},
			}

			err = lib.DeliverEmailAfterPersistence(
				func() error { return lib.SendEmail(&CreateEmailInfos) }, nil,
			)

			if err != nil {
				log.Printf("operation=AddRandevuRequest stage=%s", lib.EmailFailureStage(err))
			}
		}

		// Bildirimin tek kaynagi, basariyla kaydedilen talebin yeniden okunan halidir.
		go func(rridVal string) {
			Orm2 := utilities.Orm
			msg, prepareErr := prepareSavedRequestNotification(rridVal, func(id string) (savedRequestNotification, error) {
				GetTalep := Orm2.Select([]string{"rt.rrid", "rt.patient_first_name", "rt.patient_last_name", "rt.patient_phone", "rt.patient_email", "rt.preferred_date", "rt.preferred_time", "rt.message", "rt.created_at", "rt.status", "s.name as sube_name", "rt.sid"})
				GetTalep.Table("randevu_talepleri rt")
				GetTalep.LeftJoin("subeler s", "rt.sid", "=", "s.sid")
				GetTalep.Where("rt.rrid", "=", id)
				GetTalep.Finish()
				if err := GetTalep.Execute(); err != nil {
					return savedRequestNotification{}, err
				}
				rows, readErr := GetTalep.Rows()
				if readErr != nil {
					return savedRequestNotification{}, readErr
				}
				if len(rows) != 1 {
					return savedRequestNotification{}, errInvalidSavedRequest
				}
				row := rows[0]
				record := savedRequestNotification{
					Rrid: lib.String(row["rrid"]), Sid: lib.String(row["sid"]),
					PatientFirstName: lib.String(row["patient_first_name"]), PatientLastName: lib.String(row["patient_last_name"]),
					PatientPhone: lib.String(row["patient_phone"]), PatientEmail: lib.String(row["patient_email"]),
					Message: lib.String(row["message"]), Status: lib.String(row["status"]), SubeName: lib.String(row["sube_name"]),
				}
				record.PreferredDate, _ = row["preferred_date"].(time.Time)
				record.PreferredTime, _ = row["preferred_time"].(time.Time)
				record.CreatedAt, _ = row["created_at"].(time.Time)
				return record, nil
			}, func(event notificationevent.NewRequest) error {
				columns := []string{"message", "notification_type", "notification_level", "link"}
				values := []interface{}{event.NotificationMessage, event.NotificationType, "santral", event.RequestLink}
				if event.Sid != "" {
					columns = append(columns, "sid")
					values = append(values, event.Sid)
				}
				insertNotification := Orm2.Insert(columns, values)
				insertNotification.Table("notifications")
				insertNotification.Finish()
				return insertNotification.Execute()
			})
			if prepareErr != nil {
				return
			}
			if publishErr := notificationevent.Publish(utilities.NotificationHub, msg, notificationevent.CurrentRecipients(utilities.UserStatusReader, notificationevent.RequestRecipients(notify.BranchID(msg.Sid), func(uid notify.UserID) (notificationevent.User, bool) {
				GetRole := Orm2.Select([]string{"role"})
				GetRole.Table("users")
				GetRole.Where("uid", "=", string(uid))
				GetRole.Finish()
				if GetRole.Execute() != nil {
					return notificationevent.User{}, false
				}
				userRows, readErr := GetRole.Rows()
				if readErr != nil || len(userRows) == 0 {
					return notificationevent.User{}, false
				}
				return notificationevent.User{Role: notify.Role(lib.String(userRows[0]["role"]))}, true
			}, func(uid notify.UserID, sid notify.BranchID) bool {
				CheckPerm := Orm2.Select([]string{"can_view"})
				CheckPerm.Table("user_branch_permissions")
				CheckPerm.Where("uid", "=", string(uid))
				CheckPerm.And("sid", "=", string(sid))
				CheckPerm.And("can_view", "=", true)
				CheckPerm.Finish()
				if CheckPerm.Execute() != nil {
					return false
				}
				permRows, readErr := CheckPerm.Rows()
				return readErr == nil && len(permRows) > 0
			}))); publishErr != nil {
				log.Printf("operation=AddRandevuRequest stage=notification_publish")
			}
		}(lib.String(rrid))
		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Randevu talebi başarıyla oluşturuldu.",
			"rrid":    rrid,
		})
	}
}

func DeleteRandevuRequest(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		if ourUser.Role != "admin" && ourUser.Role != "moderator" && ourUser.Role != "santral" {
			return c.Status(403).JSON(fiber.Map{
				"status":  403,
				"message": "Forbidden: Admin access required",
			})
		}

		Rrid := c.Params("rrid")
		if Rrid == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Request ID is required",
			})
		}

		// Admin değilse can_delete iznini kontrol et
		if ourUser.Role != "admin" {
			Orm2 := utilities.Orm
			GetTalepSube := Orm2.Select([]string{"sid"})
			GetTalepSube.Table("randevu_talepleri")
			GetTalepSube.Where("rrid", "=", Rrid)
			GetTalepSube.Finish()
			_ = GetTalepSube.Execute()
			talepRows, _ := GetTalepSube.Rows()
			if len(talepRows) > 0 {
				talepSid := lib.String(talepRows[0]["sid"])
				CheckDelPerm := Orm2.Select([]string{"can_delete"})
				CheckDelPerm.Table("user_branch_permissions")
				CheckDelPerm.Where("uid", "=", ourUser.Uid)
				CheckDelPerm.And("sid", "=", talepSid)
				CheckDelPerm.Finish()
				_ = CheckDelPerm.Execute()
				delPermRows, _ := CheckDelPerm.Rows()
				if len(delPermRows) == 0 || !lib.Bool(delPermRows[0]["can_delete"]) {
					return c.Status(403).JSON(fiber.Map{"status": 403, "message": "Randevu silme yetkiniz yoktur."})
				}

			}
		}
		Orm := utilities.Orm

		// Check if request exists
		GetRequest := Orm.Select([]string{"rrid", "patient_first_name", "patient_last_name"})
		GetRequest.Table("randevu_talepleri")
		GetRequest.Where("rrid", "=", Rrid)
		GetRequest.Finish()
		err = GetRequest.Execute()

		if err != nil {
			log.Printf("operation=DeleteRandevuRequest stage=record_read")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		requestRows, err := GetRequest.Rows()
		if err != nil {
			log.Printf("operation=DeleteRandevuRequest stage=record_rows")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if len(requestRows) == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Randevu talebi bulunamadı",
			})
		}

		// Start transaction
		err = Orm.Begin()
		if err != nil {
			log.Printf("operation=DeleteRandevuRequest stage=transaction_begin")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Delete the appointment request
		DeleteRequest := Orm.Delete()
		DeleteRequest.Table("randevu_talepleri")
		DeleteRequest.Where("rrid", "=", Rrid)
		DeleteRequest.Finish()
		err = DeleteRequest.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("operation=DeleteRandevuRequest stage=record_delete")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := DeleteRequest.RowsAffected()
		if err != nil {
			Orm.Rollback()
			log.Printf("operation=DeleteRandevuRequest stage=affected_rows")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if ra == 0 {
			Orm.Rollback()
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Randevu talebi bulunamadı.",
			})
		}

		// Commit transaction
		err = Orm.Commit()
		if err != nil {
			Orm.Rollback()
			log.Printf("operation=DeleteRandevuRequest stage=transaction_commit")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// WebSocket broadcast - silme
		go func() {
			if publishErr := notificationevent.Publish(utilities.NotificationHub, notificationevent.Deleted{Type: "randevu_talebi_silindi", Rrid: Rrid}, notificationevent.CurrentRecipients(utilities.UserStatusReader, notificationevent.Recipient)); publishErr != nil {
				log.Printf("operation=DeleteRandevuRequest stage=notification_publish")
			}
		}()
		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Randevu talebi başarıyla silindi.",
		})
	}
}

func ToggleRandevuRequestStatus(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Orm := utilities.Orm

		inputs := models.RandevuRequests{}
		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("Body parse error: %v\n", err)
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Geçersiz veri formatı",
			})
		}

		if inputs.Status == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Status is required",
			})
		}

		Rrid := c.Params("rrid")
		if Rrid == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Request ID is required",
			})
		}

		if ourUser.Role != "admin" && ourUser.Role != "moderator" {
			if ourUser.Role != "santral" {
				return c.Status(403).JSON(fiber.Map{
					"status":  403,
					"message": "Forbidden: Admin, Moderator or Santral access required",
				})
			} else {
				CheckIfSantralUserHasSid := Orm.Count("users u")
				CheckIfSantralUserHasSid.Where("u.uid", "=", ourUser.Uid)
				CheckIfSantralUserHasSid.And("u.sid", "!=", nil)
				CheckIfSantralUserHasSid.Finish()
				err = CheckIfSantralUserHasSid.Execute()
				if err != nil {
					log.Printf("%v\n", err)
					return c.Status(500).JSON(fiber.Map{
						"status":  500,
						"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
					})
				}

				if CheckIfSantralUserHasSid.Length() != 0 {
					CheckIfSantralUsersExists := Orm.Count("users u")
					CheckIfSantralUsersExists.LeftJoin("subeler s", "s.sid", "=", "u.sid")
					CheckIfSantralUsersExists.LeftJoin("randevu_talepleri rt", "rt.sid", "=", "s.sid")
					CheckIfSantralUsersExists.Where("rt.rrid", "=", Rrid)
					CheckIfSantralUsersExists.And("u.role", "=", "santral")
					CheckIfSantralUsersExists.Finish()

					err = CheckIfSantralUsersExists.Execute()

					if err != nil {
						log.Printf("%v\n", err)
						return c.Status(500).JSON(fiber.Map{
							"status":  500,
							"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
						})
					}

					if CheckIfSantralUsersExists.Length() == 0 {
						return c.Status(403).JSON(fiber.Map{
							"status":  403,
							"message": "Forbidden: Santral users not found",
						})
					}
				}
			}
		}

		NewStatus := ""
		switch inputs.Status {
		case "yeni":
			NewStatus = "randevu-verildi"
		case "randevu-verildi":
			NewStatus = "randevu-verilemedi"
		case "randevu-verilemedi":
			NewStatus = "hasta-arandi"
		case "hasta-arandi":
			NewStatus = "ulasilamadi"
		case "ulasilamadi":
			NewStatus = "gelmedi"
		case "gelmedi":
			NewStatus = "hasta-vazgecti"
		case "hasta-vazgecti":
			NewStatus = "yeni"
		default:
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Invalid status",
			})
		}

		CheckIfRequestExists := Orm.Count("randevu_talepleri")
		CheckIfRequestExists.Where("rrid", "=", Rrid)
		CheckIfRequestExists.Finish()
		err = CheckIfRequestExists.Execute()

		if err != nil {
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if CheckIfRequestExists.Length() == 0 {
			return c.Status(404).JSON(fiber.Map{
				"status":  404,
				"message": "Randevu talebi bulunamadı",
			})
		}

		UpdateRequest := Orm.Update()
		UpdateRequest.Table("randevu_talepleri")
		UpdateRequest.Set("status", NewStatus)
		UpdateRequest.Set("last_modified_uid", ourUser.Uid)
		statusColMap := map[string]string{"randevu-verildi": "randevu_verildi_at", "randevu-verilemedi": "randevu_verilemedi_at", "ulasilamadi": "ulasilamadi_at", "gelmedi": "gelmedi_at", "hasta-vazgecti": "hasta_vazgecti_at", "hasta-arandi": "hasta_arandi_at"}
		if col, ok := statusColMap[NewStatus]; ok {
			UpdateRequest.Set(col, time.Now())
		}
		UpdateRequest.Where("rrid", "=", Rrid)
		UpdateRequest.Finish()
		err = UpdateRequest.Execute()

		if err != nil {
			Orm.Rollback()
			log.Printf("%v\n", err)
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		ra, err := UpdateRequest.RowsAffected()
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
				"message": "Randevu talebi bulunamadı.",
			})
		}

		Orm.Commit()
		// WebSocket broadcast - status değişimi
		go func() {
			if publishErr := notificationevent.Publish(utilities.NotificationHub, notificationevent.Status{Type: "randevu_talebi_status", Rrid: Rrid, NewStatus: NewStatus}, notificationevent.CurrentRecipients(utilities.UserStatusReader, notificationevent.Recipient)); publishErr != nil {
				log.Print("notification: publication failed")
			}
		}()

		return c.JSON(fiber.Map{
			"status":     201,
			"message":    "Randevu talebi başarıyla güncellendi.",
			"new_status": NewStatus,
		})
	}
}

func AddRandevu(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		var inputs models.Randevular
		if err := c.BodyParser(&inputs); err != nil {
			log.Printf("operation=AddRandevu stage=request_parse")
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Geçersiz veri formatı",
			})
		}

		// Validate required fields
		if inputs.PatientFirstName == "" || inputs.PatientLastName == "" || inputs.PatientPhone == "" {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Hasta adı, soyadı ve telefon numarası zorunludur",
			})
		}

		if inputs.Sid == "" || inputs.AppointmentDate.IsZero() || inputs.AppointmentTime.IsZero() {
			return c.Status(400).JSON(fiber.Map{
				"status":  400,
				"message": "Şube, randevu tarihi ve saati zorunludur",
			})
		}

		requestID, status := createRequestID(inputs.Rrid)
		if status != 0 {
			return c.Status(status).JSON(fiber.Map{"status": status, "message": "Geçersiz randevu talebi"})
		}
		if utilities == nil || utilities.Orm == nil || utilities.Orm.Pool == nil {
			return c.Status(503).JSON(fiber.Map{"status": 503, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}
		// The user row, request row and appointment insert share this request's transaction.
		tx, err := utilities.Orm.Pool.BeginTx(c.UserContext(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			log.Printf("operation=AddRandevu stage=transaction_begin")
			return c.Status(503).JSON(fiber.Map{
				"status":  503,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}
		transactionFinished := false
		defer func() {
			if !transactionFinished {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Printf("operation=AddRandevu stage=transaction_rollback")
				}
			}
		}()
		branchID, doctorID, status := authorizeCreateAppointment(c.UserContext(), tx, ourUser.Uid, inputs, requestID)
		if status != 0 {
			return createAppointmentFailure(c, status)
		}
		if status = checkCreateAppointmentSlot(c.UserContext(), tx, inputs, doctorID); status != 0 {
			return createAppointmentFailure(c, status)
		}
		appointmentSnapshot, err := appointmentworkflowsnapshot.Read(c.UserContext(), utilities.AppointmentWorkflowSnapshotReader)
		if err != nil {
			log.Printf("operation=AddRandevu stage=options_read")
			return createAppointmentFailure(c, 503)
		}

		// Insert randevu
		columns := []string{"patient_first_name", "rrid", "patient_last_name", "patient_phone", "sid", "appointment_date", "appointment_time", "duration"}
		values := []any{inputs.PatientFirstName, requestID, inputs.PatientLastName, inputs.PatientPhone, branchID, inputs.AppointmentDate, inputs.AppointmentTime, inputs.Duration}

		// Add optional fields if provided
		if inputs.Price > 0 {
			columns = append(columns, "price")
			values = append(values, inputs.Price)
		}
		if inputs.Status != "" {
			columns = append(columns, "status")
			values = append(values, inputs.Status)
		}
		if inputs.PatientEmail != "" {
			columns = append(columns, "patient_email")
			values = append(values, inputs.PatientEmail)
		}
		if inputs.PatientTcKimlik != "" {
			columns = append(columns, "patient_tc_kimlik")
			values = append(values, inputs.PatientTcKimlik)
		}
		if !inputs.PatientBirthDate.IsZero() {
			columns = append(columns, "patient_birth_date")
			values = append(values, inputs.PatientBirthDate)
		}
		if inputs.PatientGender != "" {
			columns = append(columns, "patient_gender")
			values = append(values, inputs.PatientGender)
		}
		if inputs.Drid != "" && inputs.Drid != "0" {
			columns = append(columns, "drid")
			values = append(values, inputs.Drid)
		}
		if inputs.Brid != "" && inputs.Brid != "0" {
			columns = append(columns, "brid")
			values = append(values, inputs.Brid)
		}
		if inputs.Akid != "" && inputs.Akid != "0" {
			columns = append(columns, "akid")
			values = append(values, inputs.Akid)
		}
		if inputs.Tid != "" && inputs.Tid != "0" {
			columns = append(columns, "tid")
			values = append(values, inputs.Tid)
		}
		if inputs.Tbid != "" && inputs.Tbid != "0" {
			columns = append(columns, "tbid")
			values = append(values, inputs.Tbid)
		}
		if inputs.Complaint != "" {
			columns = append(columns, "complaint")
			values = append(values, inputs.Complaint)
		}
		if inputs.Notes != "" {
			columns = append(columns, "notes")
			values = append(values, inputs.Notes)
		}

		_, err = insertCreatedAppointment(c.UserContext(), tx, columns, values)

		if err != nil {
			log.Printf("operation=AddRandevu stage=record_insert")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Commit completes the SQL transaction even when it reports an error.
		err = tx.Commit()
		transactionFinished = true
		if err != nil {
			log.Printf("operation=AddRandevu stage=transaction_commit")
			return c.Status(500).JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Send email notification if email is provided and SMTP is configured
		if appointmentSnapshot.SMTPHost != "" && appointmentSnapshot.SMTPPort != 0 && appointmentSnapshot.SMTPUsername != "" && appointmentSnapshot.SMTPPassword != "" && inputs.PatientEmail != "" {
			RootDir := os.Getenv("ROOT_DIRECTORY")
			if RootDir == "" {
				log.Printf("operation=AddRandevu stage=message_build")
			} else {
				GetLogo := ""

				if appointmentSnapshot.SiteLogoPath != "" {
					GetLogo = filepath.Join(RootDir, "static", appointmentSnapshot.SiteLogoPath)
				} else {
					GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
				}

				SubeName := ""
				DoctorName := ""

				if readErr := utilities.Orm.Pool.QueryRowContext(c.UserContext(), "SELECT name FROM subeler WHERE sid = $1", branchID).Scan(&SubeName); readErr != nil {
					log.Printf("operation=AddRandevu stage=branch_read")
					SubeName = ""
				}

				if doctorID.Valid {
					var title, firstName, lastName string
					if readErr := utilities.Orm.Pool.QueryRowContext(c.UserContext(), "SELECT title, first_name, last_name FROM doktorlar WHERE drid = $1 AND sid = $2", doctorID.Int64, branchID).Scan(&title, &firstName, &lastName); readErr != nil {
						log.Printf("operation=AddRandevu stage=doctor_read")
					} else {
						DoctorName = title + " " + firstName + " " + lastName + " - " + SubeName
					}
				}

				LogoName := filepath.Base(GetLogo)

				Html := ""

				{
					// Create professional HTML email template for appointment confirmation
					dateString := fmt.Sprintf("%d.%d.%d", inputs.AppointmentDate.Day(), int(inputs.AppointmentDate.Month()), inputs.AppointmentDate.Year())
					timeString := fmt.Sprintf("%02d:%02d", inputs.AppointmentTime.Hour(), inputs.AppointmentTime.Minute())

					// Build doctor and clinic information
					doctorInfo := ""
					if DoctorName != "" {
						doctorInfo = fmt.Sprintf(`
									<div class="appointment-doctor" style="margin-top: 15px; font-size: 16px; color: #283b6a; font-weight: 600;">
										👨‍⚕️ %s
									</div>`, DoctorName)
					} else if SubeName != "" {
						doctorInfo = fmt.Sprintf(`
									<div class="appointment-clinic" style="margin-top: 15px; font-size: 16px; color: #283b6a; font-weight: 600;">
										🏥 %s
									</div>`, SubeName)
					}

					Html = fmt.Sprintf(`
						<!DOCTYPE html>
						<html lang="tr">
						<head>
							<meta charset="UTF-8">
							<meta name="viewport" content="width=device-width, initial-scale=1.0">
							<title>Randevunuz Oluşturuldu</title>
							<style>
								body {
									margin: 0;
									padding: 0;
									font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
									background-color: #f4f7fa;
									color: #333333;
								}
								.email-container {
									max-width: 600px;
									margin: 40px auto;
									background-color: #ffffff;
									border-radius: 12px;
									overflow: hidden;
									box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
								}
								.header {
									padding: 20px 30px;
									margin-top: 20px;
									text-align: center;
									color: #ffffff;
								}
								.header img {
									max-width: 180px;
									height: auto;
									margin-bottom: 20px;
									filter: brightness(0) invert(1);
								}
								.header h1 {
									margin: 0;
									font-size: 28px;
									font-weight: 600;
									letter-spacing: -0.5px;
									color: #252525 !important;
								}
								.header .subtitle {
									margin-top: 10px;
									font-size: 16px;
									opacity: 0.95;
									font-weight: 400;
									color: #252525 !important;
								}
								.content {
									padding: 40px 30px;
								}
								.appointment-highlight {
									background: linear-gradient(135deg, #e3f2fd 0%%, #f3e5f5 100%%);
									border-radius: 12px;
									padding: 25px;
									margin: 25px 0;
									text-align: center;
								}
								.appointment-date {
									font-size: 24px;
									font-weight: 700;
									color: %s;
									margin: 10px 0;
								}
								.appointment-time {
									font-size: 20px;
									font-weight: 600;
									color: %s;
									margin: 10px 0;
								}
								.appointment-doctor, .appointment-clinic {
									font-size: 16px;
									color: #283b6a;
									font-weight: 600;
									margin: 10px 0;
								}
								.message {
									font-size: 16px;
									line-height: 1.8;
									color: #555555;
									margin-bottom: 25px;
									text-align: center;
								}
								.contact-info {
									background-color: #f0f4ff;
									border-radius: 8px;
									padding: 20px;
									margin: 25px 0;
									text-align: center;
								}
								.contact-info a {
									color: %s;
									text-decoration: none;
									font-weight: 600;
								}
								.footer {
									background-color: #f8f9fc;
									padding: 30px;
									text-align: center;
									font-size: 13px;
									color: #888888;
									border-top: 1px solid #e1e8ed;
								}
								@media only screen and (max-width: 600px) {
									.email-container {
										margin: 0;
										border-radius: 0;
									}
									.header, .content, .footer {
										padding: 25px 20px;
									}
									.header h1 {
										font-size: 24px;
									}
								}
							</style>
						</head>
						<body>
							<div class="email-container">
								<div class="header">
									<img src="cid:%s" alt="%s" />
									<h1>Randevunuz Oluşturuldu</h1>
									<div class="subtitle">Sağlığınız Bizim Önceliğimiz</div>
								</div>
								
								<div class="content">
									<div class="appointment-highlight">
										<div class="appointment-date">%s</div>
										<div class="appointment-time">%s</div>%s
									</div>
									
									<div class="message">
										Randevunuz Başarıyla Oluşturulmuştur.<br>
										Bizi tercih ettiğiniz için teşekkür ederiz.
									</div>
									
									<div class="contact-info">
										<p>Randevunuzla ilgili sorularınız için:</p>
										<p><strong>Telefon:</strong> <a href="tel:%s">%s</a></p>
										<p><strong>E-posta:</strong> <a href="mailto:%s">%s</a></p>
									</div>
								</div>
								
								<div class="footer">
									<p><strong>%s</strong></p>
									<p>© 2025 %s. Tüm hakları saklıdır.</p>
								</div>
							</div>
						</body>
						</html>
						`, appointmentSnapshot.PrimaryColor, appointmentSnapshot.SecondaryColor,
						appointmentSnapshot.PrimaryColor,
						LogoName, appointmentSnapshot.SiteName,
						dateString, timeString, doctorInfo,
						appointmentSnapshot.ContactPhone, appointmentSnapshot.ContactPhone,
						appointmentSnapshot.ContactEmail, appointmentSnapshot.ContactEmail,
						appointmentSnapshot.SiteName, appointmentSnapshot.SiteName)

				}

				CreateEmailInfos := models.EmailInfos{
					From:        appointmentSnapshot.SiteName,
					To:          []string{inputs.PatientEmail},
					Username:    appointmentSnapshot.SMTPUsername,
					Password:    appointmentSnapshot.SMTPPassword,
					Host:        appointmentSnapshot.SMTPHost,
					Port:        lib.Int64(appointmentSnapshot.SMTPPort),
					Subject:     "Randevunuz Oluşturuldu - " + appointmentSnapshot.SiteName,
					PlainText:   "Sayın " + inputs.PatientFirstName + " " + inputs.PatientLastName + ", randevunuz başarıyla oluşturulmuştur.",
					Body:        Html,
					Attachments: []string{GetLogo},
				}

				err = lib.DeliverEmailAfterPersistence(func() error {
					if addAppointmentMailHook != nil {
						return addAppointmentMailHook(&CreateEmailInfos)
					}
					return lib.SendEmail(&CreateEmailInfos)
				}, nil)

				if err != nil {
					log.Printf("operation=AddRandevu stage=%s", lib.EmailFailureStage(err))
				}
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Randevu başarıyla oluşturuldu.",
		})
	}
}

func EditRandevu(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)

		if err != nil {
			return c.JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		Rid := c.Params("rid")

		var inputs models.RandevularEdit
		err = c.BodyParser(&inputs)

		if err != nil {
			log.Printf("operation=EditRandevu stage=request_parse")
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Invalid request body",
			})
		}

		if inputs.Rid == "" {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Randevu ID is required",
			})
		}

		appointmentID, valid := validEditAppointmentID(Rid, inputs.Rid)
		if !valid {
			return c.JSON(fiber.Map{"status": 400, "message": "Invalid appointment ID"})
		}
		if utilities == nil || utilities.Orm == nil || utilities.Orm.Pool == nil {
			return c.JSON(fiber.Map{"status": 503, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}
		db := utilities.Orm.Pool
		tx, err := db.BeginTx(c.UserContext(), nil)
		if err != nil {
			log.Printf("operation=EditRandevu stage=transaction_begin")
			return c.JSON(fiber.Map{"status": 503, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}
		transactionFinished := false
		defer func() {
			if !transactionFinished {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Printf("operation=EditRandevu stage=transaction_rollback")
				}
			}
		}()
		targetSID, accessStatus := authorizeEditAppointment(c.UserContext(), tx, ourUser.Uid, appointmentID)
		if accessStatus != 0 {
			message := "Forbidden"
			if accessStatus == 404 {
				message = "Randevu not found"
			} else if accessStatus == 503 {
				message = "Server Hatası: Lütfen daha sonra tekrar deneyin."
			}
			return c.JSON(fiber.Map{"status": accessStatus, "message": message})
		}
		if destinationStatus := validateEditDestination(c.UserContext(), tx, inputs.Sid, inputs.Drid); destinationStatus != 0 {
			message := "Invalid appointment destination"
			if destinationStatus == 503 {
				message = "Server Hatası: Lütfen daha sonra tekrar deneyin."
			}
			return c.JSON(fiber.Map{"status": destinationStatus, "message": message})
		}

		appointmentSnapshot, err := appointmentworkflowsnapshot.Read(c.UserContext(), utilities.AppointmentWorkflowSnapshotReader)

		if err != nil {
			log.Printf("operation=EditRandevu stage=options_read")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if inputs.Drid != "" {
			ExactDate := inputs.AppointmentDate.Format("2006-01-02")
			ExactStartTime := inputs.AppointmentTime.Format("15:04:05")
			ExactEndTime := inputs.AppointmentTime.Add(time.Duration(inputs.Duration) * time.Minute).Format("15:04:05")

			var conflicts int64
			err = tx.QueryRowContext(c.UserContext(), `SELECT COUNT(*) FROM randevular
				WHERE drid = $1 AND rid != $2 AND appointment_date = $3
				AND (appointment_time + (duration || ' minutes')::interval) > $4
				AND appointment_time < $5`, inputs.Drid, appointmentID, ExactDate, ExactStartTime, ExactEndTime).Scan(&conflicts)

			if err != nil {
				log.Printf("operation=EditRandevu stage=availability_read")
				return c.JSON(fiber.Map{
					"status":  500,
					"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
				})
			}

			if conflicts > 0 {
				log.Printf("operation=EditRandevu stage=availability_conflict")
				return c.JSON(fiber.Map{
					"status":  400,
					"message": "Seçilen doktorun bu tarih ve saatte başka bir randevusu bulunmaktadır",
				})
			}
		}

		// Execute update
		UpdateQuery := newEditAppointmentUpdate(c.UserContext(), tx, appointmentID, targetSID)

		SomethingSet := false

		if inputs.PatientFirstName != inputs.OldPatientFirstName {
			UpdateQuery.Set("patient_first_name", inputs.PatientFirstName)
			SomethingSet = true
		}

		if inputs.PatientLastName != inputs.OldPatientLastName {
			UpdateQuery.Set("patient_last_name", inputs.PatientLastName)
			SomethingSet = true
		}

		if inputs.PatientPhone != inputs.OldPatientPhone {
			UpdateQuery.Set("patient_phone", inputs.PatientPhone)
			SomethingSet = true
		}

		if inputs.PatientEmail != inputs.OldPatientEmail {
			UpdateQuery.Set("patient_email", inputs.PatientEmail)
			SomethingSet = true
		}

		if inputs.PatientTcKimlik != inputs.OldPatientTcKimlik {
			UpdateQuery.Set("patient_tc_kimlik", inputs.PatientTcKimlik)
			SomethingSet = true
		}

		if inputs.PatientBirthDate != inputs.OldPatientBirthDate {
			UpdateQuery.Set("patient_birth_date", inputs.PatientBirthDate)
			SomethingSet = true
		}

		if inputs.PatientGender != inputs.OldPatientGender {
			UpdateQuery.Set("patient_gender", inputs.PatientGender)
			SomethingSet = true
		}

		if inputs.Brid != inputs.OldBrid {
			if inputs.Brid == "" {
				UpdateQuery.Set("brid", nil)
			} else {
				UpdateQuery.Set("brid", inputs.Brid)
			}

			SomethingSet = true
		}

		if !sameEditBranch(inputs.Sid, targetSID) {
			if inputs.Sid == "" {
				UpdateQuery.Set("sid", nil)
				UpdateQuery.Set("drid", nil)
			} else {
				UpdateQuery.Set("sid", inputs.Sid)

				if inputs.Drid == "" {
					UpdateQuery.Set("drid", nil)
				} else {
					UpdateQuery.Set("drid", inputs.Drid)
				}
			}

			SomethingSet = true
		}

		if sameEditBranch(inputs.Sid, targetSID) && inputs.Drid != inputs.OldDrid {
			if inputs.Drid == "" {
				UpdateQuery.Set("drid", nil)
			} else {
				UpdateQuery.Set("drid", inputs.Drid)
			}

			SomethingSet = true
		}

		if inputs.Akid != inputs.OldAkid {
			if inputs.Akid == "" {
				UpdateQuery.Set("akid", nil)
			} else {
				UpdateQuery.Set("akid", inputs.Akid)
			}

			SomethingSet = true
		}

		if inputs.Tid != inputs.OldTid {
			if inputs.Tid == "" {
				UpdateQuery.Set("tid", nil)
			} else {
				UpdateQuery.Set("tid", inputs.Tid)
			}

			SomethingSet = true
		}

		if inputs.Tbid != inputs.OldTbid {
			if inputs.Tbid == "" {
				UpdateQuery.Set("tbid", nil)
			} else {
				UpdateQuery.Set("tbid", inputs.Tbid)
			}

			SomethingSet = true
		}

		if inputs.AppointmentDate != inputs.OldAppointmentDate {
			UpdateQuery.Set("appointment_date", inputs.AppointmentDate)
			SomethingSet = true
		}

		if inputs.AppointmentTime != inputs.OldAppointmentTime {
			UpdateQuery.Set("appointment_time", inputs.AppointmentTime)
			SomethingSet = true
		}

		if inputs.Duration != inputs.OldDuration {
			UpdateQuery.Set("duration", inputs.Duration)
			SomethingSet = true
		}

		if inputs.Status != inputs.OldStatus {
			UpdateQuery.Set("status", inputs.Status)
			SomethingSet = true
		}

		if inputs.Notes != inputs.OldNotes {
			UpdateQuery.Set("notes", inputs.Notes)
			SomethingSet = true
		}

		if inputs.Complaint != inputs.OldComplaint {
			UpdateQuery.Set("complaint", inputs.Complaint)
			SomethingSet = true
		}

		if inputs.CancelReason != inputs.OldCancelReason {
			UpdateQuery.Set("cancel_reason", inputs.CancelReason)
			SomethingSet = true
		}

		if inputs.ReminderSent != inputs.OldReminderSent {
			UpdateQuery.Set("reminder_sent", inputs.ReminderSent)
			SomethingSet = true
		}

		if inputs.ConfirmationCode != inputs.OldConfirmationCode {
			UpdateQuery.Set("confirmation_code", inputs.ConfirmationCode)
			SomethingSet = true
		}

		if inputs.Price != inputs.OldPrice {
			UpdateQuery.Set("price", inputs.Price)
			SomethingSet = true
		}

		if inputs.PaymentStatus != inputs.OldPaymentStatus {
			UpdateQuery.Set("payment_status", inputs.PaymentStatus)
			SomethingSet = true
		}

		if !SomethingSet {
			return c.JSON(fiber.Map{
				"status":  400,
				"message": "Nothing changed",
			})
		}

		UpdateQuery.Set("updated_at", "NOW()")

		err = UpdateQuery.Execute()

		if err != nil {
			log.Printf("operation=EditRandevu stage=record_update")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		// Check if any rows were affected
		rowsAffected, err := UpdateQuery.RowsAffected()
		if err != nil {
			log.Printf("operation=EditRandevu stage=affected_rows")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}

		if rowsAffected == 0 {
			return c.JSON(fiber.Map{
				"status":  404,
				"message": "Randevu not found or no changes made",
			})
		}

		// Commit transaction
		err = tx.Commit()
		// database/sql closes the transaction even when Commit returns an error.
		transactionFinished = true
		if err != nil {
			log.Printf("operation=EditRandevu stage=transaction_commit")
			return c.JSON(fiber.Map{
				"status":  500,
				"message": "Server Hatası: Lütfen daha sonra tekrar deneyin.",
			})
		}
		if (inputs.AppointmentDate != inputs.OldAppointmentDate || inputs.AppointmentTime != inputs.OldAppointmentTime || inputs.Duration != inputs.OldDuration || inputs.Drid != inputs.OldDrid) &&
			inputs.Drid != "" && inputs.Drid != "0" &&
			(appointmentSnapshot.SMTPHost != "" && appointmentSnapshot.SMTPPort != 0 && appointmentSnapshot.SMTPUsername != "" && appointmentSnapshot.SMTPPassword != "" && inputs.PatientEmail != "") {
			RootDir := os.Getenv("ROOT_DIRECTORY")

			if RootDir == "" {
				log.Printf("operation=EditRandevu stage=message_build")
			} else {
				GetLogo := ""

				if appointmentSnapshot.SiteLogoPath != "" {
					GetLogo = filepath.Join(RootDir, "static", appointmentSnapshot.SiteLogoPath)
				} else {
					GetLogo = filepath.Join(RootDir, "static", "files", "defaults", "logo", "n-hospital-logo.png")
				}

				SubeName := ""
				DoctorName := ""

				if inputs.Sid != "" && inputs.Sid != "0" {
					if err = db.QueryRowContext(c.UserContext(), "SELECT name FROM subeler WHERE sid = $1", inputs.Sid).Scan(&SubeName); err != nil {
						log.Printf("operation=EditRandevu stage=branch_read")
						SubeName = ""
					}
				}

				if inputs.Drid != "" && inputs.Drid != "0" {
					var title, firstName, lastName string
					if err = db.QueryRowContext(c.UserContext(), "SELECT title, first_name, last_name FROM doktorlar WHERE drid = $1", inputs.Drid).Scan(&title, &firstName, &lastName); err != nil {
						log.Printf("operation=EditRandevu stage=doctor_read")
					} else {
						DoctorName = title + " " + firstName + " " + lastName + " - " + SubeName
					}
				}

				LogoName := filepath.Base(GetLogo)

				Html := ""

				{
					// Create professional HTML email template for appointment confirmation
					dateString := fmt.Sprintf("%d.%d.%d", inputs.AppointmentDate.Day(), int(inputs.AppointmentDate.Month()), inputs.AppointmentDate.Year())
					timeString := fmt.Sprintf("%02d:%02d", inputs.AppointmentTime.Hour(), inputs.AppointmentTime.Minute())

					// Build doctor and clinic information
					doctorInfo := ""
					if DoctorName != "" {
						doctorInfo = fmt.Sprintf(`
									<div class="appointment-doctor" style="margin-top: 15px; font-size: 16px; color: #283b6a; font-weight: 600;">
										👨‍⚕️ %s
									</div>`, DoctorName)
					} else if SubeName != "" {
						doctorInfo = fmt.Sprintf(`
									<div class="appointment-clinic" style="margin-top: 15px; font-size: 16px; color: #283b6a; font-weight: 600;">
										🏥 %s
									</div>`, SubeName)
					}

					Html = fmt.Sprintf(`
						<!DOCTYPE html>
						<html lang="tr">
						<head>
							<meta charset="UTF-8">
							<meta name="viewport" content="width=device-width, initial-scale=1.0">
							<title>Randevunuz Güncellendi</title>
							<style>
								body {
									margin: 0;
									padding: 0;
									font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
									background-color: #f4f7fa;
									color: #333333;
								}
								.email-container {
									max-width: 600px;
									margin: 40px auto;
									background-color: #ffffff;
									border-radius: 12px;
									overflow: hidden;
									box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
								}
								.header {
									padding: 20px 30px;
									margin-top: 20px;
									text-align: center;
									color: #ffffff;
								}
								.header img {
									max-width: 180px;
									height: auto;
									margin-bottom: 20px;
									filter: brightness(0) invert(1);
								}
								.header h1 {
									margin: 0;
									font-size: 28px;
									font-weight: 600;
									letter-spacing: -0.5px;
									color: #252525 !important;
								}
								.header .subtitle {
									margin-top: 10px;
									font-size: 16px;
									opacity: 0.95;
									font-weight: 400;
									color: #252525 !important;
								}
								.content {
									padding: 40px 30px;
								}
								.appointment-highlight {
									background: linear-gradient(135deg, #e3f2fd 0%%, #f3e5f5 100%%);
									border-radius: 12px;
									padding: 25px;
									margin: 25px 0;
									text-align: center;
								}
								.appointment-date {
									font-size: 24px;
									font-weight: 700;
									color: %s;
									margin: 10px 0;
								}
								.appointment-time {
									font-size: 20px;
									font-weight: 600;
									color: %s;
									margin: 10px 0;
								}
								.appointment-doctor, .appointment-clinic {
									font-size: 16px;
									color: #283b6a;
									font-weight: 600;
									margin: 10px 0;
								}
								.message {
									font-size: 16px;
									line-height: 1.8;
									color: #555555;
									margin-bottom: 25px;
									text-align: center;
								}
								.contact-info {
									background-color: #f0f4ff;
									border-radius: 8px;
									padding: 20px;
									margin: 25px 0;
									text-align: center;
								}
								.contact-info a {
									color: %s;
									text-decoration: none;
									font-weight: 600;
								}
								.footer {
									background-color: #f8f9fc;
									padding: 30px;
									text-align: center;
									font-size: 13px;
									color: #888888;
									border-top: 1px solid #e1e8ed;
								}
								@media only screen and (max-width: 600px) {
									.email-container {
										margin: 0;
										border-radius: 0;
									}
									.header, .content, .footer {
										padding: 25px 20px;
									}
									.header h1 {
										font-size: 24px;
									}
								}
							</style>
						</head>
						<body>
							<div class="email-container">
								<div class="header">
									<img src="cid:%s" alt="%s" />
									<h1>Randevu Bilgileriniz Değiştirilmiştir. <h1>
									<div class="subtitle">Sağlığınız Bizim Önceliğimiz</div>
								</div>
								
								<div class="content">
									<div class="appointment-highlight">
										<div class="appointment-date">%s</div>
										<div class="appointment-time">%s</div>%s
									</div>
									
									<div class="message">
										Randevu bilgileriniz değiştirilmiştir.<br>
										Bizi tercih ettiğiniz için teşekkür ederiz.
									</div>
									
									<div class="contact-info">
										<p>Randevunuzla ilgili sorularınız için:</p>
										<p><strong>Telefon:</strong> <a href="tel:%s">%s</a></p>
										<p><strong>E-posta:</strong> <a href="mailto:%s">%s</a></p>
									</div>
								</div>
								
								<div class="footer">
									<p><strong>%s</strong></p>
									<p>© 2025 %s. Tüm hakları saklıdır.</p>
								</div>
							</div>
						</body>
						</html>
						`, appointmentSnapshot.PrimaryColor, appointmentSnapshot.SecondaryColor,
						appointmentSnapshot.PrimaryColor,
						LogoName, appointmentSnapshot.SiteName,
						dateString, timeString, doctorInfo,
						appointmentSnapshot.ContactPhone, appointmentSnapshot.ContactPhone,
						appointmentSnapshot.ContactEmail, appointmentSnapshot.ContactEmail,
						appointmentSnapshot.SiteName, appointmentSnapshot.SiteName)

				}

				CreateEmailInfos := models.EmailInfos{
					From:        appointmentSnapshot.SiteName,
					To:          []string{inputs.PatientEmail},
					Username:    appointmentSnapshot.SMTPUsername,
					Password:    appointmentSnapshot.SMTPPassword,
					Host:        appointmentSnapshot.SMTPHost,
					Port:        lib.Int64(appointmentSnapshot.SMTPPort),
					Subject:     "Randevunuz yeniden düzenlendi - " + appointmentSnapshot.SiteName,
					PlainText:   "Sayın " + inputs.PatientFirstName + " " + inputs.PatientLastName + ", randevu bilgilerinizde değişiklik yapılmıştır.",
					Body:        Html,
					Attachments: []string{GetLogo},
				}

				err = lib.DeliverEmailAfterPersistence(
					func() error {
						if editAppointmentMailHook != nil {
							return editAppointmentMailHook(&CreateEmailInfos)
						}
						return lib.SendEmail(&CreateEmailInfos)
					}, nil,
				)

				if err != nil {
					log.Printf("operation=EditRandevu stage=%s", lib.EmailFailureStage(err))
				}
			}
		}

		return c.JSON(fiber.Map{
			"status":  201,
			"message": "Randevu updated successfully",
		})
	}
}

func DeleteRandevu(states *models.AppState, utilities *models.Utilities) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := lib.CheckAuth(c)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"status":  401,
				"message": "Unauthorized",
			})
		}

		appointmentID, valid := editPositiveID(c.Params("rid"))
		if !valid {
			return c.Status(400).JSON(fiber.Map{"status": 400, "message": "Invalid appointment ID"})
		}
		if utilities == nil || utilities.Orm == nil || utilities.Orm.Pool == nil {
			return c.Status(503).JSON(fiber.Map{"status": 503, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}

		// This request owns its transaction. The current user and target row are
		// locked before deletion; no client-supplied branch is an authority source.
		tx, err := utilities.Orm.Pool.BeginTx(c.UserContext(), nil)
		if err != nil {
			log.Printf("operation=DeleteRandevu stage=transaction_begin")
			return c.Status(503).JSON(fiber.Map{"status": 503, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}
		transactionFinished := false
		defer func() {
			if !transactionFinished {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Printf("operation=DeleteRandevu stage=transaction_rollback")
				}
			}
		}()

		targetSID, accessStatus := authorizeEditAppointment(c.UserContext(), tx, ourUser.Uid, appointmentID)
		if accessStatus != 0 {
			message := "Forbidden"
			if accessStatus == 404 {
				message = "Randevu bulunamadı"
			} else if accessStatus == 503 {
				message = "Server Hatası: Lütfen daha sonra tekrar deneyin."
			}
			return c.Status(accessStatus).JSON(fiber.Map{"status": accessStatus, "message": message})
		}

		result, err := tx.ExecContext(c.UserContext(), "DELETE FROM randevular WHERE rid = $1 AND sid IS NOT DISTINCT FROM $2", appointmentID, targetSID)
		if err != nil {
			log.Printf("operation=DeleteRandevu stage=record_delete")
			return c.Status(500).JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil || rowsAffected != 1 {
			if err != nil || rowsAffected > 1 {
				log.Printf("operation=DeleteRandevu stage=affected_rows")
				return c.Status(500).JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
			}
			return c.Status(404).JSON(fiber.Map{"status": 404, "message": "Randevu bulunamadı"})
		}

		err = tx.Commit()
		// database/sql closes the transaction even when Commit reports failure.
		transactionFinished = true
		if err != nil {
			log.Printf("operation=DeleteRandevu stage=transaction_commit")
			return c.Status(500).JSON(fiber.Map{"status": 500, "message": "Server Hatası: Lütfen daha sonra tekrar deneyin."})
		}

		return c.JSON(fiber.Map{
			"status":  200,
			"message": "Randevu başarıyla silindi.",
		})
	}
}
