# Where things live

A feature keeps its files together:

```text
pages/root/poll.templ     markup and Go
pages/root/poll.css       styles for the poll
poll_voting_test.go       behaviour tests
slides/stack.md           a slide like this one
schema.sql                the whole database
```

Notes:

- Group by feature, not by technical role: everything about the poll sits side by side.
- AGENTS.md has the rules for people and AI agents alike. Read it before your first PR.
- `go tool task start` runs it all locally with hot reload.
