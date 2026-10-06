package opener

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/speech"
)

// Audio returns a clip's MP3, synthesizing and storing it the first time.
// It returns ErrNoClip for unknown or retired clips.
func (l *Library) Audio(ctx context.Context, clipID uuid.UUID, tts speech.Synthesizer) ([]byte, error) {
	clip, err := l.q.GetOpenerClip(ctx, clipID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !clip.Active) {
		return nil, ErrNoClip
	}
	if err != nil {
		return nil, fmt.Errorf("read clip: %w", err)
	}
	mp3, err := l.q.GetOpenerClipAudio(ctx, clipID)
	if err == nil {
		return mp3, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("read clip audio: %w", err)
	}
	return l.synthesize(ctx, clip, tts)
}

// SynthesizeMissing makes audio for every active clip that has none and
// returns how many it made.
func (l *Library) SynthesizeMissing(ctx context.Context, tts speech.Synthesizer) (int, error) {
	clips, err := l.q.ListOpenerClipsWithoutAudio(ctx)
	if err != nil {
		return 0, fmt.Errorf("list clips without audio: %w", err)
	}
	for i, c := range clips {
		if _, err := l.synthesize(ctx, c, tts); err != nil {
			return i, err
		}
	}
	return len(clips), nil
}

func (l *Library) synthesize(ctx context.Context, clip db.OpenerClip, tts speech.Synthesizer) ([]byte, error) {
	mp3, err := tts.Synthesize(ctx, clip.Text, clip.Voice)
	if err != nil {
		return nil, fmt.Errorf("synthesize clip %s: %w", clip.ID, err)
	}
	if err := l.q.SaveOpenerClipAudio(ctx, db.SaveOpenerClipAudioParams{ClipID: clip.ID, Mp3: mp3}); err != nil {
		return nil, fmt.Errorf("save clip audio: %w", err)
	}
	return mp3, nil
}
