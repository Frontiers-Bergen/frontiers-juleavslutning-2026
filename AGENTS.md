# Agent Instructions

Read the **Why** section in `README.md` first. Every change should serve it: the people at the Frontiers juleavslutning having fun.
The README also holds the checklist of planned work and the non-goals.

## Engineering principles

The goal while coding is to create a maintainable, high-quality solution that is easy to understand, change, and verify.

- Keep in mind good engineering practices such as KISS and YAGNI.
- Aim for low coupling, high cohesion and separation of concerns. Code that changes together should stay together.
- Avoid adding new dependencies without a clear benefit.
- Prefer clear names, straightforward control flow, and familiar project conventions.
- Do not add accidental complexity without discussing it with the user first. Complexity is our enemy.
- Be pragmatic. If the rules conflict, use your judgment to decide which is more important for the current change.

### Abstraction and duplication

Do not prematurely abstract. Only share code when it represents the same underlying concept or business rule.
Never duplicate a business rule; duplicate incidental code freely until the third copy (Martin Fowler's rule of three).

### Small changes

Strive for small, cohesive changes. Solve one problem completely and keep unrelated cleanup separate.
Small changes are easier to review, understand, and roll back. Work towards the smallest coherent change.

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
  Use the project's BDD helper when it has one; otherwise use three comment lines.
- Split the body into three visible sections: `// Given`, `// When`, `// Then`.
- Start the Given section with the expected result, so the test's purpose is clear before the setup details.
- Keep each test to one behavior and give it a specific name. Split creation, idempotency, validation, and error cases into separate tests.
- Move noisy setup and repeated assertions into local test helpers when that makes the test read like behavior.

```go
// Given a poll nobody has voted on.
// When the results are counted.
// Then every option has zero votes.
func TestPollResults_NoVotes_EveryOptionHasZeroVotes(t *testing.T) {
	// Given
	expected := map[string]int{"yes": 0, "no": 0}
	db := createTestDatabase(t)
	poll := createPoll(t, db, "yes", "no")

	// When
	actual := countVotes(t, db, poll)

	// Then
	assertVoteCounts(t, expected, actual)
}
```

## Go

- Handle every error explicitly. Never ignore an error; make failure paths obvious.
- Prefer concrete types. Introduce interfaces only at the point of use, when they are needed.
- Avoid packages named `helpers`, `utils` or `common`; put code next to the feature that uses it.
- Prefer explicit parameters over hidden state.

## Project conventions

- Stack: Go, templ, Datastar (SSE), embedded NATS JetStream KV for live updates, SQLite.
- Group code by feature, not by technical role. A simple feature can be a single file; a feature with several related files gets its own folder.
- Write explicit SQL, keep it close to where it is used, and handle every database error.
- Schema changes belong in `schema.sql`; never edit the runtime database file.
- Keep templates close to their data access. If a feature both reads and writes, keep both in the same file or feature folder.
- Slide order is defined in `pages/root/active_slide.templ`; slide content lives in `slides/`.
- The event is called **juleavslutning** in code and docs. Do not use "julebord".
- Events are run in English, so user-facing text and code are in English.

## Generated files

Do not edit `*_templ.go` files. Edit the `.templ` source and run `go tool templ generate` (or `task build:templ`).

templ reference for LLMs: https://templ.guide/llms.md
