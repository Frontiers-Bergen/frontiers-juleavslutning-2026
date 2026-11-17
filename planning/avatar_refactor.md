# Avatar refactor: planning draft

Status: the first landing-page prototype is implemented; the scope remains limited to the builder.

## Why

Give people at the juleavslutning a fun way to create an avatar. The experience should be simple on a phone and easy
for contributors to build on.

## Scope

Replace the current single-emoji chooser on the name and avatar landing page with a prototype avatar builder. An
attendee chooses a **head**, a **hat** and a **set of eyes**; the selections make one avatar in the preview.
The avatar has no body. Every option must be representable as a simple, temporary SVG, using shapes and/or colours
rather than finished artwork.

This work is about the builder on the existing page. Joining the deck is a separate feature: leave the name field and
disabled Continue behavior as they are. Persistence, using avatars elsewhere, shared-screen display and integration
with games are out of scope for this prototype. Do not design those features as prerequisites for the builder.

## User story

As an attendee, I want to combine a head, a hat and a set of eyes so that I can create an avatar of my own.

## Agreed behavior

- The page provides three choices each for head, hat and eyes, with one of each selected initially. Each choice
  combines its shape and colour; there is no separate colour picker.
- Labelled head, hat and eyes tabs let the attendee switch directly between groups on all screen sizes; Head opens
  initially.
- The combined preview is above the tabs, remains available while switching groups, and scrolls with the page.
- The preview shows the selected parts together as one avatar; all options can be represented as simple, temporary SVGs.
- Changing one part updates the preview immediately without changing the other selections or automatically switching
  tabs.
- Within each group, choices are shown as SVG thumbnails without visible text labels; “no hat” is represented by a
  bare head. An outline and a checkmark mark the selected choice.
- Design mobile-first, with touch as the primary interaction.
- The name field and disabled Continue button remain on the landing page; the builder does not submit or save a profile.
- Presenter access to the deck remains unchanged.

## Agreed prototype appearance

These are deliberately simple SVG placeholders, not final artwork.

| Part | Options | Default |
| --- | --- | --- |
| Head | Peach circle, mint oval, lavender rounded square | Peach circle |
| Hat | No hat, blue beanie, red pointed hat | Blue beanie |
| Eyes | Round dots, tall ovals, upward-curved smiling eyes; all the same dark colour | Round dots |

The hat sits at the top of the head and the pair of eyes sits on the face; the “no hat” choice leaves the head bare.

## Recommended implementation details

- Use simple SVG shapes and paths rather than detailed artwork; keep the parts aligned across combinations.
- Make the choices distinguishable without relying on colour alone and keep tap targets at least 3rem high.
- Keep the controls usable by keyboard and give each choice an accessible name despite the thumbnail-only presentation.
- No animation is needed for this prototype.

## Current behavior

- The attendee landing screen renders the local SVG avatar builder described above. Reloading restores its defaults.
- Continue is disabled, there is no submission, and neither the name nor avatar is saved.
- An attendee opening the invite link sees the landing screen, not the deck. A presenter still sees the deck.
- Existing attendee session IDs identify a browser for other features, but the landing screen does not create a
  session or store a profile.

Sources: [landing template](../pages/root/attendee_landing.templ),
[landing styles](../pages/root/attendee_landing.css), [landing tests](../pages/root/attendee_landing_test.go),
[root route](../pages/root/root_index.templ), and [session ID](../pages/root/domain_types.go).

## Verification

Verify the default combination and changing each part independently, prioritizing touch use. Use a real browser for the
builder interaction and touch-sized controls, and check the phone and big-screen layouts. Do not test landing-page
text content as part of this work.
