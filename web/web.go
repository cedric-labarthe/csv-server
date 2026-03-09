package web

import "embed"

// Static holds all files under web/static/, served at /static/.
//
//go:embed static
var Static embed.FS
