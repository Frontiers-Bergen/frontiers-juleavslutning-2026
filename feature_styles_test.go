package main

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestFeatureStyles_EveryCSSFileNextToATemplFile_IsServed(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given feature CSS files next to their .templ files anywhere in the repo.",
		When:  "When a browser loads the feature stylesheet.",
		Then:  "Then every one of them is imported into the features layer and served as CSS.",
	})

	// Given
	expectedFiles := featureCSSFilesInRepo(t)
	app := startTestApp(t)
	browser := app.newAttendee()

	// When
	importStatus, _, importList := app.get(browser, "/feature-styles.css")

	// Then
	if importStatus != http.StatusOK {
		t.Fatalf("expected /feature-styles.css to return %d, got %d", http.StatusOK, importStatus)
	}
	for _, path := range expectedFiles {
		url := featureStylesURLPrefix + path
		if !strings.Contains(importList, `@import url("`+url+`") layer(features);`) {
			t.Errorf("%s is not imported; add its folder to the go:embed patterns in feature_styles.go\nimport list:\n%s", path, importList)
			continue
		}
		status, contentType, _ := app.get(browser, url)
		if status != http.StatusOK || !strings.HasPrefix(contentType, "text/css") {
			t.Errorf("%s: expected %d text/css, got %d %q", url, http.StatusOK, status, contentType)
		}
	}
}

func TestFeatureStyles_SourceFiles_AreNotServed(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given feature CSS is served from the same folders as the source code.",
		When:  "When someone requests a source file through the feature styles URL.",
		Then:  "Then it is not found; only CSS files are ever served.",
	})

	// Given
	expectedStatus := http.StatusNotFound
	app := startTestApp(t)
	stranger := app.newAttendee()

	for _, sourcePath := range []string{"pages/root/poll.templ", "pages/root/poll_templ.go", "main.go", ".env", "schema.sql"} {
		// When
		actualStatus, _, _ := app.get(stranger, featureStylesURLPrefix+sourcePath)

		// Then
		if actualStatus != expectedStatus {
			t.Errorf("%s: expected status %d, got %d", sourcePath, expectedStatus, actualStatus)
		}
	}
}

// featureCSSFilesInRepo lists every .css file next to the source code, the way go:embed paths look.
// static/ holds the shared index.css, which is linked on its own.
func featureCSSFilesInRepo(t *testing.T) []string {
	t.Helper()
	notFeatureCode := map[string]bool{".git": true, "static": true, "tmp": true, "bin": true, "node_modules": true}
	files := []string{}
	err := fs.WalkDir(os.DirFS("."), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && notFeatureCode[path] {
			return fs.SkipDir
		}
		if !entry.IsDir() && filepath.Ext(path) == ".css" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list CSS files in repo: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("found no feature CSS files; is the test running from the repo root?")
	}
	return files
}

// get returns the status, content type and body of a GET request.
func (app *testApp) get(client *http.Client, path string) (status int, contentType string, body string) {
	app.t.Helper()
	response, err := client.Get(app.server.URL + path)
	if err != nil {
		app.t.Fatalf("GET %s failed: %v", path, err)
	}
	defer closeBody(app.t, response)
	content, err := io.ReadAll(response.Body)
	if err != nil {
		app.t.Fatalf("failed to read %s: %v", path, err)
	}
	return response.StatusCode, response.Header.Get("Content-Type"), string(content)
}
