package db

import (
	"cgl/obj"
	"testing"

	"github.com/sqlc-dev/pqtype"
)

func TestPublicSlugBase(t *testing.T) {
	cases := map[string]string{
		"Robotik-AG":           "robotik-ag",
		"Ärger & Spaß!":        "aerger-spass",
		"  Sommer_Camp 2026  ": "sommer-camp-2026",
		"Café Übung":           "cafe-uebung",
		"!!!":                  "",
		"Ein sehr langer Workshopname mit vielen Wörtern": "ein-sehr-langer-workshopname",
		"Einsehrlangerworkshopnameohneleerzeichen":        "einsehrlangerworkshopnameohnel",
	}
	for name, want := range cases {
		if got := publicSlugBase(name); got != want {
			t.Errorf("publicSlugBase(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestValidatePublicSlug(t *testing.T) {
	for _, ok := range []string{"abc", "robotik-ag-mokassin-hausboot", "2026-camp"} {
		if err := validatePublicSlug(ok); err != nil {
			t.Errorf("validatePublicSlug(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"ab", "-abc", "abc-", "a--b", "a b", "äbc", "ABC"} {
		if err := validatePublicSlug(bad); err == nil {
			t.Errorf("validatePublicSlug(%q) = nil, want an error", bad)
		}
	}
}

func TestValidatePublicLinksRejectsNonHTTP(t *testing.T) {
	bad := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"chatgamelab.eu",
		"//chatgamelab.eu",
		"https://",
		"",
	}
	for _, raw := range bad {
		_, err := ValidatePublicLinks([]obj.PublicWorkshopLink{{Title: "Programm", URL: raw}})
		if err == nil {
			t.Errorf("ValidatePublicLinks(%q) = nil, want an error", raw)
		}
	}
}

func TestValidatePublicLinks(t *testing.T) {
	links, err := ValidatePublicLinks([]obj.PublicWorkshopLink{
		{Title: "  Programm  ", Description: "  Das Heft  ", URL: "  https://example.org/heft  "},
		{Title: "", Description: "", URL: ""}, // a row left blank is dropped
		{Title: "Fotos", URL: "http://example.org/fotos"},
	})
	if err != nil {
		t.Fatalf("ValidatePublicLinks = %v, want nil", err)
	}
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2", len(links))
	}
	if links[0].Title != "Programm" || links[0].Description != "Das Heft" || links[0].URL != "https://example.org/heft" {
		t.Errorf("first link not trimmed: %+v", links[0])
	}

	if _, err := ValidatePublicLinks([]obj.PublicWorkshopLink{{URL: "https://example.org"}}); err == nil {
		t.Error("a link without a title was accepted")
	}

	many := make([]obj.PublicWorkshopLink, PublicLinksMax+1)
	for i := range many {
		many[i] = obj.PublicWorkshopLink{Title: "x", URL: "https://example.org"}
	}
	if _, err := ValidatePublicLinks(many); err == nil {
		t.Errorf("%d links were accepted, want an error", len(many))
	}
}

func TestPublicLinksRoundTrip(t *testing.T) {
	want := []obj.PublicWorkshopLink{{Title: "Programm", Description: "Das Heft", URL: "https://example.org"}}
	raw, err := marshalPublicLinks(want)
	if err != nil {
		t.Fatalf("marshalPublicLinks = %v", err)
	}
	got := unmarshalPublicLinks(raw)
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("round trip gave %+v, want %+v", got, want)
	}
	if empty, _ := marshalPublicLinks(nil); empty.Valid {
		t.Error("no links should store NULL")
	}
	if got := unmarshalPublicLinks(pqtype.NullRawMessage{}); got != nil {
		t.Errorf("NULL gave %+v, want nil", got)
	}
}
