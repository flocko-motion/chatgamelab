package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cgl/api/apitest"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

func TestGetLocaleFile(t *testing.T) {
	route := schemafapi.NewRoute[LocaleReq, LocaleResp](GetLocaleFileEndpoint{}, "", "")

	rec := apitest.Serve(t, route, httptest.NewRequest(http.MethodGet, "/api/languages/de", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", rec.Code, rec.Body)
	}
	var translations map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&translations); err != nil || len(translations) == 0 {
		t.Fatalf("translations %v, err %v", translations, err)
	}

	rec = apitest.Serve(t, route, httptest.NewRequest(http.MethodGet, "/api/languages/xx", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown language: status %d, want 404", rec.Code)
	}
}
