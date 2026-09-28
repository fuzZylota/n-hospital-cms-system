package database

import (
	"context"
	"errors"
	"models"
	"models/data"
	"reflect"
	"strings"
	"testing"
)

type frontendSiteOptionsReader struct {
	active, testing           data.SiteOptions
	activeFound, testingFound bool
	err, testingErr           error
	calls                     []data.OptionSetSelection
}

func (r *frontendSiteOptionsReader) ReadSiteOptions(ctx context.Context, selection data.OptionSetSelection) (data.SiteOptions, bool, error) {
	if ctx == nil {
		panic("nil context")
	}
	r.calls = append(r.calls, selection)
	if r.err != nil {
		return data.SiteOptions{}, false, r.err
	}
	if selection == data.ActiveOptionSet {
		return r.active, r.activeFound, nil
	}
	if r.testingErr != nil {
		return data.SiteOptions{}, false, r.testingErr
	}
	return r.testing, r.testingFound, nil
}

func frontendOptionFixture(id string, active, testing bool) data.SiteOptions {
	return data.SiteOptions{
		Set:      data.OptionSetIdentity{ID: id, IsActive: active, IsTesting: testing},
		SiteName: id, SiteDescription: "description", MaintenanceMode: true,
		Preloader: "preloader", FacebookURL: "facebook", TwitterURL: "twitter",
		InstagramURL: "instagram", LinkedInURL: "linkedin", ContactEmail: "contact",
		ContactPhone: "phone", MainPageMetaTitle: "meta title",
		MainPageMetaDescription: "meta description", GoogleAnalytics: "analytics",
		PrimaryColor: "primary", SecondaryColor: "secondary", AccentColor: "accent",
		BackgroundColor: "background", FontColor: "font", FontFamily: "family",
		ItemsPerPage: 17, EnableTestimonials: true, MaximumSublinksOnMenuItem: 8,
		ShowDoctorSocialMedia: true, ShowDoctorAppointmentFee: true,
		ShowPartnerPictures: true, RecaptchaSiteKey: "public site key",
		SiteLogo:         data.PublicMedia{Path: id + "/logo", AltText: "logo alt", Title: "logo title"},
		SiteLightLogo:    data.PublicMedia{Path: id + "/light", AltText: "light alt", Title: "light title"},
		Favicon:          data.PublicMedia{Path: id + "/favicon"},
		DefaultPageMedia: data.PublicMedia{Path: id + "/default", AltText: "default alt", Title: "default title"},
	}
}

func frontendCacheState(reader data.SiteOptionsReader) *models.AppState {
	return &models.AppState{
		SiteOptionsReader: reader,
		HeaderButtons:     []models.HeaderButton{{}}, NewsLinks: []models.NewsLink{{}},
		SubelerLinks: []models.SubeLink{{}}, TibbiBirimlerLinks: []models.TibbiBirimLink{{}},
		TedkiklerLinks: []models.TedkikLink{{}},
	}
}

func fetchFrontendForTest(t *testing.T, state *models.AppState, authenticated bool) (Options, error) {
	t.Helper()
	input := &models.FrontendOptions{States: state}
	if authenticated {
		input.User.Uid = "synthetic-user"
	}
	return (&Options{}).FetchOptionsForFrontendWithCache(input)
}

func TestFrontendSiteOptionsSelectionCacheAndReset(t *testing.T) {
	reader := &frontendSiteOptionsReader{
		active: frontendOptionFixture("active", true, false), activeFound: true,
		testing: frontendOptionFixture("testing", false, true), testingFound: true,
	}
	state := frontendCacheState(reader)
	for _, step := range []struct {
		name          string
		authenticated bool
		wantID        string
		wantCalls     []data.OptionSetSelection
	}{
		{"anonymous active", false, "active", []data.OptionSetSelection{data.ActiveOptionSet}},
		{"anonymous cache hit", false, "active", []data.OptionSetSelection{data.ActiveOptionSet}},
		{"authenticated testing", true, "testing", []data.OptionSetSelection{data.ActiveOptionSet, data.TestingOptionSet}},
		{"authenticated cache hit", true, "testing", []data.OptionSetSelection{data.ActiveOptionSet, data.TestingOptionSet}},
		{"anonymous after testing", false, "active", []data.OptionSetSelection{data.ActiveOptionSet, data.TestingOptionSet}},
	} {
		t.Run(step.name, func(t *testing.T) {
			got, err := fetchFrontendForTest(t, state, step.authenticated)
			if err != nil {
				t.Fatal(err)
			}
			if got.Options.Oid != step.wantID || (*got.Medias)[0].FilePath != step.wantID+"/logo" {
				t.Fatalf("selected set/media = %q/%q", got.Options.Oid, (*got.Medias)[0].FilePath)
			}
			if !reflect.DeepEqual(reader.calls, step.wantCalls) {
				t.Fatalf("reader calls = %v, want %v", reader.calls, step.wantCalls)
			}
		})
	}
	reader.active = frontendOptionFixture("active-reset", true, false)
	state.ActiveOptions = models.Options{}
	got, err := fetchFrontendForTest(t, state, false)
	if err != nil || got.Options.Oid != "active-reset" || (*got.Medias)[0].FilePath != "active-reset/logo" {
		t.Fatalf("active reset result = %+v, %v", got.Options, err)
	}
	reader.testing = frontendOptionFixture("testing-reset", false, true)
	state.TestingOptions = models.Options{}
	got, err = fetchFrontendForTest(t, state, true)
	if err != nil || got.Options.Oid != "testing-reset" || (*got.Medias)[0].FilePath != "testing-reset/logo" {
		t.Fatalf("testing reset result = %+v, %v", got.Options, err)
	}
	if want := []data.OptionSetSelection{data.ActiveOptionSet, data.TestingOptionSet, data.ActiveOptionSet, data.TestingOptionSet}; !reflect.DeepEqual(reader.calls, want) {
		t.Fatalf("reset calls = %v, want %v", reader.calls, want)
	}
}

func TestFrontendSiteOptionsMissingTestingFallsBackAndRetries(t *testing.T) {
	reader := &frontendSiteOptionsReader{active: frontendOptionFixture("active", true, false), activeFound: true}
	state := frontendCacheState(reader)
	for i := 0; i < 2; i++ {
		got, err := fetchFrontendForTest(t, state, true)
		if err != nil || got.Options.Oid != "active" || (*got.Medias)[0].FilePath != "active/logo" {
			t.Fatalf("fallback = %+v, %v", got.Options, err)
		}
	}
	if want := []data.OptionSetSelection{data.ActiveOptionSet, data.TestingOptionSet, data.TestingOptionSet}; !reflect.DeepEqual(reader.calls, want) {
		t.Fatalf("missing testing calls = %v, want %v", reader.calls, want)
	}
}

func TestFrontendSiteOptionsBothFlagsUseTestingForAuthenticated(t *testing.T) {
	both := frontendOptionFixture("both", true, true)
	reader := &frontendSiteOptionsReader{active: both, activeFound: true, testing: both, testingFound: true}
	state := frontendCacheState(reader)
	for _, authenticated := range []bool{false, true} {
		got, err := fetchFrontendForTest(t, state, authenticated)
		if err != nil || got.Options.Oid != "both" || !got.Options.OptionSetIsActive || !got.Options.OptionSetIsTestingNow {
			t.Fatalf("both flags result = %+v, %v", got.Options, err)
		}
	}
	if want := []data.OptionSetSelection{data.ActiveOptionSet, data.TestingOptionSet}; !reflect.DeepEqual(reader.calls, want) {
		t.Fatalf("both flags calls = %v", reader.calls)
	}
}

func TestFrontendSiteOptionsFailuresAreSafe(t *testing.T) {
	for _, tc := range []struct {
		name         string
		reader       data.SiteOptionsReader
		want         string
		contextError error
	}{
		{"missing active", &frontendSiteOptionsReader{}, "active site options are unavailable", nil},
		{"reader unavailable", nil, "active site options reader is unavailable", nil},
		{"reader failure", &frontendSiteOptionsReader{err: errors.New("private connection detail")}, "active site options read failed", nil},
		{"canceled", &frontendSiteOptionsReader{err: context.Canceled}, "active site options read canceled", context.Canceled},
		{"deadline", &frontendSiteOptionsReader{err: context.DeadlineExceeded}, "active site options read timed out", context.DeadlineExceeded},
		{"invalid active", &frontendSiteOptionsReader{active: frontendOptionFixture("wrong", false, true), activeFound: true}, "invalid selected set", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fetchFrontendForTest(t, frontendCacheState(tc.reader), false)
			if err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "private connection detail") {
				t.Fatalf("unsafe or missing error: %v", err)
			}
			if tc.contextError != nil && !errors.Is(err, tc.contextError) {
				t.Fatalf("context error lost: %v", err)
			}
		})
	}
	reader := &frontendSiteOptionsReader{
		active: frontendOptionFixture("active", true, false), activeFound: true,
		testingErr: errors.New("private testing detail"),
	}
	state := frontendCacheState(reader)
	_, err := fetchFrontendForTest(t, state, true)
	if err == nil || err.Error() != "testing site options read failed" || state.TestingOptions.Oid != "" {
		t.Fatalf("testing read failure = %v, cached testing = %+v", err, state.TestingOptions)
	}
}

func TestFrontendSiteOptionsRenderProjectionAndMediaOrder(t *testing.T) {
	source := frontendOptionFixture("active", true, false)
	reader := &frontendSiteOptionsReader{active: source, activeFound: true}
	got, err := fetchFrontendForTest(t, frontendCacheState(reader), false)
	if err != nil {
		t.Fatal(err)
	}
	want := models.Options{
		Oid: "active", OptionSetIsActive: true, SiteName: "active", SiteDescription: "description",
		MaintenanceMode: true, Preloader: "preloader", FacebookUrl: "facebook", TwitterUrl: "twitter",
		InstagramUrl: "instagram", LinkedinUrl: "linkedin", ContactEmail: "contact", ContactPhone: "phone",
		MainPageMetaTitle: "meta title", MainPageMetaDescription: "meta description", GoogleAnalytics: "analytics",
		PrimaryColor: "primary", SecondaryColor: "secondary", AccentColor: "accent", BackgroundColor: "background",
		FontColor: "font", FontFamily: "family", ItemsPerPage: 17, EnableTestimonials: true,
		MaximumSublinksOnAMenuItem: 8, ShowDoctorSocialMedia: true, ShowDoctorAppointmentFee: true,
		ShowAnlasmaliKurumPictures: true, RecaptchaSiteKey: "public site key",
		DefaultPageMediaPath: "active/default", DefaultPageMediaAltText: "default alt", DefaultPageMediaTitle: "default title",
	}
	if !reflect.DeepEqual(*got.Options, want) {
		t.Fatalf("render projection differs: got %+v, want %+v", *got.Options, want)
	}
	if got.Options.SMTPHost != "" || got.Options.SMTPPort != 0 || got.Options.SMTPUsername != "" || got.Options.SMTPPassword != "" || got.Options.SMTPEncryption != "" || got.Options.RecaptchaSecretKey != "" {
		t.Fatal("server-only values reached frontend render model")
	}
	wantMedia := []models.Medias{
		{FilePath: "active/logo", AltText: "logo alt", Title: "logo title"},
		{FilePath: "active/light", AltText: "light alt", Title: "light title"},
		{FilePath: "active/favicon"},
		{FilePath: "active/default", AltText: "default alt", Title: "default title"},
	}
	if !reflect.DeepEqual(*got.Medias, wantMedia) {
		t.Fatalf("render media order differs: got %+v, want %+v", *got.Medias, wantMedia)
	}
}
