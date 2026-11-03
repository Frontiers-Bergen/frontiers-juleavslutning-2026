const EmojiBalloon = (() => {

const REDUCED_MOTION_VISIBLE_MS = 800;

const GLOBAL = {
	balloons: []
}

const animate = () => {
	requestAnimationFrame(animate);
	for (let i = 0; i < GLOBAL.balloons.length; i++) {
		const b = GLOBAL.balloons[i];
		
		b._position.y = b._position.y + b._velocity.y;
		b._position.x = b._position.x + b._velocity.x;
		
		// moving left
		if (b._driftDirection === "LEFT") {
			const xLessThanMaxDrift = b._position.x < b._startPositionX - b._maxDriftX;
			if (xLessThanMaxDrift) {
				b._driftDirection = "RIGHT";
				b._velocity.x = b._velocity.x * -1;
			}
		} else {
			const xGreaterThanMaxDrift = b._position.x > b._startPositionX + b._maxDriftX;
			if (xGreaterThanMaxDrift) {
				b._driftDirection = "LEFT";
				b._velocity.x = b._velocity.x * -1;
			}
		}
		b.style.top = b._position.y + "px"; 
		b.style.left = b._position.x + "px"; 
	}
	GLOBAL.balloons.forEach(b => {
		if (b._position.y <= -32) {
			document.body.removeChild(b);
		}
	})
	GLOBAL.balloons = GLOBAL.balloons.filter(b => b._delete === false)
};
animate();

class _EmojiBalloon extends HTMLElement {
	_delete = false;
	_velocity = { x: -1, y: -2.5 }; // px per tick (16ms)
	_position = { x: 0, y: 0 };
	
	_startPositionX = 0;
	_maxDriftX = 50
	_driftDirection = "LEFT"

	constructor() {
		super();

		this._driftDirection = randomInt(0, 1) === 0 ? "LEFT" : "RIGHT";
		this._velocity.x = this._driftDirection === "LEFT" ? -1 : 1;
		

		this._maxDriftX = randomInt(25, 75);
		this._velocity.y = this._velocity.y * randomInt(0.5, 2)
		this._velocity.x = this._velocity.x * randomInt(0.5, 2)


		this.attachShadow({ mode: "open" })

		const emoji = this.getAttribute("emoji") || "";
		if (!emoji) {
			return
		}

		// position the <emoji-balloon> at the location of the button
		const buttons = document.body.querySelectorAll(".emoji-balloon-button");
		let button = null;
		buttons.forEach(b => {
			if (b.innerHTML.includes(emoji)) {
				button = b;
			}
		});
		if (button) {
			const rect = button.getBoundingClientRect();	
			this._position.x = rect.x;
			this._position.y = rect.y;
			this._startPositionX = rect.x;
		}

		this.shadowRoot.innerHTML = `<div>${emoji}</div>`;

		this.style.position = "absolute";
		this.style.userSelect = "none";
	}

	static get observedAttributes() {
		return ["emoji"];
	}

	connectedCallback() {
		// With "reduce motion" on, a smash still shows its balloon at the button, but it does not fly:
		// it stays for a moment and disappears. Every smash stays visible without the movement.
		if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
			setTimeout(() => this.remove(), REDUCED_MOTION_VISIBLE_MS);
			return;
		}
		GLOBAL.balloons.push(this)
	}

	disconnectedCallback() {
		this._delete = true;	
	}

}

return _EmojiBalloon
})();

if (!customElements.get("emoji-balloon")) {
	customElements.define("emoji-balloon", EmojiBalloon);
}

function randomInt(min, max) {
  // Ensure integers
  min = Math.ceil(min);
  max = Math.floor(max);

  // Inclusive of both min and max
  return Math.floor(Math.random() * (max - min + 1)) + min;
}
