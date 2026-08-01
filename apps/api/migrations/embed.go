package migrations

import "embed"

// Files contains the ordered SQL migrations shipped with the API binary.
//
//go:embed *.sql
var Files embed.FS
