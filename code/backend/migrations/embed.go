// Package migrations carries the schema's SQL inside the binary, so the image
// needs nothing but /app/server to bring an empty database up to date.
package migrations

import "embed"

// FS holds every file of this directory. db.Migrate applies the *.up.sql files
// in filename order; the README and the Go files ride along and are ignored.
//
//go:embed *
var FS embed.FS
