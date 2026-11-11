# Style guide

Frontiers Bergen has no official style guide, and none will arrive before the juleavslutning, so this is ours.
It takes its look from the Frontiers Bergen website (frontiersbergen.dev) and adds what a live, phone-first app needs.

The values live in code: the design tokens at the top of `static/index.css` are the source of truth.
This guide explains what each one is for and the rules around them. Change both together, and check the result
with `go tool task screenshots`.

## The feel

Friendly, a little nerdy, and fun. The app should feel like the Frontiers website came to the Christmas party:
the same purple, the same monospace font, the same markdown-style headings, with festive touches on top.

## Colours

| Token | Value | Use |
|---|---|---|
| `--color-bg-base` | `#1e1e3f` | Page background, the stage, empty preview frames |
| `--color-surface` | `#2d2b55` | Anything that sits on the background: presenter bar and sidebar, result cards, buttons, code blocks, notes |
| `--color-border` | `#3a3779` | Borders on surfaces; the background of a reaction button while pressed |
| `--color-text-strong` | `#fff` | Headings and the most important text |
| `--color-text-soft` | `#c5c0e8` | Body text |
| `--color-text-muted` | `#a599e9` | Secondary text: labels, hints, list markers |
| `--color-accent` | `#ff9d00` | The `#` before headings. Use sparingly; it is the brand's spark |
| `--color-link` | `#80ffea` | Links and inline code |
| `--color-link-hover` | `#fad000` | Links on hover |
| `--color-team-teal` | `#07d9b2` | Team colour in team games (the logo's teal); near-black text on it |
| `--color-team-orange` | `#ff9200` | Team colour in team games (the logo's orange); near-black text on it |
| `--color-primary-text` | `--color-text-soft` | The page's default text colour |
| `--surface-presenter` | `--color-surface` | The presenter's bar and sidebar |
| `--surface-backstage` | `--color-surface` | Presenter notes |
| `--surface-glass` | `--color-bg-base` at 90% | The reaction bar over the stage |
| `--border-subtle` | 1px `--color-border` | Borders on surfaces |
| `--font-visual` | `1.5rem` | Diagrams, charts and tiles on the big screen |

Rules:
- Use the tokens, never raw colour values, in feature CSS.
- Do not add brand colours. Festive colour comes from emoji, illustrations and motion, not from new tokens.
- The exception is the poll options, which keep their Kahoot colours and shapes so people find their answer at a glance:
  red ▲, blue ◆, yellow ●, green ■ (`kahootThemeForIndex` in `pages/root/poll.templ`). Colour is never the only cue;
  the shape always comes with it. Text is white on red, blue and green, and near-black (`--color-bg-base`) on yellow,
  where white would be unreadable.

## Typography

- **JetBrains Mono** everywhere, like the website. Regular (400) for text, Bold (700) for headings and emphasis.
  It is self-hosted in `static/fonts` (SIL Open Font License); do not load fonts from a CDN, the venue wifi may be slow.
- Sizes come from the type scale tokens: `--font-h1` for slide headings, `--font-body` for slide text, `--font-small`
  for presenter notes and small print. They grow with the screen, from phone to big screen.
- Slide headings start with an orange `#`, like the markdown headings on the website. It is added by CSS
  (hidden from screen readers); do not type it into the heading text.
- Monospace is wide: keep headings short so they wrap into two or three lines on a phone, not five.

## Logo

- `static/frontiers-logo.svg`, the same file as the website's logo.
- Use it on the dark background only, at its own colours. Do not recolour, stretch or put it on a light surface.
- On a slide it is 4rem high (`slides/welcome.css`).

## Layout and spacing

- **Mobile first.** Base CSS is for a phone; `@media (width >= 40rem)` adds the big-screen layout.
- Phones get one column; the big screen may use two (the poll and its results).
- Spacing comes from `--space-1` (0.5rem) to `--space-5` (3rem); corners use `--radius`.
- **Tap targets are at least 3rem (48px)** tall. The reaction buttons are meant to be smashed with a thumb.
- The reaction bar is always at the bottom of the screen; nothing should hide behind it (`--reactions-min-height`).

## Components

- **Slides:** one idea per slide; a heading and a few short lines or at most about five bullets. Content slides are
  markdown (`slides/*.md`, format in `slides/markdown_slides.go`); talking points go in the `Notes:` section,
  not on the slide.
- **Buttons:** surface background, a border, strong text. The reaction buttons take the border colour as their background while pressed, so every smash is felt.
- **Cards** (poll results): surface background with a border, no shadows.
- **Code blocks:** surface background with a border; on a phone they scroll sideways instead of breaking lines.
- **The QR code** stays black on white, whatever the theme: scanners need the contrast.

## Festive touches

The juleavslutning may look like Christmas, on top of the Frontiers look:
- Emoji are welcome everywhere (🎄 ❄️ 🎁), in slides, buttons and balloons.
- Motion is fun (balloons, confetti, snow) as long as it never hides content, slows a phone down, or ignores "reduce motion".
- Keep the purple background and the palette; do not switch to red and green.
- Illustrations are our own flat SVG drawings in the palette's colours, on a transparent background
  (`static/illustrations/`). They are decoration: the slide's text carries the message, and the picture stands
  beside the text, never on top of it (the `Illustration:` line in a slide's markdown).

## Voice and tone

- English (Frontiers runs its events in English); the domain terms in `domain_dictionary.md`, like juleavslutning, stay as they are.
- Friendly and informal, short sentences, humour welcome. Talk to people as "you".
- Say what to do, not what went wrong in technical terms ("Open the invite link you received", not "401 Unauthorized").

## Accessibility

- Text needs a contrast of at least 4.5:1 against its background. All text tokens pass on both the background and
  surfaces (lowest: muted text on a surface, 5.3:1).
- Poll options: white text on red, blue and green passes; yellow gets near-black text (6.7:1 instead of 2.4:1 for white).
- Motion respects the "reduce motion" setting (`prefers-reduced-motion`): emoji balloons appear at their button and
  disappear without flying (`static/emoji-balloon.js`). New animations must do the same; `go tool task browser-check`
  checks the balloons.
