// Package briefing writes the arrival briefing of a finished pickup session
// and updates the elder's interest keywords (time-decayed scores).
package briefing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/jobs"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/notify"
)

// Fixed summaries for sessions Claude does not summarize.
const (
	SilentSummary = "어르신이 이번 대기 시간에 말씀을 거의 하지 않으셨음."
	// quotedPrefix starts the summary when Claude is unavailable: the
	// caregiver gets the elder's own words instead of nothing.
	quotedPrefix = "AI 요약을 만들지 못해 어르신 말씀을 그대로 옮김: "
	maxQuoted    = 3
)

// Worker runs jobs.BriefingArgs.
type Worker struct {
	river.WorkerDefaults[jobs.BriefingArgs]
	Pool       *pgxpool.Pool
	Summarizer llm.Summarizer
	Logger     *slog.Logger
	Now        func() time.Time
}

// Timeout bounds one briefing (a Haiku call plus a few writes).
func (w *Worker) Timeout(*river.Job[jobs.BriefingArgs]) time.Duration { return time.Minute }

// Work implements river.Worker. It is idempotent: a briefing that already
// exists is left alone, so a retried job never bumps keywords twice.
func (w *Worker) Work(ctx context.Context, job *river.Job[jobs.BriefingArgs]) error {
	id := job.Args.SessionID
	q := db.New(w.Pool)
	if _, err := q.GetBriefingBySession(ctx, id); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read briefing: %w", err)
	}

	owner, err := q.GetSessionOwner(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(fmt.Errorf("session %s not found", id))
	}
	if err != nil {
		return fmt.Errorf("read session: %w", err)
	}
	sess := owner.ConversationSession
	if sess.EndedAt == nil {
		return fmt.Errorf("session %s has not ended yet", id) // retried later
	}

	utts, err := q.ListSessionUtterances(ctx, id)
	if err != nil {
		return fmt.Errorf("read utterances: %w", err)
	}
	escs, err := q.ListSessionEscalations(ctx, id)
	if err != nil {
		return fmt.Errorf("read escalations: %w", err)
	}

	in := Input(owner.ElderName, utts, escs)
	b, err := w.brief(ctx, in, escs)
	if err != nil {
		return err
	}
	return w.save(ctx, sess, b)
}

// brief asks Claude, or builds a briefing without it.
func (w *Worker) brief(ctx context.Context, in llm.BriefingInput, escs []db.EscalationEvent) (llm.Briefing, error) {
	var quotes []string
	for _, l := range in.Lines {
		if l.Elder {
			quotes = append(quotes, l.Text)
		}
	}
	if len(quotes) == 0 {
		return withEscalations(llm.Briefing{Summary: SilentSummary}, escs), nil
	}
	b, err := w.Summarizer.Brief(ctx, in)
	switch {
	case err == nil:
		return b, nil
	case errors.Is(err, llm.ErrUnavailable):
		if len(quotes) > maxQuoted {
			quotes = quotes[len(quotes)-maxQuoted:]
		}
		return withEscalations(llm.Briefing{Summary: quotedPrefix + "“" + strings.Join(quotes, "”, “") + "”"}, escs), nil
	default:
		return llm.Briefing{}, err // retried; River backs off
	}
}

// withEscalations fills the emotion fields of a briefing Claude did not
// write: any escalation makes the session UNUSUAL.
func withEscalations(b llm.Briefing, escs []db.EscalationEvent) llm.Briefing {
	if len(escs) == 0 {
		return b
	}
	b.EmotionTag = "UNUSUAL"
	labels := make([]string, 0, len(escs))
	for _, e := range escs {
		labels = append(labels, notify.TriggerLabels[e.TriggerType])
	}
	flag := "위급 감지: " + strings.Join(dedupe(labels), ", ")
	b.EmotionFlag = &flag
	return b
}

// save writes the briefing, the session's emotion tag and the keywords in
// one transaction.
func (w *Worker) save(ctx context.Context, sess db.ConversationSession, b llm.Briefing) error {
	names := make([]string, len(b.Keywords))
	for i, k := range b.Keywords {
		names[i] = k.Keyword
	}
	top, err := json.Marshal(names)
	if err != nil {
		return err
	}
	var model *string
	if b.Model != "" {
		model = &b.Model
	}
	mentionedAt := *sess.EndedAt
	err = pgx.BeginFunc(ctx, w.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		// A run that already committed (job retried after commit) wins.
		if _, err := q.GetBriefingBySession(ctx, sess.ID); err == nil {
			return errAlreadyWritten
		}
		if _, err := q.UpsertBriefing(ctx, db.UpsertBriefingParams{
			SessionID: sess.ID, SummaryText: b.Summary, TopKeywords: top,
			EmotionFlag: b.EmotionFlag, Model: model,
		}); err != nil {
			return fmt.Errorf("save briefing: %w", err)
		}
		if b.EmotionTag != "" {
			if err := q.SetSessionEmotionTag(ctx, db.SetSessionEmotionTagParams{ID: sess.ID, Tag: &b.EmotionTag}); err != nil {
				return fmt.Errorf("save emotion tag: %w", err)
			}
		}
		for _, k := range b.Keywords {
			if _, err := q.BumpKeyword(ctx, db.BumpKeywordParams{
				ElderID: sess.ElderID, Keyword: k.Keyword, Category: k.Category,
				Mentions: int32(k.Mentions), EmotionTone: k.EmotionTone, MentionedAt: mentionedAt,
			}); err != nil {
				return fmt.Errorf("bump keyword %q: %w", k.Keyword, err)
			}
		}
		return nil
	})
	if errors.Is(err, errAlreadyWritten) {
		return nil
	}
	if err != nil {
		return err
	}
	w.Logger.InfoContext(ctx, "briefing written", "session_id", sess.ID, "keywords", len(b.Keywords),
		"emotion_tag", b.EmotionTag, "model", b.Model, "prompt_version", llm.BriefingPromptVersion)
	return nil
}

var errAlreadyWritten = errors.New("briefing already written")

// Input turns a session's records into the summarizer's input. AI sentences
// of one turn are joined, like the conversation history.
func Input(elderName string, utts []db.Utterance, escs []db.EscalationEvent) llm.BriefingInput {
	in := llm.BriefingInput{ElderName: elderName}
	for _, u := range utts {
		elder := u.Speaker == "ELDER"
		if n := len(in.Lines); n > 0 && !elder && !in.Lines[n-1].Elder {
			in.Lines[n-1].Text += " " + u.Text
			continue
		}
		in.Lines = append(in.Lines, llm.Line{Elder: elder, Text: u.Text})
	}
	byID := map[string]string{}
	for _, u := range utts {
		byID[u.ID.String()] = u.Text
	}
	for _, e := range escs {
		s := notify.TriggerLabels[e.TriggerType]
		switch {
		case e.UtteranceID != nil && byID[e.UtteranceID.String()] != "":
			s += ": " + byID[e.UtteranceID.String()]
		case e.Reason != nil:
			s += ": " + *e.Reason
		}
		in.Escalations = append(in.Escalations, s)
	}
	return in
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	out := s[:0]
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
