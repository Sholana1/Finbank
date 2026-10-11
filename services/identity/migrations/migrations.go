// Package migrations embeds the SQL schema files into the migrate binary.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
