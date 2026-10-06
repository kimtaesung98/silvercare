// Package auth issues and checks credentials: caregiver and guardian access
// tokens (temporary center-issued logins) and elder tablet device tokens.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidToken is returned for malformed, forged or expired tokens.
var ErrInvalidToken = errors.New("invalid token")

const (
	caregiverTokenPrefix = "cg1"
	guardianTokenPrefix  = "gd1"
)

// Tokens signs and verifies access tokens with HMAC-SHA256. A token is
// "<prefix>.<account id>.<expiry unix>.<signature>", where the prefix says
// whose token it is ("cg1" a caregiver's, "gd1" a guardian's), so a
// caregiver token cannot be used on a guardian endpoint.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokens returns a Tokens that issues tokens valid for ttl.
func NewTokens(secret string, ttl time.Duration) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// IssueCaregiver returns an access token for the caregiver and its expiry.
func (t *Tokens) IssueCaregiver(caregiverID uuid.UUID) (string, time.Time) {
	return t.issue(caregiverTokenPrefix, caregiverID)
}

// IssueGuardian returns an access token for the guardian and its expiry.
func (t *Tokens) IssueGuardian(guardianID uuid.UUID) (string, time.Time) {
	return t.issue(guardianTokenPrefix, guardianID)
}

func (t *Tokens) issue(prefix string, id uuid.UUID) (string, time.Time) {
	exp := t.now().Add(t.ttl).Truncate(time.Second)
	payload := prefix + "." + id.String() + "." + strconv.FormatInt(exp.Unix(), 10)
	return payload + "." + t.sign(payload), exp
}

// VerifyCaregiver returns the caregiver ID in a valid, unexpired token.
func (t *Tokens) VerifyCaregiver(token string) (uuid.UUID, error) {
	return t.verify(caregiverTokenPrefix, token)
}

// VerifyGuardian returns the guardian ID in a valid, unexpired token.
func (t *Tokens) VerifyGuardian(token string) (uuid.UUID, error) {
	return t.verify(guardianTokenPrefix, token)
}

func (t *Tokens) verify(prefix, token string) (uuid.UUID, error) {
	i := strings.LastIndexByte(token, '.')
	if i < 0 {
		return uuid.Nil, ErrInvalidToken
	}
	payload, sig := token[:i], token[i+1:]
	if !hmac.Equal([]byte(sig), []byte(t.sign(payload))) {
		return uuid.Nil, ErrInvalidToken
	}
	parts := strings.Split(payload, ".")
	if len(parts) != 3 || parts[0] != prefix {
		return uuid.Nil, ErrInvalidToken
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	exp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || !t.now().Before(time.Unix(exp, 0)) {
		return uuid.Nil, ErrInvalidToken
	}
	return id, nil
}

func (t *Tokens) sign(payload string) string {
	m := hmac.New(sha256.New, t.secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// HashPassword returns a bcrypt hash of password.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}

// CheckPassword reports whether password matches the bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash keeps login timing the same whether or not the login ID exists.
var dummyHash, _ = HashPassword("not-a-real-password")

// CheckPasswordOrDummy is CheckPassword that still spends bcrypt time when
// the account has no hash, so unknown login IDs are not distinguishable by timing.
func CheckPasswordOrDummy(hash *string, password string) bool {
	if hash == nil {
		CheckPassword(dummyHash, password)
		return false
	}
	return CheckPassword(*hash, password)
}

// NewDeviceToken returns a random tablet token. Only its HashDeviceToken is stored.
func NewDeviceToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// HashDeviceToken is the value stored in device.token_hash.
func HashDeviceToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// BearerToken extracts the token from an "Authorization: Bearer <token>" header value.
func BearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}

// Caregiver is the authenticated caregiver of a request.
type Caregiver struct {
	ID       uuid.UUID
	CenterID uuid.UUID
}

// Guardian is the authenticated guardian of a request.
type Guardian struct {
	ID uuid.UUID
}

// Tablet is the authenticated elder tablet of a request.
type Tablet struct {
	DeviceID uuid.UUID
	ElderID  uuid.UUID
}

type ctxKey int

const (
	caregiverKey ctxKey = iota
	guardianKey
	tabletKey
)

// WithCaregiver stores the authenticated caregiver in ctx.
func WithCaregiver(ctx context.Context, c Caregiver) context.Context {
	return context.WithValue(ctx, caregiverKey, c)
}

// CaregiverFrom returns the caregiver stored by WithCaregiver.
func CaregiverFrom(ctx context.Context) (Caregiver, bool) {
	c, ok := ctx.Value(caregiverKey).(Caregiver)
	return c, ok
}

// WithGuardian stores the authenticated guardian in ctx.
func WithGuardian(ctx context.Context, g Guardian) context.Context {
	return context.WithValue(ctx, guardianKey, g)
}

// GuardianFrom returns the guardian stored by WithGuardian.
func GuardianFrom(ctx context.Context) (Guardian, bool) {
	g, ok := ctx.Value(guardianKey).(Guardian)
	return g, ok
}

// WithTablet stores the authenticated tablet in ctx.
func WithTablet(ctx context.Context, t Tablet) context.Context {
	return context.WithValue(ctx, tabletKey, t)
}

// TabletFrom returns the tablet stored by WithTablet.
func TabletFrom(ctx context.Context) (Tablet, bool) {
	t, ok := ctx.Value(tabletKey).(Tablet)
	return t, ok
}
