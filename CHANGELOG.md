# Changelog

All notable changes to this project are documented here, in [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/) format, per SPEC.md §13.1.

Nothing is tagged or released yet — this file tracks what's landed on `main` since the
project's first commit, ahead of the first real version number.

## [Unreleased]

### Added
- `internal/rubric` — the 5/7-Rs decision engine from SPEC.md §5.3: all eight trigger
  functions, the tie-break resolution order (Retain absent a forcing driver, then Retire,
  then narrower-evidence Rs, then lowest-effort-first), and the telemetry-absence guard
  against a false Retire call (§12). Full table-driven test coverage, including the exact
  tie-break example already stated in the spec's own prose.
- `cmd/oporder` — the CLI entry point. Honest stub: `version` and `help` work, `scan` reports
  not-implemented rather than pretending to run an assessment.
- CI pipeline (`.github/workflows/ci.yml`) — gofmt, `go vet`, `staticcheck`, build, and test
  on every push and PR, gating merge per §7.
- This changelog.
