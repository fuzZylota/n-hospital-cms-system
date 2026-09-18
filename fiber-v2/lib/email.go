package lib

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"mime"
	"mime/quotedprintable"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type emailMessage struct {
	From        string
	To          []string
	Username    string
	Password    string
	Host        string
	Port        int64
	Subject     string
	PlainText   string
	Body        string
	Attachments []string
}

type emailBuilder func(emailMessage) ([]byte, error)
type emailSender func(emailMessage, []byte) error

// EmailStage is a closed set of safe categories, never derived from input.
type EmailStage uint8

const (
	EmailMessageBuild EmailStage = iota + 1
	EmailSMTPDelivery
	EmailReplyStatePersist
)

func (stage EmailStage) String() string {
	switch stage {
	case EmailMessageBuild:
		return "message_build"
	case EmailSMTPDelivery:
		return "smtp_delivery"
	case EmailReplyStatePersist:
		return "reply_state_persist"
	default:
		return "unknown"
	}
}

// EmailError exposes only a fixed stage. Unwrap is for errors.Is/As; callers
// must not log the underlying cause, which may contain credentials or PII.
type EmailError struct {
	stage EmailStage
	cause error
}

func (e *EmailError) Error() string     { return "email operation failed: " + e.stage.String() }
func (e *EmailError) Unwrap() error     { return e.cause }
func (e *EmailError) Stage() EmailStage { return e.stage }

// EmailFailureStage is safe log metadata even when err is an unexpected error.
func EmailFailureStage(err error) string {
	var failure *EmailError
	if errors.As(err, &failure) && failure != nil {
		return failure.stage.String()
	}
	return "unknown"
}

func sendEmail(message emailMessage, build emailBuilder, send emailSender) error {
	content, err := build(message)
	if err != nil {
		return &EmailError{stage: EmailMessageBuild, cause: err}
	}

	if err := send(message, content); err != nil {
		return &EmailError{stage: EmailSMTPDelivery, cause: err}
	}

	return nil
}

func sendEmailSMTP(message emailMessage, content []byte) error {
	auth := smtp.PlainAuth("", message.Username, message.Password, message.Host)
	address := message.Host + ":" + strconv.FormatInt(message.Port, 10)
	return smtp.SendMail(address, auth, message.Username, message.To, content)
}

func buildEmailMessage(message emailMessage) ([]byte, error) {
	encodedFrom := mime.QEncoding.Encode("utf-8", "Gönderen Adı")
	from := fmt.Sprintf("%s <%s>", encodedFrom, message.From)
	return buildHTMLWithAttachments(from, message.To, message.Subject, message.PlainText, message.Body, message.Attachments)
}

func buildHTMLWithAttachments(from string, to []string, subject, plainText, html string, files []string) ([]byte, error) {
	var b bytes.Buffer

	mixedBoundary := generateEmailBoundary("MIXED")
	altBoundary := generateEmailBoundary("ALT")

	if strings.Contains(from, "<") && strings.Contains(from, ">") {
		start := strings.Index(from, "<") + 1
		end := strings.Index(from, ">")
		from = strings.TrimSpace(from[start:end])
	}

	encodedSubject := mime.QEncoding.Encode("utf-8", subject)
	toHeader := strings.Join(to, ", ")
	headers := map[string]string{
		"From":         from,
		"To":           toHeader,
		"Subject":      encodedSubject,
		"MIME-Version": "1.0",
		"Content-Type": `multipart/mixed; boundary="` + mixedBoundary + `"`,
	}

	for key, value := range headers {
		fmt.Fprintf(&b, "%s: %s\r\n", key, value)
	}
	fmt.Fprintf(&b, "\r\n")

	fmt.Fprintf(&b, "--%s\r\n", mixedBoundary)
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)

	fmt.Fprintf(&b, "--%s\r\n", altBoundary)
	fmt.Fprintf(&b, "Content-Type: text/plain; charset=\"utf-8\"\r\n")
	fmt.Fprintf(&b, "Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	plainWriter := quotedprintable.NewWriter(&b)
	if _, err := plainWriter.Write([]byte(plainText)); err != nil {
		return nil, err
	}
	if err := plainWriter.Close(); err != nil {
		return nil, err
	}
	fmt.Fprintf(&b, "\r\n")

	fmt.Fprintf(&b, "--%s\r\n", altBoundary)
	fmt.Fprintf(&b, "Content-Type: text/html; charset=\"utf-8\"\r\n")
	fmt.Fprintf(&b, "Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	htmlWriter := quotedprintable.NewWriter(&b)
	if _, err := htmlWriter.Write([]byte(html)); err != nil {
		return nil, err
	}
	if err := htmlWriter.Close(); err != nil {
		return nil, err
	}
	fmt.Fprintf(&b, "\r\n")
	fmt.Fprintf(&b, "--%s--\r\n", altBoundary)

	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		filename := filepath.Base(filePath)
		mimeType := mime.TypeByExtension(filepath.Ext(filename))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		encoded := base64.StdEncoding.EncodeToString(data)

		fmt.Fprintf(&b, "--%s\r\n", mixedBoundary)
		fmt.Fprintf(&b, "Content-Type: %s; name=\"%s\"\r\n", mimeType, filename)
		fmt.Fprintf(&b, "Content-Transfer-Encoding: base64\r\n")
		if strings.Contains(html, "cid:"+filename) {
			fmt.Fprintf(&b, "Content-Disposition: inline; filename=\"%s\"\r\n", filename)
			fmt.Fprintf(&b, "Content-ID: <%s>\r\n\r\n", filename)
		} else {
			fmt.Fprintf(&b, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", filename)
		}

		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			fmt.Fprintf(&b, "%s\r\n", encoded[i:end])
		}
		fmt.Fprintf(&b, "\r\n")
	}

	fmt.Fprintf(&b, "--%s--\r\n", mixedBoundary)
	return b.Bytes(), nil
}

func generateEmailBoundary(prefix string) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] // #nosec G404 -- MIME boundary is not security-sensitive.
	}
	return prefix + "-" + string(b)
}

// SendEmailThenMarkReplied preserves the invariant that a record is marked as
// replied only after the primary e-mail delivery succeeds.
func SendEmailThenMarkReplied(send func() error, markReplied func() error) error {
	if err := send(); err != nil {
		stage := EmailSMTPDelivery
		var failure *EmailError
		if errors.As(err, &failure) && failure != nil && failure.stage == EmailMessageBuild {
			stage = EmailMessageBuild
		}
		return &EmailError{stage: stage, cause: err}
	}
	if err := markReplied(); err != nil {
		return &EmailError{stage: EmailReplyStatePersist, cause: err}
	}
	return nil
}

// EmailReplyResponse preserves the existing JSON contract and distinguishes a
// delivered reply whose database state could not be saved. It never exposes err.
type EmailReplyResponse struct {
	Status         int    `json:"status"`
	Message        string `json:"message"`
	PartialSuccess bool   `json:"partial_success,omitempty"`
	Code           string `json:"code,omitempty"`
}

func ReplyEmailResponse(err error) EmailReplyResponse {
	if err == nil {
		return EmailReplyResponse{Status: 201, Message: "Cevap başarıyla gönderildi."}
	}
	var failure *EmailError
	if errors.As(err, &failure) && failure != nil && failure.stage == EmailReplyStatePersist {
		return EmailReplyResponse{
			Status:         500,
			Message:        "E-posta gönderildi ancak yanıt durumu kaydedilemedi. Aynı yanıtı yeniden göndermeyin.",
			PartialSuccess: true,
			Code:           "email_sent_state_not_saved",
		}
	}
	return EmailReplyResponse{Status: 500, Message: "Cevap gönderimi tamamlanamadı. Lütfen daha sonra tekrar deneyin."}
}
