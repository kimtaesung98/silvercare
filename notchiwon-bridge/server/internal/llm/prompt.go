package llm

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"strings"
	"text/template"
)

//go:embed prompts/*.tmpl
var promptFiles embed.FS

var templates = template.Must(template.New("").
	Funcs(template.FuncMap{"join": strings.Join}).
	ParseFS(promptFiles, "prompts/*.tmpl"))

// PromptVersion identifies the prompt templates. It is a hash of the files, so
// any edit changes it; sessions record it in conversation_session.prompt_version.
var PromptVersion = func() string {
	h := sha256.New()
	_ = fs.WalkDir(promptFiles, "prompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := promptFiles.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	return "conv-" + hex.EncodeToString(h.Sum(nil))[:10]
}()

func render(name string, data any) string {
	var b bytes.Buffer
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		// The templates are embedded and the data types fixed, so this is a bug.
		panic(err)
	}
	return strings.TrimSpace(b.String())
}

// SystemPrompt returns the system prompt blocks for a session. The first block
// depends only on the mode, so it is the same for every elder and is cached;
// the second holds the elder's name and top interest keywords.
func SystemPrompt(mode, elderName string, keywords []string) []string {
	return []string{
		render("system.tmpl", struct{ Mode string }{mode}),
		render("profile.tmpl", struct {
			ElderName string
			Keywords  []string
		}{elderName, keywords}),
	}
}

// Kickoff is the user message placed before the AI greeting that opens every
// session (docs/specs/conversation-loop.md section 2).
func Kickoff() string { return render("kickoff.tmpl", nil) }

// Continuation tells Claude that sentence 0 (opener) was already spoken, so
// its reply starts at sentence 1.
func Continuation(opener string) string {
	return render("continue.tmpl", struct{ Opener string }{opener})
}
