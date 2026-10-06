// Package migrations embeds the goose SQL migrations so the server and tests
// can apply them without reading the file system.
package migrations

import "embed"

// FS holds every *.sql migration in this directory.
//
//go:embed *.sql
var FS embed.FS
