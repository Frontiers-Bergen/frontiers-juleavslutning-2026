# How a live update flows

1. You tap a button on your phone
2. Datastar sends it to the Go server
3. Go writes to SQLite and publishes a NATS message
4. Every phone's open stream renders fresh HTML
5. Datastar patches it into every page

```html
<button data-on:click="@post('/poll/vote?option=smash-the-halls')">
```

Notes:

- The server owns the state; phones just show it. That's why it stays simple.
- Emoji smashes use one NATS message per smash, so every balloon shows up.
