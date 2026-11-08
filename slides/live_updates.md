# Redraw everything, on every smash

1. You tap; your phone sends a short POST
2. Go writes it to SQLite and announces "something changed"
3. Every open page renders itself again, whole, from SQLite
4. Datastar morphs only the difference into the page

A page renders in 0.1–0.4 ms. An update costs a phone about 38 bytes.

Notes:

- No client state, no partial updates, no debouncing: the server owns the state, the phones show it.
- It works because rendering is cheap, morphing keeps focus and scroll, and the stream is compressed.
- Balloons are events, not state: one message per smash, so every balloon shows up.
- The rules are in "Page life cycle" in AGENTS.md. Read it before you design anything that updates a page.
