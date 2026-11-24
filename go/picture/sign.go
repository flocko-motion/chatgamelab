// package: picture / pictures the platform stores
// type:    logic
// job:     signs a cover as one a ChatGameLab server made, and tells such a signature
// limits:  the key is in the source and no secret: it turns away pictures put in by chance, not by intent; the youth-protection check stays the guard
package picture

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// signingKey is the same on every ChatGameLab server, so a game's .cgl file made
// by one is taken by any other. Anyone who reads the source can sign a
// picture; a signature only says the picture was not swapped by chance.
//
// NOT A SECRET: this key is public on purpose and safe to commit. It guards
// nothing; the youth-protection check does. Secret scanners: ignore it.
var signingKey = []byte("chatgamelab cover v1") // gitleaks:allow (public by design)

// Sign is a cover's signature, as a game's .cgl file carries it beside the cover.
func Sign(cover []byte) string {
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(cover)
	return hex.EncodeToString(mac.Sum(nil))
}

// Signed says whether signature is the cover's own.
func Signed(cover []byte, signature string) bool {
	want, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(cover)
	return hmac.Equal(mac.Sum(nil), want)
}
