// Browser check: clicks real buttons in headless Chromium and verifies what the server and the page show
// (go tool task browser-check). It covers what the Go tests cannot: Datastar running in a real browser.
//
// Builds the app from this checkout and runs it in a temp dir with a fresh database and dev keys, so the
// local presentation.db and .env are never touched. Chromium is driven over the DevTools protocol with
// Node's built-in fetch and WebSocket, so there are no npm dependencies.
//
// Needs: Node 22+, Go, chromium or google-chrome. Exits non-zero when a check fails.

import { execFileSync, spawn } from "node:child_process";
import { cpSync, existsSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const repo = join(dirname(fileURLToPath(import.meta.url)), "..");
const appPort = 8097;
const devtoolsPort = 9334;
const deckURL = `http://localhost:${appPort}/dev-key/`;
const api = `${deckURL}root/api`;
const presenterKey = "browser-check-key";
const appBinary = process.platform === "win32" ? "app.exe" : "app";

const work = mkdtempSync(join(tmpdir(), "browser-check-"));
const processes = [];
let failures = 0;

function cleanUp() {
	for (const child of processes) child.kill();
	// Chrome may still be writing its profile for a moment after being stopped; retry, and never let
	// cleanup turn passing checks into a failure.
	try {
		rmSync(work, { recursive: true, force: true, maxRetries: 10, retryDelay: 200 });
	} catch (error) {
		console.warn(`could not remove ${work}: ${error.message}`);
	}
}
process.on("SIGINT", () => {
	cleanUp();
	process.exit(130);
});

function check(name, passed, details) {
	console.log(`${passed ? "PASS" : "FAIL"}  ${name}${details ? `  ${JSON.stringify(details)}` : ""}`);
	if (!passed) failures++;
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function waitFor(url) {
	for (let attempt = 0; attempt < 80; attempt++) {
		try {
			await fetch(url);
			return;
		} catch {
			await sleep(250);
		}
	}
	throw new Error(`${url} did not come up`);
}

// Finds Chrome or Chromium on Linux, macOS and Windows without a shell (Windows has no `command -v`).
// Google Chrome first: Ubuntu's chromium can be a snap, which cannot use a profile directory in /tmp.
// BROWSER_PATH overrides the search.
function findBrowser() {
	if (process.env.BROWSER_PATH) return process.env.BROWSER_PATH;
	const names = ["google-chrome-stable", "google-chrome", "chromium", "chromium-browser"];
	const extensions = process.platform === "win32" ? [".exe", ""] : [""];
	for (const directory of (process.env.PATH ?? "").split(delimiter)) {
		for (const name of names) {
			for (const extension of extensions) {
				const candidate = join(directory, name + extension);
				if (existsSync(candidate)) return candidate;
			}
		}
	}
	const knownInstallPaths = [
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
	];
	const installed = knownInstallPaths.find((path) => existsSync(path));
	if (installed) return installed;
	throw new Error("Chrome or Chromium is required");
}

// The presenter is a plain HTTP client: log in, keep the session cookie, move the deck.
async function presenterSession() {
	const login = await fetch(`http://localhost:${appPort}/presenter/login`, {
		method: "POST",
		body: new URLSearchParams({ key: presenterKey }),
		redirect: "manual",
	});
	const cookie = login.headers.getSetCookie().map((header) => header.split(";")[0]).join("; ");
	const moveToSlide = (index) => fetch(`${api}/slides/set?index=${index}`, { method: "POST", headers: { cookie } });
	// What an attendee sees: only the active slide. (The presenter's page also previews every slide.)
	const attendeeHTML = async () => (await fetch(deckURL)).text();
	// Steps through the deck until a slide matching the pattern is shown, like a presenter clicking "next",
	// so the check does not depend on slide order.
	const moveToSlideWith = async (pattern) => {
		for (let index = 0; index < 100; index++) {
			await moveToSlide(index);
			const html = await attendeeHTML();
			if (pattern.test(html)) return html;
			const shown = Number(/slideIndex:\s*(\d+)/.exec(html)?.[1]);
			if (shown < index) break;
		}
		throw new Error(`no slide matches ${pattern}`);
	};
	const startTugOfWar = () => fetch(`${api}/tug-of-war/start`, { method: "POST", headers: { cookie } });
	return { moveToSlideWith, startTugOfWar };
}

// One browser tab, like one attendee's phone. reducedMotion emulates the "reduce motion" accessibility setting.
// touch emulates a phone (a coarse pointer), which the snow globe needs before it listens to the motion sensor.
async function openTab(url, { reducedMotion = false, touch = false } = {}) {
	const target = await (await fetch(`http://127.0.0.1:${devtoolsPort}/json/new?${url}`, { method: "PUT" })).json();
	const socket = new WebSocket(target.webSocketDebuggerUrl);
	await new Promise((resolve) => socket.addEventListener("open", resolve));
	let nextId = 1;
	const pending = new Map();
	const problems = [];
	socket.addEventListener("message", (event) => {
		const message = JSON.parse(event.data);
		if (message.id && pending.has(message.id)) {
			pending.get(message.id)(message);
			pending.delete(message.id);
		}
		if (message.method === "Runtime.consoleAPICalled" && ["error", "warning"].includes(message.params.type)) {
			problems.push(message.params.args.map((arg) => arg.value ?? arg.description).join(" "));
		}
		if (message.method === "Runtime.exceptionThrown") {
			problems.push(message.params.exceptionDetails.exception?.description ?? message.params.exceptionDetails.text);
		}
	});
	const send = (method, params = {}) =>
		new Promise((resolve) => {
			const id = nextId++;
			pending.set(id, resolve);
			socket.send(JSON.stringify({ id, method, params }));
		});
	await send("Runtime.enable");
	// Simulate a phone on busy venue wifi. On localhost every request finishes in about a millisecond,
	// so rapid taps never overlap and request-cancellation bugs stay invisible.
	await send("Network.enable");
	await send("Network.emulateNetworkConditions", { offline: false, latency: 200, downloadThroughput: -1, uploadThroughput: -1 });
	if (reducedMotion) {
		await send("Emulation.setEmulatedMedia", { features: [{ name: "prefers-reduced-motion", value: "reduce" }] });
	}
	if (touch) {
		await send("Emulation.setTouchEmulationEnabled", { enabled: true, maxTouchPoints: 5 });
	}
	if (reducedMotion || touch) {
		await send("Page.reload");
	}
	await sleep(2000); // let Datastar connect its SSE streams
	return {
		problems,
		run: async (javascript) => {
			const response = await send("Runtime.evaluate", { expression: javascript, awaitPromise: true, returnByValue: true });
			if (response.result?.exceptionDetails) throw new Error(response.result.exceptionDetails.exception?.description);
			return response.result.result.value;
		},
		close: () => socket.close(),
	};
}

try {
	execFileSync("go", ["tool", "templ", "generate", "-log-level", "error"], { cwd: repo, stdio: "inherit" });
	execFileSync("go", ["build", "-o", join(work, appBinary), "."], { cwd: repo, stdio: "inherit" });
	cpSync(join(repo, "static"), join(work, "static"), { recursive: true });

	processes.push(spawn(join(work, appBinary), { cwd: work, env: { ...process.env, PORT: String(appPort), PRESENTER_KEY: presenterKey }, stdio: "ignore" }));
	const browserArguments = ["--headless=new", "--disable-gpu", `--user-data-dir=${join(work, "profile")}`, `--remote-debugging-port=${devtoolsPort}`];
	// GitHub's Ubuntu runners block Chrome's sandbox (AppArmor), and Chrome exits at once. Only on CI.
	if (process.env.CI) browserArguments.push("--no-sandbox");
	const browserPath = findBrowser();
	console.log(`browser: ${browserPath}`);
	const browser = spawn(browserPath, [...browserArguments, "about:blank"], { stdio: ["ignore", "ignore", "pipe"] });
	let browserErrors = "";
	browser.stderr.on("data", (chunk) => (browserErrors += chunk));
	processes.push(browser);
	await waitFor(deckURL);
	try {
		await waitFor(`http://127.0.0.1:${devtoolsPort}/json/version`);
	} catch (error) {
		throw new Error(`${error.message}; the browser said:\n${browserErrors.slice(-2000)}`);
	}

	const presenter = await presenterSession();
	const phone = await openTab(deckURL);

	// Every smash counts: a burst of clicks must all reach the server and all show a balloon.
	const smashes = 10;
	const afterSmashing = await phone.run(`(async () => {
		const thumbsUp = document.querySelector(".emoji-balloon-button");
		// Count balloons as they arrive: each one floats off screen and removes itself after a while.
		let balloons = 0;
		new MutationObserver((changes) => {
			for (const change of changes) for (const node of change.addedNodes) if (node.nodeName === "EMOJI-BALLOON") balloons++;
		}).observe(document.body, { childList: true });
		for (let i = 0; i < ${smashes}; i++) {
			thumbsUp.click();
			await new Promise((resolve) => setTimeout(resolve, 50)); // a fast thumb: 20 taps per second
		}
		await new Promise((resolve) => setTimeout(resolve, 2500));
		return { shown: Number(thumbsUp.querySelector("span").textContent), balloons };
	})()`);
	check(`${smashes} fast 👍 clicks are all shown`, afterSmashing.shown === smashes, afterSmashing);
	check(`${smashes} fast 👍 clicks each give a balloon`, afterSmashing.balloons === smashes, afterSmashing);

	// Balloons fly, except for people who turned on "reduce motion": then a balloon appears at its button,
	// stays still, and disappears, so every smash is still seen.
	const balloonMotion = `(async () => {
		const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
		const appeared = new Promise((resolve) => new MutationObserver((changes, observer) => {
			for (const change of changes) for (const node of change.addedNodes) {
				if (node.nodeName === "EMOJI-BALLOON") { observer.disconnect(); resolve(node); }
			}
		}).observe(document.body, { childList: true }));
		document.querySelector(".emoji-balloon-button").click();
		const balloon = await appeared;
		await sleep(100);
		const firstTop = balloon.getBoundingClientRect().top;
		await sleep(400);
		const laterTop = balloon.getBoundingClientRect().top;
		await sleep(900);
		return { moved: Math.abs(laterTop - firstTop) > 1, stillThereAfterTwoSeconds: balloon.isConnected };
	})()`;
	const normalMotion = await phone.run(balloonMotion);
	check("balloons fly up", normalMotion.moved === true, normalMotion);

	// Live update: the presenter moves the deck and the open phone follows without a reload.
	await presenter.moveToSlideWith(/class="slide poll"/);
	await sleep(1500);
	const showsPoll = await phone.run(`document.querySelector(".poll-option") !== null`);
	check("an open phone follows the presenter to the poll slide", showsPoll === true);

	// Voting by tapping an option.
	const afterVoting = await phone.run(`(async () => {
		const option = () => [...document.querySelectorAll(".poll-option")].find((button) => button.textContent.includes("Smash the Halls"));
		option().click();
		await new Promise((resolve) => setTimeout(resolve, 2500));
		return { pressed: option().getAttribute("aria-pressed") };
	})()`);
	check("a tapped poll option shows as selected", afterVoting.pressed === "true", afterVoting);
	const resultsHTML = await presenter.moveToSlideWith(/data-poll-results="app-name"/);
	const counted = /data-option-key="smash-the-halls"\s+data-vote-count="1"/.test(resultsHTML);
	check("the tapped vote is counted in the results", counted);

	// Tug of war: the presenter starts a round; after the 5 second countdown, emoji smashes pull for the
	// phone's team and the rope moves through the live updates.
	await presenter.moveToSlideWith(/data-tug-of-war-phase/);
	await sleep(1500);
	await presenter.startTugOfWar();
	await sleep(5500);
	const tugOfWar = await phone.run(`(async () => {
		const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
		const smashButton = document.querySelector(".emoji-balloon-button");
		for (let i = 0; i < 10; i++) {
			smashButton.click();
			await sleep(50);
		}
		await sleep(1500);
		const slide = document.querySelector(".tug-of-war");
		const ownPulls = Number(slide.dataset.team === "teal" ? slide.dataset.tealPulls : slide.dataset.orangePulls);
		return { phase: slide.dataset.tugOfWarPhase, team: slide.dataset.team, ownPulls, rope: slide.style.getPropertyValue("--rope").trim() };
	})()`);
	check("10 emoji smashes during a tug of war count 10 pulls for your team", tugOfWar.ownPulls === 10, tugOfWar);
	check("the rope moves toward the team that pulls", tugOfWar.rope !== "" && tugOfWar.rope !== "0.000", tugOfWar);

	// Someone else shakes (a plain request, its own session): it snows on this phone, and the snowfall
	// removes itself when it has fallen, through its Datastar data-init, with no component.
	await fetch(`${api}/snow/shake`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ shakes: 3 }) });
	const snowHere = await phone.run(`(async () => {
		const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
		await sleep(1500);
		const snowfalls = document.querySelectorAll(".snowfall").length;
		await sleep(17500); // a snowfall stays 18 s, so the slowest flake finishes falling
		return { snowfalls, snowfallsAfterFalling: document.querySelectorAll(".snowfall").length };
	})()`);
	check("another phone's shakes make it snow here", snowHere.snowfalls === 1, snowHere);
	check("the snowfall cleans itself up when it has fallen", snowHere.snowfallsAfterFalling === 0, snowHere);

	check("no console errors or exceptions in the phone tab", phone.problems.length === 0, phone.problems);
	phone.close();

	// Snow globe: shaking a phone shows puffs of snow on it at once, and when the shaking stops, the other
	// screens get a snowfall with the energy of the shakes. "Another screen" is a plain stream reader here,
	// because a second browser tab would hide the first, and Datastar closes the stream of a hidden page.
	const otherScreen = await fetch(`${api}/snow/sse`);
	const otherScreenReader = otherScreen.body.getReader();
	let otherScreenText = "";
	const otherScreenSnow = (async () => {
		const decoder = new TextDecoder();
		while (!/class="snowfall"/.test(otherScreenText)) {
			const { value, done } = await otherScreenReader.read();
			if (done) return;
			otherScreenText += decoder.decode(value);
		}
	})();
	const shakingPhone = await openTab(deckURL, { touch: true });
	const shaking = await shakingPhone.run(`(async () => {
		const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
		const shake = () => window.dispatchEvent(new DeviceMotionEvent("devicemotion", {
			acceleration: { x: 20, y: 0, z: 0 }, accelerationIncludingGravity: { x: 20, y: 0, z: 9.8 },
			rotationRate: { alpha: 0, beta: 0, gamma: 0 }, interval: 16,
		}));
		const buttonBeforeShaking = document.querySelector(".snow-globe-enable") !== null;
		shake();
		await sleep(50);
		const puffsRightAfterFirstShake = document.querySelectorAll(".snowfall").length;
		const buttonAfterFirstShake = document.querySelector(".snow-globe-enable") !== null;
		for (let i = 0; i < 4; i++) {
			await sleep(250);
			shake();
		}
		return { buttonBeforeShaking, buttonAfterFirstShake, puffsRightAfterFirstShake, puffsAfterFiveShakes: document.querySelectorAll(".snowfall").length };
	})()`);
	check("shaking a phone shows a puff of snow on it at once, one per shake", shaking.puffsRightAfterFirstShake === 1 && shaking.puffsAfterFiveShakes === 5, shaking);
	// Chrome has requestPermission like iOS but sends motion events without asking (already on page load), so no
	// "Shake for snow" button may be left. On an iPhone it stays until tapped; only a real iPhone can check that.
	check("a phone that sends motion events gets no 'Shake for snow' button", shaking.buttonAfterFirstShake === false, shaking);
	await Promise.race([otherScreenSnow, sleep(3000)]);
	await otherScreenReader.cancel();
	const otherScreenEnergy = Number(/class="snowfall" style="--energy: (\d+);"/.exec(otherScreenText)?.[1]);
	check("five shakes make it snow on the other screens with the energy of five shakes", otherScreenEnergy === 5, { otherScreenEnergy });
	check("no console errors or exceptions in the shaking phone's tab", shakingPhone.problems.length === 0, shakingPhone.problems);
	shakingPhone.close();

	// Last, because a second tab hides the first one, and Datastar closes the live stream of a hidden page.
	const calmPhone = await openTab(deckURL, { reducedMotion: true });
	const reducedMotion = await calmPhone.run(balloonMotion);
	check("with reduce motion, a balloon stays still and disappears", reducedMotion.moved === false && reducedMotion.stillThereAfterTwoSeconds === false, reducedMotion);
	check("no console errors or exceptions in the reduce-motion tab", calmPhone.problems.length === 0, calmPhone.problems);
	calmPhone.close();
} catch (error) {
	console.error(`browser check could not run: ${error.message}`);
	failures++;
} finally {
	cleanUp();
}

console.log(failures === 0 ? "browser check passed" : `browser check failed: ${failures} problem(s)`);
process.exit(failures === 0 ? 0 : 1);
