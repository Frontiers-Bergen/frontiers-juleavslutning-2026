# The stack

- **Go** serves everything
- **templ** writes the HTML
- **Datastar** keeps every phone live over server-sent events
- **SQLite** stores what must survive the evening
- **NATS**, embedded, passes messages between phones

No frontend build step. No JavaScript framework.

Notes:

- Keep this short: context, not a sales pitch.
- One binary, one SQLite file. The server renders HTML; Datastar patches it into the page.
- A web component only when the browser must do something the server cannot, like reading a sensor (the snow globe).
