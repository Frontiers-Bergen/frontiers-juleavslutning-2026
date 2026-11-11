package slides

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// Content slides are markdown files in this folder, one per slide (welcome.md). The deck order lives in
// pages/root/active_slide.templ; a markdown slide is added there with MustMarkdownSlide("welcome").
//
// File format:
//   - The first "# heading" is the slide title, shown on the slide and in the presenter's slide list.
//   - Everything after a line containing only "Notes:" is presenter notes (the reveal.js convention).
//   - An optional line "Illustration: /static/illustrations/smash-button.svg" gives the slide a picture: beside the
//     text on the big screen, under it on a phone. The picture is decoration (alt=""): the text carries the
//     message. Its size can be changed in the slide's CSS (.markdown-slide-illustration).
//   - Optional per-slide CSS: welcome.css next to welcome.md, nested under .slide-welcome.
//
// The markdown is rendered once at startup. Raw HTML in markdown is allowed and not sanitized: slides are
// written by us and reviewed in pull requests, never by attendees at runtime.
//
// Interactive slides (polls, games) stay templ components.

//go:embed *.md
var markdownFiles embed.FS

const (
	presenterNotesMarker = "Notes:"
	illustrationMarker   = "Illustration:"
)

type MarkdownSlide struct {
	// Name is the file name without .md; the slide gets the CSS class "slide-<Name>".
	Name        string
	Title       string
	ContentHTML string
	NotesHTML   string
	// Illustration is the URL of the slide's picture, or "" for a slide without one.
	Illustration string
}

// File is where the slide comes from, relative to the repository root; the presenter view shows it.
func (slide MarkdownSlide) File() string {
	return "slides/" + slide.Name + ".md"
}

var markdownSlides = mustLoadMarkdownSlides(markdownFiles)

// MustMarkdownSlide returns the slide from <name>.md. It panics for an unknown name, like MustPoll:
// a typo in the deck is a programming error, and every test that renders the deck catches it.
func MustMarkdownSlide(name string) MarkdownSlide {
	slide, found := markdownSlides[name]
	if !found {
		panic(fmt.Sprintf("unknown markdown slide %q: add slides/%s.md", name, name))
	}
	return slide
}

func mustLoadMarkdownSlides(files fs.FS) map[string]MarkdownSlide {
	slidesByName, err := loadMarkdownSlides(files)
	if err != nil {
		panic(err)
	}
	return slidesByName
}

func loadMarkdownSlides(files fs.FS) (map[string]MarkdownSlide, error) {
	paths, err := fs.Glob(files, "*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to list markdown slides: %w", err)
	}
	slidesByName := map[string]MarkdownSlide{}
	for _, path := range paths {
		source, err := fs.ReadFile(files, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		name := strings.TrimSuffix(path, ".md")
		slide, err := parseMarkdownSlide(name, string(source))
		if err != nil {
			return nil, fmt.Errorf("slides/%s: %w", path, err)
		}
		slidesByName[name] = slide
	}
	return slidesByName, nil
}

func parseMarkdownSlide(name string, source string) (MarkdownSlide, error) {
	content, notes := splitPresenterNotes(source)
	content, illustration, err := splitIllustration(content)
	if err != nil {
		return MarkdownSlide{}, err
	}
	title, err := slideTitle(content)
	if err != nil {
		return MarkdownSlide{}, err
	}
	return MarkdownSlide{
		Name:         name,
		Title:        title,
		ContentHTML:  markdownToHTML(content),
		NotesHTML:    markdownToHTML(notes),
		Illustration: illustration,
	}, nil
}

// splitIllustration takes the "Illustration:" line out of the content, so it is not shown as text.
func splitIllustration(content string) (rest string, illustration string, err error) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		value, isIllustration := strings.CutPrefix(strings.TrimSpace(line), illustrationMarker)
		if !isIllustration {
			continue
		}
		illustration = strings.TrimSpace(value)
		if illustration == "" {
			return "", "", errors.New(`"Illustration:" needs the picture's URL, like /static/illustrations/smash-button.svg`)
		}
		return strings.Join(append(lines[:i:i], lines[i+1:]...), "\n"), illustration, nil
	}
	return content, "", nil
}

func splitPresenterNotes(source string) (content string, notes string) {
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == presenterNotesMarker {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	return source, ""
}

func slideTitle(content string) (string, error) {
	for line := range strings.SplitSeq(content, "\n") {
		if title, isHeading := strings.CutPrefix(strings.TrimSpace(line), "# "); isHeading {
			return strings.TrimSpace(title), nil
		}
	}
	return "", errors.New(`a slide needs a "# heading"; it is the slide title`)
}

func markdownToHTML(source string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	markdownParser := parser.NewWithExtensions(parser.CommonExtensions | parser.NoEmptyLineBeforeBlock)
	renderer := html.NewRenderer(html.RendererOptions{Flags: html.CommonFlags | html.HrefTargetBlank})
	return string(markdown.ToHTML([]byte(source), markdownParser, renderer))
}
