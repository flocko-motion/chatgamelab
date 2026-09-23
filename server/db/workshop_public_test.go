package db

import "testing"

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
