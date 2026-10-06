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

// Target is one entry of escalation_event.notified_targets.
type Target struct {
	DeviceID uuid.UUID `json:"deviceId"`
	SentAt   time.Time `json:"sentAt"`
}

// EscalationWorker pushes an escalation to the caregiver of the visit whose
// pickup session raised it.
//
// Escalations in a companion session have no caregiver; the guardian alert
// for those comes in stage 6, and until then they are only recorded.
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
		log.WarnContext(ctx, "escalation in a companion session: guardian alerts are not built yet")
		return nil
	}

	phones, err := w.Queries.ListCaregiverPhones(ctx, d.CaregiverID)
	if err != nil {
		return fmt.Errorf("list caregiver phones: %w", err)
	}
	if len(phones) == 0 {
		log.WarnContext(ctx, "caregiver has no phone registered for alerts", "caregiver_id", *d.CaregiverID)
		return nil
	}

	msg := EscalationMessage(d)
	var (
		sent    []Target
		lastErr error
	)
	for _, p := range phones {
		err := w.Sender.Send(ctx, *p.FcmToken, msg)
		switch {
		case err == nil:
			sent = append(sent, Target{DeviceID: p.ID, SentAt: w.now()})
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
		if err := w.Queries.SetEscalationNotified(ctx, db.SetEscalationNotifiedParams{ID: id, NotifiedTargets: targets}); err != nil {
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
