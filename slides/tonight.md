# How tonight works

1. Tell Gerhard your GitHub username to get push access
2. Make a branch and keep the change small
3. Open a pull request; it gets its own preview link
4. We merge it live, straight onto the big screen 🎉

`go tool task check` before you push. Ask questions, help each other, make us laugh.

Notes:

- Add people with `gh api -X PUT repos/frontiers-bergen/frontiers-juleavslutning-2026/collaborators/<username> -f permission=push` (`deploy/README.md`).
- No push access? A pull request from a fork works too, just without a preview.
- Small pull requests, one idea each. Test the behaviour, not the implementation.
- AI-written code is welcome; the same rules apply to it. If it makes the juleavslutning more fun, it's in scope.
