// package: picture / pictures the platform stores
// type:    logic
// job:     scales a picture down to the JPEG size the platform stores a cover at
// limits:  reads PNG and JPEG only; drawing pictures is the engine's
package picture

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // a cover arrives as a PNG
)

// CoverWidth is the width, in pixels, a cover is stored at: sharp on a game's
// tile on a sharp screen, and in the editor.
const CoverWidth = 600

// quality is the JPEG quality: small, and still clean at the size shown.
const quality = 80

// Scale scales a PNG or JPEG down to width pixels, keeping its aspect, and
// encodes it as a JPEG. A picture already that narrow is only re-encoded.
func Scale(data []byte, width int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("reading the picture: %w", err)
	}
	b := src.Bounds()
	if b.Dx() > width {
		src = shrink(src, width, b.Dy()*width/b.Dx())
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, src, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("writing the picture: %w", err)
	}
	return out.Bytes(), nil
}

// shrink scales src down to w×h by averaging the source pixels each target
// pixel covers, which keeps a downscaled picture smooth where picking one
// pixel per target would alias.
func shrink(src image.Image, w, h int) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, w, max(h, 1)))
	for y := 0; y < dst.Bounds().Dy(); y++ {
		y0 := b.Min.Y + y*b.Dy()/dst.Bounds().Dy()
		y1 := max(b.Min.Y+(y+1)*b.Dy()/dst.Bounds().Dy(), y0+1)
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*b.Dx()/w
			x1 := max(b.Min.X+(x+1)*b.Dx()/w, x0+1)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
				}
			}
			dst.SetRGBA64(x, y, color.RGBA64{R: uint16(r / n), G: uint16(g / n), B: uint16(bl / n), A: uint16(a / n)})
		}
	}
	return dst
}
