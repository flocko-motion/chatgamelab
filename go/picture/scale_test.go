package picture

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"testing"
)

func TestScaleShrinksToWidthKeepingAspect(t *testing.T) {
	// Noise, as a generated picture's detail is: a PNG of it is large.
	random := rand.New(rand.NewSource(1))
	src := image.NewRGBA(image.Rect(0, 0, 1024, 1536))
	for y := 0; y < 1536; y++ {
		for x := 0; x < 1024; x++ {
			src.Set(x, y, color.RGBA{R: uint8(random.Intn(256)), G: uint8(random.Intn(256)), B: uint8(random.Intn(256)), A: 255})
		}
	}
	var in bytes.Buffer
	if err := png.Encode(&in, src); err != nil {
		t.Fatal(err)
	}
	scaled, err := Scale(in.Bytes(), CoverWidth)
	if err != nil {
		t.Fatalf("Scale: %v", err)
	}
	got, err := jpeg.Decode(bytes.NewReader(scaled))
	if err != nil {
		t.Fatalf("the scaled picture is no JPEG: %v", err)
	}
	if b := got.Bounds(); b.Dx() != CoverWidth || b.Dy() != 900 {
		t.Errorf("scaled to %dx%d, want %dx900", b.Dx(), b.Dy(), CoverWidth)
	}
	if len(scaled) >= in.Len() {
		t.Errorf("the scaled picture (%d bytes) is not smaller than the original (%d)", len(scaled), in.Len())
	}
}

func TestScaleRefusesWhatIsNoPicture(t *testing.T) {
	if _, err := Scale([]byte("not a picture"), CoverWidth); err == nil {
		t.Error("Scale accepted bytes that are no picture")
	}
}
