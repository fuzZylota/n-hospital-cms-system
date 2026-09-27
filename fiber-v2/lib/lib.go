package lib

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gofiber/contrib/websocket"
	"golang.org/x/crypto/bcrypt"

	"github.com/lib/pq"

	"lib/userstatus"
	"models"
	"models/data"

	"github.com/gofiber/fiber/v2"
)

func GenerateRandomString(length int) []byte {
	b := make([]byte, length)
	rand.Read(b)
	return b
}

func Encrypt(plaintext []byte) (string, error) {
	key := os.Getenv("ENCRYPTION_KEY")

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize()) // genelde 12
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ct := aead.Seal(nil, nonce, plaintext, nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt: base64(nonce|ciphertext) alır ve çözer.
func Decrypt(b64 string) ([]byte, error) {
	key := os.Getenv("ENCRYPTION_KEY")

	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ns := aead.NonceSize()
	if len(data) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce := data[:ns]
	ct := data[ns:]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func Int64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func String(v interface{}) string {
	switch n := v.(type) {
	case string:
		return n
	case int64:
		return strconv.FormatInt(n, 10)
	case int:
		return strconv.Itoa(n)
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(n)
	case nil:
		return ""
	default:
		return ""
	}
}

func Bool(v interface{}) bool {
	switch n := v.(type) {
	case bool:
		return n
	case int64, int, float64:
		if n == 0 {
			return false
		}
		return true
	case string:
		n = strings.ToLower(n)
		if n == "" || n == "0" || n == "false" {
			return false
		}

		return true
	default:
		return false
	}
}

func Float64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case float32:
		return float64(n)
	case string:
		nFloat, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0
		}
		return nFloat
	default:
		return 0
	}
}

func StringArray(v interface{}) []string {
	switch n := v.(type) {
	case []string:
		return n
	case string:
		var arr pq.StringArray
		if err := arr.Scan(n); err != nil {
			return []string{}
		}
		return []string(arr)
	default:
		return []string{}
	}
}

func Time(v interface{}) time.Time {
	switch n := v.(type) {
	case time.Time:
		if n.Format("2006-01-02 15:04:05.999999-07") == "0001-01-01 00:00:00 +0000 UTC" {
			return time.Time{}
		} else {
			return n
		}
	default:
		return time.Time{}
	}
}

func CreateJWT(User models.AuthenticatedUser) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)

	claims["Nickname"] = User.Name
	claims["Email"] = User.Email
	claims["Phone"] = User.Phone
	claims["Name"] = User.Name
	claims["Surname"] = User.Surname
	claims["Uid"] = User.Uid
	claims["LastLogin"] = User.LastLogin.Format("2006-01-02 15:04:05.999999-07")
	claims["Role"] = User.Role
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	claims["iat"] = time.Now().Unix()

	secret := os.Getenv("JWT_SECRET")

	tokenString, err := token.SignedString([]byte(secret))

	if err != nil {
		log.Printf("Error is: %s \n", err)
		return "", errors.New("couldn't sign that token")
	}

	return tokenString, nil
}

func VerifyJWT(tokenString string) error {
	_, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		_, tokenCorrect := t.Method.(*jwt.SigningMethodHMAC)

		if !tokenCorrect {
			return "", errors.New("malformed token")
		}

		secret := os.Getenv("JWT_SECRET")

		return []byte(secret), nil
	})

	if err != nil {
		log.Printf("Error is: %s \n", err)
		return err
	}

	return nil
}

func GetJWT(c any) (models.AuthenticatedUser, error) {
	ourAuthCookie := ""
	switch conn := c.(type) {
	case *fiber.Ctx:
		cookieName := os.Getenv("AUTH_COOKIE_NAME")

		if cookieName == "" {
			cookieName = "n-hospital-auth"
		}

		ourAuthCookie = conn.Cookies(cookieName)
	case *websocket.Conn:
		cookieName := os.Getenv("AUTH_COOKIE_NAME")

		if cookieName == "" {
			cookieName = "n-hospital-auth"
		}

		ourAuthCookie = conn.Cookies(cookieName)
	default:
		return models.AuthenticatedUser{}, errors.New("invalid argument given to the GetJWT function")
	}

	if ourAuthCookie == "" {
		return models.AuthenticatedUser{}, nil
	}

	err := VerifyJWT(ourAuthCookie)

	if err != nil {
		log.Printf("Error is: %s \n", err)
		return models.AuthenticatedUser{}, errors.New("malformed Token")
	}

	ourToken, err := jwt.Parse(ourAuthCookie, func(t *jwt.Token) (interface{}, error) {
		_, tokenCorrect := t.Method.(*jwt.SigningMethodHMAC)

		if !tokenCorrect {
			return "", errors.New("malformed token")
		}

		secret := os.Getenv("JWT_SECRET")

		return []byte(secret), nil
	})

	if err != nil {
		log.Printf("error is: %s \n", err)
		return models.AuthenticatedUser{}, errors.New("malformed token")
	}

	actualToken, _ := ourToken.Claims.(jwt.MapClaims)

	OurUser := models.AuthenticatedUser{
		Name:    actualToken["Name"].(string),
		Email:   actualToken["Email"].(string),
		Phone:   actualToken["Phone"].(string),
		Surname: actualToken["Surname"].(string),
		Role:    actualToken["Role"].(string),
		Uid:     String(actualToken["Uid"]),
	}

	if actualToken["IsActive"] != nil {
		OurUser.IsActive = actualToken["IsActive"].(bool)
	}

	if actualToken["Remember"] != nil {
		OurUser.Remember = actualToken["Remember"].(bool)
	}

	if actualToken["LastLogin"] != nil {
		OurUser.LastLogin = StandardizePostgresTimestampWithTimeZone(actualToken["LastLogin"].(string))
	}

	return OurUser, nil
}

func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// "c.Cookies()" ile çerezleri elde edilir
		cookieName := os.Getenv("AUTH_COOKIE_NAME")

		if cookieName == "" {
			cookieName = "n-hospital-auth"
		}

		ourAuthCookie := c.Cookies(cookieName)

		if ourAuthCookie == "" {
			return c.Next()
		}

		err := VerifyJWT(ourAuthCookie)
		if err != nil {
			log.Printf("Error is: %s \n", err)
			return c.Next()
		}

		ourUser, err := GetJWT(c)
		if err != nil {
			log.Printf("Error is: %s \n", err)
			return c.Next()
		}

		tokenString, err := CreateJWT(ourUser)
		if err != nil {
			log.Printf("Error is: %s \n", err)
			return c.Next()
		}

		// Yeni çerez oluşturup tarayıcıya ekleyin
		cookie := fiber.Cookie{
			Name:     cookieName,
			Value:    tokenString,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Lax",
		}

		if ourUser.Remember {
			cookie.MaxAge = 7200 * 12 * 30
		} else {
			cookie.MaxAge = 7200
		}

		c.Cookie(&cookie)

		return c.Next()
	}
}

func HandleUserBanning(reader data.UserStatusReader) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ourUser, err := GetJWT(c)
		if err == nil && ourUser.Uid != "" && userstatus.ShouldExpireAuthCookies(c.UserContext(), reader, ourUser.Uid) {
			cookieName := os.Getenv("AUTH_COOKIE_NAME")

			if cookieName == "" {
				cookieName = "n-hospital-auth"
			}

			cookie := fiber.Cookie{
				Name:     cookieName,
				Value:    "",
				Expires:  time.Now().Add(-time.Hour * 24),
				HTTPOnly: true,
				MaxAge:   0,
			}

			c.Cookie(&cookie)
		}

		return c.Next()
	}
}

// warning, it's only compatible with that crate: "github.com/gofiber/websocket/v2"
func CheckAuth(c any) (models.AuthenticatedUser, error) {
	switch c.(type) {
	case *fiber.Ctx, *websocket.Conn:
		ourUser, err := GetJWT(c)

		if err != nil {
			return models.AuthenticatedUser{}, err
		}

		if ourUser.Uid == "" {
			return models.AuthenticatedUser{}, errors.New("unauthorized")
		}

		return ourUser, nil
	default:
		return models.AuthenticatedUser{}, errors.New("Invalid argument on CheckAuth method.")
	}

}

func StandardizePostgresTimestampWithTimeZone(timeString string) time.Time {
	layout := "2006-01-02 15:04:05.999999-07"
	t, err := time.Parse(layout, timeString)

	if err != nil {
		log.Printf("Error is: %s \n", err)
		return time.Time{}
	}

	return t.UTC()
}

// if it returns true, the value shouldnt be sent to the database
func IsShouldntSentToDatabase(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	case reflect.Struct:
		// Tüm struct field’larını kontrol et, hepsi sıfırsa true döndür
		for i := 0; i < v.NumField(); i++ {
			if !IsShouldntSentToDatabase(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

type UniqueFilePathResponse struct {
	BaseName  string
	FilePath  string
	Extension string
	Error     error
}

func UniqueFilePath(path string) (UniqueFilePathResponse, error) {
	// Dosya mevcut değilse direk onu kullan
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return UniqueFilePathResponse{
			BaseName:  filepath.Base(path),
			FilePath:  path,
			Extension: filepath.Ext(path),
			Error:     nil,
		}, nil
	} else if err != nil {
		return UniqueFilePathResponse{
			BaseName:  "",
			FilePath:  "",
			Extension: "",
			Error:     fmt.Errorf("dosya kontrol edilemedi: %w", err),
		}, nil
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Sayacımız
	counter := 1
	for {
		newName := fmt.Sprintf("%s (%d)%s", name, counter, ext)
		newPath := filepath.Join(dir, newName)

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return UniqueFilePathResponse{
				BaseName:  newName,
				FilePath:  newPath,
				Extension: ext,
				Error:     nil,
			}, nil
		} else if err != nil {
			return UniqueFilePathResponse{
				BaseName:  "",
				FilePath:  "",
				Extension: "",
				Error:     fmt.Errorf("dosya kontrol edilemedi: %w", err),
			}, nil
		}
		counter++
	}
}

// Yüklenen dosyanın uzantısının izin verilen türler arasında olup olmadığını kontrol eder.
// Güvenlik: .php, .exe, .sh gibi tehlikeli dosyaların yüklenmesini engeller.
var allowedUploadExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".gif": true, ".svg": true, ".pdf": true, ".mp4": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".txt": true, ".csv": true,
	".webm": true, ".ogg": true, ".mp3": true, ".wav": true,
	".ico": true, ".bmp": true, ".tiff": true, ".tif": true,
}

func ValidateUploadedFile(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return fmt.Errorf("dosya uzantısı bulunamadı: %s", filename)
	}
	if !allowedUploadExtensions[ext] {
		return fmt.Errorf("izin verilmeyen dosya türü: %s", ext)
	}
	return nil
}

func SaveFileWithBuffering(dstDir string, fileHeader multipart.FileHeader) error {
	// Hedef klasör yoksa oluştur
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("hedef klasör oluşturulamadı: %w", err)
	}

	if err := ValidateUploadedFile(fileHeader.Filename); err != nil {
		return err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Hedef dosyayı oluştur
	dst, err := os.Create(dstDir + "/" + fileHeader.Filename)
	if err != nil {
		return fmt.Errorf("hedef dosya oluşturulamadı: %w", err)
	}
	defer dst.Close()

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				if err == io.EOF {
					break
				} else {
					return werr
				}

			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	// Diske sync et
	if err := dst.Sync(); err != nil {
		return fmt.Errorf("dosya sync edilemedi: %w", err)
	}

	return nil
}

func SaveFileWithBufferingWithRenaming(dstDir, fileName string, fileHeader multipart.FileHeader) error {
	// Hedef klasör yoksa oluştur
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("hedef klasör oluşturulamadı: %w", err)
	}

	if err := ValidateUploadedFile(fileHeader.Filename); err != nil {
		return err
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	defer file.Close()

	// Hedef dosyayı oluştur
	dst, err := os.Create(dstDir + "/" + fileName)
	if err != nil {
		return fmt.Errorf("hedef dosya oluşturulamadı: %w", err)
	}
	defer dst.Close()

	buf := make([]byte, 64*1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				if err == io.EOF {
					break
				} else {
					log.Printf("%v\n", werr)
					return werr
				}

			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("%v\n", err)
			return err
		}
	}

	// Diske sync et
	if err := dst.Sync(); err != nil {
		log.Printf("%v\n", err)
		return fmt.Errorf("dosya sync edilemedi: %w", err)
	}

	return nil
}

func ReadDirectory(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		if errors.Is(err, os.ErrNotExist) {
			return []os.DirEntry{}, nil
		}

		return nil, err
	}

	return entries, nil
}

func DeleteFile(filePath string) error {
	err := os.Remove(filePath)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		log.Printf("Cannot remove file: %v %T\n", err, err)
		return err
	}

	return nil
}

func MakeTimeHumanReadable(GivenTime time.Time, UserTimezone string) string {
	loc, err := time.LoadLocation(UserTimezone)
	if err != nil {
		// Hatalı timezone verilirse UTC fallback
		loc = time.UTC
	}
	localTime := GivenTime.In(loc)
	return localTime.Format("15:04, 02/01/2006")
}

func MakeTimeHumanReadableWithoutNormalization(GivenTime time.Time) string {
	return GivenTime.Format("02.01.2006 15:04")
}

func ConvertTimeForTheDateInput(GivenTime time.Time, UserTimezone string) string {
	loc, err := time.LoadLocation(UserTimezone)
	if err != nil {
		// Hatalı timezone verilirse UTC fallback
		loc = time.UTC
	}
	localTime := GivenTime.In(loc)
	return localTime.Format("2006-01-02")
}

func ConvertTimeForDateTimeLocalInput(GivenTime time.Time, UserTimezone string) string {
	loc, err := time.LoadLocation(UserTimezone)
	if err != nil {
		// Hatalı timezone verilirse UTC fallback
		loc = time.UTC
	}
	localTime := GivenTime.In(loc)
	return localTime.Format("2006-01-02T15:04")
}

func ConvertTimeForTheDateForFrontend(GivenTime time.Time) string {
	return GivenTime.Format("2006-01-02")
}

func ConvertTimeForTheTimeForFrontend(GivenTime time.Time) string {
	return GivenTime.Format("15:04")
}

func ConvertTimeForTheMonthForFrontend(GivenTime time.Time) string {
	month := GivenTime.Month().String()

	GetLang := os.Getenv("LANG")

	if GetLang == "" {
		GetLang = "tr"
	}

	switch GetLang {
	case "en":
		return month
	case "tr", "türkiye", "turkey", "türkçe", "turkiye":
		switch month {
		case "January":
			return "Ocak"
		case "February":
			return "Şubat"
		case "March":
			return "Mart"
		case "April":
			return "Nisan"
		case "May":
			return "Mayıs"
		case "June":
			return "Haziran"
		case "July":
			return "Temmuz"
		case "August":
			return "Ağustos"
		case "September":
			return "Eylül"
		case "October":
			return "Ekim"
		case "November":
			return "Kasım"
		case "December":
			return "Aralık"
		}
	}

	return month
}

func ConvertTimeForTheDayForFrontend(GivenTime time.Time) int {
	return GivenTime.Day()
}

// FormatFrontendDate produces a Turkish-localized "15 Mart 2026" style date string.
var turkishFrontendMonthNames = map[time.Month]string{
	time.January:   "Ocak",
	time.February:  "Şubat",
	time.March:     "Mart",
	time.April:     "Nisan",
	time.May:       "Mayıs",
	time.June:      "Haziran",
	time.July:      "Temmuz",
	time.August:    "Ağustos",
	time.September: "Eylül",
	time.October:   "Ekim",
	time.November:  "Kasım",
	time.December:  "Aralık",
}

func FormatFrontendDate(GivenTime time.Time) string {
	if GivenTime.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d %s %d", GivenTime.Day(), turkishFrontendMonthNames[GivenTime.Month()], GivenTime.Year())
}

// IsDifferentDay reports whether two timestamps fall on different calendar days,
// used to decide whether a "last updated" date should be shown alongside the publish date.
func IsDifferentDay(a time.Time, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return false
	}
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay != by || am != bm || ad != bd
}

func ShowDateOfTimeInput(t time.Time, UserTimezone string) string {
	loc, err := time.LoadLocation(UserTimezone)
	if err != nil {
		// Hatalı timezone verilirse UTC fallback
		loc = time.UTC
	}
	localTime := t.In(loc)
	return localTime.Format("02/01/2006")
}

// ShowTimeOfTimeInput -> 14:05 gibi döner
func ShowTimeOfTimeInput(t time.Time, UserTimezone string) string {
	loc, err := time.LoadLocation(UserTimezone)
	if err != nil {
		// Hatalı timezone verilirse UTC fallback
		loc = time.UTC
	}
	localTime := t.In(loc)
	return localTime.Format("15:04")
}

func ContainsWrapper(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TurnStructIntoJson(v interface{}) string {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return string(template.JS(""))
	}

	return string(jsonBytes)
}

func SendEmail(infos *models.EmailInfos) error {
	if infos == nil {
		return &EmailError{stage: EmailMessageBuild, cause: errors.New("e-posta bilgileri eksik")}
	}

	return sendEmail(emailMessage{
		From:        infos.From,
		To:          infos.To,
		Username:    infos.Username,
		Password:    infos.Password,
		Host:        infos.Host,
		Port:        infos.Port,
		Subject:     infos.Subject,
		PlainText:   infos.PlainText,
		Body:        infos.Body,
		Attachments: infos.Attachments,
	}, buildEmailMessage, sendEmailSMTP)
}

func ShortenTextForFrontend(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max] // saçma durumlar için
	}
	return s[:max-3] + "..."
}

func WebsocketHandshake(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		// burası handshake aşaması
		// subprotocol’ü kontrol edebilirsin
		proto := c.Get("Sec-WebSocket-Protocol")
		//fmt.Println("client subprotocol:", proto)

		c.Locals("protocol", proto)

		log.Printf("Websocket protocol: %s", proto)

		c.Locals("allowed", true)

		return c.Next()
	}

	log.Printf("Websocket not upgraded")

	return fiber.ErrUpgradeRequired
}

func VerifyRecaptcha(token string, secretKey string) bool {
	data := url.Values{
		"secret":   {secretKey},
		"response": {token},
	}

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", data)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result models.RecaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	return result.Success
}

// SQL kimlik doğrulayıcıları (kolon adı / sıralama yönü) - panel liste
// sayfalarındaki "sort_by"/"sort_order" query parametreleri dogrudan
// kullanicidan geldigi ve ham SQL'e (ORDER BY) eklendigi icin, SQL
// injection'i engellemek amaciyla eklendi. Sadece harf/rakam/alt cizgi
// ve tek bir nokta (tablo takma adi icin, orn "rt.created_at") kabul
// edilir; uymayan her deger guvenli varsayilana dusurulur.
var validSqlIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

func SanitizeSortColumn(column string, fallback string) string {
	if validSqlIdentifier.MatchString(column) {
		return column
	}

	return fallback
}

func SanitizeSortOrder(order string) string {
	upper := strings.ToUpper(strings.TrimSpace(order))

	if upper == "ASC" || upper == "DESC" {
		return upper
	}

	return "DESC"
}

// Kullanici sifreleri onceden geri-donusturulebilir AES sifrelemesiyle
// (Encrypt/Decrypt) saklaniyordu - ENCRYPTION_KEY sizarsa tum sifreler
// duz metin olarak geri elde edilebilirdi. Bundan sonra tek yonlu bcrypt
// hash kullaniliyor; Encrypt/Decrypt sadece geriye donuk uyumluluk ve
// diger kullanim yerleri (SMTP sifresi gibi) icin dosyada kalmaya devam
// ediyor, sifreler icin artik kullanilmiyor.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func ComparePasswordHash(hash string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}

// PanelAuthMiddleware — panel ve backend route’ları için merkezi yetki kontrolü.
// Her controller’da tekrarlanan if OurUser.Role != "admin" kontrollerini
// ortadan kaldırır. Yetkilendirme unutulursa privilege escalation’ı önler.
func PanelAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := GetJWT(c)
		if err != nil || user.Uid == "" {
			// AJAX isteklerinde JSON, sayfa isteklerinde yönlendirme
			if c.Get("X-Requested-With") == "XMLHttpRequest" || c.Get("Accept") == "application/json" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  401,
					"message": "Bu işlemi gerçekleştirmek için giriş yapmanız gerekiyor.",
				})
			}
			return c.Redirect("/giris")
		}
		return c.Next()
	}
}
