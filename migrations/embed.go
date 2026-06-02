package migrations

import "embed"

// FS содержит все SQL-файлы миграций из текущего каталога.
//
//go:embed *.sql
var FS embed.FS
