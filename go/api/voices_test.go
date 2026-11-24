package api

import (
	"context"
	"net/http"
	"testing"

	"cgl/api/apitest"
)

// A voice is heard and made in the editor only for a voice set to custom,
// in a genre that offers it:
// with a built-in choice the session casts its own voice, which a preview
// would misrepresent, and no voice is made for it.
func TestVoicesAreHeardAndMadeOnlyForCustom(t *testing.T) {
	for _, form := range []VoiceSampleReq{
		{Genre: "npc-live", VoiceStyle: "deep"},
		{Genre: "npc-live", VoiceStyle: "deep", Voice: "male"},
		{Genre: "npc-live", VoiceStyle: "deep", Voice: "female"},
		{Genre: "npc-live", VoiceStyle: "deep", Voice: "any"},
		// An Adventure's narrator is always cast: it offers no custom.
		{Genre: "adventure", VoiceStyle: "deep", Voice: "custom"},
	} {
		for name, handle := range map[string]func(context.Context, VoiceSampleReq) (VoiceSampleResp, error){
			"sample": VoiceSampleEndpoint{}.Handle,
			"new":    NewVoiceEndpoint{}.Handle,
		} {
			_, err := handle(context.Background(), form)
			if err == nil {
				t.Fatalf("%s for %+v: no refusal", name, form)
			}
			rec, body := apitest.ServeError(t, func() error { return err })
			if rec.Code != http.StatusBadRequest || body["code"] != codeVoiceNotCustom {
				t.Errorf("%s for %+v: %d %v, want 400 %s", name, form, rec.Code, body, codeVoiceNotCustom)
			}
		}
	}
}
