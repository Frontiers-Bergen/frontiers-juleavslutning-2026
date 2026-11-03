package slides

import (
	"strings"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
)

func TestMarkdownSlide_HeadingAndNotes_BecomeTitleAndPresenterNotes(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a markdown slide with a heading, some content and a Notes: section.",
		When:  "When the slide is parsed.",
		Then:  "Then the heading is the title, the content is on the slide and the notes are kept apart for the presenter.",
	})

	// Given
	expectedTitle := "Every smash counts"
	source := "# Every smash counts\n\n- No debounce\n\nNotes:\n\n- Tell the retro tool story\n"

	// When
	slide, err := parseMarkdownSlide("smash", source)

	// Then
	if err != nil {
		t.Fatalf("expected the slide to parse, got %v", err)
	}
	if slide.Title != expectedTitle {
		t.Fatalf("expected title %q, got %q", expectedTitle, slide.Title)
	}
	if !strings.Contains(slide.ContentHTML, "<li>No debounce</li>") || strings.Contains(slide.ContentHTML, "retro tool") {
		t.Fatalf("content should hold the list and not the notes:\n%s", slide.ContentHTML)
	}
	if !strings.Contains(slide.NotesHTML, "<li>Tell the retro tool story</li>") {
		t.Fatalf("notes should hold the presenter notes:\n%s", slide.NotesHTML)
	}
}

func TestMarkdownSlide_WithoutHeading_IsRejected(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a markdown slide without a # heading.",
		When:  "When the slide is parsed.",
		Then:  "Then it is rejected, because the presenter's slide list needs a title.",
	})

	// Given
	expectedErrorText := "# heading"
	source := "Just some text\n"

	// When
	_, err := parseMarkdownSlide("untitled", source)

	// Then
	if err == nil || !strings.Contains(err.Error(), expectedErrorText) {
		t.Fatalf("expected an error mentioning %q, got %v", expectedErrorText, err)
	}
}

func TestMarkdownSlides_EveryFileInTheFolder_Loads(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the markdown slides in the slides folder.",
		When:  "When they are loaded at startup.",
		Then:  "Then every one of them parses and has a title.",
	})

	// Given
	files := markdownFiles

	// When
	slidesByName, err := loadMarkdownSlides(files)

	// Then
	if err != nil {
		t.Fatalf("expected every markdown slide to load, got %v", err)
	}
	if len(slidesByName) == 0 {
		t.Fatalf("expected at least one markdown slide")
	}
	for name, slide := range slidesByName {
		if slide.Title == "" {
			t.Errorf("%s.md has no title", name)
		}
	}
}
