// Package speech turns elder audio into text (STT) and AI sentences into
// audio (TTS) with Naver Clova. The tablet only records and plays: it never
// holds a Clova key.
package speech

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Recognizer is speech to text.
type Recognizer interface {
	// Recognize returns what was said in audio. format is an elder.audio
	// format (pcm16, opus, aac); sampleRate applies to pcm16.
	Recognize(ctx context.Context, audio []byte, format string, sampleRate int) (string, error)
}

// Synthesizer is text to speech.
type Synthesizer interface {
	// Synthesize returns MP3 audio of text spoken in voice ("default" is the
	// server's default speaker).
	Synthesize(ctx context.Context, text, voice string) ([]byte, error)
}

// ErrUnavailable means no Clova key is configured.
var ErrUnavailable = errors.New("clova is not configured (NAVER_CLOVA_CLIENT_ID / NAVER_CLOVA_CLIENT_SECRET are empty)")

// Unavailable is used without Clova keys: recognition fails and no audio is
// made, so the tablet falls back to text (stage 3 text mode still works).
type Unavailable struct{}

// Recognize implements Recognizer.
func (Unavailable) Recognize(context.Context, []byte, string, int) (string, error) {
	return "", ErrUnavailable
}

// Synthesize implements Synthesizer.
func (Unavailable) Synthesize(context.Context, string, string) ([]byte, error) {
	return nil, ErrUnavailable
}

// ClovaConfig configures Clova Speech Recognition (CSR) and Clova Voice.
type ClovaConfig struct {
	ClientID     string
	ClientSecret string
	STTURL       string // CSR short-form endpoint (lang=Kor is added)
	TTSURL       string // Clova Voice Premium endpoint
	// DefaultSpeaker is the Clova speaker for the "default" voice.
	DefaultSpeaker string
	// TTSSpeed is Clova's speed: -5 (faster) .. 5 (slower). Elders follow a
	// slightly slower voice better.
	TTSSpeed int
}

// Clova calls the Naver Cloud Clova APIs.
type Clova struct {
	cfg  ClovaConfig
	http *http.Client
}

// NewClova returns a Clova client.
func NewClova(cfg ClovaConfig) *Clova {
	return &Clova{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Clova) auth(req *http.Request) {
	req.Header.Set("X-NCP-APIGW-API-KEY-ID", c.cfg.ClientID)
	req.Header.Set("X-NCP-APIGW-API-KEY", c.cfg.ClientSecret)
}

func (c *Clova) do(req *http.Request) ([]byte, error) {
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clova %s: %s: %s", req.URL.Path, resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// Recognize implements Recognizer. CSR takes WAV but not raw PCM, so pcm16
// gets a WAV header first.
func (c *Clova) Recognize(ctx context.Context, audio []byte, format string, sampleRate int) (string, error) {
	if format == "pcm16" {
		if sampleRate == 0 {
			sampleRate = 16000
		}
		audio = WAV(audio, sampleRate)
	}
	u, err := url.Parse(c.cfg.STTURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("lang", "Kor")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(audio))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	body, err := c.do(req)
	if err != nil {
		return "", err
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("clova stt response: %w", err)
	}
	return strings.TrimSpace(out.Text), nil
}

// Synthesize implements Synthesizer.
func (c *Clova) Synthesize(ctx context.Context, text, voice string) ([]byte, error) {
	if voice == "" || voice == "default" {
		voice = c.cfg.DefaultSpeaker
	}
	form := url.Values{
		"speaker": {voice},
		"text":    {text},
		"format":  {"mp3"},
		"speed":   {strconv.Itoa(c.cfg.TTSSpeed)},
		"volume":  {"0"},
		"pitch":   {"0"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TTSURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	audio, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, errors.New("clova tts returned no audio")
	}
	return audio, nil
}

// WAV wraps 16-bit little-endian mono PCM in a WAV header.
func WAV(pcm []byte, sampleRate int) []byte {
	var b bytes.Buffer
	b.Grow(44 + len(pcm))
	w := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	w(uint32(36 + len(pcm)))
	b.WriteString("WAVEfmt ")
	w(uint32(16))             // fmt chunk size
	w(uint16(1))              // PCM
	w(uint16(1))              // mono
	w(uint32(sampleRate))     // sample rate
	w(uint32(sampleRate * 2)) // byte rate
	w(uint16(2))              // block align
	w(uint16(16))             // bits per sample
	b.WriteString("data")
	w(uint32(len(pcm)))
	b.Write(pcm)
	return b.Bytes()
}
