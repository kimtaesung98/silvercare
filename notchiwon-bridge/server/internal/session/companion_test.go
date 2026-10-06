package session

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/llm"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// companionSession ends the fixture's pickup session and opens a companion
// one for the same elder.
func companionSession(t *testing.T, e *env) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	if _, err := e.q.EndSession(ctx, db.EndSessionParams{ID: e.session, EndedReason: ptr("CAREGIVER_ARRIVED")}); err != nil {
		t.Fatal(err)
	}
	sess, err := e.q.CreateCompanionSession(ctx, db.CreateCompanionSessionParams{ElderID: e.elderID, StartedBy: "ELDER"})
	if err != nil {
		t.Fatal(err)
	}
	e.session = sess.ID
	return sess.ID
}

// fakeBudget answers the daily token question.
type fakeBudget struct {
	over bool
	err  error
}

func (b fakeBudget) Exhausted(context.Context, uuid.UUID) (bool, error) { return b.over, b.err }

// fakeCompactor records the summaries the engine asked for.
type fakeCompactor struct{ upTo []int32 }

func (c *fakeCompactor) Compact(_ context.Context, _ uuid.UUID, upTo int32) {
	c.upTo = append(c.upTo, upTo)
}

func TestCompanionGoodbyeEndsTheTalk(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	id := companionSession(t, e)

	res, err := e.engine.HandleElderText(context.Background(), id, "이제 그만할게, 고마워", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res.End != EndElderDeclined {
		t.Fatalf("end = %q", res.End)
	}
	if len(res.Replies) == 0 || res.Replies[0].Text != GoodbyeSentence {
		t.Fatalf("replies = %+v", res.Replies)
	}
	if n := len(e.llm.requests()); n != 0 {
		t.Errorf("asked Claude %d times to say goodbye", n)
	}
	// The elder's line and the goodbye are both saved.
	us := e.utterances(t)
	if len(us) != 2 || us[0].Speaker != "ELDER" || us[1].Text != GoodbyeSentence {
		t.Errorf("utterances = %+v", us)
	}
}

func TestCompanionKeepsTalkingOtherwise(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	id := companionSession(t, e)

	res, err := e.engine.HandleElderText(context.Background(), id, "손주가 그만 간다고 하네", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res.End != "" {
		t.Errorf("end = %q on a line that only mentions stopping", res.End)
	}
	if len(e.llm.requests()) != 1 {
		t.Errorf("Claude calls = %d", len(e.llm.requests()))
	}
}

func TestCompanionTokenLimitEndsTheTalk(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	id := companionSession(t, e)
	e.engine.SetBudget(fakeBudget{over: true})

	res, err := e.engine.HandleElderText(context.Background(), id, "오늘 날씨가 좋네", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res.End != EndTokenLimit || res.Replies[0].Text != TokenLimitSentence {
		t.Fatalf("end = %q, replies = %+v", res.End, res.Replies)
	}
	if n := len(e.llm.requests()); n != 0 {
		t.Errorf("spent %d more calls past the limit", n)
	}
}

func TestCompanionTalksOnWhenTheBudgetCannotBeRead(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	id := companionSession(t, e)
	e.engine.SetBudget(fakeBudget{err: context.DeadlineExceeded})

	res, err := e.engine.HandleElderText(context.Background(), id, "오늘 날씨가 좋네", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if res.End != "" {
		t.Errorf("end = %q: a read error should not cut the elder off", res.End)
	}
}

func TestPickupModeIgnoresGoodbyeAndBudget(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	e.engine.SetBudget(fakeBudget{over: true})

	res, err := e.engine.HandleElderText(context.Background(), e.session, "이제 그만할게", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	// The pickup bridge runs until the caregiver arrives, whatever is said.
	if res.End != "" || len(e.llm.requests()) != 1 {
		t.Errorf("end = %q, calls = %d", res.End, len(e.llm.requests()))
	}
}

func TestLongTalkIsSummarizedAndTrimmed(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	id := companionSession(t, e)
	c := &fakeCompactor{}
	e.engine.SetCompactor(c)
	ctx := context.Background()

	// A short talk is not summarized.
	for seq := int32(0); seq < 10; seq++ {
		e.say(t, seq, "ELDER", "짧은 이야기")
	}
	if _, err := e.engine.HandleElderText(ctx, id, "아직 짧네", newRecSink()); err != nil {
		t.Fatal(err)
	}
	if len(c.upTo) != 0 {
		t.Fatalf("summarized a short talk: %v", c.upTo)
	}

	// Past compactAfter unsummarized seqs, everything but the last keepSeqs goes.
	for seq := int32(12); seq < 12+compactAfter; seq++ {
		e.say(t, seq, "ELDER", "계속 이야기")
	}
	res, err := e.engine.HandleElderText(ctx, id, "이야기가 길어졌네", newRecSink())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.upTo) != 1 || c.upTo[0] != res.Turn-keepSeqs {
		t.Fatalf("upTo = %v, want one at %d", c.upTo, res.Turn-keepSeqs)
	}
}

func TestSummaryReplacesTheEarlyTurns(t *testing.T) {
	e := newEnv(t, DefaultConfig)
	id := companionSession(t, e)
	ctx := context.Background()

	for seq := int32(0); seq < 6; seq++ {
		e.say(t, seq, "ELDER", "예전 이야기")
	}
	summary := "큰아들과 손녀 이야기를 나누셨다."
	if err := e.q.SetHistorySummary(ctx, db.SetHistorySummaryParams{
		ID: id, Summary: &summary, Through: ptr(int32(3)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.engine.HandleElderText(ctx, id, "새 이야기", newRecSink()); err != nil {
		t.Fatal(err)
	}
	req := e.llm.requests()[0]
	if !strings.Contains(strings.Join(req.System, "\n"), summary) {
		t.Fatalf("summary not in the system prompt: %+v", req.System)
	}
	// Only the seqs after the summary travel: two of the six old lines plus
	// the new one, merged into one user turn.
	var elder []string
	for _, m := range req.Messages {
		if m.Role == llm.User {
			elder = append(elder, m.Text)
		}
	}
	joined := strings.Join(elder, "|")
	if strings.Count(joined, "예전 이야기") != 2 {
		t.Errorf("old lines sent = %d, want the two after the summary: %q", strings.Count(joined, "예전 이야기"), joined)
	}
	if !strings.Contains(joined, "새 이야기") {
		t.Errorf("new line missing: %q", joined)
	}
}
