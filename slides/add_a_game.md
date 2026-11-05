# Add your own mini-game

Two example games show the way:

- **Tug of war**, next slide: only Go, SQLite and Datastar; your emoji smashes are the pulls
- Shake the snow globe: a web component built with Rocket (coming soon)

Start with the first. Most games don't need more.

Notes:

- Tug of war is one templ file, one CSS file and one test file: `pages/root/tug_of_war.*`, `tug_of_war_test.go`.
- JavaScript is only needed when the phone must react on its own: sensors, dragging, physics. Animation alone is CSS.
- Draft: link to the snow globe and the guide once they exist.
