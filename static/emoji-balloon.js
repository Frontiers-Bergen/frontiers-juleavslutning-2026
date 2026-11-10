// <emoji-balloon emoji="🔥">: one balloon per emoji smash, rising from the smashed button. The emoji stream
// (emoji_counter.templ) appends one to <body> for every smash, on every screen.
//
// The browser animates the flight itself, with two Web Animations on transform: one rising, one swaying from side
// to side. Both start at the moment the balloon arrived, so the flight follows the clock:
// - a window the browser does not draw (on another workspace) still receives balloons; when it is drawn again,
//   the old balloons have already flown away and the recent ones are mid-flight, instead of all taking off at once;
// - the speed is the same on 60 Hz and 120 Hz screens;
// - no JavaScript runs per frame, which matters on slow phones with hundreds of balloons in the air.
// The balloon content lives in a shadow DOM: Datastar watches everything added to <body> for data-* attributes,
// and measured with 400 balloons, content outside a shadow DOM cost about 45% more JavaScript.

const REDUCED_MOTION_VISIBLE_MS = 800;
// Each balloon picks one of each, so the balloons spread out.
const RISE_PX_PER_SECOND = [150, 300];
const SWAY_PX_PER_SECOND = [60, 120];
const SWAY_WIDTH_PX = { min: 25, max: 75 };
// Above the top of the screen: the flight ends out of sight.
const FLIGHT_END_PX = -32;

// Where each button is, read once per frame. A balloon has just been added to the page when it asks, so reading a
// position makes the browser lay out the page again; with many balloons arriving, doing that for every balloon was
// the biggest cost (measured: layout -57% with 400 or 1500 balloons). The buttons do not move between frames.
const buttonPositions = new Map();
function buttonPosition(emoji) {
	if (!buttonPositions.has(emoji)) {
		if (buttonPositions.size === 0) requestAnimationFrame(() => buttonPositions.clear());
		const button = document.querySelector(`.emoji-balloon-button[data-emoji="${CSS.escape(emoji)}"]`);
		buttonPositions.set(emoji, button?.getBoundingClientRect());
	}
	return buttonPositions.get(emoji);
}

class EmojiBalloon extends HTMLElement {
	connectedCallback() {
		const arrivedAt = performance.now(); // the document timeline counts from the same origin
		const emoji = this.getAttribute("emoji") ?? "";
		const start = emoji && buttonPosition(emoji);
		if (!start) {
			this.remove();
			return;
		}

		Object.assign(this.style, { position: "fixed", left: `${start.x}px`, top: `${start.y}px`, userSelect: "none", pointerEvents: "none" });
		if (!this.shadowRoot) {
			this.attachShadow({ mode: "open" }).innerHTML = `<div>${emoji}</div>`;
		}

		// With "reduce motion" on, a smash still shows its balloon at the button, but it does not fly:
		// it stays for a moment and disappears. Every smash stays visible without the movement.
		if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
			setTimeout(() => this.remove(), REDUCED_MOTION_VISIBLE_MS);
			return;
		}

		const rise = start.y - FLIGHT_END_PX;
		const flight = this.animate([{ transform: "translateY(0)" }, { transform: `translateY(${-rise}px)` }], {
			duration: (rise / pick(RISE_PX_PER_SECOND)) * 1000,
			fill: "forwards",
		});
		flight.startTime = arrivedAt;
		flight.finished.then(
			() => this.remove(),
			(error) => {
				// Removing the balloon cancels its flight; anything else is a real problem.
				if (error.name !== "AbortError") throw error;
			},
		);

		// Sway out to one side, back through the start, out to the other side, and back, for as long as it flies.
		const width = SWAY_WIDTH_PX.min + Math.random() * (SWAY_WIDTH_PX.max - SWAY_WIDTH_PX.min);
		const side = Math.random() < 0.5 ? -1 : 1;
		const sway = this.shadowRoot.firstElementChild.animate(
			[0, side, 0, -side, 0].map((direction) => ({ transform: `translateX(${direction * width}px)` })),
			{ duration: ((4 * width) / pick(SWAY_PX_PER_SECOND)) * 1000, iterations: Infinity },
		);
		sway.startTime = arrivedAt;
	}
}

function pick(choices) {
	return choices[Math.floor(Math.random() * choices.length)];
}

if (!customElements.get("emoji-balloon")) {
	customElements.define("emoji-balloon", EmojiBalloon);
}
