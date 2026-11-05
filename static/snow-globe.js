// <snow-globe>: shake your phone and it snows, like shaking a real snow globe.
//
// The only part of the snow globe that needs JavaScript is the motion sensor: devicemotion is only available
// in JavaScript, and iOS only grants it after a tap. So this Rocket component:
//   1. listens to the motion sensor (phones only: laptops have none, and mice are not "coarse pointers"),
//   2. shows an immediate puff of snow on this phone for every shake, without waiting for the network,
//   3. when the shaking stops, emits a `shaken` event with the number of shakes.
// Datastar markup on the element (pages/root/snow_globe.templ) turns `shaken` into a POST, and the server
// sends a snowfall with the same energy to every other screen. Props down, events up: the component knows
// nothing about URLs or the server.
//
// The markup of a snowfall matches the server's (Snowfall in pages/root/snow_globe.templ); the CSS for
// both is pages/root/snow_globe.css. Motion is skipped entirely with "reduce motion".

import { rocket } from "/static/datastar-rocket.js";

// m/s², acceleration without gravity: a deliberate shake, not walking around with the phone in a pocket.
const SHAKE_THRESHOLD = 14;
// One shake swings the phone back and forth; don't count the swing back as a second shake.
const MIN_MS_BETWEEN_SHAKES = 200;
// Shaking has stopped after this much stillness; then the shakes are sent as one burst.
const BURST_ENDS_AFTER_MS = 600;
const MAX_SHAKES_PER_BURST = 20;
// Longer than the slowest flake: 400 ms delay + 14 s fall.
const SNOWFALL_VISIBLE_MS = 16000;

const random = (min, max) => min + Math.random() * (max - min);

// A snowfall with `flakes` flakes swirling with `energy`, the same markup and variation the server sends
// (randomSnowflakes and Snowfall in snow_globe.templ): the outer span swirls and falls, the inner one sways.
function snowfall(energy, flakes) {
	const container = document.createElement("div");
	container.className = "snowfall";
	container.style.setProperty("--energy", energy);
	for (let i = 0; i < flakes; i++) {
		const flake = document.createElement("span");
		flake.className = "snowflake";
		flake.style.setProperty("--x", `${random(5, 95).toFixed(1)}%`);
		flake.style.setProperty("--top", `${random(0, 10).toFixed(1)}%`);
		flake.style.setProperty("--drift", `${random(-3, 3).toFixed(2)}vw`);
		flake.style.setProperty("--size", random(0.6, 1.2).toFixed(2));
		flake.style.setProperty("--delay", `${Math.round(random(0, 400))}ms`);
		flake.style.setProperty("--duration", `${Math.round(random(10000, 14000))}ms`);
		flake.style.setProperty("--fall", `${Math.round(random(75, 95))}vh`);
		flake.style.setProperty("--sway", `${random(1, 3).toFixed(2)}vw`);
		flake.style.setProperty("--sway-duration", `${Math.round(random(2500, 4500))}ms`);
		const sway = document.createElement("span");
		sway.className = "snowflake-sway";
		sway.textContent = "❄";
		flake.append(sway);
		container.append(flake);
	}
	return container;
}

function puffOfSnow(energy) {
	if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
	const puff = snowfall(energy, 3);
	document.body.append(puff);
	setTimeout(() => puff.remove(), SNOWFALL_VISIBLE_MS);
}

rocket("snow-globe", {
	mode: "light",
	setup({ host, emit, cleanup }) {
		const hasMotionSensor = "DeviceMotionEvent" in window && window.matchMedia("(pointer: coarse)").matches;
		if (!hasMotionSensor) return;

		let shakesInBurst = 0;
		let lastShakeAt = 0;
		let burstTimer;

		const onMotion = (event) => {
			const a = event.acceleration;
			if (!a || a.x === null) return;
			const strength = Math.hypot(a.x, a.y, a.z);
			const now = performance.now();
			if (strength < SHAKE_THRESHOLD || now - lastShakeAt < MIN_MS_BETWEEN_SHAKES) return;

			lastShakeAt = now;
			shakesInBurst = Math.min(shakesInBurst + 1, MAX_SHAKES_PER_BURST);
			// Each shake swirls a little harder, so five shakes start with five growing puffs.
			puffOfSnow(shakesInBurst);
			clearTimeout(burstTimer);
			burstTimer = setTimeout(() => {
				// Bubbles, so Datastar markup on a wrapper element hears it (see SnowGlobe in snow_globe.templ).
				emit("shaken", { count: shakesInBurst }, { bubbles: true });
				shakesInBurst = 0;
			}, BURST_ENDS_AFTER_MS);
		};

		window.addEventListener("devicemotion", onMotion);
		cleanup(() => {
			window.removeEventListener("devicemotion", onMotion);
			clearTimeout(burstTimer);
		});

		// iOS only sends motion events after the user taps and allows them. Recent Chrome has requestPermission too,
		// but sends events without asking, so the button only stays until the first motion event arrives.
		if (typeof DeviceMotionEvent.requestPermission !== "function") return;
		const button = document.createElement("button");
		button.type = "button";
		button.className = "snow-globe-enable";
		button.textContent = "❄️ Shake for snow";
		button.addEventListener("click", async () => {
			button.remove();
			await DeviceMotionEvent.requestPermission();
		});
		window.addEventListener("devicemotion", () => button.remove(), { once: true });
		host.append(button);
	},
});
