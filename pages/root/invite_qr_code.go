package root

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
)

// Invite QR code: people at the venue join by scanning the big screen instead of typing a URL.
//
// GET /{inviteKey}/qr.svg returns the invite link as a QR code. A slide shows it with a relative URL
// (`<img src="qr.svg">`), so markdown slides need no templating.
//
// The link uses the host the page was opened with. On a LAN that is the machine's IP; in production it is
// the public domain, with https detected from the reverse proxy's X-Forwarded-Proto header. Opening the big
// screen as "localhost" gives a QR code a phone cannot open.

func InviteQRCodeRoute(router chi.Router, logger *slog.Logger) {
	router.Get("/qr.svg", func(w http.ResponseWriter, r *http.Request) {
		svg, err := qrCodeSVG(inviteLink(r, chi.URLParam(r, "inviteKey")))
		if err != nil {
			logger.Error("failed to draw invite QR code", slog.String("error", err.Error()))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "no-cache")
		if _, err := w.Write([]byte(svg)); err != nil {
			return // the client went away
		}
	})
}

func inviteLink(r *http.Request, inviteKey string) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/", scheme, r.Host, inviteKey)
}

// qrCodeSVG draws content as a QR code: black modules on a white background, including the quiet zone
// scanners need around the code. One viewBox unit is one module, so it stays sharp on a projector.
// Adapted from conorganizer's components/qrcode.
func qrCodeSVG(content string) (string, error) {
	if content == "" {
		return "", errors.New("qr code content is empty")
	}
	code, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to encode qr code: %w", err)
	}
	bitmap := code.Bitmap()
	size := len(bitmap)

	var path strings.Builder
	for y, row := range bitmap {
		for x, dark := range row {
			if dark {
				fmt.Fprintf(&path, "M%d %dh1v1h-1z", x, y)
			}
		}
	}
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges"><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="%s"/></svg>`,
		size, size, size, size, path.String(),
	), nil
}
