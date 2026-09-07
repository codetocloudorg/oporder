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

## Status

Early. Building in public. No released binary yet — this repo currently holds the shape of
the idea, not the tool. Follow along, open an issue if you want to help shape where it goes,
or drop into [Code To Cloud's Discord](https://discord.gg/vwfwq2EpXJ) if you'd rather talk it
through than write it up.

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
