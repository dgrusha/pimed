// Package web embeds the built SvelteKit SPA (backend/web/dist/, produced by
// the frontend build). The pattern also matches this file, so the package
// compiles even before the frontend has been built.
package web

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:*
var files embed.FS

// Dist returns the SPA build output rooted at dist/. It may be empty if the
// frontend has not been built.
func Dist() (fs.FS, error) {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		return nil, fmt.Errorf("web: sub dist: %w", err)
	}
	return sub, nil
}
