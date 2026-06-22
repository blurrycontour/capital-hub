// Package web embeds the compiled SvelteKit frontend and serves it as a SPA.
package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Assets returns the embedded frontend build as a filesystem rooted at the
// SPA's static files.
func Assets() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// SPAHandler serves static assets from the given filesystem, falling back to
// index.html for client-side routes (paths without a file extension).
func SPAHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" {
			upath = "index.html"
		}

		if f, err := assets.Open(upath); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Unknown path with no extension -> let the SPA router handle it.
		if path.Ext(upath) == "" {
			serveIndex(w, r, assets)
			return
		}

		http.NotFound(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, assets fs.FS) {
	f, err := assets.Open("index.html")
	if err != nil {
		http.Error(w, "frontend not built", http.StatusNotFound)
		return
	}
	defer f.Close()

	seeker, ok := f.(io.ReadSeeker)
	if !ok {
		http.Error(w, "frontend not seekable", http.StatusInternalServerError)
		return
	}
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "frontend not built", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", stat.ModTime(), seeker)
}
