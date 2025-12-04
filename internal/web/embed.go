package web

import (
	"embed"
	"io/fs"
)

//go:embed *.html
var files embed.FS

// FS provides access to embedded web files
var FS fs.FS = files

