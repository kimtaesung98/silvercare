package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/api"
)

func compileSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(api.WSEventsSchema))
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource("ws-events.schema.json", doc); err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile("ws-events.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return s
}

// eventTypes lists the message types the schema defines, read from its top-level oneOf.
func eventTypes(t *testing.T) []string {
	t.Helper()
	var doc struct {
		OneOf []struct {
			Ref string `json:"$ref"`
		} `json:"oneOf"`
	}
	if err := json.Unmarshal(api.WSEventsSchema, &doc); err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, o := range doc.OneOf {
		types = append(types, strings.TrimPrefix(o.Ref, "#/$defs/"))
	}
	return types
}

func TestWSEventExamplesMatchSchema(t *testing.T) {
	schema := compileSchema(t)

	raw, err := os.ReadFile("ws-events.examples.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples []json.RawMessage
	if err := json.Unmarshal(raw, &examples); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for i, ex := range examples {
		v, err := jsonschema.UnmarshalJSON(bytes.NewReader(ex))
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(v); err != nil {
			t.Errorf("example %d %s: %v", i, ex, err)
		}
		var head struct{ Type string }
		_ = json.Unmarshal(ex, &head)
		seen[head.Type] = true
	}

	// Every event in the contract has at least one example, so ws-events.md stays honest.
	for _, typ := range eventTypes(t) {
		if !seen[typ] {
			t.Errorf("no example for %q in ws-events.examples.json", typ)
		}
	}
}

func TestWSEventsDocumentedInMarkdown(t *testing.T) {
	md, err := os.ReadFile("ws-events.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range eventTypes(t) {
		if !bytes.Contains(md, []byte("`"+typ+"`")) {
			t.Errorf("ws-events.md does not mention `%s`", typ)
		}
	}
}

func TestWSEventSchemaRejects(t *testing.T) {
	schema := compileSchema(t)
	cases := map[string]string{
		"unknown type":         `{"type":"elder.shout","data":{}}`,
		"missing data":         `{"type":"ping"}`,
		"unknown field":        `{"type":"ping","data":{"nonce":"1","extra":true}}`,
		"unknown top field":    `{"type":"ping","data":{"nonce":"1"},"sessionId":"x"}`,
		"reply index 0":        `{"type":"ai.reply","data":{"sessionId":"9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d","turnId":2,"index":0,"utteranceId":"4d5e6f7a-8b9c-4d0e-9f1a-3b4c5d6e7f8a","text":"네","audio":null}}`,
		"bad uuid":             `{"type":"session.end","data":{"sessionId":"not-a-uuid"}}`,
		"pickup request":       `{"type":"session.request","data":{"mode":"PICKUP_BRIDGE"}}`,
		"audio without length": `{"type":"elder.audio","data":{"sessionId":"9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d","clientId":"u","audio":{"format":"pcm16"}}}`,
		"filler as opener":     `{"type":"ai.opener","data":{"sessionId":"9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d","turnId":1,"clipId":"2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e","category":"FILLER"}}`,
	}
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			v, err := jsonschema.UnmarshalJSON(strings.NewReader(cases[name]))
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(v); err == nil {
				t.Errorf("schema accepted %s", cases[name])
			}
		})
	}
}
