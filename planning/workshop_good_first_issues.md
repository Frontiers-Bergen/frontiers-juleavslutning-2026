# Good first issues for the workshop (draft)

Small, fun, self-contained tasks for 12 November. Each fits in an evening, touches few files, and makes the
juleavslutning more fun. Review, then turn the keepers into GitHub issues with the `good first issue` label.

Each item: a user story, what done looks like, and where to start.

## 1. A Christmas tree reaction 🎄
*As an attendee, I want a 🎄 reaction button, so I can show festive spirit.*
Done: a sixth button that counts and sends balloons like the others; tests still pass.
Start: `reactionEmojis` and `EmojiVoteWidget` in `pages/root/emoji_counter.templ`.

## 2. Smash milestones
*As the room, we want the big screen to celebrate when we reach 1,000 smashes together, so smashing has a goal.*
Done: at every 1,000 total smashes, the big screen shows a short celebration; a test covers the milestone rule.
Start: `pages/root/emoji_counter.templ` (counts live in the `emoji_counter` table).

## 3. Snow when we smash ❄️
*As an attendee, I want the big screen to snow harder the more we smash, so fidgeting during talks looks pretty.*
Done: snowfall intensity follows recent smashes; no layout shift on phones.
Start: a CSS-only or Rocket web component on the big screen; see `AGENTS.md` on when a web component is needed.

## 4. A hidden easter egg 🥚
*As a curious developer, I want to find a secret, so exploring the app is rewarding.*
Done: something delightful happens on a secret input (Konami code, tapping the logo seven times...). Keep it a surprise.
Start: anywhere; keep it in its own feature file.

## 5. Countdown slide ⏳
*As the organiser, I want a slide counting down to the juleavslutning, so people know when it is.*
Done: a slide showing days, hours and minutes left; the time comes from one place in the code.
Start: a templ slide in `slides/`, added to the deck in `pages/root/active_slide.templ`.

## 6. Your fun fact slide 💬
*As an attendee, I want to add a slide about my favourite Frontiers moment, so the deck feels like ours.*
Done: a new markdown slide with a heading, a few lines and presenter notes.
Start: copy `slides/stack.md`; see the format in `slides/markdown_slides.go`.

## 7. Christmas colours 🎨
*As the room, we want the app to look festive, so it feels like the juleavslutning.*
Done: a Christmas palette in the design tokens; screenshots before and after (`go tool task screenshots`).
Start: the tokens layer in `static/index.css`.

## 8. Fastest thumb 👍
*As an attendee, I want to see who smashed the most, so there is a friendly competition.*
Done: a slide with the top five smashers by session (no names needed: an emoji avatar per phone is fine).
Start: count smashes per attendee session (`attendeeSessionID` in `pages/root/live_updates.go`); needs a schema change in `schema.sql`.

## 9. Tug of war: room against remote 🪢
*As the audience, we want the people in the room to compete against the remote viewers, so both feel part of it.*
Done: a slide where each side smashes to pull the rope; the invite key decides the side.
Start: the poll already splits local and remote by invite key (`pages/root/poll.templ`).

## 10. Result reveal drumroll 🥁
*As the presenter, I want poll results to build up slowly, so the reveal is exciting.*
Done: result bars grow from zero when the results slide appears.
Start: `pages/root/poll_results.css`; mind people who prefer reduced motion.
