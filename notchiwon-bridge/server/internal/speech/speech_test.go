package speech

import (
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWAV(t *testing.T) {
	pcm := []byte{1, 0, 2, 0, 3, 0}
	w := WAV(pcm, 16000)
	if len(w) != 44+len(pcm) || string(w[0:4]) != "RIFF" || string(w[8:16]) != "WAVEfmt " || string(w[36:40]) != "data" {
		t.Fatalf("header = %q", w[:44])
	}
	if sr := binary.LittleEndian.Uint32(w[24:28]); sr != 16000 {
		t.Errorf("sample rate = %d", sr)
	}
	if n := binary.LittleEndian.Uint32(w[40:44]); n != uint32(len(pcm)) {
		t.Errorf("data size = %d", n)
	}
}

func TestClova(t *testing.T) {
	var gotSTT, gotTTS *http.Request
	var sttBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-NCP-APIGW-API-KEY-ID") != "id" || r.Header.Get("X-NCP-APIGW-API-KEY") != "secret" {
			http.Error(w, `{"error":"auth"}`, http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/stt":
			gotSTT = r
			sttBody, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte(`{"text":" 손주가 어제 왔어 "}`))
		case "/tts":
			_ = r.ParseForm()
			gotTTS = r
			_, _ = w.Write([]byte("ID3mp3"))
		}
	}))
	defer srv.Close()
	c := NewClova(ClovaConfig{ClientID: "id", ClientSecret: "secret", STTURL: srv.URL + "/stt", TTSURL: srv.URL + "/tts", DefaultSpeaker: "nara", TTSSpeed: 1})
	ctx := context.Background()

	text, err := c.Recognize(ctx, []byte{0, 0, 1, 0}, "pcm16", 16000)
	if err != nil || text != "손주가 어제 왔어" {
		t.Fatalf("Recognize = %q, %v", text, err)
	}
	if gotSTT.URL.Query().Get("lang") != "Kor" || string(sttBody[:4]) != "RIFF" {
		t.Errorf("stt request: %s, body %q", gotSTT.URL, sttBody[:4])
	}

	audio, err := c.Synthesize(ctx, "안녕하세요.", "default")
	if err != nil || string(audio) != "ID3mp3" {
		t.Fatalf("Synthesize = %q, %v", audio, err)
	}
	if f := gotTTS.Form; f.Get("speaker") != "nara" || f.Get("text") != "안녕하세요." || f.Get("format") != "mp3" || f.Get("speed") != "1" {
		t.Errorf("tts form = %v", f)
	}

	bad := NewClova(ClovaConfig{ClientID: "id", ClientSecret: "wrong", STTURL: srv.URL + "/stt", TTSURL: srv.URL + "/tts"})
	if _, err := bad.Synthesize(ctx, "안녕", "nara"); err == nil {
		t.Error("expected an error for a rejected key")
	}
}
