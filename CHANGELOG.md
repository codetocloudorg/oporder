# Changelog

All notable changes to this project are documented here, in [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/) format, per SPEC.md §13.1.

## [Unreleased]

Nothing yet.

## [0.1.0] — 2026-09-08

First real, tagged release. Six platform binaries (darwin/linux × amd64/arm64, plus Windows),
built and checksummed by the tested `.goreleaser.yaml` pipeline; `npm install -g oporder` and
`brew install oporder` both download and run the real binary.

### Added
- `internal/codescan` — workload-boundary detection per SPEC.md §5.0's three-tier method
  (deploy manifest / module boundary+entrypoint / ambiguous-flagged-not-guessed), including a
  Go-monorepo special case: a module with multiple `cmd/*/main.go` binaries is reported as one
  workload per binary, not one for the whole module.
- `internal/depscan` — proprietary-managed-dependency detection (AWS/Azure/GCP SDK matching
  across go.mod, package.json, requirements.txt, pyproject.toml). Needs no cloud account; a
  detected dependency alone rules out a Rehost call in the rubric.
- `internal/correlate` — tag-based and name-based matching between detected code workloads and
  live cloud resources, with explicit unmatched-on-both-sides reporting (§5.0 tiers 2-4).
- `internal/rubric` — the 5/7-Rs decision engine (§5.3): all eight trigger functions, the
  tie-break resolution order, and the telemetry-absence guard against a false Retire call.
- `internal/debtdelta` — projected technical-debt trajectory per R (§5.8).
- `internal/narrative` — a plain-English paragraph per detected workload, composed from the
  evidence the rest of the pipeline already gathered. No LLM call.
- Read-only live connectors for AWS, Azure, GCP, and Cloudflare (`internal/provider/*`), plus
  AWS CloudWatch and Azure Monitor CPU utilization as a real Retire-candidate signal.
- `oporder scan` — wires all of the above together. Always analyzes local code; adds live
  correlation, a Mermaid diagram, and a partial-evidence 5/7-Rs call when a cloud account is
  configured. Writes `SITUATION.md` and `MISSION.md`.
- `docs/rubric.md` — the 5/7-Rs and debt-delta logic in plain language, with real worked
  examples and a Mermaid decision tree.
- `.goreleaser.yaml` + `.github/workflows/release.yml` — the real release pipeline this version
  was built by.
- Real npm (`oporder`) and Homebrew (`codetocloudorg/tap`) installers, both downloading the
  actual GitHub Release binary for the current platform.
- CI pipeline (`.github/workflows/ci.yml`) — gofmt, `go vet`, `staticcheck`, build, and test on
  every push and PR.

### Fixed
- npm placeholder package had no `bin` entry at all — `npm install -g oporder` never created
  an `oporder` command on PATH. Found and fixed by actually running the documented install
  command, not by reading the source and assuming it worked.
- Homebrew tap's placeholder formula crashed instead of showing its own message: a
  nonexistent-tag `url` left `version` unparseable, and the same URL 404'd during Homebrew's
  fetch step, which runs before `install`. Also documented the `brew trust` step current
  Homebrew requires for any third-party tap. Both replaced entirely once v0.1.0 gave the
  formula a real release to point at.

### Known gaps (see SPEC.md §10 for the full milestone status)
- GCP's connector is unverified against a real account.
- Well-Architected scoring, security baseline, SDLC scoring, cost/effort estimation, the TUI,
  and the HTML report don't exist yet.
- Utilization gathering (the Retire signal) is AWS/Azure-only; GCP and Cloudflare aren't
  wired in, by deliberate choice — see SPEC.md §11 on why that stopped there.
