package picture

import "testing"

func TestASignatureFitsItsCoverOnly(t *testing.T) {
	cover := []byte{0xFF, 0xD8, 0xFF, 1, 2, 3}
	sig := Sign(cover)
	if !Signed(cover, sig) {
		t.Error("a cover's own signature is refused")
	}
	if Signed(append([]byte{}, cover[:5]...), sig) {
		t.Error("a changed picture passes")
	}
	if Signed(cover, "") || Signed(cover, "not hex") || Signed(cover, sig[:10]) {
		t.Error("a missing or broken signature passes")
	}
}
