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
// Longer than the slowest flake: its fall starts at most 1.8 s in and takes at most 14 s.
const SNOWFALL_VISIBLE_MS = 18000;

const random = (min, max) => min + Math.random() * (max - min);

// A snowfall with `flakes` flakes swirling with `energy`, the same markup and variation the server sends
// (randomSnowflakes and Snowfall in snow_globe.templ): three nested spans, one per movement: swirl, fall, sway.
function snowfall(energy, flakes) {
	const container = document.createElement("div");
	container.className = "snowfall";
	container.style.setProperty("--energy", energy);
	for (let i = 0; i < flakes; i++) {
		const delay = Math.round(random(0, 400));
		const flake = document.createElement("span");
		flake.className = "snowflake";
		const set = (name, value) => flake.style.setProperty(name, value);
		set("--x", `${random(5, 95).toFixed(1)}%`);
		set("--top", `${random(0, 10).toFixed(1)}%`);
		set("--drift", `${random(-3, 3).toFixed(2)}vw`);
		set("--size", random(0.6, 1.2).toFixed(2));
		set("--delay", `${delay}ms`);
		set("--swirl-duration", `${Math.round(random(1000, 1800))}ms`);
		set("--fall-delay", `${delay + Math.round(random(600, 1400))}ms`);
		set("--duration", `${Math.round(random(10000, 14000))}ms`);
		set("--fall", `${Math.round(random(75, 95))}vh`);
		set("--fall-drift", `${random(-2, 2).toFixed(2)}vw`);
		set("--sway", `${random(1, 3).toFixed(2)}vw`);
		set("--sway-duration", `${Math.round(random(2500, 4500))}ms`);
		const fall = document.createElement("span");
		fall.className = "snowflake-fall";
		const sway = document.createElement("span");
		sway.className = "snowflake-sway";
		sway.textContent = "❄";
		fall.append(sway);
		flake.append(fall);
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
