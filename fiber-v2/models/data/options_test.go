package data_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"models/data"
)

type siteOptionsReaderFunc func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error)

func (f siteOptionsReaderFunc) ReadSiteOptions(ctx context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
	return f(ctx, selection)
}

type uploadPolicyReaderFunc func(context.Context) (data.UploadPolicy, bool, error)

func (f uploadPolicyReaderFunc) ReadUploadPolicy(ctx context.Context) (data.UploadPolicy, bool, error) {
	return f(ctx)
}

type passwordPolicyReaderFunc func(context.Context) (data.PasswordPolicy, bool, error)

func (f passwordPolicyReaderFunc) ReadPasswordPolicy(ctx context.Context) (data.PasswordPolicy, bool, error) {
	return f(ctx)
}

type mailDeliveryOptionsReaderFunc func(context.Context) (data.MailDeliveryOptions, bool, error)

func (f mailDeliveryOptionsReaderFunc) ReadMailDeliveryOptions(ctx context.Context) (data.MailDeliveryOptions, bool, error) {
	return f(ctx)
}

type captchaVerificationOptionsReaderFunc func(context.Context) (data.CaptchaVerificationOptions, bool, error)

func (f captchaVerificationOptionsReaderFunc) ReadCaptchaVerificationOptions(ctx context.Context) (data.CaptchaVerificationOptions, bool, error) {
	return f(ctx)
}

type optionMediaReferencesReaderFunc func(context.Context) (data.OptionMediaReferences, bool, error)

func (f optionMediaReferencesReaderFunc) ReadOptionMediaReferences(ctx context.Context) (data.OptionMediaReferences, bool, error) {
	return f(ctx)
}

var (
	_ data.SiteOptionsReader                = siteOptionsReaderFunc(nil)
	_ data.UploadPolicyReader               = uploadPolicyReaderFunc(nil)
	_ data.PasswordPolicyReader             = passwordPolicyReaderFunc(nil)
	_ data.MailDeliveryOptionsReader        = mailDeliveryOptionsReaderFunc(nil)
	_ data.CaptchaVerificationOptionsReader = captchaVerificationOptionsReaderFunc(nil)
	_ data.OptionMediaReferencesReader      = optionMediaReferencesReaderFunc(nil)
)

func TestOptionSetSelectionsAreExplicit(t *testing.T) {
	if data.ActiveOptionSet == 0 || data.TestingOptionSet == 0 {
		t.Fatal("active and testing selections must not use the invalid zero value")
	}
	if data.ActiveOptionSet == data.TestingOptionSet {
		t.Fatal("active and testing selections must be distinct")
	}
}

func TestSiteOptionsIsPublicAndPanelSafe(t *testing.T) {
	setType := reflect.TypeOf(data.OptionSetIdentity{})
	mediaType := reflect.TypeOf(data.PublicMedia{})
	stringType := reflect.TypeOf("")
	int64Type := reflect.TypeOf(int64(0))
	boolType := reflect.TypeOf(false)
	assertStructFields(t, data.SiteOptions{}, []fieldContract{
		{name: "Set", typeOf: setType},
		{name: "SiteName", typeOf: stringType},
		{name: "SiteDescription", typeOf: stringType},
		{name: "MaintenanceMode", typeOf: boolType},
		{name: "Preloader", typeOf: stringType},
		{name: "FacebookURL", typeOf: stringType},
		{name: "TwitterURL", typeOf: stringType},
		{name: "InstagramURL", typeOf: stringType},
		{name: "LinkedInURL", typeOf: stringType},
		{name: "ContactEmail", typeOf: stringType},
		{name: "ContactPhone", typeOf: stringType},
		{name: "MainPageMetaTitle", typeOf: stringType},
		{name: "MainPageMetaDescription", typeOf: stringType},
		{name: "GoogleAnalytics", typeOf: stringType},
		{name: "PrimaryColor", typeOf: stringType},
		{name: "SecondaryColor", typeOf: stringType},
		{name: "AccentColor", typeOf: stringType},
		{name: "BackgroundColor", typeOf: stringType},
		{name: "FontColor", typeOf: stringType},
		{name: "FontFamily", typeOf: stringType},
		{name: "ItemsPerPage", typeOf: int64Type},
		{name: "EnableTestimonials", typeOf: boolType},
		{name: "MaximumSublinksOnMenuItem", typeOf: int64Type},
		{name: "ShowDoctorSocialMedia", typeOf: boolType},
		{name: "ShowDoctorAppointmentFee", typeOf: boolType},
		{name: "ShowPartnerPictures", typeOf: boolType},
		{name: "RecaptchaSiteKey", typeOf: stringType},
		{name: "SiteLogo", typeOf: mediaType},
		{name: "SiteLightLogo", typeOf: mediaType},
		{name: "Favicon", typeOf: mediaType},
		{name: "DefaultPageMedia", typeOf: mediaType},
	})

	typeOfOptions := reflect.TypeOf(data.SiteOptions{})
	for index := 0; index < typeOfOptions.NumField(); index++ {
		field := typeOfOptions.Field(index)
		name := strings.ToLower(field.Name)
		for _, forbidden := range []string{"password", "secret", "smtp", "username"} {
			if strings.Contains(name, forbidden) {
				t.Errorf("SiteOptions unexpectedly exposes %s", field.Name)
			}
		}
		if field.Tag != "" {
			t.Errorf("SiteOptions.%s has serialization/infrastructure tag %q", field.Name, field.Tag)
		}
	}

	assertStructFields(t, data.OptionSetIdentity{}, []fieldContract{
		{name: "ID", typeOf: reflect.TypeOf("")},
		{name: "IsActive", typeOf: reflect.TypeOf(false)},
		{name: "IsTesting", typeOf: reflect.TypeOf(false)},
	})
	assertStructFields(t, data.PublicMedia{}, []fieldContract{
		{name: "Path", typeOf: reflect.TypeOf("")},
		{name: "AltText", typeOf: reflect.TypeOf("")},
		{name: "Title", typeOf: reflect.TypeOf("")},
	})
}

func TestInternalOptionProjectionsHaveExactFieldsAndNoTags(t *testing.T) {
	setType := reflect.TypeOf(data.OptionSetIdentity{})
	stringType := reflect.TypeOf("")
	int64Type := reflect.TypeOf(int64(0))

	assertStructFields(t, data.UploadPolicy{}, []fieldContract{
		{name: "Set", typeOf: setType},
		{name: "MaxBytes", typeOf: int64Type},
	})
	assertStructFields(t, data.PasswordPolicy{}, []fieldContract{
		{name: "Set", typeOf: setType},
		{name: "RequireStrong", typeOf: reflect.TypeOf(false)},
	})
	assertStructFields(t, data.MailDeliveryOptions{}, []fieldContract{
		{name: "Set", typeOf: setType},
		{name: "Host", typeOf: stringType},
		{name: "Port", typeOf: int64Type},
		{name: "Username", typeOf: stringType},
		{name: "Password", typeOf: stringType},
	})
	if _, exists := reflect.TypeOf(data.MailDeliveryOptions{}).FieldByName("Encryption"); exists {
		t.Fatal("MailDeliveryOptions must not claim an unproven encryption setting")
	}
	assertStructFields(t, data.CaptchaVerificationOptions{}, []fieldContract{
		{name: "Set", typeOf: setType},
		{name: "SiteKey", typeOf: stringType},
		{name: "SecretKey", typeOf: stringType},
	})
	assertStructFields(t, data.OptionMediaReferences{}, []fieldContract{
		{name: "Set", typeOf: setType},
		{name: "SiteLogoID", typeOf: reflect.TypeOf((*string)(nil))},
		{name: "SiteLightLogoID", typeOf: reflect.TypeOf((*string)(nil))},
		{name: "FaviconID", typeOf: reflect.TypeOf((*string)(nil))},
		{name: "DefaultPageMediaID", typeOf: reflect.TypeOf((*string)(nil))},
	})
}

func TestSiteOptionsReaderOutcomes(t *testing.T) {
	repositoryFailure := errors.New("safe option lookup failure")
	invalidSelectionFailure := errors.New("invalid option set selection")
	active := data.SiteOptions{
		Set:      data.OptionSetIdentity{ID: "12", IsActive: true},
		SiteName: "Nivgoz",
	}
	testingOptions := data.SiteOptions{
		Set:      data.OptionSetIdentity{ID: "13", IsTesting: true},
		SiteName: "Nivgoz preview",
	}

	tests := []struct {
		name      string
		context   func() context.Context
		reader    data.SiteOptionsReader
		selection data.OptionSetSelection
		want      data.SiteOptions
		wantFound bool
		wantError error
	}{
		{
			name:      "active success",
			context:   context.Background,
			selection: data.ActiveOptionSet,
			reader: siteOptionsReaderFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
				if selection != data.ActiveOptionSet {
					t.Fatalf("selection = %v, want active", selection)
				}
				return active, true, nil
			}),
			want:      active,
			wantFound: true,
		},
		{
			name:      "testing success",
			context:   context.Background,
			selection: data.TestingOptionSet,
			reader: siteOptionsReaderFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
				if selection != data.TestingOptionSet {
					t.Fatalf("selection = %v, want testing", selection)
				}
				return testingOptions, true, nil
			}),
			want:      testingOptions,
			wantFound: true,
		},
		{
			name:      "empty selected result",
			context:   context.Background,
			selection: data.ActiveOptionSet,
			reader: siteOptionsReaderFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
				return data.SiteOptions{}, false, nil
			}),
		},
		{
			name:      "lookup error",
			context:   context.Background,
			selection: data.ActiveOptionSet,
			reader: siteOptionsReaderFunc(func(context.Context, data.OptionSetSelection) (data.SiteOptions, bool, error) {
				return data.SiteOptions{}, false, repositoryFailure
			}),
			wantError: repositoryFailure,
		},
		{
			name: "context canceled",
			context: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			selection: data.ActiveOptionSet,
			reader:    siteOptionsReaderFunc(contextAwareSiteOptions),
			wantError: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			context: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				t.Cleanup(cancel)
				return ctx
			},
			selection: data.TestingOptionSet,
			reader:    siteOptionsReaderFunc(contextAwareSiteOptions),
			wantError: context.DeadlineExceeded,
		},
		{
			name:      "zero selection rejected",
			context:   context.Background,
			selection: 0,
			reader: siteOptionsReaderFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
				if selection == data.ActiveOptionSet || selection == data.TestingOptionSet {
					t.Fatalf("selection = %v, want invalid", selection)
				}
				return data.SiteOptions{}, false, invalidSelectionFailure
			}),
			wantError: invalidSelectionFailure,
		},
		{
			name:      "unknown selection rejected",
			context:   context.Background,
			selection: data.OptionSetSelection(255),
			reader: siteOptionsReaderFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
				if selection == data.ActiveOptionSet || selection == data.TestingOptionSet {
					t.Fatalf("selection = %v, want invalid", selection)
				}
				return data.SiteOptions{}, false, invalidSelectionFailure
			}),
			wantError: invalidSelectionFailure,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, found, err := test.reader.ReadSiteOptions(test.context(), test.selection)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if found != test.wantFound {
				t.Fatalf("found = %v, want %v", found, test.wantFound)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("result = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestSiteOptionsReaderMissingTestingSetDoesNotFallbackToActive(t *testing.T) {
	activeRead := false

	// This fake demonstrates the N05A contract only. N05B must prove that the
	// production repository follows the same no-fallback behavior.
	reader := siteOptionsReaderFunc(func(_ context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
		switch selection {
		case data.ActiveOptionSet:
			activeRead = true
			return data.SiteOptions{Set: data.OptionSetIdentity{ID: "12", IsActive: true}}, true, nil
		case data.TestingOptionSet:
			return data.SiteOptions{}, false, nil
		default:
			return data.SiteOptions{}, false, errors.New("invalid option set selection")
		}
	})

	got, found, err := reader.ReadSiteOptions(context.Background(), data.TestingOptionSet)
	if err != nil || found || !reflect.DeepEqual(got, data.SiteOptions{}) {
		t.Fatalf("testing lookup = (%#v, %v, %v), want zero, false, nil", got, found, err)
	}
	if activeRead {
		t.Fatal("missing testing set unexpectedly read the active set")
	}
}

func TestInternalOptionReadersHaveContextOnlySignatures(t *testing.T) {
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	readers := []struct {
		name       string
		readerType reflect.Type
		methodName string
	}{
		{name: "upload policy", readerType: reflect.TypeOf((*data.UploadPolicyReader)(nil)).Elem(), methodName: "ReadUploadPolicy"},
		{name: "password policy", readerType: reflect.TypeOf((*data.PasswordPolicyReader)(nil)).Elem(), methodName: "ReadPasswordPolicy"},
		{name: "mail delivery options", readerType: reflect.TypeOf((*data.MailDeliveryOptionsReader)(nil)).Elem(), methodName: "ReadMailDeliveryOptions"},
		{name: "captcha verification options", readerType: reflect.TypeOf((*data.CaptchaVerificationOptionsReader)(nil)).Elem(), methodName: "ReadCaptchaVerificationOptions"},
		{name: "option media references", readerType: reflect.TypeOf((*data.OptionMediaReferencesReader)(nil)).Elem(), methodName: "ReadOptionMediaReferences"},
	}

	for _, reader := range readers {
		t.Run(reader.name, func(t *testing.T) {
			method, exists := reader.readerType.MethodByName(reader.methodName)
			if !exists {
				t.Fatalf("%s method is missing", reader.methodName)
			}
			if method.Type.NumIn() != 1 || method.Type.In(0) != contextType {
				t.Fatalf("%s inputs = %v, want only context.Context", reader.methodName, method.Type)
			}
		})
	}
}

func TestInternalOptionReaderFakeOutcomes(t *testing.T) {
	repositoryFailure := errors.New("safe active option lookup failure")
	upload := data.UploadPolicy{Set: data.OptionSetIdentity{ID: "21", IsActive: true}, MaxBytes: 1048576}
	password := data.PasswordPolicy{Set: data.OptionSetIdentity{ID: "21", IsActive: true}, RequireStrong: true}
	mail := data.MailDeliveryOptions{Set: data.OptionSetIdentity{ID: "21", IsActive: true}, Host: "smtp.example.test", Port: 587, Username: "mailer", Password: "internal-only"}
	captcha := data.CaptchaVerificationOptions{Set: data.OptionSetIdentity{ID: "21", IsActive: true}, SiteKey: "public-key", SecretKey: "internal-only"}
	logoID := "31"
	media := data.OptionMediaReferences{Set: data.OptionSetIdentity{ID: "21", IsActive: true}, SiteLogoID: &logoID}

	type readerContract struct {
		name   string
		normal interface{}
		zero   interface{}
		read   func(outcome string) (interface{}, bool, error)
	}

	readers := []readerContract{
		{
			name: "upload policy", normal: upload, zero: data.UploadPolicy{},
			read: func(outcome string) (interface{}, bool, error) {
				return uploadPolicyReaderFunc(func(context.Context) (data.UploadPolicy, bool, error) {
					switch outcome {
					case "normal":
						return upload, true, nil
					case "empty":
						return data.UploadPolicy{}, false, nil
					default:
						return data.UploadPolicy{}, false, repositoryFailure
					}
				}).ReadUploadPolicy(context.Background())
			},
		},
		{
			name: "password policy", normal: password, zero: data.PasswordPolicy{},
			read: func(outcome string) (interface{}, bool, error) {
				return passwordPolicyReaderFunc(func(context.Context) (data.PasswordPolicy, bool, error) {
					switch outcome {
					case "normal":
						return password, true, nil
					case "empty":
						return data.PasswordPolicy{}, false, nil
					default:
						return data.PasswordPolicy{}, false, repositoryFailure
					}
				}).ReadPasswordPolicy(context.Background())
			},
		},
		{
			name: "mail delivery options", normal: mail, zero: data.MailDeliveryOptions{},
			read: func(outcome string) (interface{}, bool, error) {
				return mailDeliveryOptionsReaderFunc(func(context.Context) (data.MailDeliveryOptions, bool, error) {
					switch outcome {
					case "normal":
						return mail, true, nil
					case "empty":
						return data.MailDeliveryOptions{}, false, nil
					default:
						return data.MailDeliveryOptions{}, false, repositoryFailure
					}
				}).ReadMailDeliveryOptions(context.Background())
			},
		},
		{
			name: "captcha verification options", normal: captcha, zero: data.CaptchaVerificationOptions{},
			read: func(outcome string) (interface{}, bool, error) {
				return captchaVerificationOptionsReaderFunc(func(context.Context) (data.CaptchaVerificationOptions, bool, error) {
					switch outcome {
					case "normal":
						return captcha, true, nil
					case "empty":
						return data.CaptchaVerificationOptions{}, false, nil
					default:
						return data.CaptchaVerificationOptions{}, false, repositoryFailure
					}
				}).ReadCaptchaVerificationOptions(context.Background())
			},
		},
		{
			name: "option media references", normal: media, zero: data.OptionMediaReferences{},
			read: func(outcome string) (interface{}, bool, error) {
				return optionMediaReferencesReaderFunc(func(context.Context) (data.OptionMediaReferences, bool, error) {
					switch outcome {
					case "normal":
						return media, true, nil
					case "empty":
						return data.OptionMediaReferences{}, false, nil
					default:
						return data.OptionMediaReferences{}, false, repositoryFailure
					}
				}).ReadOptionMediaReferences(context.Background())
			},
		},
	}

	// These are fake contract checks, not evidence that the N05B production
	// repository already implements active-only lookup behavior.
	for _, reader := range readers {
		for _, outcome := range []string{"normal", "empty", "error"} {
			t.Run(reader.name+"/"+outcome, func(t *testing.T) {
				got, found, err := reader.read(outcome)
				want := reader.zero
				wantFound := false
				var wantError error
				if outcome == "normal" {
					want = reader.normal
					wantFound = true
				} else if outcome == "error" {
					wantError = repositoryFailure
				}

				if !errors.Is(err, wantError) {
					t.Fatal("unexpected reader error")
				}
				if found != wantFound {
					t.Fatal("unexpected reader found state")
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatal("unexpected reader result")
				}
			})
		}
	}
}

func contextAwareSiteOptions(ctx context.Context, _ data.OptionSetSelection) (data.SiteOptions, bool, error) {
	if err := ctx.Err(); err != nil {
		return data.SiteOptions{}, false, err
	}
	return data.SiteOptions{}, false, nil
}

type fieldContract struct {
	name   string
	typeOf reflect.Type
}

func assertStructFields(t *testing.T, value interface{}, want []fieldContract) {
	t.Helper()
	typeOfValue := reflect.TypeOf(value)
	if typeOfValue.NumField() != len(want) {
		t.Fatalf("%s has %d fields, want %d", typeOfValue.Name(), typeOfValue.NumField(), len(want))
	}
	for index, expected := range want {
		field := typeOfValue.Field(index)
		if field.Name != expected.name || field.Type != expected.typeOf {
			t.Errorf("%s field %d = %s %s, want %s %s", typeOfValue.Name(), index, field.Name, field.Type, expected.name, expected.typeOf)
		}
		if field.Tag != "" {
			t.Errorf("%s.%s has serialization/infrastructure tag %q", typeOfValue.Name(), field.Name, field.Tag)
		}
	}
}
