package adapters

import (
	"context"
	"encoding/binary"
)

// Hearing hears one clip of the player's speech, with instructions, and
// answers in a schema: a tool call that takes audio. The reply is the JSON
// the model gave, unchecked; the caller checks it against the schema, as it
// does any structured reply. A refusal or a reply cut short is ErrBadReply.
type Hearing interface {
	Hear(ctx context.Context, system string, clip Clip, schema Schema) (string, Usage, error)
}

// Clip is one utterance of the player's: PCM16, little-endian, mono, at
// ClipRate, as the playback-engine records it while the button is held.
type Clip []byte

// ClipRate is a clip's sample rate, the realtime APIs' own.
const ClipRate = 24_000

// Seconds is how long the clip lasts.
func (c Clip) Seconds() float64 { return float64(len(c)/2) / ClipRate }

// Silent reports a clip with no sample louder than a whisper, as a held
// button with nobody speaking gives.
func (c Clip) Silent() bool {
	const whisper = 64
	for i := 0; i+1 < len(c); i += 2 {
		sample := int16(binary.LittleEndian.Uint16(c[i:]))
		if sample > whisper || sample < -whisper {
			return false
		}
	}
	return true
}

// WAV wraps the clip in a WAV file's header, for a provider that wants a file.
func (c Clip) WAV() []byte {
	const (
		channels      = 1
		bitsPerSample = 16
		headerSize    = 44
	)
	data := len(c) &^ 1
	out := make([]byte, 0, headerSize+data)
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(36+data))
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16) // the fmt chunk's size
	out = binary.LittleEndian.AppendUint16(out, 1)  // PCM
	out = binary.LittleEndian.AppendUint16(out, channels)
	out = binary.LittleEndian.AppendUint32(out, ClipRate)
	out = binary.LittleEndian.AppendUint32(out, ClipRate*channels*bitsPerSample/8)
	out = binary.LittleEndian.AppendUint16(out, channels*bitsPerSample/8)
	out = binary.LittleEndian.AppendUint16(out, bitsPerSample)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(data))
	return append(out, c[:data]...)
}
