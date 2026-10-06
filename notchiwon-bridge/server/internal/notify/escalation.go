package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
)

// TriggerLabels are how the caregiver app names each escalation trigger.
var TriggerLabels = map[string]string{
	"FALL_MENTION":       "낙상 언급",
	"PAIN_COMPLAINT":     "통증 호소",
	"SELF_OR_OTHER_HARM": "자해·타해 언급",
	"OTHER_ANOMALY":      "이상 징후",
}

// Target is one entry of escalation_event.notified_targets: a caregiver
// phone that got the push, or a guardian phone number that got the text.
type Target struct {
	DeviceID *uuid.UUID `json:"deviceId,omitempty"`
	Phone    string     `json:"phone,omitempty"`
	SentAt   time.Time  `json:"sentAt"`
}

// EscalationWorker pushes an escalation to whoever must act on it: the
// caregiver of the visit whose pickup session raised it, or, in a companion
// session after daycare, the elder's guardian.
type EscalationWorker struct {
	river.WorkerDefaults[jobs.EscalationAlertArgs]
	Queries *db.Queries
	Sender  Sender
	Logger  *slog.Logger
	Now     func() time.Time
}

// Work implements river.Worker.
func (w *EscalationWorker) Work(ctx context.Context, job *river.Job[jobs.EscalationAlertArgs]) error {
	id := job.Args.EscalationID
	d, err := w.Queries.GetEscalationDetail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(fmt.Errorf("escalation %s not found", id))
	}
	if err != nil {
		return fmt.Errorf("read escalation: %w", err)
	}
	ev := d.EscalationEvent
	log := w.Logger.With("escalation_id", id, "session_id", ev.SessionID)
	switch {
	case ev.AcknowledgedAt != nil:
		return nil // seen already (a retry after someone opened the app)
	case d.CaregiverID == nil:
		return w.alertGuardian(ctx, d, log)
	}

	phones, err := w.Queries.ListCaregiverPhones(ctx, d.CaregiverID)
	if err != nil {
		return fmt.Errorf("list caregiver phones: %w", err)
	}
	if len(phones) == 0 {
		log.WarnContext(ctx, "caregiver has no phone registered for alerts", "caregiver_id", *d.CaregiverID)
		return nil
	}

	return w.push(ctx, d, phones, log)
}

// push sends the alert to every phone and records who got it. It returns an
// error only when nobody did, so River retries.
func (w *EscalationWorker) push(ctx context.Context, d db.GetEscalationDetailRow, phones []db.Device, log *slog.Logger) error {
	msg := EscalationMessage(d)
	var (
		sent    []Target
		lastErr error
	)
	for _, p := range phones {
		err := w.Sender.Send(ctx, *p.FcmToken, msg)
		switch {
		case err == nil:
			sent = append(sent, Target{DeviceID: &p.ID, SentAt: w.now()})
		case errors.Is(err, ErrUnavailable):
			log.WarnContext(ctx, "fcm not configured: escalation alert not pushed")
			return nil
		case errors.Is(err, ErrUnregistered):
			log.InfoContext(ctx, "fcm token unregistered, revoking device", "device_id", p.ID)
			if err := w.Queries.RevokeDeviceByFcmToken(ctx, p.FcmToken); err != nil {
				log.ErrorContext(ctx, "revoke device failed", "device_id", p.ID, "err", err)
			}
		default:
			log.WarnContext(ctx, "fcm send failed", "device_id", p.ID, "err", err)
			lastErr = err
		}
	}
	if len(sent) > 0 {
		targets, err := json.Marshal(sent)
		if err != nil {
			return err
		}
		if err := w.Queries.SetEscalationNotified(ctx, db.SetEscalationNotifiedParams{
			ID: d.EscalationEvent.ID, NotifiedTargets: targets,
		}); err != nil {
			// The push went out; only the record is missing. Retrying would
			// push again, so log instead.
			log.ErrorContext(ctx, "record notified targets failed", "err", err)
		}
		log.InfoContext(ctx, "escalation alert sent", "devices", len(sent))
		return nil
	}
	// Nobody got it: retry (River backs off) unless every token was dead.
	return lastErr
}

// alertGuardian pushes an escalation from a companion session to the
// guardian's phones.
func (w *EscalationWorker) alertGuardian(ctx context.Context, d db.GetEscalationDetailRow, log *slog.Logger) error {
	elder, err := w.Queries.GetElder(ctx, d.ElderID)
	if err != nil {
		return fmt.Errorf("read elder: %w", err)
	}
	phones, err := w.Queries.ListGuardianPhones(ctx, &elder.GuardianID)
	if err != nil {
		return fmt.Errorf("list guardian phones: %w", err)
	}
	if len(phones) == 0 {
		log.WarnContext(ctx, "guardian has no phone registered for alerts", "guardian_id", elder.GuardianID)
		return nil
	}
	return w.push(ctx, d, phones, log)
}

func (w *EscalationWorker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// EscalationMessage is the push for an escalation: who, what kind, and what
// they said. Data carries the ids the app opens the alert screen with.
func EscalationMessage(d db.GetEscalationDetailRow) Message {
	ev := d.EscalationEvent
	label, ok := TriggerLabels[ev.TriggerType]
	if !ok {
		label = TriggerLabels["OTHER_ANOMALY"]
	}
	body := label
	switch {
	case d.UtteranceText != nil && *d.UtteranceText != "":
		body += ": \u201c" + *d.UtteranceText + "\u201d"
	case ev.Reason != nil && *ev.Reason != "":
		body += ": " + *ev.Reason
	}
	data := map[string]string{
		"type":         "escalation",
		"escalationId": ev.ID.String(),
		"sessionId":    ev.SessionID.String(),
	}
	if d.VisitID != nil {
		data["visitId"] = d.VisitID.String()
	}
	return Message{Title: "위급 알림: " + d.ElderName + " 어르신", Body: body, Data: data}
}
