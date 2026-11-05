# Add your own mini-game

Two example games show the way:

- **Tug of war**, next slide: only Go, SQLite and Datastar; your emoji smashes are the pulls
- **Shake the snow globe**: shake your phone and it snows everywhere; a small web component built with Rocket reads the motion sensor

Start with the first. Most games don't need more.

Notes:

- Tug of war is one templ file, one CSS file and one test file: `pages/root/tug_of_war.*`, `tug_of_war_test.go`.
- JavaScript is only needed when the phone must react on its own: sensors, dragging, physics. Animation alone is CSS.
- The snow globe shows when JavaScript is worth it: `static/snow-globe.js` reads the sensor; the snowfall on every other screen is server-rendered HTML and CSS.
- The step-by-step guide is `MINI_GAMES.md` in the repo root.
