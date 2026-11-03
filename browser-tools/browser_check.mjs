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
	rmSync(work, { recursive: true, force: true });
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
function findBrowser() {
	const names = ["chromium", "chromium-browser", "google-chrome-stable", "google-chrome"];
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
	return { moveToSlideWith };
}

// One browser tab, like one attendee's phone.
async function openTab(url) {
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
	processes.push(
		spawn(findBrowser(), ["--headless=new", "--disable-gpu", `--user-data-dir=${join(work, "profile")}`, `--remote-debugging-port=${devtoolsPort}`, "about:blank"], { stdio: "ignore" }),
	);
	await waitFor(deckURL);
	await waitFor(`http://127.0.0.1:${devtoolsPort}/json/version`);

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

	check("no console errors or exceptions in the phone tab", phone.problems.length === 0, phone.problems);
	phone.close();
} catch (error) {
	console.error(`browser check could not run: ${error.message}`);
	failures++;
} finally {
	cleanUp();
}

console.log(failures === 0 ? "browser check passed" : `browser check failed: ${failures} problem(s)`);
process.exit(failures === 0 ? 0 : 1);
