// Package migrations embeds all SQL migration files for use with goose.
package migrations

import "embed"

// FS holds all migration SQL files embedded at compile time.
//
//go:embed *.sql
var FS embed.FS
