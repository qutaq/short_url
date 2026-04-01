package migrations

import "embed"

// FS contains all SQL migration files from this directory.
//
//go:embed *.sql
var FS embed.FS
