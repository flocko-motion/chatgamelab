package mock

import (
	"bytes"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
)

// The mock draws a real picture rather than returning a placeholder string, so
// the player decodes and displays it through the same path a generated image
// will use. Nothing here is art: the point is that a PNG of the right shape
// arrives and renders.
//
// Drawn with the standard library alone. Rectangles and circles are a few lines
// of pixel arithmetic, and a drawing dependency would buy nothing a mock needs.
const (
	imageWidth  = 512
	imageHeight = 320
	shapeCount  = 14
)

// paint renders a prompt as coloured geometry. The prompt seeds the randomness,
// so the same scenario always yields the same picture — which keeps a test that
// looks at one from being flaky, and makes a changed prompt visible at a glance.
func paint(prompt string) ([]byte, error) {
	seed := fnv.New64a()
	_, _ = seed.Write([]byte(prompt))
	random := rand.New(rand.NewSource(int64(seed.Sum64())))

	canvas := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))
	hue := random.Float64()
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{shade(hue, 0.25)}, image.Point{}, draw.Src)

	for i := 0; i < shapeCount; i++ {
		fill := shade(hue+random.Float64()*0.4, 0.45+random.Float64()*0.45)
		x, y := random.Intn(imageWidth), random.Intn(imageHeight)

		if random.Intn(2) == 0 {
			w, h := 30+random.Intn(160), 20+random.Intn(120)
			draw.Draw(canvas, image.Rect(x, y, x+w, y+h), &image.Uniform{fill}, image.Point{}, draw.Over)
			continue
		}

		radius := 18 + random.Intn(70)
		disc(canvas, x, y, radius, fill)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func disc(canvas *image.RGBA, cx, cy, radius int, fill color.Color) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= radius*radius {
				canvas.Set(x, y, fill)
			}
		}
	}
}

// shade turns a hue in turns and a brightness into a colour, so a picture holds
// together instead of looking like a test pattern.
func shade(hue, value float64) color.RGBA {
	hue -= float64(int(hue))
	if hue < 0 {
		hue += 1
	}

	region := hue * 6
	offset := region - float64(int(region))
	high := uint8(value * 255)
	low := uint8(value * 90)
	mid := uint8(value * (90 + 165*offset))
	dim := uint8(value * (255 - 165*offset))

	switch int(region) {
	case 0:
		return color.RGBA{high, mid, low, 255}
	case 1:
		return color.RGBA{dim, high, low, 255}
	case 2:
		return color.RGBA{low, high, mid, 255}
	case 3:
		return color.RGBA{low, dim, high, 255}
	case 4:
		return color.RGBA{mid, low, high, 255}
	default:
		return color.RGBA{high, low, dim, 255}
	}
}
