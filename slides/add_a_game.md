# Add your own mini-game

Tug of war, next slide, is three files:

```text
pages/root/tug_of_war.templ   markup and Go
pages/root/tug_of_war.css     its styles
tug_of_war_test.go            behaviour tests
```

Most games need no JavaScript. The snow globe shows when it is worth it: a sensor.

Notes:

- Group by feature, not by technical role: everything about a game sits side by side.
- JavaScript only when the phone must react on its own: sensors, dragging, physics. Animation alone is CSS.
- The step-by-step guide is MINI_GAMES.md; AGENTS.md has the rules for people and AI agents alike.
- Next: play a round of tug of war. Your emoji smashes are the pulls.
