// Package migrations embeds the goose SQL migration files so they can be applied
// programmatically (by cmd/migrate and the e2e tests) without shipping loose
// files alongside the binary.
package migrations

import "embed"

// FS holds every *.sql migration in this directory.
//
//go:embed *.sql
var FS embed.FS
