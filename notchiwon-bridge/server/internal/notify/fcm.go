// Package notify sends push notifications to caregiver phones through
// Firebase Cloud Messaging (HTTP v1 API).
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Message is one push notification.
type Message struct {
	Title string
	Body  string
	// Data reaches the app even when it is in the background; the app reads
	// "type" and the id fields to open the right screen.
	Data map[string]string
	// Channel is the Android notification channel. Empty means
	// AndroidChannel: an alert that should wake the phone.
	Channel string
}

// channel returns the Android channel this message goes out on.
func (m Message) channel() string {
	if m.Channel == "" {
		return AndroidChannel
	}
	return m.Channel
}

// priority is HIGH for an alert and NORMAL for news that can wait for the
// phone to wake on its own.
func (m Message) priority() string {
	if m.channel() == AndroidChannel {
		return "HIGH"
	}
	return "NORMAL"
}

// Sender delivers a Message to one FCM registration token.
type Sender interface {
	Send(ctx context.Context, token string, m Message) error
}

var (
	// ErrUnregistered means the token is no longer valid (app removed or
	// token rotated); the device should not be sent to again.
	ErrUnregistered = errors.New("fcm token is not registered")
	// ErrUnavailable is what Unavailable returns.
	ErrUnavailable = errors.New("fcm is not configured (FCM_CREDENTIALS_FILE is empty)")
)

// Unavailable is the Sender used when no Firebase service account is set.
// Escalations are still recorded and shown in the app's open list.
type Unavailable struct{}

// Send implements Sender.
func (Unavailable) Send(context.Context, string, Message) error { return ErrUnavailable }

// fcmScope is the OAuth scope for the FCM HTTP v1 API.
const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// Notification channels the apps create. AndroidChannel carries escalation
// alerts (high importance, sound on); DigestChannel carries the guardian's
// daily news, which should not sound like an alarm.
const (
	AndroidChannel = "escalation"
	DigestChannel  = "digest"
)

// FCM is the Sender backed by the FCM HTTP v1 API.
type FCM struct {
	projectID string
	client    *http.Client
	// baseURL is overridden in tests.
	baseURL string
}

// NewFCM returns a Sender for the Firebase project of the service account
// key in credentialsJSON.
func NewFCM(ctx context.Context, credentialsJSON []byte) (*FCM, error) {
	creds, err := google.CredentialsFromJSONWithType(ctx, credentialsJSON, google.ServiceAccount, fcmScope)
	if err != nil {
		return nil, fmt.Errorf("read firebase service account: %w", err)
	}
	if creds.ProjectID == "" {
		return nil, errors.New("firebase service account has no project_id")
	}
	return &FCM{
		projectID: creds.ProjectID,
		client:    oauth2.NewClient(ctx, creds.TokenSource),
		baseURL:   "https://fcm.googleapis.com",
	}, nil
}

// newFCMForTest skips OAuth.
func newFCMForTest(projectID, baseURL string, client *http.Client) *FCM {
	return &FCM{projectID: projectID, client: client, baseURL: baseURL}
}

type fcmRequest struct {
	Message fcmMessage `json:"message"`
}

type fcmMessage struct {
	Token        string            `json:"token"`
	Notification fcmNotification   `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
	Android      fcmAndroid        `json:"android"`
}

type fcmNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type fcmAndroid struct {
	// HIGH wakes the phone even in Doze; NORMAL waits.
	Priority     string                 `json:"priority"`
	Notification fcmAndroidNotification `json:"notification"`
}

type fcmAndroidNotification struct {
	ChannelID string `json:"channel_id"`
	Sound     string `json:"sound"`
}

// Send implements Sender.
func (f *FCM) Send(ctx context.Context, token string, m Message) error {
	body, err := json.Marshal(fcmRequest{Message: fcmMessage{
		Token:        token,
		Notification: fcmNotification{Title: m.Title, Body: m.Body},
		Data:         m.Data,
		Android: fcmAndroid{
			Priority:     m.priority(),
			Notification: fcmAndroidNotification{ChannelID: m.channel(), Sound: "default"},
		},
	}})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/v1/projects/%s/messages:send", f.baseURL, f.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("fcm send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	// 404 NOT_FOUND / UNREGISTERED: the app was removed or the token rotated.
	if resp.StatusCode == http.StatusNotFound || strings.Contains(string(raw), "UNREGISTERED") {
		return ErrUnregistered
	}
	return fmt.Errorf("fcm send: HTTP %d: %s", resp.StatusCode, raw)
}
