package mock

import (
	_ "embed"
	"encoding/binary"
	"math"
)

// voiceSample is the mock's voice preview: one spoken line ("Hello! This is
// the mock voice. …"), raw PCM16 at 24 kHz mono, made once with OpenAI's
// speech API (gpt-4o-mini-tts, voice marin) by the project's owner, whose
// output it is. A designer previewing a voice on the mock hears speech.
//
//go:embed voice-sample.pcm
var voiceSample []byte

// In conversation the mock speaks in tones rather than words. Raw PCM16 at
// 24 kHz is exactly what a realtime session sends, so the player decodes this
// through the same path it will use for real speech — which is the point of a
// mock: exercise the machinery, not the content.
const (
	sampleRate    = 24_000
	frameSamples  = 480 // 20 ms, the granularity real audio streams at
	utteranceSecs = 0.6
)

// Pitches differ per utterance so consecutive lines are audibly distinct, which
// is what you want when checking that a reply actually arrived.
var pitches = []float64{196.00, 246.94, 293.66, 329.63, 392.00}

// speech renders one utterance as a sequence of PCM16 frames.
func speech(index int) [][]byte { return tone(index, utteranceSecs) }

// tone renders a tone of the index's pitch, lasting seconds, as PCM16 frames.
func tone(index int, seconds float64) [][]byte {
	pitch := pitches[index%len(pitches)]
	total := int(seconds * sampleRate)

	var frames [][]byte
	for start := 0; start < total; start += frameSamples {
		end := min(start+frameSamples, total)
		frame := make([]byte, 0, (end-start)*2)
		for i := start; i < end; i++ {
			t := float64(i) / sampleRate
			// A soft attack and decay, so the tone sounds spoken rather than
			// switched on: an abrupt edge clicks in a speaker.
			envelope := math.Sin(math.Pi * float64(i) / float64(total))
			sample := math.Sin(2*math.Pi*pitch*t) * envelope * 0.35
			frame = binary.LittleEndian.AppendUint16(frame, uint16(int16(sample*math.MaxInt16)))
		}
		frames = append(frames, frame)
	}
	return frames
}
