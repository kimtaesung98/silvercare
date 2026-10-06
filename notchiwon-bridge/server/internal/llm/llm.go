// Package llm calls Claude for the conversation and owns its prompt templates.
// Claude is called only from the server: the tablet never holds an API key.
package llm

import "context"

// Role is a message role.
type Role string

// Roles.
const (
	User      Role = "user"
	Assistant Role = "assistant"
)

// Message is one conversation turn.
type Message struct {
	Role Role
	Text string
}

// Request is one conversation call.
type Request struct {
	// System blocks; the first one is cached.
	System []string
	// Messages alternate user/assistant and start and end with user.
	Messages []Message
	// Continuation, when set, is appended to the last user message as its own
	// text block (see Continuation).
	Continuation string
}

// Concern is a flag_concern tool call: a worrying sign the keyword rules may
// have missed.
type Concern struct {
	Type   string
	Reason string
}

// Result is what a call returned besides the streamed text.
type Result struct {
	Model        string
	InputTokens  int
	OutputTokens int
	Concerns     []Concern
	// Refused is true when Claude declined (stop_reason "refusal"); the text
	// streamed so far should not be trusted as a complete answer.
	Refused bool
}

// Client streams one reply. onText receives the text as it arrives, from the
// calling goroutine. A cancelled ctx stops the stream.
type Client interface {
	Stream(ctx context.Context, req Request, onText func(string)) (Result, error)
}
