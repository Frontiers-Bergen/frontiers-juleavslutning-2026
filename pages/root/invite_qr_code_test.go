package root_test

import (
	"image"
	"image/color"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"testing"

	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/apptest"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/bdd"
	"github.com/makiuchi-d/gozxing"
	zxingqr "github.com/makiuchi-d/gozxing/qrcode"
)

func TestInviteQRCode_Scanning_OpensTheDeck(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the big screen shows the invite QR code.",
		When:  "When someone scans it with their phone.",
		Then:  "Then it opens the deck with the invite link on the same host.",
	})

	// Given
	app := apptest.Start(t)
	expectedLink := app.URL() + "/" + apptest.InviteKey + "/"
	bigScreen := app.NewAttendee()

	// When
	status, contentType, svg := app.Get(bigScreen, "/"+apptest.InviteKey+"/qr.svg")
	actualLink := scanQRCodeSVG(t, svg)

	// Then
	if status != http.StatusOK || contentType != "image/svg+xml" {
		t.Fatalf("expected 200 image/svg+xml, got %d %q", status, contentType)
	}
	if actualLink != expectedLink {
		t.Fatalf("expected the QR code to open %q, got %q", expectedLink, actualLink)
	}
}

func TestRepositoryQRCode_Scanning_OpensTheRepository(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the big screen shows the repository QR code on the slide where people start building.",
		When:  "When someone scans it with their phone.",
		Then:  "Then it opens the repository on GitHub.",
	})

	// Given
	expectedLink := "https://github.com/frontiers-bergen/frontiers-juleavslutning-2026"
	app := apptest.Start(t)
	bigScreen := app.NewAttendee()

	// When
	status, contentType, svg := app.Get(bigScreen, "/"+apptest.InviteKey+"/repo-qr.svg")
	actualLink := scanQRCodeSVG(t, svg)

	// Then
	if status != http.StatusOK || contentType != "image/svg+xml" {
		t.Fatalf("expected 200 image/svg+xml, got %d %q", status, contentType)
	}
	if actualLink != expectedLink {
		t.Fatalf("expected the QR code to open %q, got %q", expectedLink, actualLink)
	}
}

func TestInviteQRCode_BehindHTTPSProxy_LinksToHTTPS(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given the app runs behind a reverse proxy that terminates HTTPS.",
		When:  "When the big screen loads the invite QR code.",
		Then:  "Then the QR code links to https, so phones do not land on plain http.",
	})

	// Given
	app := apptest.Start(t)
	request, err := http.NewRequest(http.MethodGet, app.URL()+"/"+apptest.InviteKey+"/qr.svg", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	request.Host = "juleavslutning.example"
	request.Header.Set("X-Forwarded-Proto", "https")
	expectedLink := "https://juleavslutning.example/" + apptest.InviteKey + "/"

	// When
	response, err := app.NewAttendee().Do(request)
	if err != nil {
		t.Fatalf("GET qr.svg failed: %v", err)
	}
	defer apptest.CloseBody(t, response)
	svg, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read qr.svg: %v", err)
	}
	actualLink := scanQRCodeSVG(t, string(svg))

	// Then
	if actualLink != expectedLink {
		t.Fatalf("expected the QR code to open %q, got %q", expectedLink, actualLink)
	}
}

var (
	qrModule  = regexp.MustCompile(`M(\d+) (\d+)h1v1h-1z`)
	qrViewBox = regexp.MustCompile(`viewBox="0 0 (\d+) (\d+)"`)
)

// scanQRCodeSVG paints every dark module of the SVG into an image (10px per module) and decodes it,
// so the test checks the drawing a phone would scan, not only the encoder. From conorganizer's qrtest.
func scanQRCodeSVG(t *testing.T, svg string) string {
	t.Helper()
	box := qrViewBox.FindStringSubmatch(svg)
	if box == nil {
		t.Fatalf("no viewBox in QR SVG: %.200s", svg)
	}
	size, err := strconv.Atoi(box[1])
	if err != nil {
		t.Fatalf("invalid viewBox size %q: %v", box[1], err)
	}
	const scale = 10
	img := image.NewGray(image.Rect(0, 0, size*scale, size*scale))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	for _, module := range qrModule.FindAllStringSubmatch(svg, -1) {
		x, _ := strconv.Atoi(module[1]) // the regexp only matches digits
		y, _ := strconv.Atoi(module[2])
		for dy := range scale {
			for dx := range scale {
				img.SetGray(x*scale+dx, y*scale+dy, color.Gray{Y: 0})
			}
		}
	}
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatalf("failed to build bitmap: %v", err)
	}
	result, err := zxingqr.NewQRCodeReader().Decode(bitmap, nil)
	if err != nil {
		t.Fatalf("failed to decode QR code: %v", err)
	}
	return result.GetText()
}
