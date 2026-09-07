# Contributing to OpOrder

The short version: if you had to read more than this one file to send a real PR, that's a
bug in the project, not in you. Open an issue.

**Found a security issue** — a credential leak, a prompt-injection path, anything that could
expose an account or a repo, not just a wrong recommendation? Stop, don't file it here — see
[SECURITY.md](SECURITY.md) for private disclosure instead.

**Want to talk it through before writing anything** — a design question, a "would this PR
even be welcome" check, or just thinking out loud? [Code To Cloud's
Discord](https://discord.gg/vwfwq2EpXJ) is faster than an issue for that kind of thing.

## Before you write code

Read [SPEC.md](SPEC.md) §1 (non-negotiable design principles) and §4.2 (complexity is
earned, not assumed). Everything else follows from those two sections. In particular:

- A recommendation must be able to reach "retain," "repatriate," or "retire" on real
  evidence. If your change makes the scoring lean toward a migration path by default,
  it won't be merged, however reasonable it looks in isolation.
- v0.1–v0.3 are a **plain Go CLI**, on purpose — no MCP servers, no skills, no workflow
  orchestration yet. Don't add that indirection ahead of schedule; see §4.2 for why.

## The easiest first contribution: add a provider check

Whatever the current architecture is at the time you're reading this (check §10's roadmap
table for the phase), "add a check for provider X" is designed to always be the simplest
path in. Look at `docs/architecture/` for the current pattern — it should read like a
worked example, not a spec you have to reverse-engineer from existing code.

## Rubric changes

Any change to the 5/7-Rs logic (§5.3), the Well-Architected normalization (§5.4), the SDLC
maturity dimensions (§5.5), the cost model (§5.6), or the technical debt delta model (§5.8)
needs:
1. The reasoning stated in plain language in the PR description — why this evidence should
   move the score, not just that it does.
2. A new eval fixture (§7) that would have caught the old behavior as wrong, if one doesn't
   already exist.

A rubric change with no fixture attached is a guess, not a fix.

## Reporting a wrong recommendation

This is the single most valuable kind of issue you can file. Include the repo/account shape
(anonymized is fine), what OpOrder said, and what should have been said instead. Every
confirmed wrong call becomes a permanent eval fixture — see SPEC.md §11's devil's-advocate
note on this being expected, not a failure state.

## Code style

Standard `gofmt`/`go vet` clean, plus `staticcheck` — no framework beyond the Go standard
library and the minimum needed for provider SDKs and MCP — see SPEC.md §1 on avoiding
unearned complexity. Held to the [Google Go Style
Guide](https://google.github.io/styleguide/go/) on top of `gofmt`, since `gofmt` settles
formatting but not naming, package structure, or error-handling conventions, and this project
would rather adopt a real, widely-reviewed standard than invent its own.

**Every exported function, type, and package has a doc comment** — Go's own convention,
enforced by `staticcheck`, not optional. Comments explain *why*, not *what* — the same
standard this document has held itself to throughout: a comment restating what the code
already says is dead weight; a comment explaining a non-obvious constraint or a workaround is
what actually helps the next person, including future-you.

**Repository layout** follows the [Standard Go Project
Layout](https://github.com/golang-standards/project-layout) — `cmd/` for the binary entry
point, `internal/` for code not meant to be imported by other projects, `pkg/` only if
something is genuinely meant to be reused externally. Not a Google or Apple internal
convention specifically — there isn't one publicly documented for Go — but the closest thing
the Go community has to a load-bearing consensus, which is the more honest bar to cite than
naming a company that doesn't actually publish a Go layout standard.

## License

By contributing, you agree your contribution is licensed under the Apache License 2.0,
same as the rest of the repo.
