# How to add your own mini-game

A mini-game is a slide that everyone plays from their phone while the big screen shows the result.
Two finished games show the way; copy the one closest to your idea:

- **Tug of war** (`pages/root/tug_of_war.*`, including `tug_of_war_test.go`): only Go, SQLite and Datastar, no JavaScript.
  Two random, even teams; the emoji reaction buttons are the pulls. Start here: most games need nothing more.
- **Shake the snow globe** (`static/snow-globe.js`, `pages/root/snow_globe.*`, including `snow_globe_test.go`): a small
  Rocket web component reads the phone's motion sensor; the snowfall on every other screen is server-rendered HTML and CSS.

Read "Page life cycle" in `AGENTS.md` first. In short: the server holds the state, every tap is a short POST,
and every open page renders itself again, whole, after every change. No client state, no partial updates, no debouncing.

Ideas waiting for someone: "Mini-game ideas" in `README.md`.

## A game without JavaScript (like tug of war)

1. **Files.** Create `pages/root/<game>.templ`, with `<game>.css` and `<game>_test.go` next to it: a game's files stay together.
   The CSS is picked up automatically; nest all of it under the game's root class (`.my-game { ... }`).
2. **State in SQLite.** Add your tables to `schema.sql`. It runs on every startup, so use `CREATE TABLE IF NOT EXISTS`
   and `INSERT OR IGNORE`. Changed a table you already created? Delete your local `presentation.db`.
3. **Writes.** Add `<Game>Routes(router, ...)` in your `.templ` file and register it in `RootLayoutRoute`
   (`pages/root/root_index.templ`), next to `TugOfWarRoutes`. A route validates, writes SQLite, calls
   `publishDeckChanged(nc)` and answers `204`. It never returns HTML: every open page re-renders on its own.
   Presenter-only actions (start, reset) check `isPresenterFromSession` and answer `403` otherwise.
4. **The slide.** A templ component `<Game>Slide(db, inviteKey, sessionID, isPresenter)` loads its state from SQLite
   and renders it. Add it to the deck in `pages/root/active_slide.templ`. The same page is the phone controller and
   the big screen: start mobile first, one-handed, tap targets at least 3rem tall, and use `@media (width >= 40rem)`
   for the big screen.
5. **Input.** Buttons use `data-on:click="@post('/<invite key>/root/api/<game>/<action>',{requestCancellation: 'disabled'})"`.
   Or reuse the emoji reaction buttons, which are always on screen: tug of war hooks into `IncrementEmojiRoute`
   through `countSmashForTugOfWar`.
6. **Time.** Only things that change with time alone need a timer (a countdown); see `tickTugOfWarClock`.
7. **Tests.** HTTP tests in `<game>_test.go` (`package root_test`) start the app with `apptest.Start(t)`: phones are
   `app.NewAttendee()`, the presenter is `app.NewPresenter()`. Put semantic `data-` attributes on the slide (`data-tug-of-war-phase`) and read them in tests.
   If the game matters on the evening, add it to `browser-tools/browser_check.mjs`.

## A game that needs the browser (like the snow globe)

Only when the phone must do something the server cannot: read a sensor, drag at 60 fps, run physics.
Animation alone is CSS.

1. Write a Rocket component in `static/<game>.js` and load it in `components/base_layout.templ`.
2. Props down, events up: the component emits a bubbling event (`emit("shaken", {count}, {bubbles: true})`) and knows nothing about URLs.
3. Wrap it in an element that carries the Datastar markup (`data-on:shaken="... @post(...)"`). Never put `@get` or `@post`
   on the Rocket element itself: Rocket rewrites them into calls to the component's own actions.
4. If every single event must show up on its own (one snowfall per shake), publish it on its own NATS subject and let
   a stream append a small patch for it (`SnowGlobeRoutes`). Anything that must be remembered still goes in SQLite.
5. Respect `prefers-reduced-motion`.

## Before your pull request

- `go tool task check` passes.
- `go tool task screenshots` and look at the result; try the game on a real phone (`SETUP.md` explains the LAN address).
- `go tool task browser-check` when you changed buttons, Datastar attributes or streams.
