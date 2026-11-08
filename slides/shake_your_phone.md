# Shake your phone ❄️

Shake it like a snow globe: it snows on your phone, and on the big screen.

On an iPhone, tap **Shake for snow** first.

Notes:

- The more you shake, the harder it swirls on the other screens.
- The only part that needs JavaScript is reading the motion sensor: a small Rocket web component,
  `static/snow-globe.js`. The snow on every other screen is server-rendered HTML and CSS.
- Laptops have no motion sensor, so nothing happens there. iPhones only allow the sensor after a tap, hence the button.
