# Add your own mini-game

```text
pages/root/tug_of_war.templ   markup and Go
pages/root/tug_of_war.css     its styles
tug_of_war_test.go            behaviour tests
```

Most games need no JavaScript. The snow globe shows when it is worth it.

Notes:

- Cohesion: what changes together stays together.
- JavaScript only when the phone must react on its own: sensors, dragging, physics. Animation alone is CSS.
