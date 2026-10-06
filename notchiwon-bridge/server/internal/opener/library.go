package opener

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// DefaultVoice is used for elders without preferred_tts_voice, and for
// voices that have no clips recorded yet.
const DefaultVoice = "default"

// ErrNoClip means no active clip exists for the category in either voice.
var ErrNoClip = errors.New("no opener clip")

// Library reads the active clips from opener_clip.
type Library struct {
	q *db.Queries
}

// NewLibrary returns a Library.
func NewLibrary(q *db.Queries) *Library { return &Library{q: q} }

// VoiceOf returns the voice an elder's clips are recorded in.
func VoiceOf(e db.Elder) string {
	if e.PreferredTtsVoice != nil && *e.PreferredTtsVoice != "" {
		return *e.PreferredTtsVoice
	}
	return DefaultVoice
}

// Manifest is the clip list the tablet caches.
type Manifest struct {
	Voice   string
	Version string
	Clips   []db.OpenerClip
}

// Manifest lists the clips for voice, falling back to the default voice when
// the voice has none.
func (l *Library) Manifest(ctx context.Context, voice string) (Manifest, error) {
	clips, err := l.q.ListActiveOpenerClips(ctx, voice)
	if err != nil {
		return Manifest{}, fmt.Errorf("list opener clips: %w", err)
	}
	if len(clips) == 0 && voice != DefaultVoice {
		voice = DefaultVoice
		if clips, err = l.q.ListActiveOpenerClips(ctx, voice); err != nil {
			return Manifest{}, fmt.Errorf("list opener clips: %w", err)
		}
	}
	return Manifest{Voice: voice, Version: version(clips), Clips: clips}, nil
}

// version hashes what the tablet caches, so it changes whenever a clip is
// added, retired, re-recorded or re-worded.
func version(clips []db.OpenerClip) string {
	h := sha256.New()
	for _, c := range clips {
		fmt.Fprintf(h, "%s|%s|%s|%s|%d\n", c.ID, c.Category, c.Text, c.AudioRef, c.DurationMs)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Pick returns a random clip of the category in the manifest, avoiding
// `avoid` (the clip played last) when there is another choice.
func (m Manifest) Pick(c Category, avoid uuid.UUID) (db.OpenerClip, error) {
	var choices []db.OpenerClip
	for _, clip := range m.Clips {
		if clip.Category == string(c) && clip.ID != avoid {
			choices = append(choices, clip)
		}
	}
	if len(choices) == 0 {
		for _, clip := range m.Clips {
			if clip.Category == string(c) {
				choices = append(choices, clip)
			}
		}
	}
	if len(choices) == 0 {
		return db.OpenerClip{}, fmt.Errorf("%w: %s in voice %s", ErrNoClip, c, m.Voice)
	}
	return choices[rand.IntN(len(choices))], nil
}
