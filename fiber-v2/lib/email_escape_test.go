package lib

import (
	"strings"
	"testing"
)

func TestEscapeEmailText(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"script", `<script>alert(1)</script>`, `&lt;script&gt;alert(1)&lt;/script&gt;`},
		{"anchor", `<a href="https://evil.example">x</a>`, `&lt;a href=&#34;https://evil.example&#34;&gt;x&lt;/a&gt;`},
		{"quotes", `a"b'c`, `a&#34;b&#39;c`},
		{"ampersand", `a&b&amp;c`, `a&amp;b&amp;amp;c`},
		{"plain", `Ayşe Yılmaz`, `Ayşe Yılmaz`},
		{"empty", ``, ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EscapeEmailText(tc.in); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"a@b.c\r\nBcc: x@y.z", "a@b.cBcc: x@y.z"},
		{"line1\nline2", "line1line2"},
		{"cr\rcr", "crcr"},
		{"nul\x00x", "nulx"},
		{"clean subject", "clean subject"},
	} {
		if got := SanitizeHeaderValue(tc.in); got != tc.want {
			t.Fatalf("got %q want %q", got, tc.want)
		}
	}
}

func TestBuildEmailHeadersRejectInjection(t *testing.T) {
	raw, err := buildHTMLWithAttachments("Site <from@x.y>", []string{"victim@x.y\r\nBcc: evil@x.y"}, "Konu\r\nBcc: evil2@x.y", "t", "<p>h</p>", nil)
	if err != nil {
		t.Fatal(err)
	}
	headerBlock := strings.SplitN(string(raw), "\r\n\r\n", 2)[0]
	for _, line := range strings.Split(headerBlock, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Fatal("header injection produced an extra header line")
		}
	}
}
