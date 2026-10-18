package session

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSaveAudio(t *testing.T) {
	dir := t.TempDir()
	pcm := []byte{1, 0, 2, 0, 3, 0}
	path, err := saveAudio(dir, uuid.New(), "pcm16", 16000, pcm)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, ".wav") {
		t.Fatalf("path = %s, want .wav", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 44+len(pcm) || string(got[0:4]) != "RIFF" || string(got[8:12]) != "WAVE" || string(got[36:40]) != "data" {
		t.Fatalf("bad wav header: % x", got[:44])
	}
	if string(got[44:]) != string(pcm) {
		t.Fatal("pcm not copied after the header")
	}

	path, err = saveAudio(dir, uuid.New(), "opus", 48000, []byte("ogg"))
	if err != nil || !strings.HasSuffix(path, ".opus") {
		t.Fatalf("opus: path = %s, err = %v", path, err)
	}
}
