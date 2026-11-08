# How to contribute

- **Cohesion:** what changes together stays together; your feature keeps its files side by side
- **Low coupling:** your change should not break anyone else's
- **Small pull requests,** one idea each; `go tool task check` before you push
- **Merge conflicts will happen:** we all add to the same few lists; pull `main` often, and ask if you get stuck
- **Don't be the Grinch:** there are a hundred ways to ruin the evening for everybody. We can't stop you. Please don't.

Notes:

- The rules for people and AI agents alike are in AGENTS.md: keep it simple, test the behaviour in given-when-then form.
- The shared lists: the slide order (`pages/root/active_slide.templ`), the routes (`pages/root/root_index.templ`) and
  the database (`schema.sql`). A conflict there is usually two additions next to each other: keep both.
- There is no technical gatekeeping: with push access, your change reaches the big screen when it is merged. We trust you.
- AI-written code is welcome; the same rules apply to it.
