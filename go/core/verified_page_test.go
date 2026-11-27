package core

import (
	"errors"
	"strings"
	"testing"

	"cgl/obj"
)

// codeOf is the code an error answers with, or "" for none.
func codeOf(err error) string {
	var coded interface{ Code() string }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return ""
}

// A page's addresses are https or plain http, on any host and in either
// mode; anything else is refused with the field's own code. A test instance
// runs in development, so production is pinned down here.
func TestPageURLBySchemeHostAndMode(t *testing.T) {
	cases := []struct {
		name string
		dev  bool
		raw  string
		want string // the code; "" accepts
	}{
		{"https in production", false, "https://example.org/play", ""},
		{"https in development", true, "https://example.org/play", ""},
		{"http in production", false, "http://example.org/play", ""},
		{"http on loopback in production", false, "http://localhost:8080/play", ""},
		{"http in development, a public host", true, "http://example.org/play", ""},
		{"http in development, a LAN address", true, "http://192.168.1.20:8080/play", ""},
		{"http in development, a local name", true, "http://my-machine.local/play", ""},
		{"no scheme", true, "example.org/play", obj.ErrCodePageURLInvalid},
		{"another scheme", true, "ftp://example.org/play", obj.ErrCodePageURLInvalid},
		{"no host", true, "https:///play", obj.ErrCodePageURLInvalid},
		{"user info", true, "https://me@example.org/play", obj.ErrCodePageURLInvalid},
		{"a fragment", true, "https://example.org/play#start", obj.ErrCodePageURLInvalid},
		{"empty", true, "", obj.ErrCodePageURLInvalid},
		{"too long", true, "https://example.org/" + strings.Repeat("a", maxPageURL), obj.ErrCodePageURLInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.dev {
				t.Setenv("SCHEMAF_DEV", "1")
			} else {
				t.Setenv("SCHEMAF_DEV", "")
			}
			_, err := pageURL(c.raw, pageField)
			if got := codeOf(err); got != c.want {
				t.Errorf("pageURL(%q) refused with %q, want %q (err: %v)", c.raw, got, c.want, err)
			}
		})
	}
}

// Each refusal of a page's addresses names its field by its code, and the
// callback is on the page's site.
func TestCheckPageURLsCodes(t *testing.T) {
	t.Setenv("SCHEMAF_DEV", "")
	cases := []struct {
		name           string
		page, callback string
		want           string
	}{
		{"both https on one site", "https://example.org/play", "https://example.org:8443/cb", ""},
		{"callback on http, page on https: another site", "https://example.org/play", "http://example.org/cb", obj.ErrCodeCallbackOffSite},
		{"page invalid", "example.org/play", "https://example.org/cb", obj.ErrCodePageURLInvalid},
		{"callback invalid", "https://example.org/play", "not an address", obj.ErrCodeCallbackURLInvalid},
		{"callback on another host", "https://example.org/play", "https://other.example/cb", obj.ErrCodeCallbackOffSite},
		{"callback on www", "https://example.org/play", "https://www.example.org/cb", obj.ErrCodeCallbackOffSite},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			site, err := checkPageURLs(VerifiedPageDraft{PageURL: c.page, CallbackURL: c.callback})
			if got := codeOf(err); got != c.want {
				t.Fatalf("refused with %q, want %q (err: %v)", got, c.want, err)
			}
			if c.want == "" && site != "https://example.org" {
				t.Errorf("site %q, want https://example.org", site)
			}
		})
	}
}

// A plain http page's site keeps its scheme, in production too: the admin
// approves http://example.org apart from https://example.org.
func TestAPlainHTTPPagesSiteKeepsItsScheme(t *testing.T) {
	t.Setenv("SCHEMAF_DEV", "")
	site, err := checkPageURLs(VerifiedPageDraft{PageURL: "http://Example.org:8080/play", CallbackURL: "http://example.org/cb"})
	if err != nil {
		t.Fatal(err)
	}
	if site != "http://example.org" {
		t.Errorf("site %q, want http://example.org", site)
	}
}
