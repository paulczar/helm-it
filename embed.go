//go:build embed
// +build embed

package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:static
var staticFS embed.FS

func fileServer() http.Handler {
	// Use a sub-filesystem to remove the "static" prefix from the path.
	fsys, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create sub filesystem: %v", err)
	}
	return http.FileServer(http.FS(fsys))
}
