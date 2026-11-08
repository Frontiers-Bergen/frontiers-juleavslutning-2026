# Redraw everything, on every smash

1. You tap; your phone sends a short POST
2. Go writes it to SQLite and announces "something changed"
3. Every open page renders itself again, whole, from SQLite
4. Datastar morphs only the difference into the page

A page renders in 0.1–0.4 ms. An update costs a phone about 38 bytes.

Notes:

- It re-renders everything every time.
- This took me some time to get used to.
- It's very different from how we usually think.
- It's so fast that we don't need to worry about partial updates, debouncing, or client state.
- This removes a lot of complexity.
- No client state, no partial updates, no debouncing: the server owns the state, the phones show it.
- The agents struggle with this. The rules are in "Page life cycle" in AGENTS.md.
