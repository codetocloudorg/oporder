# Changelog

All notable changes to this project are documented here, in [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/) format, per SPEC.md §13.1.

Nothing is tagged or released yet — this file tracks what's landed on `main` since the
project's first commit, ahead of the first real version number.

## [Unreleased]

### Fixed
- npm placeholder package had no `bin` entry at all — `npm install -g oporder` never created
  an `oporder` command on PATH, and the honest "no binary yet" message only printed with
  `--foreground-scripts` (current npm hides postinstall output by default), so a real,
  default `npm install -g oporder` run showed nothing and left nothing runnable. Added
  `bin/oporder.js`, registered in `package.json`, sharing its message with `postinstall.js`
  via `lib/message.js`; exits non-zero so it's never mistaken for success. Found and fixed by
  actually running the documented install command, not by reading the source and assuming it
  worked.
- Homebrew tap's placeholder formula crashed instead of showing its own message: a
  nonexistent-tag `url` left `version` unparseable on current Homebrew ("invalid attribute for
  formula: version (nil)"), and even after fixing that, the same nonexistent URL 404's during
  Homebrew's fetch step, which runs before `install` — so the formula's own honest `odie`
  message was unreachable either way. Fixed by pointing `url`/`sha256` at a real, immutable
  commit snapshot instead of a fake tag. Also documented the `brew trust` step current
  Homebrew requires for any third-party tap, found the same way — by actually running
  `brew install oporder` from a clean state instead of assuming the documented commands
  worked.

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
