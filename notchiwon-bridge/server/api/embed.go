// Package api embeds the contract files shared with the Flutter apps:
// the REST spec (openapi.yaml) and the /ws/elder message schema.
package api

import _ "embed"

// OpenAPI is openapi.yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte

// WSEventsSchema is ws-events.schema.json, the JSON Schema of every
// /ws/elder text frame.
//
//go:embed ws-events.schema.json
var WSEventsSchema []byte
