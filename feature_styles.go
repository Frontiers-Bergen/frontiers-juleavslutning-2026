package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Feature CSS lives in a .css file next to the feature's .templ file (pages/root/poll.css next to poll.templ),
// so editors give full CSS tooling and the browser inspector shows which file a rule comes from.
//
// The files are embedded into the binary and served under /feature-styles/<path in repo>. Only files
// matching the patterns below are embedded, so no source code is ever served. /feature-styles.css is
// generated at startup: one @import per file, each into `layer(features)`, so the feature files do not
// wrap themselves in @layer. The base layout links it once in <head>; live updates never touch <head>,
// so slides do not flash unstyled when they first appear.
//
// Embed patterns have no "**": a new folder with feature CSS needs its own pattern here.
// TestFeatureStyles_EveryCSSFileNextToATemplFile_IsServed fails when one is missing.
//
//go:embed pages/*.css pages/root/*.css slides/*.css
var featureStyleFiles embed.FS

const featureStylesURLPrefix = "/feature-styles/"

func featureStylesRoutes(router chi.Router) error {
	importList, err := featureStylesImportList(featureStyleFiles)
	if err != nil {
		return err
	}

	// no-cache: browsers revalidate on every load, so a deploy is visible at once. The files are a few KB.
	router.Get("/feature-styles.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write([]byte(importList)); err != nil {
			return // the client went away
		}
	})

	files := http.StripPrefix(featureStylesURLPrefix, http.FileServerFS(featureStyleFiles))
	router.Get(featureStylesURLPrefix+"*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
	return nil
}

func featureStylesImportList(files fs.FS) (string, error) {
	var importList strings.Builder
	err := fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".css") {
			return nil
		}
		fmt.Fprintf(&importList, "@import url(%q) layer(features);\n", featureStylesURLPrefix+path)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to list feature styles: %w", err)
	}
	return importList.String(), nil
}
