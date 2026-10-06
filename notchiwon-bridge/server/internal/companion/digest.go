package companion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/notify"
)

// NoTalkSummary is the digest of a day with no companion conversation.
const NoTalkSummary = "오늘은 말동무 대화가 없었어요."

// quotedPrefix starts the digest when Claude is unavailable: the guardian
// gets the elder's own words instead of nothing.
const (
	quotedPrefix = "AI 요약을 만들지 못해 어르신 말씀을 그대로 옮겨요: "
	maxQuoted    = 5
)

// DigestWorker writes one elder's daily digest and pushes it to the
// guardian's phones (jobs.DailyDigestArgs). The row is unique per elder and
// day, so a retry after the push went out neither rewrites it nor notifies
// again.
type DigestWorker struct {
	river.WorkerDefaults[jobs.DailyDigestArgs]
	Queries  *db.Queries
	Digester llm.Digester
	Sender   notify.Sender
	Policy   Policy
	Logger   *slog.Logger
}

// Timeout bounds one digest (a Haiku call, a push and two writes).
func (w *DigestWorker) Timeout(*river.Job[jobs.DailyDigestArgs]) time.Duration {
	return time.Minute
}

// Work implements river.Worker.
func (w *DigestWorker) Work(ctx context.Context, job *river.Job[jobs.DailyDigestArgs]) error {
	elderID, date := job.Args.ElderID, job.Args.Date
	log := w.Logger.With("elder_id", elderID, "date", date)
	g, err := w.Queries.GetElderGuardian(ctx, elderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(fmt.Errorf("elder %s has no guardian", elderID))
	}
	if err != nil {
		return fmt.Errorf("read guardian: %w", err)
	}
	if existing, err := w.Queries.GetDailyDigest(ctx, db.GetDailyDigestParams{
		ElderID: elderID, DigestDate: mustDate(date),
	}); err == nil {
		// Written by an earlier attempt: only the text may still be owed.
		return w.send(ctx, existing, g, log)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read digest: %w", err)
	}

	s, err := w.Policy.Schedule(ctx, elderID)
	if err != nil {
		return err
	}
	day, err := time.ParseInLocation(time.DateOnly, date, s.Location)
	if err != nil {
		return river.JobCancel(fmt.Errorf("date %q: %w", date, err))
	}
	next := day.AddDate(0, 0, 1)

	sessions, err := w.Queries.ListElderSessionsBetween(ctx, db.ListElderSessionsBetweenParams{
		ElderID: elderID, FromTime: day, ToTime: next,
	})
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	escs, err := w.Queries.ListElderEscalationsBetween(ctx, db.ListElderEscalationsBetweenParams{
		ElderID: elderID, FromTime: day, ToTime: next,
	})
	if err != nil {
		return fmt.Errorf("list escalations: %w", err)
	}

	in, tokens, quotes, err := w.input(ctx, g.ElderName, date, sessions, escs)
	if err != nil {
		return err
	}
	d := w.digest(ctx, in, escs, quotes)

	row, err := w.Queries.InsertDailyDigest(ctx, db.InsertDailyDigestParams{
		ElderID: elderID, DigestDate: mustDate(date), SummaryText: d.Summary,
		EmotionFlag: d.EmotionFlag, SessionCount: int32(len(sessions)),
		EscalationCount: int32(len(escs)), TotalTokens: int32(tokens), Model: model(d.Model),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // another attempt wrote and sent it
	}
	if err != nil {
		return fmt.Errorf("save digest: %w", err)
	}
	log.InfoContext(ctx, "daily digest written", "sessions", len(sessions), "escalations", len(escs))
	return w.send(ctx, row, g, log)
}

// send pushes the digest to the guardian's phones and marks it sent. A
// guardian with no phone registered reads it in the app instead.
func (w *DigestWorker) send(ctx context.Context, d db.DailyDigest, g db.GetElderGuardianRow, log *slog.Logger) error {
	if d.SentAt != nil {
		return nil
	}
	phones, err := w.Queries.ListGuardianPhones(ctx, &g.GuardianID)
	if err != nil {
		return fmt.Errorf("list guardian phones: %w", err)
	}
	if len(phones) == 0 {
		log.WarnContext(ctx, "guardian has no phone registered: digest written but not pushed")
		return nil
	}
	body := d.SummaryText
	if d.EmotionFlag != nil {
		body += "\n살펴볼 점: " + *d.EmotionFlag
	}
	msg := notify.Message{
		Title:   g.ElderName + " 어르신 하루 소식",
		Body:    body,
		Channel: notify.DigestChannel,
		Data: map[string]string{
			"type":    "digest",
			"elderId": d.ElderID.String(),
			"date":    d.DigestDate.Time.Format(time.DateOnly),
		},
	}
	var (
		sent    int
		lastErr error
	)
	for _, p := range phones {
		switch err := w.sender().Send(ctx, *p.FcmToken, msg); {
		case err == nil:
			sent++
		case errors.Is(err, notify.ErrUnavailable):
			log.WarnContext(ctx, "fcm not configured: digest written but not pushed")
			return nil
		case errors.Is(err, notify.ErrUnregistered):
			if err := w.Queries.RevokeDeviceByFcmToken(ctx, p.FcmToken); err != nil {
				log.ErrorContext(ctx, "revoke device failed", "device_id", p.ID, "err", err)
			}
		default:
			lastErr = err
		}
	}
	if sent == 0 && lastErr != nil {
		return fmt.Errorf("push digest: %w", lastErr) // retried
	}
	if err := w.Queries.MarkDailyDigestSent(ctx, d.ID); err != nil {
		// The push went out; retrying would notify twice, so log instead.
		log.ErrorContext(ctx, "mark digest sent failed", "err", err)
	}
	log.InfoContext(ctx, "daily digest sent", "phones", sent)
	return nil
}

// input gathers the day for Claude: the per-session summaries when they
// exist, else the day's utterances, plus what the elder said for the
// fallback digest.
func (w *DigestWorker) input(
	ctx context.Context,
	elderName, date string,
	sessions []db.ListElderSessionsBetweenRow,
	escs []db.EscalationEvent,
) (llm.DigestInput, int64, []string, error) {
	in := llm.DigestInput{ElderName: elderName, Date: date, SessionCount: len(sessions)}
	var (
		tokens int64
		quotes []string
	)
	for _, s := range sessions {
		if s.SummaryText != nil {
			in.Summaries = append(in.Summaries, *s.SummaryText)
		}
		utts, err := w.Queries.ListSessionUtterances(ctx, s.ConversationSession.ID)
		if err != nil {
			return in, 0, nil, fmt.Errorf("read utterances: %w", err)
		}
		for _, u := range utts {
			if u.InputTokens != nil {
				tokens += int64(*u.InputTokens)
			}
			if u.OutputTokens != nil {
				tokens += int64(*u.OutputTokens)
			}
			elder := u.Speaker == "ELDER"
			if elder {
				quotes = append(quotes, u.Text)
			}
			if s.SummaryText != nil {
				continue
			}
			if n := len(in.Lines); n > 0 && !elder && !in.Lines[n-1].Elder {
				in.Lines[n-1].Text += " " + u.Text
				continue
			}
			in.Lines = append(in.Lines, llm.Line{Elder: elder, Text: u.Text})
		}
	}
	for _, e := range escs {
		label := notify.TriggerLabels[e.TriggerType]
		if e.Reason != nil && *e.Reason != "" {
			label += ": " + *e.Reason
		}
		in.Escalations = append(in.Escalations, label)
	}
	return in, tokens, quotes, nil
}

// digest asks Claude, or writes the digest without it.
func (w *DigestWorker) digest(
	ctx context.Context,
	in llm.DigestInput,
	escs []db.EscalationEvent,
	quotes []string,
) llm.Digest {
	if in.SessionCount == 0 || len(quotes) == 0 {
		return withEscalations(llm.Digest{Summary: NoTalkSummary}, escs)
	}
	d, err := w.Digester.Digest(ctx, in)
	if err == nil {
		return d
	}
	if !errors.Is(err, llm.ErrUnavailable) {
		w.Logger.WarnContext(ctx, "claude digest failed, quoting the elder instead", "err", err)
	}
	if len(quotes) > maxQuoted {
		quotes = quotes[len(quotes)-maxQuoted:]
	}
	return withEscalations(llm.Digest{
		Summary: quotedPrefix + "“" + strings.Join(quotes, "”, “") + "”",
	}, escs)
}

// withEscalations fills the flag of a digest Claude did not write.
func withEscalations(d llm.Digest, escs []db.EscalationEvent) llm.Digest {
	if len(escs) == 0 {
		return d
	}
	labels := make([]string, 0, len(escs))
	for _, e := range escs {
		labels = append(labels, notify.TriggerLabels[e.TriggerType])
	}
	flag := "위급 감지: " + strings.Join(dedupe(labels), ", ") + " (바로 연락드렸어요)"
	d.EmotionFlag = &flag
	return d
}

func dedupe(ss []string) []string {
	seen := map[string]bool{}
	out := ss[:0]
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func (w *DigestWorker) sender() notify.Sender {
	if w.Sender == nil {
		return notify.Unavailable{}
	}
	return w.Sender
}

func model(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// mustDate parses a YYYY-MM-DD the tick wrote; a bad one is a bug.
func mustDate(s string) pgtype.Date {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return pgtype.Date{Time: t, Valid: true}
}
