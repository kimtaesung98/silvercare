package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCaregiverToken(t *testing.T) {
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	tokens := NewTokens(strings.Repeat("k", 32), time.Hour)
	tokens.now = func() time.Time { return now }
	id := uuid.New()

	tok, exp := tokens.IssueCaregiver(id)
	if !exp.Equal(now.Add(time.Hour)) {
		t.Errorf("exp = %v", exp)
	}
	got, err := tokens.VerifyCaregiver(tok)
	if err != nil || got != id {
		t.Fatalf("verify = %v, %v", got, err)
	}

	other := NewTokens(strings.Repeat("x", 32), time.Hour)
	other.now = tokens.now
	forged, _ := other.IssueCaregiver(id)

	parts := strings.Split(tok, ".")
	tampered := strings.Join([]string{parts[0], uuid.NewString(), parts[2], parts[3]}, ".")

	for name, bad := range map[string]string{
		"empty":         "",
		"no signature":  "cg1." + id.String(),
		"other secret":  forged,
		"swapped id":    tampered,
		"garbage":       "a.b.c.d",
		"trailing junk": tok + "x",
	} {
		if _, err := tokens.VerifyCaregiver(bad); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	now = now.Add(time.Hour)
	if _, err := tokens.VerifyCaregiver(tok); err == nil {
		t.Error("expired token accepted")
	}
}

func TestPassword(t *testing.T) {
	h, err := HashPassword("pw-1234")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "pw-1234") || CheckPassword(h, "pw-1235") {
		t.Error("password check wrong")
	}
	if CheckPasswordOrDummy(nil, "pw-1234") {
		t.Error("nil hash accepted")
	}
}

func TestBearerToken(t *testing.T) {
	cases := map[string]struct {
		tok string
		ok  bool
	}{
		"Bearer abc": {"abc", true},
		"bearer abc": {"abc", true},
		"Bearer ":    {"", false},
		"Basic abc":  {"", false},
		"":           {"", false},
		"Bearerabc":  {"", false},
	}
	for in, want := range cases {
		tok, ok := BearerToken(in)
		if tok != want.tok || ok != want.ok {
			t.Errorf("BearerToken(%q) = %q, %v", in, tok, ok)
		}
	}
}

func TestDeviceToken(t *testing.T) {
	a, b := NewDeviceToken(), NewDeviceToken()
	if a == b || len(a) < 40 {
		t.Errorf("tokens %q %q", a, b)
	}
	if len(HashDeviceToken(a)) != 32 {
		t.Error("hash length")
	}
}
