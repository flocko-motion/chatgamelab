package obj

import "testing"

func TestImageHash(t *testing.T) {
	if got := ImageHash(nil); got != "" {
		t.Errorf("ImageHash(nil) = %q, want empty", got)
	}
	if got := ImageHash([]byte{}); got != "" {
		t.Errorf("ImageHash(empty) = %q, want empty", got)
	}

	a := ImageHash([]byte("scene-one"))
	if len(a) != 16 {
		t.Errorf("hash length = %d, want 16 (%q)", len(a), a)
	}
	if a != ImageHash([]byte("scene-one")) {
		t.Error("hash is not stable for identical input")
	}
	if a == ImageHash([]byte("scene-two")) {
		t.Error("different input produced the same hash")
	}
}
