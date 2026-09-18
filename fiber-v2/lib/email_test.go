package lib

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSendEmailReturnsMIMEBuildErrorWithoutStoppingProcess(t *testing.T) {
	buildErr := errors.New("mime build failed")
	senderCalled := false

	err := sendEmail(emailMessage{}, func(emailMessage) ([]byte, error) {
		return nil, buildErr
	}, func(emailMessage, []byte) error {
		senderCalled = true
		return nil
	})

	if !errors.Is(err, buildErr) {
		t.Fatalf("expected wrapped MIME error, got %v", err)
	}
	if senderCalled {
		t.Fatal("SMTP sender must not run after MIME creation fails")
	}

	if err := sendEmail(emailMessage{}, func(emailMessage) ([]byte, error) {
		return []byte("next message"), nil
	}, func(emailMessage, []byte) error {
		return nil
	}); err != nil {
		t.Fatalf("subsequent delivery proves process survival; got %v", err)
	}
}

func TestSendEmailReturnsSMTPErrorWithoutStoppingProcess(t *testing.T) {
	smtpErr := errors.New("smtp delivery failed")

	err := sendEmail(emailMessage{}, func(emailMessage) ([]byte, error) {
		return []byte("message"), nil
	}, func(emailMessage, []byte) error {
		return smtpErr
	})

	if !errors.Is(err, smtpErr) {
		t.Fatalf("expected wrapped SMTP error, got %v", err)
	}
}

func TestSendEmailReturnsNilOnSuccess(t *testing.T) {
	message := emailMessage{To: []string{"recipient@example.test"}, Body: "body"}
	var calls []string
	err := sendEmail(message, func(got emailMessage) ([]byte, error) {
		calls = append(calls, "build")
		if !reflect.DeepEqual(got, message) {
			t.Fatal("builder did not receive the original message")
		}
		return []byte("built message"), nil
	}, func(got emailMessage, content []byte) error {
		calls = append(calls, "send")
		if !reflect.DeepEqual(got, message) || string(content) != "built message" {
			t.Fatal("sender did not receive the original envelope and built content")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected successful delivery, got %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"build", "send"}) {
		t.Fatalf("wrong call count or order: %v", calls)
	}
}

func TestSendEmailRedactsCredentialsAndMessageData(t *testing.T) {
	message := emailMessage{
		From:        "from@example.test",
		To:          []string{"to@example.test"},
		Username:    "smtp-user",
		Password:    "smtp-password",
		Host:        "smtp.example.test",
		Subject:     "private subject",
		PlainText:   "private text",
		Body:        "private body",
		Attachments: []string{"private/path/logo.png"},
	}
	underlying := errors.New(strings.Join([]string{
		message.From,
		message.To[0],
		message.Username,
		message.Password,
		message.Host,
		message.Subject,
		message.PlainText,
		message.Body,
		message.Attachments[0],
	}, " "))

	err := sendEmail(message, func(emailMessage) ([]byte, error) {
		return nil, underlying
	}, func(emailMessage, []byte) error {
		return nil
	})

	if !errors.Is(err, underlying) {
		t.Fatal("expected the sanitized wrapper to preserve the underlying error")
	}
	for _, sensitive := range []string{
		message.From,
		message.To[0],
		message.Username,
		message.Password,
		message.Host,
		message.Subject,
		message.PlainText,
		message.Body,
		message.Attachments[0],
	} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("error leaked sensitive value %q: %v", sensitive, err)
		}
	}
}

func TestBuildEmailMessageReturnsMissingAttachmentError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-logo.png")
	err := sendEmail(emailMessage{Attachments: []string{missing}}, buildEmailMessage, func(emailMessage, []byte) error {
		t.Fatal("SMTP sender must not run when an attachment cannot be read")
		return nil
	})

	if err == nil {
		t.Fatal("expected missing attachment to return an error")
	}
	if strings.Contains(err.Error(), missing) {
		t.Fatalf("error leaked attachment path: %v", err)
	}
}

func TestSendEmailThenMarkRepliedDoesNotMarkAfterDeliveryFailure(t *testing.T) {
	deliveryErr := errors.New("delivery failed")
	marked := false

	err := SendEmailThenMarkReplied(func() error {
		return deliveryErr
	}, func() error {
		marked = true
		return nil
	})

	if !errors.Is(err, deliveryErr) {
		t.Fatalf("expected wrapped delivery error, got %v", err)
	}
	if marked {
		t.Fatal("record must not be marked replied after delivery failure")
	}
}

func TestSendEmailThenMarkRepliedMarksAfterSuccessfulDelivery(t *testing.T) {
	marked := false
	sent := false
	var calls []string

	err := SendEmailThenMarkReplied(func() error {
		calls = append(calls, "send")
		sent = true
		return nil
	}, func() error {
		if !sent {
			t.Fatal("status write preceded delivery")
		}
		calls = append(calls, "mark")
		marked = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected successful reply workflow, got %v", err)
	}
	if !marked {
		t.Fatal("record was not marked replied after successful delivery")
	}
	if !reflect.DeepEqual(calls, []string{"send", "mark"}) {
		t.Fatalf("wrong call count or order: %v", calls)
	}
}

func TestEmailErrorsExposeOnlyFixedStages(t *testing.T) {
	// Deliberately unrelated to emailMessage, overlapping, encoded and partial:
	// safety must not depend on finding known message fields in an error string.
	private := "credential=fixture-token user@EXAMPLE.test private subject body-fragment /private/cv.docx C:\\private\\cv.docx Zml4dHVyZQ=="
	cause := &textproto.Error{Code: 550, Msg: private}
	cases := []struct {
		name  string
		stage EmailStage
		want  string
		run   func() error
	}{
		{"build", EmailMessageBuild, "message_build", func() error {
			return sendEmail(emailMessage{}, func(emailMessage) ([]byte, error) { return nil, cause }, func(emailMessage, []byte) error {
				t.Fatal("sender called after build failure")
				return nil
			})
		}},
		{"smtp", EmailSMTPDelivery, "smtp_delivery", func() error {
			return sendEmail(emailMessage{}, func(emailMessage) ([]byte, error) { return nil, nil }, func(emailMessage, []byte) error { return cause })
		}},
		{"reply_state", EmailReplyStatePersist, "reply_state_persist", func() error {
			return SendEmailThenMarkReplied(func() error { return nil }, func() error { return cause })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			var failure *EmailError
			if !errors.As(err, &failure) || failure.Stage() != tc.stage {
				t.Fatal("wrong typed error stage")
			}
			var original *textproto.Error
			if !errors.Is(err, cause) || !errors.As(err, &original) || original != cause {
				t.Fatal("programmatic cause was not preserved")
			}
			want := "email operation failed: " + tc.want
			for _, rendered := range []string{err.Error(), fmt.Sprintf("%v", err), fmt.Sprintf("%s", err), fmt.Sprintf("%+v", err)} {
				if rendered != want {
					t.Fatal("error exposed something other than the fixed stage")
				}
			}
			if EmailFailureStage(err) != tc.want {
				t.Fatal("unsafe or incorrect log metadata")
			}
		})
	}
	if EmailFailureStage(cause) != "unknown" || EmailFailureStage(nil) != "unknown" {
		t.Fatal("unexpected errors must not become log metadata")
	}
	if (&EmailError{stage: 255, cause: cause}).Error() != "email operation failed: unknown" {
		t.Fatal("unknown stage was not constrained")
	}
}

func TestReplyEmailResponseOutcomes(t *testing.T) {
	cause := errors.New("private underlying failure")
	cases := []struct {
		name                              string
		buildErr, sendErr, stateErr       error
		wantStage                         string
		wantStatus, wantSends, wantWrites int
		partial                           bool
	}{
		{"build_failure", cause, nil, nil, "message_build", 500, 0, 0, false},
		{"send_failure", nil, cause, nil, "smtp_delivery", 500, 1, 0, false},
		{"success", nil, nil, nil, "unknown", 201, 1, 1, false},
		{"sent_state_failed", nil, nil, cause, "reply_state_persist", 500, 1, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sends, writes := 0, 0
			sent := false
			err := SendEmailThenMarkReplied(func() error {
				return sendEmail(emailMessage{}, func(emailMessage) ([]byte, error) { return []byte("content"), tc.buildErr }, func(emailMessage, []byte) error {
					sends++
					sent = tc.sendErr == nil
					return tc.sendErr
				})
			}, func() error {
				if !sent {
					t.Fatal("status callback ran before successful send")
				}
				writes++
				return tc.stateErr
			})
			if sends != tc.wantSends || writes != tc.wantWrites || EmailFailureStage(err) != tc.wantStage {
				t.Fatal("wrong workflow order, count or classification")
			}
			response := ReplyEmailResponse(err)
			if response.Status != tc.wantStatus || response.PartialSuccess != tc.partial {
				t.Fatal("wrong client outcome")
			}
			encoded, marshalErr := json.Marshal(response)
			if marshalErr != nil || bytes.Contains(encoded, []byte(cause.Error())) {
				t.Fatal("unsafe JSON response")
			}
			if tc.partial {
				if response.Code != "email_sent_state_not_saved" || !strings.Contains(response.Message, "E-posta gönderildi") || !strings.Contains(response.Message, "kaydedilemedi") || !strings.Contains(response.Message, "yeniden göndermeyin") || strings.Contains(response.Message, "tekrar deneyin") {
					t.Fatal("partial success must describe delivery and prohibit resending")
				}
			} else {
				if bytes.Contains(encoded, []byte("partial_success")) || bytes.Contains(encoded, []byte("code")) {
					t.Fatal("ordinary response contract changed")
				}
				if tc.wantStatus == 201 && response.Message != "Cevap başarıyla gönderildi." {
					t.Fatal("success message changed")
				}
			}
		})
	}
}

func TestBuildEmailMessagePreservesMIMEAndAttachments(t *testing.T) {
	for _, withAttachments := range []bool{false, true} {
		t.Run(fmt.Sprintf("attachments_%t", withAttachments), func(t *testing.T) {
			message := emailMessage{From: "from@example.test", To: []string{"one@example.test", "two@example.test"}, Subject: "Türkçe konu", PlainText: "Düz metin", Body: "<p>Gövde</p><img src=\"cid:logo.png\">"}
			attachmentData := bytes.Repeat([]byte("fixture"), 30)
			if withAttachments {
				for _, filename := range []string{"logo.png", "document.bin"} {
					path := filepath.Join(t.TempDir(), filename)
					if err := os.WriteFile(path, attachmentData, 0600); err != nil {
						t.Fatal(err)
					}
					message.Attachments = append(message.Attachments, path)
				}
			}
			calls := 0
			err := sendEmail(message, buildEmailMessage, func(got emailMessage, content []byte) error {
				calls++
				if !reflect.DeepEqual(got, message) {
					t.Fatal("envelope changed")
				}
				parsed, err := mail.ReadMessage(bytes.NewReader(content))
				if err != nil {
					t.Fatal(err)
				}
				if parsed.Header.Get("From") != message.From || parsed.Header.Get("To") != strings.Join(message.To, ", ") {
					t.Fatal("address headers changed")
				}
				subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
				if err != nil || subject != message.Subject {
					t.Fatal("subject encoding changed")
				}
				kind, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
				if err != nil || kind != "multipart/mixed" {
					t.Fatal("invalid mixed MIME")
				}
				mixed := multipart.NewReader(parsed.Body, params["boundary"])
				alternative, err := mixed.NextPart()
				if err != nil {
					t.Fatal(err)
				}
				kind, params, err = mime.ParseMediaType(alternative.Header.Get("Content-Type"))
				if err != nil || kind != "multipart/alternative" {
					t.Fatal("invalid alternative MIME")
				}
				alternatives := multipart.NewReader(alternative, params["boundary"])
				for i, want := range []string{message.PlainText, message.Body} {
					part, err := alternatives.NextPart()
					if err != nil {
						t.Fatal(err)
					}
					body, err := io.ReadAll(part)
					if err != nil || string(body) != want {
						t.Fatal("body encoding changed")
					}
					wantType := []string{"text/plain", "text/html"}[i]
					if !strings.HasPrefix(part.Header.Get("Content-Type"), wantType) {
						t.Fatal("body type changed")
					}
				}
				if _, err := alternatives.NextPart(); err != io.EOF {
					t.Fatal("unexpected alternative part")
				}
				for i, path := range message.Attachments {
					part, err := mixed.NextPart()
					if err != nil {
						t.Fatal(err)
					}
					disposition, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
					if err != nil || params["filename"] != filepath.Base(path) {
						t.Fatal("attachment name changed")
					}
					if i == 0 && (disposition != "inline" || part.Header.Get("Content-ID") != "<logo.png>") {
						t.Fatal("inline attachment changed")
					}
					if i == 1 && disposition != "attachment" {
						t.Fatal("normal attachment changed")
					}
					encoded, err := io.ReadAll(part)
					if err != nil {
						t.Fatal(err)
					}
					for _, line := range bytes.Split(encoded, []byte("\r\n")) {
						if len(line) > 76 {
							t.Fatal("base64 line too long")
						}
					}
					decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, bytes.NewReader(encoded)))
					if err != nil || !bytes.Equal(decoded, attachmentData) {
						t.Fatal("attachment data changed")
					}
				}
				if _, err := mixed.NextPart(); err != io.EOF {
					t.Fatal("unexpected mixed part")
				}
				return nil
			})
			if err != nil || calls != 1 {
				t.Fatal("MIME success did not reach sender once")
			}
		})
	}
}
