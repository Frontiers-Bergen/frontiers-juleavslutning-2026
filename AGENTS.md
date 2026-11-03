# Agent Instructions

Read the **Why** section in `README.md` first. Every change should serve it: the people at the Frontiers juleavslutning having fun.
The README also holds the checklist of planned work and the non-goals.

## Engineering principles

The goal while coding is to create a maintainable, high-quality solution that is easy to understand, change, and verify.

- Keep in mind good engineering practices such as KISS and YAGNI.
- Aim for low coupling, high cohesion and separation of concerns. Code that changes together should stay together.
- Avoid adding new dependencies without a clear benefit.
- Prefer clear names, straightforward control flow, and familiar project conventions.
- Complexity is our enemy. Avoid accidental complexity.
- Be pragmatic. If the rules conflict, use your judgment to decide which is more important for the current change.

### Abstraction and duplication

Do not prematurely abstract. Only share code when it represents the same underlying concept or business rule.
Never duplicate a business rule; duplicate incidental code freely until the third copy (Martin Fowler's rule of three).

### Avoid primitive obsession

Give domain concepts their own types instead of passing raw strings and ints around.
A named type (`type InviteKey string`, `type PollKey string`, `type SessionID string`) lets the compiler catch mixed-up arguments,
gives the concept a home for its rules, and makes function signatures read like the domain.
Parse and validate at the boundary (HTTP request, database row) and pass the typed value inward.
Use a named type when a value has rules or is easily confused with another value of the same primitive type; do not wrap values that are just data.

### Small changes

Strive for small, cohesive changes. Solve one problem completely and keep unrelated cleanup separate.
Small changes are easier to review, understand, and roll back. Work towards the smallest coherent change.

## Working with the user

- First inspect the relevant code and tests, then discuss.
- Ask clarifying questions, or interview the user, when the request is unclear.
- Always share your thoughts and recommendations.
- Challenge the user's assumptions when you see a better approach.
- Do not over-engineer. Discuss with the user before adding accidental complexity.

## Pull request workflow

Every feature follows three steps:

1. **Tidy first.** Before adding the feature, look at the code it will touch. If a refactor or improvement makes the feature easier to add, do that first, without changing behavior.
2. **Add the feature.** The behavior change, with tests.
3. **Refactor before finalizing.** Clean up what adding the feature revealed, again without changing behavior.

Keep structural changes (tidying, refactoring) and behavior changes apart so each can be reviewed on its own terms.
For small tidyings, separate commits in one pull request are enough.
When a step is big enough to deserve its own review, use stacked pull requests (GitHub stacked PRs, `gh stack` CLI extension): each step is one PR based on the previous, merged bottom-up.
Skip steps that are not needed; an empty tidy or refactor step is fine.

Run `go tool task check` before opening a pull request: formatting, vet, `go fix`, golangci-lint and tests. Fix with `go tool task fmt` where it applies.
For UI changes, run `go tool task screenshots` before and after the change and look at the result; for a pure refactor, the screenshots should be pixel-identical.

## Comments and documentation

Comments and documentation are primarily written for other agents.
Add them where they explain purpose, business rules, constraints, and non-obvious decisions — context that cannot easily be read from the code.
Do not repeat implementation details. Update existing comments and documentation when behavior changes.

Document features with a user story focused on one user goal.
The word "and" can indicate that a story contains unrelated goals, but it is not a problem by itself; keep related behavior together when it serves one coherent goal.
Use other documentation formats when a user story is not a good fit.

The code is the source of truth for what the system currently does. Requirements and acceptance criteria describe what it should do.
If the code, tests, and documentation disagree, point out the discrepancy.
Do not assume the current implementation is correct, and do not change tests or documentation just to match it.

## Testing

Test behavior, not implementation. Verify meaningful outcomes, important failure cases, and regressions.
A test should keep passing when implementation details change and the behavior stays the same.

Write tests in given-when-then form so they read from intent to verification:

- Start each test with three lines of BDD metadata describing the test in plain words: Given some initial context, When an event occurs, Then the expected outcome.
  Use the project's BDD helper, `bdd.Behavior` in `bdd/bdd.go`. The metadata is written in English.
- Split the body into three visible sections: `// Given`, `// When`, `// Then`.
- Start the Given section with the expected result, so the test's purpose is clear before the setup details.
- Keep each test to one behavior and give it a specific name. Split creation, idempotency, validation, and error cases into separate tests.
- Move noisy setup and repeated assertions into local test helpers when that makes the test read like behavior.

```go
func TestPollResults_NoVotes_EveryOptionHasZeroVotes(t *testing.T) {
	bdd.Behavior(t, bdd.BDD{
		Given: "Given a poll nobody has voted on.",
		When:  "When the results are counted.",
		Then:  "Then every option has zero votes.",
	})

	// Given
	expected := map[string]int{"yes": 0, "no": 0}
	db := openTestDatabase(t, filepath.Join(t.TempDir(), "presentation.db"))
	poll := createPoll(t, db, "yes", "no")

	// When
	actual := countVotes(t, db, poll)

	// Then
	assertVoteCounts(t, expected, actual)
}
```

Tests exist to let agents change the code safely, not to reach a coverage number.
Prefer tests at feature boundaries with real dependencies (a real SQLite file in `t.TempDir()`, real embedded NATS) over mocks.
Run them with `go tool task test`. `database_test.go` is a working example.
HTTP-level tests use the harness in `app_test.go`: `startTestApp` runs the real router from `newApp` with a fresh database,
and each attendee or presenter is an `http.Client` with its own cookies. `slide_control_test.go` is a working example.

## Go

- Handle every error explicitly. Never ignore an error; make failure paths obvious.
- Prefer concrete types. Introduce interfaces only at the point of use, when they are needed.
- Avoid packages named `helpers`, `utils` or `common`; put code next to the feature that uses it.
- Prefer explicit parameters over hidden state.

## Project conventions

- Stack: Go, templ, Datastar (SSE), embedded NATS JetStream KV for live updates, SQLite.
- Group code by feature, not by technical role. A simple feature can be a single file; a feature with several related files gets its own folder.
- Develop mobile first. Attendees use their phones as controllers, so every feature must work well on a small touch screen, one-handed.
  The big screen is the shared stage; design for both, starting with the phone.
- Styling is plain CSS. No Tailwind or other CSS frameworks; the project started from the northstar example and its Tailwind leftovers were removed on purpose.
- Shared CSS (design tokens, element defaults, app layout) lives in `static/index.css`, which declares the layers `tokens, base, layout, features`.
  A feature's CSS lives in a `.css` file next to its `.templ` file (`pages/root/poll.css` next to `poll.templ`), nested under the feature's
  root class (`.poll { ... }`), so features cannot break each other. Do not wrap it in `@layer`: `/feature-styles.css` imports every
  feature file into `layer(features)`, which always wins over the shared layout. Reuse the tokens (`var(--space-2)`, `var(--radius)`, ...).
  The files are embedded and served by `feature_styles.go`; a new folder with CSS needs a `go:embed` pattern there (a test fails until it is added).
  No `<style>` blocks in templates, and no inline styles except to pass dynamic values as custom properties (`style="--share: 42%"`).
- Write explicit SQL, keep it close to where it is used, and handle every database error.
- `schema.sql` is the single source of truth for the database and is applied on every startup, so every statement must be idempotent (`CREATE TABLE IF NOT EXISTS`, `INSERT OR IGNORE`).
  There are no migrations: changing an existing table means deleting the local database file. Schema changes are frozen on the juleavslutning evening.
- Data that must survive the evening (ideas, votes, feedback, emoji counts) lives in SQLite, not in NATS KV; KV is for live updates and has a TTL.
- Live updates go through embedded NATS. A KV watcher delivers the latest value, so bursts merge into one update: fine for state like "current slide".
  When every single event matters (one balloon per emoji smash), publish each event as a core NATS message instead.
- Keep templates close to their data access. If a feature both reads and writes, keep both in the same file or feature folder.
- Slide order is defined in `pages/root/active_slide.templ`; slide content lives in `slides/`.
- Domain terms are listed in `domain_dictionary.md` and are used unchanged everywhere: code, database, UI and docs.
  Everything else is in English. Frontiers runs its events in English, so the app is English only (no bilingual frontend).

## Generated files

Do not edit `*_templ.go` files. Edit the `.templ` source and run `go tool templ generate` (or `task build:templ`).

templ reference for LLMs: https://templ.guide/llms.md
