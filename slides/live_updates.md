# Redraw everything, on every smash

<div class="flow" role="img" aria-label="You tap on your phone; a short POST goes to the Go server, which saves it in SQLite and announces that something changed; then every open page, on every phone and the big screen, renders itself again, whole, and Datastar morphs the difference into the page.">
<div class="flow-node flow-tap"><span class="flow-icon">📱</span><span class="flow-text">You tap</span></div>
<div class="flow-arrow flow-post"><span class="flow-text">POST</span></div>
<div class="flow-node flow-server"><span class="flow-icon">🐹</span><span class="flow-text">Go saves it in SQLite</span></div>
<div class="flow-arrow flow-changed"><span class="flow-text">"something changed"</span></div>
<div class="flow-screens">
<div class="flow-node flow-screen"><span class="flow-icon">📱</span><span class="flow-text">renders the whole page</span></div>
<div class="flow-node flow-screen"><span class="flow-icon">📱</span><span class="flow-text">renders the whole page</span></div>
<div class="flow-node flow-screen"><span class="flow-icon">💻</span><span class="flow-text">the big screen too</span></div>
</div>
</div>

A page renders in 0.1–0.4 ms. An update costs a phone about 38 bytes.

Notes:

- It re-renders everything every time.
- This took me some time to get used to.
- It's very different from how we usually think.
- It's so fast that we don't need to worry about partial updates, debouncing, or client state.
- This removes a lot of complexity.
- No client state, no partial updates, no debouncing: the server owns the state, the phones show it.
- The agents struggle with this. The rules are in "Page life cycle" in AGENTS.md.
