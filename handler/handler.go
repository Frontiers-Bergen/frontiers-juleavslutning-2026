package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/components"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/pages"
	"github.com/frontiers-bergen/frontiers-juleavslutning-2026/services"
	"github.com/starfederation/datastar-go/datastar"
)

type demoHandler struct {
	demoService  *services.DemoService
	emojiService *services.EmojiBalloonService
}

func NewDemoHandler(ns *embeddednats.Server, emojiService *services.EmojiBalloonService) (*demoHandler, error) {
	demoService, err := services.NewDemoService(ns, emojiService)
	if err != nil {
		return nil, fmt.Errorf("failed to create demo service: %w", err)
	}
	dm := demoHandler{demoService: demoService, emojiService: emojiService}
	return &dm, nil
}

func (dm *demoHandler) DemoRoute(w http.ResponseWriter, r *http.Request) {
	err := components.BaseLayout("demo", pages.DemoPage()).Render(r.Context(), w)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (dm *demoHandler) DemoSSE(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sse := datastar.NewSSE(w, r)

	updateWatcher, err := dm.demoService.WatchUpdates(ctx)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	emojiWatcher, err := dm.demoService.WatchEmojiCounter(ctx)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	thumbs_up := dm.emojiService.Counter.ThumbsUpEmojiCount
	confetti := dm.emojiService.Counter.ConfettiEmojiCount
	cry_laugh := dm.emojiService.Counter.CryLaughEmojiCount
	fire := dm.emojiService.Counter.FireEmojiCount
	heart := dm.emojiService.Counter.HeartEmojiCount

	for {
		select {
		case <-ctx.Done():
			return

		case <-updateWatcher.Updates():
			if err := sse.ExecuteScript("console.log('update watcher fired')"); err != nil {
				return
			}

		case <-emojiWatcher.Updates():
			b, err := json.Marshal(dm.emojiService)

			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			err = sse.PatchSignals(b)

			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}

			emoji := ""

			if thumbs_up != dm.emojiService.Counter.ThumbsUpEmojiCount {
				thumbs_up = dm.emojiService.Counter.ThumbsUpEmojiCount
				emoji = "👍"
			}
			if confetti != dm.emojiService.Counter.ConfettiEmojiCount {
				confetti = dm.emojiService.Counter.ConfettiEmojiCount
				emoji = "🎉"
			}
			if cry_laugh != dm.emojiService.Counter.CryLaughEmojiCount {
				cry_laugh = dm.emojiService.Counter.CryLaughEmojiCount
				emoji = "😂"
			}
			if fire != dm.emojiService.Counter.FireEmojiCount {
				fire = dm.emojiService.Counter.FireEmojiCount
				emoji = "🔥"
			}
			if heart != dm.emojiService.Counter.HeartEmojiCount {
				heart = dm.emojiService.Counter.HeartEmojiCount
				emoji = "❤️"
			}

			if emoji != "" {
				if err := sse.PatchElementTempl(components.DemoEmojBalloon(emoji), datastar.WithModeAppend(), datastar.WithSelectorID("demo-container")); err != nil {
					return
				}
			}

			if err := sse.ExecuteScript(`
				  
document.body.querySelector("#demo-container").scrollTo({
  top: document.body.querySelector("#demo-container").scrollHeight,
  behavior: "smooth"
});
				`); err != nil {
				return
			}

		}
	}
}
