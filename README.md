# OpOrder

**Vendor-neutral assessment for cloud migration and application modernization.**

OpOrder reads your code and your live infrastructure, tells you honestly what's actually
there, and hands you a plan — without an opinion about which cloud you should end up on.

## Why

Every existing agentic migration tool is built by a cloud provider with a stake in the
answer. AWS Transform recommends AWS. Azure's tooling recommends Azure. That's not a flaw
in those tools — it's their business model. OpOrder has no cloud to sell, so it can say
things they structurally can't:

- "Retain this. Migrating it isn't worth it."
- "Repatriate this to on-prem — it's cheaper there."
- "Retire this. Nobody uses it."

A recommendation that costs the tool nothing to make is more trustworthy. That's the whole
premise.

This applies just as much when nothing is moving providers at all — modernizing a legacy
.NET Framework monolith or containerizing a VM-based app, in place, is exactly as much
OpOrder's job as a cross-cloud migration is.

## What it produces: an OpOrder

Named after the military planning document, a real OpOrder has three parts, and so does
this tool's output:

| Section | What it means here |
|---|---|
| **Situation** | The honest as-is: architecture diagram, dependency map, documentation — generated from your actual code and live infrastructure, not from what the wiki claims |
| **Mission** | The recommendation: rehost, replatform, refactor, rearchitect, repurchase, retire, or retain — per workload, with the reasoning shown |
| **Execution** | Cost estimate, effort estimate, and sequencing — everything needed to hand to a build team, yours or anyone else's |

## What it deliberately doesn't do

OpOrder assesses and recommends. It does not execute. The moment the tool that makes the
recommendation also profits from carrying it out, the recommendation stops being
trustworthy — so OpOrder stops at the plan. If you want the migration or modernization work
actually done, that's a separate, explicitly-scoped engagement with whoever you choose,
[Code To Cloud](https://codetocloud.io) included.

## Try it right now

No released binary yet, but the reasoning engine is real and working today. Three commands,
copy-paste, nothing to configure:

```
git clone https://github.com/codetocloudorg/oporder.git
cd oporder
go run ./cmd/sample-report
```

That runs the actual 5/7-Rs rubric and debt-delta engine against five illustrative sample
workloads and prints real, computed JSON — the genuine reasoning, on fixture data. Requires
[Go](https://go.dev/dl/) installed, nothing else.

**`oporder scan` is real too, and needs no cloud account to be useful:**

```
go run ./cmd/oporder scan
```

Run from any repo, this detects workload boundaries in the code (one per `Dockerfile`, one
per Go module's `cmd/*/main.go`, etc. — see [`docs/rubric.md`](docs/rubric.md) for the
reasoning model these feed, and SPEC.md §5.0 for the detection rules themselves) and writes a
real `SITUATION.md`, including a Mermaid diagram of what it found.

**Add a real AWS, Azure, GCP, or Cloudflare account** and it also pulls live inventory and
correlates it against the code — matched, unmatched-on-either-side, all shown, never guessed:

```
export AWS_REGION=us-east-1                          # uses your existing AWS credentials
export OPORDER_AZURE_SUBSCRIPTION_ID=<subscription>   # uses your existing `az login` session
export OPORDER_GCP_PROJECT_ID=<project>               # uses Application Default Credentials
echo "<cloudflare-token>" > ~/.cloudflare_token       # from dash.cloudflare.com/profile/api-tokens

go run ./cmd/oporder scan
```

Any subset works; skip what you don't have. A correlated workload with real utilization data
(AWS CloudWatch or Azure Monitor) or a detected proprietary-SDK dependency (any provider) also
gets a genuine, partial-evidence 5/7-Rs call, written to `MISSION.md` — see
[`docs/rubric.md`](docs/rubric.md) for exactly what evidence is real today versus what's still
missing (see SPEC.md §10's M2 for what's left).

## Status

Early. Building in public. No released binary yet, but `oporder scan` genuinely works end to
end today: code workload-boundary detection, proprietary-dependency scanning, live read-only
connectors for AWS, Azure, GCP, and Cloudflare, tag/name correlation between code and
infrastructure, and a real, partial-evidence 5/7-Rs call backed by live utilization data
(AWS/Azure) and detected dependencies (any provider). See [`docs/rubric.md`](docs/rubric.md)
for how that call gets made, or SPEC.md §10 for the full milestone-by-milestone status, including
what's still missing (GCP's connector is unverified against a real account; Well-Architected
scoring, the TUI, and the HTML report don't exist yet). Follow along, open an issue if you
want to help shape where it goes, or drop into [Code To Cloud's
Discord](https://discord.gg/vwfwq2EpXJ) if you'd rather talk it through than write it up.

The name is reserved everywhere it needs to be ahead of a real release — the domain
([oporder.dev](https://oporder.dev)), [npm](https://www.npmjs.com/package/oporder)
(`oporder@0.0.1`, an honest placeholder — `npm install -g oporder` today just tells you
there's nothing to install yet), and a [Homebrew
tap](https://github.com/codetocloudorg/homebrew-tap) (`brew tap codetocloudorg/tap && brew
install oporder` — same honesty, it fails with a clear message and a link to the working
`sample-report` command rather than pretending to install a binary that doesn't exist).
GitHub Releases will be the primary way to get the binary once one exists; npm and Homebrew
both become real installers at the same time, for anyone who'd rather use the package manager
they already have than download a release directly.

## Planned architecture

- An **MCP server** for read-only cloud inventory, cost, and Well-Architected-style checks —
  connects to your account, sends nothing anywhere you haven't chosen.
- A **skill** encoding the assessment methodology — the Rs reasoning, SDLC-maturity scoring,
  what a good architecture doc looks like.
- A **workflow** that fans the assessment out per service, verifies findings with a fresh
  reviewer, and synthesizes the report.

## License

Apache License 2.0 — see [LICENSE](LICENSE).

---

Built by [Code To Cloud](https://codetocloud.io) —
[Discord](https://discord.gg/vwfwq2EpXJ) ·
[GitHub](https://github.com/codetocloudorg) ·
[Podcast](https://open.spotify.com/show/1iOZfFVamUk7CJPOvtU00v)
