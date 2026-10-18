package session

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// saveAudio writes one elder recording into dir so a developer can listen to
// what the tablet sent (DEBUG_AUDIO_DIR). PCM16 gets a WAV header; other
// formats are written as they came.
func saveAudio(dir string, sessionID uuid.UUID, format string, sampleRate int, audio []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405.000"), sessionID.String()[:8])
	if format != "pcm16" {
		path := filepath.Join(dir, name+"."+format)
		return path, os.WriteFile(path, audio, 0o644)
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	path := filepath.Join(dir, name+".wav")
	return path, os.WriteFile(path, wav(audio, sampleRate), 0o644)
}

// wav wraps mono little-endian PCM16 in a 44-byte RIFF header.
func wav(pcm []byte, sampleRate int) []byte {
	out := make([]byte, 44+len(pcm))
	le := binary.LittleEndian
	copy(out[0:], "RIFF")
	le.PutUint32(out[4:], uint32(36+len(pcm)))
	copy(out[8:], "WAVEfmt ")
	le.PutUint32(out[16:], 16)
	le.PutUint16(out[20:], 1) // PCM
	le.PutUint16(out[22:], 1) // mono
	le.PutUint32(out[24:], uint32(sampleRate))
	le.PutUint32(out[28:], uint32(sampleRate*2))
	le.PutUint16(out[32:], 2)
	le.PutUint16(out[34:], 16)
	copy(out[36:], "data")
	le.PutUint32(out[40:], uint32(len(pcm)))
	copy(out[44:], pcm)
	return out
}
