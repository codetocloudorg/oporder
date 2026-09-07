# OpOrder — Specification v0.1

> Status: design document. Nothing in this repo executes yet. This is the plan the build
> gets held to, and it gets revised in the open as reality pushes back on it.

---

## 0. Vision

A developer runs one command against a codebase and a cloud account. Ten minutes later they
have an honest picture of what they're actually running, a straight answer on whether to
rehost, replatform, refactor, rearchitect, repurchase, retire, or retain it, what it will
cost in real dollars either way, and how long it will take. Nobody who built the tool makes
more money depending on which answer comes out.

That's the whole product. Everything below is how it gets built without quietly becoming
something else.

The point of building it isn't to hand organizations a different authority to defer to. It's
to put the engineer, the developer, the organization back in the driving seat — with a real,
evidenced choice about what's actually right for their situation, instead of the choice a
vendor with something to sell them already made on their behalf. "Free to choose" only means
something if the choice is backed by the same rigor a hyperscaler's own tooling has, which is
why §13 holds this project to real versioning, release, and governance discipline rather than
letting "it's open source" stand in for engineering seriousness.

---

## 1. Non-negotiable design principles

These aren't aspirations — they're constraints the architecture has to enforce mechanically,
not just promise in a README, because a promise a competitor can't structurally keep isn't a
differentiator, and neither is one you can quietly abandon under commercial pressure.

| Principle | What it rules out |
|---|---|
| **The recommendation must be able to cost the tool nothing.** "Retain," "repatriate," and "retire" have to be reachable outcomes, weighted identically to every migration path. | Any scoring logic that can't reach those three conclusions on real evidence is disqualified, full stop, before it ships. |
| **Assessment and execution stay structurally separate.** OpOrder produces the plan. It never runs the migration, writes the production code, or touches the client's deploy pipeline. | No "auto-fix" mode. No `oporder execute`. Ever. |
| **Every scoring decision is visible in the repo.** Anyone can read exactly why a workload got scored "replatform" instead of "refactor." | No opaque model call standing in for a documented rubric. If the LLM makes the call, the rubric it was given ships in the same PR. |
| **Nothing leaves the user's boundary they didn't approve.** Self-hosted by default. | No default telemetry, no default cloud callback, no "phone home for a nicer report." |
| **A worker never grades its own homework.** Every finding gets checked by a fresh-context agent before it reaches the report. | No self-verification loops. See [[08-Harness-and-Loops / 09-Agent-Design-Patterns / 20-Graph-Engineering]] pattern already validated in the Code To Cloud vault. |
| **A contributor should be able to submit a real PR after reading one file.** Architectural elegance that requires understanding four subsystems before anyone can help is a tax on exactly the community growth this project depends on. | No feature ships behind more indirection than it currently needs. See §4.2 — the fancier architecture is *earned*, not assumed from day one. |
| **Vendor-neutrality applies to the tooling ecosystem too, not just the cloud.** The plain CLI has to work with zero AI agent host installed. | No feature that only works inside one vendor's agent product (Claude Code, Cursor, etc.) is allowed to become load-bearing. See §4.3. |
| **Every recommendation shows pros and cons, never a bare verdict — and every surface looks considered, not default-tool ugly.** | No CLI output, TUI view, or report ships that only shows the "why we're right" side of a call, or that reads like an afternoon's unstyled output. See §6. |

---

## 2. What everyone else has already built, and what that tells us

Condensed from the full competitive research already done for this project — this is the
evidence the design below is built against, not a guess.

### 2.0 The proof this category works at real scale: Government of Alberta

Before the vendor landscape below, the single strongest piece of evidence for this whole
project, and it's local: in July 2026, Alberta's Ministry of Technology and Innovation used
**Claude Agent SDK + Claude Code (Opus and Sonnet)** to scan **466 million lines of code
across 27 provincial ministries — 1,280 applications, 3,400 repositories — in about 20
hours**, a task they estimate would otherwise take **6.5 years and roughly $2 billion**.
Some legacy systems were then rebuilt in **4–5 days** versus an original 5-month estimate.

Their architecture independently validates several of this spec's design choices, arrived at
separately:
- **A "red team / blue team" agent split** — one agent probes for exploitation paths, a
  separate one assesses defenses and drafts remediation — the same fresh-context,
  never-grade-your-own-homework discipline as §1's verification principle and §5.7, just
  named differently.
- **A two-stage pipeline**: a deterministic rules engine flags known patterns first, then an
  LLM does the judgment-call review with exact file/line citations — the same
  computational-vs-inferential split this spec applies throughout (§7's testing strategy is
  built on the identical distinction).
- **Mixed model tiers** (Opus for judgment, Sonnet for volume) — the same cheap-model-for-
  boring-nodes, strong-model-for-judgment-nodes pattern already validated in the Code To
  Cloud Agentic Engineering vault's Graph Engineering work.

Alberta published their methodology as **The Velocity Papers** (thevelocitywhitepapers.com)
— free, open, step-by-step technical white papers, explicitly offered as "a roadmap other
governments could follow." **What they did not publish is a reusable, installable tool** —
it's documentation of a bespoke effort built by a dedicated ministry team over roughly 18
months, not something a mid-market company or a solo engineer can `go install` and point at
their own estate. That gap — a proven, government-validated methodology with no productized
way for anyone else to run it — is exactly what OpOrder exists to close. The Velocity Papers
are required reading before finalizing §5's rubric design, not just a citation.

**The honest caveat**: Alberta's numbers reflect a provincial government's resourcing,
direct scale, and presumably direct vendor support — they are the upper bound of what's
possible with this category of tooling, not a baseline OpOrder v0.1 should imply it matches
on day one. Cite the pattern, not the throughput.

| Category | Who | What we take from them | What we deliberately don't repeat |
|---|---|---|---|
| Hyperscaler-native agentic migration | AWS Transform, Copilot app modernization, watsonx Code Assistant for Z | Real agentic pipelines work at scale (AWS Transform: 4.5B+ lines processed year one). Full assessment → plan → code pipelines are technically proven. | Every one only ever recommends its own cloud. Structurally can't be trusted for a neutral second opinion — this is the entire gap OpOrder exists to fill. |
| Closed enterprise modernization | Blitzy ($1.4B valuation), Mechanical Orchard | Reverse-engineering a codebase into a dependency graph before touching it is the right first move — we do this too. | Closed source, enterprise-only, no way for a mid-market team or a skeptical engineer to read the reasoning. Also: their own published numbers (Bun rewrite: ~750K lines, 99.8% test pass, 11 days) get inflated in secondhand retellings — a lesson in citing primary sources, applied to our own eventual case studies too. |
| Open-source code modernization | Konveyor/Kai (Red Hat), Move2Kube | RAG over a rules corpus plus static analysis, rather than a bare LLM call, is how you get modernization suggestions that are actually checkable. We adopt the same shape for the code-analysis layer. | Narrow scope (Java EE→Quarkus, K8s only). We're the assessment layer *above* this kind of tool, not a replacement for it — OpOrder can recommend "refactor," Konveyor-style tooling is a fine choice for who actually executes that. |
| Auto architecture diagrams | Hava.io, Cloudockit | Live-account diagramming that stays current is genuinely useful and undersupplied in open source (CloudMapper's diagram feature is abandoned upstream). | Closed, paid SaaS, cloud-account-only — no code-level understanding, no recommendation layer on top. |
| Cost / Well-Architected tooling | Infracost, Powerpipe/Steampipe AWS Well-Architected mod | Live pricing APIs beat baked-in numbers. Both are Go, single-binary, self-hosted — the exact distribution shape we want. | Single-cloud (AWS-centric) and IaC-diff focused, not migration-decision focused. We borrow the pricing-API pattern, not the scope. |
| Migration execution/orchestration | AWS Cloud Migration Factory (OSS, Apache 2.0) | Wave planning and cutover orchestration are a solved, separate problem from assessment. | This is the *execution* layer. We don't compete with it or replicate it — a completed OpOrder should be able to feed straight into a tool like this. |

**The gap, stated precisely:** nobody combines live-infrastructure assessment, code-level
understanding, an honest 5/7-Rs call (including "don't migrate"), cross-provider
Well-Architected scoring, and real cost/effort numbers, in one open, self-hosted,
vendor-neutral tool. That's the whole product.

---

## 3. The output: what an OpOrder actually contains

```
oporder-report/
├── SITUATION.md       # the as-is: diagram, inventory, dependency map, tech debt signals
├── MISSION.md         # per-workload 5/7-Rs recommendation, with the rubric and evidence shown
├── EXECUTION.md        # cost estimate, effort estimate, sequencing, target service mapping
├── waf-scorecard.json  # cross-provider Well-Architected pillar scores, machine-readable
├── sdlc-maturity.json  # branching/CI/test/release maturity signals feeding the effort estimate
├── debt-delta.json     # per-workload technical debt trajectory: mitigated / unchanged / at-risk of increasing (§5.8)
└── architecture.svg    # the live-state diagram, regenerable on every run
```

### 3.1 SITUATION — what's actually there

- Architecture diagram generated from **live infrastructure state**, not from source code
  alone (source tells you intent; the account tells you truth, and they drift — this is the
  gap CloudMapper's abandoned diagram feature and Archify's code-only diagrams both leave
  open).
- Dependency graph: service-to-service, service-to-managed-resource, and code-level module
  dependencies, merged into one graph.
- Plain-English inventory: what's running, what's talking to what, what nobody seems to be
  using (usage telemetry, where the provider exposes it, feeds the "retire" candidate list).

### 3.2 MISSION — the honest call

For every workload, one of: **rehost · replatform · refactor · rearchitect · repurchase ·
retire · retain** (the 7 Rs — see §5.3), with:
- The specific evidence that drove the call (e.g. "3 of 4 managed-service dependencies have
  no equivalent on the target platform → refactor, not rehost").
- A stated confidence level, not false precision.
- What would have to change for the call to flip (the anchor a human can push back against).

### 3.3 EXECUTION — what it costs, in both directions

- **Destination hosting cost**, live-priced (§5.6), compared across every viable target —
  including "stay exactly where you are" and "repatriate to owned hardware" as first-class
  options, not footnotes.
- **The cost of doing the work**: engineering effort estimate (§5.6), plus transparent
  reporting of what generating *this report* itself cost in tokens — if OpOrder isn't willing
  to show its own cost, it has no business estimating anyone else's.
- **Sequencing**: which workloads block which, surfaced from the dependency graph, not
  guessed.

---

## 4. System architecture

```mermaid
flowchart TB
    U["Developer / Operator"] -->|"oporder scan ."| CLI["OpOrder CLI (Go binary)"]
    CLI --> SKILL["Assessment Skill\n(the methodology: rubrics, scoring, report shape)"]
    CLI --> WF["Dynamic Workflow\n(fan out → reduce → verify → synthesize)"]
    WF --> MCP1["MCP: Cloud Inventory\n(read-only)"]
    WF --> MCP2["MCP: Pricing\n(live, per-provider)"]
    WF --> MCP3["MCP: WAF / Posture Checks"]
    MCP1 --> AWS[("AWS")]
    MCP1 --> GCP[("GCP")]
    MCP1 --> AZ[("Azure")]
    MCP1 --> CF[("Cloudflare")]
    WF --> VER["Fresh-context Verifiers\n(one per finding, never the worker's chat)"]
    VER --> REP["OpOrder Report\nSituation / Mission / Execution"]
```

### 4.1 Why this shape, not a monolith

- **The CLI is the only thing users install.** A single Go binary, no runtime dependency,
  same reasoning Infracost, Steampipe, and driftctl already made — it's why they all cross-
  compile cleanly to Linux, macOS, and WSL2 with zero friction. We follow the same precedent
  rather than inventing a new one.
- **The MCP servers are the only thing that touches a cloud account or a pricing API.**
  Isolating I/O behind MCP means the assessment logic (the skill) never needs its own
  bespoke per-provider client code, and a contributor can add a fifth provider by writing one
  MCP server, not by touching the reasoning engine.
- **The skill carries the methodology, not the code.** The 5/7-Rs rubric, the WAF scoring
  normalization, the SDLC-maturity signals — all human-readable, versioned, diffable. This is
  what principle #3 (§1) actually looks like in the repo, not just in the README.
- **The workflow is the diamond pattern** already documented for this exact use case in the
  Code To Cloud Agentic Engineering vault (`20-Graph-Engineering.md`): fan out per workload,
  reduce in plain code, verify each finding with a **fresh-context** agent, synthesize once.
  This is not a new architecture invented for this spec — it's the validated pattern applied.

### 4.2 Complexity is earned, not assumed

The four-piece architecture in §4 (CLI, MCP servers, skill, workflow) is the *end state*,
not the v0.1 starting point — and that distinction matters more for this project's actual
survival than any diagram does. Concretely:

- **v0.1 ships as a plain Go CLI** that calls provider SDKs directly. No MCP servers, no
  skill/workflow split. A Go developer can open `main.go`, understand the whole program,
  and submit a PR to add a check without knowing what MCP even stands for.
- **The MCP/skill/workflow decomposition arrives only when something real forces it** —
  specifically, when the assessment logic needs to run inside other agent hosts (Claude
  Code, Cursor, etc.) rather than just this CLI, which is a genuine, distinct requirement,
  not a speculative one. Until that day, the extra indirection has no justification and is
  pure contribution friction.
- **"Add a fifth provider" should always be the easy PR.** Whatever the current
  architecture is at any given phase, that specific contribution path gets kept simple and
  documented (§8's `docs/architecture/` requirement applies from v0.1, even before there's
  anything architecturally complex to document) — it's the single most likely first
  contribution from an outside developer, and it's the one that should never require a
  maintainer's hand-holding to complete.

### 4.3 Interoperability with agentic standards, not lock-in to one

Vendor-neutrality applies recursively — to which AI tooling can use OpOrder, not just which
cloud it recommends:

- **MCP (Model Context Protocol)** stays the integration surface once §4.2's "something real
  forces it" threshold is met — it's already cross-vendor (Anthropic, OpenAI, and Google all
  support it), which is exactly why it's the right choice over a bespoke integration API.
- **AGENTS.md compatibility**: any coding agent reading a repo's `AGENTS.md` should be able to
  learn how to invoke `oporder` in that repo. A cheap addition with real reach, since
  AGENTS.md is a cross-vendor convention, not specific to any one agent product.
- **A Claude Code Plugin is one distribution channel, not the architecture.** Packaging the
  assessment skill and MCP server as an installable Claude Code plugin (once §4.2 justifies
  MCP existing at all) is a legitimate way to reach that specific audience — it should never
  become a reason the core logic only works inside Claude Code.
- **A2A (Agent2Agent protocol)** is worth watching, not building against yet: the day
  OpOrder's plan needs to hand off to a separate execution agent programmatically rather than
  to a human, A2A is the emerging cross-vendor way to do that handoff without OpOrder needing
  to know anything about the specific agent receiving it.
- **`llms.txt`** on the eventual docs site (§9) — a small, low-cost addition that makes the
  documentation itself legible to any agent evaluating whether to recommend OpOrder, not just
  to search engines.

**The test for adopting any of these:** does it stay optional, and does the plain CLI still
work with zero of them installed? If yes, add it. If a feature only works inside one vendor's
agent host, it doesn't belong in this project — same discipline as refusing to favor one
cloud, applied to the tooling ecosystem itself.

### 4.4 Platform support matrix

| Platform | Support level | Notes |
|---|---|---|
| Linux (x86_64, arm64) | First-class | Primary CI target |
| macOS (Intel, Apple Silicon) | First-class | |
| WSL2 | First-class | Same binary as Linux; no WSL-specific shims needed if we avoid anything that assumes a native Windows filesystem path |
| Native Windows (non-WSL) | Not targeted for v1 | Revisit only if real demand shows up — see Gap Analysis §12 |

### 4.5 Provider support matrix — containers and serverless first

| Provider | Container target | Serverless target | Live pricing source | Notes |
|---|---|---|---|---|
| AWS | ECS, EKS, Fargate | Lambda | AWS Price List API | Most mature target; broadest existing tooling to compare against |
| GCP | GKE, Cloud Run | Cloud Functions | Cloud Billing Catalog API (confirmed live, REST + RPC) | Cloud Run blurs container/serverless — treat as its own row in output, not forced into one bucket |
| Azure | AKS, Container Apps | Azure Functions | Azure Retail Prices API — `https://prices.azure.com/api/retail/prices` (confirmed live) | |
| Cloudflare | Workers (+ Containers, verify current GA status at build time) | Workers | **No public pricing API as of this writing** — needs a maintained scrape/manual-update path, flagged honestly in §12 | Real migration path already documented by Cloudflare: DNS → R2 → D1 → Workers. R2's zero egress fee is a first-class input to the cost model, not a footnote — it can flip a recommendation on its own for egress-heavy workloads. |

---

## 5. The assessment engine

### 5.1 Code analysis layer

- Static dependency graph (tree-sitter–based, language-agnostic parsing — the approach
  used by the `codeknow` open-source project, which already proves this works with zero API
  keys and no fine-tuning).
- Proprietary-SDK detection: flags direct calls to vendor-specific APIs (Step Functions,
  EventBridge triggers, Azure Logic Apps connectors) — this is where "rehost" quietly becomes
  "refactor" and most naive migration estimates go wrong.
- SDLC-maturity signals: branch protection, CI presence, test coverage where measurable,
  release cadence from git history. Feeds directly into the effort estimate (§5.6) — an
  untested monolith costs more to move than a well-tested one, regardless of size.

### 5.2 Live infrastructure layer (via MCP)

- Inventory: what's deployed, where, at what scale.
- Utilization: CPU/memory/traffic where the provider exposes it — the primary signal for
  flagging "retire" candidates, since nobody manually goes looking for unused resources.
- IAM and network topology: input to the WAF security-pillar scoring, and to blast-radius
  estimation for sequencing.

### 5.3 The 5/7-Rs decision rubric

Each workload is scored against explicit, documented triggers — not a bare model call with
no visible reasoning:

| R | Primary trigger |
|---|---|
| **Retain** | Low business criticality, acceptable current cost, no compliance or EOL driver forcing action |
| **Repatriate** | Steady-state predictable load + high egress cost + cloud premium exceeds owned-hardware TCO over the depreciation horizon |
| **Rehost** | Containerizable or VM-portable, no proprietary managed-service dependencies detected, migration deadline is the dominant constraint |
| **Replatform** | 1:1 managed-service swap available (e.g. self-managed DB → managed DB) with minimal code change |
| **Refactor** | Proprietary dependencies with no target-platform equivalent, but the core logic is sound |
| **Rearchitect** | Monolith requiring decomposition to meet a stated goal (e.g. the container/serverless targets in §4.5) |
| **Repurchase** | Equivalent SaaS/COTS functionality is cheaper than continued maintenance |
| **Retire** | Usage telemetry shows no or negligible traffic; redundant with another system |

Every trigger above must resolve from **evidence gathered in §5.1/5.2**, not from the model's
unstated priors. Where evidence is ambiguous, the report says so explicitly rather than
picking a confident-sounding answer — a stated "insufficient evidence, here's what would
resolve it" beats a wrong confident one.

### 5.4 Well-Architected scoring, normalized across providers

AWS, Azure, and GCP each publish their own well-architected framework, with different pillar
names and counts. OpOrder maps all three onto one canonical rubric (Security, Reliability,
Performance, Cost, Operations, Sustainability) so a cross-cloud comparison is actually
apples-to-apples — this normalization step is real, non-trivial work and is called out
explicitly in the Gap Analysis (§12) as a place the mapping will need active maintenance as
providers update their own frameworks.

### 5.5 SDLC maturity scoring

A lens, not a platform — the line drawn earlier in this project's history still holds: SDLC
*guidance* is in scope, SDLC *execution* never is (§1). OpOrder never touches a client's
pipeline; it reports on the one that's already there, with the same evidence discipline as
every other pillar in this spec.

**What gets scored**, each dimension independently, each with the evidence shown:

| Dimension | What's checked | Why it matters to the recommendation |
|---|---|---|
| **Version control hygiene** | Branch protection, trunk-based vs. long-lived branches, commit/PR conventions | A team without branch protection attempting a Rearchitect is a materially higher-risk bet than one with it — the report says so, not just implies it |
| **CI presence and quality** | Does CI exist, does it gate merges, how long does it take, is it actually green or perpetually red | Red or absent CI isn't a style preference, it's a direct input to the effort estimate (§5.6) and the debt-delta risk (§5.8) |
| **Test pyramid shape** | Unit vs. integration vs. end-to-end ratio, measurable coverage where available | The single strongest predictor of whether a Refactor/Rearchitect is likely to land clean or regress silently |
| **Release cadence and rollback readiness** | How often does this ship, is there a documented or practiced rollback path | A team that ships weekly with a working rollback carries migration risk very differently than one that ships quarterly with none |
| **Code review practice** | PR review requirements, average time-to-merge, whether reviews are substantive or rubber-stamped where inferable from history | Feeds confidence level, not a pass/fail gate — this is the softest signal and the report treats it that way |

**Output**: `sdlc-maturity.json` (already in the tree, §3), one score per dimension, each
traceable to the exact repository signal that produced it — same "no opaque model call
standing in for a documented rubric" discipline as §1 applies everywhere else. This scorecard
directly feeds two other outputs, not just itself: the effort estimate (§5.6) and the
technical debt trajectory (§5.8) both read from it rather than re-deriving their own version
of "is this team's process healthy," which would risk the two disagreeing with each other for
no good reason.

### 5.6 Cost engine

**Destination hosting cost** — live-queried at report time, never baked in:
- AWS Price List API, Azure Retail Prices API, GCP Cloud Billing Catalog API — all confirmed
  live and queryable without a sales call.
- Cloudflare — no equivalent public API exists today; see §12 for the honest maintenance
  plan this requires.

**Cost of doing the work** — two numbers, both shown, both distrusted by default until
verified:
1. **Token cost of the assessment itself.** Every OpOrder report states what it cost in LLM
   spend to generate. A tool unwilling to show its own bill has no standing to estimate
   anyone else's.
2. **Engineering effort estimate**, derived from codebase size, complexity, SDLC maturity
   (§5.5), and the specific R selected — expressed as a range with the assumptions shown,
   never a bare number. Ranges get audited against real completed engagements over time,
   the same way any estimating model has to earn trust.

### 5.7 Verification discipline

Every finding is checked by a **fresh-context verifier agent** before it reaches the report
— never the same conversation that produced the finding. Three parallel checks per finding,
matching the pattern already validated in this vault's Graph Engineering work: is it
correct, is it current, is the source real. A finding that fails majority verification is
dropped, not softened.

### 5.8 Technical debt delta — mitigated or introduced

Cost and effort (§5.6) answer "what does this take." They don't answer the question that
actually determines whether the work was worth doing: **does the debt this organization is
carrying go down, stay flat, or go up as a result?** A rehost that just relocates an
unmaintained mess to a new address, or a rushed rewrite that trades known problems for
unfamiliar ones, can pass every cost and timeline check in this spec and still be a bad
decision. This has to be scored explicitly, not left implicit in the R recommendation.

**Baseline debt fingerprint** (current state, measurable directly from §5.1/§5.5 signals):
- Cyclomatic complexity and duplication (`jscpd`-style detection — the same mechanism this
  project's own research already trusts more than vendor debt-quantification claims: see
  the Code To Cloud vault's Devil's Advocate note on testing GitClear's duplication claim
  on your own repo rather than trusting either side of that argument).
- Dependency staleness and EOL exposure — how much of the debt is "the framework version
  itself is the risk," which several Rs (rehost, replatform) leave completely untouched.
- Test coverage and SDLC maturity (§5.5) — debt that isn't covered by tests is debt nobody
  can safely touch, which is itself a compounding risk independent of the code's age.

**Projected debt trajectory** (per candidate R, shown alongside the recommendation in
MISSION.md):

| R | Typical debt trajectory | Why |
|---|---|---|
| Retain | Unchanged | By definition — nothing moves |
| Rehost | **Usually unchanged, sometimes worse** | Lift-and-shift relocates the codebase without touching it; report this plainly rather than let "we migrated" imply "we fixed something" |
| Replatform | Modest reduction | Swapping to a managed service typically removes the operational debt of self-managing that piece, without touching application-level debt |
| Refactor / Rearchitect | **Reduction if executed well, real risk of increase if rushed** | The widest variance of any R — see the caution below |
| Repurchase | Reduction | Retiring in-house code for a maintained product removes that code's debt entirely, at the cost of new integration debt — both sides get shown |
| Retire | Full elimination | The only R that's unambiguous |

> [!warning] The "introduced" side is a projection, not a measurement, and the report says so.
> Baseline debt is measurable today. Debt introduced by a rewrite that doesn't exist yet is
> an estimate conditioned on execution quality, team familiarity with the target stack, and
> timeline pressure — none of which OpOrder can observe in advance. Where a client is using
> AI-assisted tooling to execute a Refactor/Rearchitect, GitClear's own findings (already
> cited in this vault's Devil's Advocate note) are directly relevant and get surfaced in the
> report as context, not as a prediction: duplicated code blocks rose roughly 8× in frequency
> industry-wide as AI-generated code volume grew, and "moved lines" (their refactoring proxy)
> fell from ~25% to under 10% of changed lines over the same period. That's a mechanism, not
> a guarantee — it's a reason to weight the "introduced debt" side of a Refactor/Rearchitect
> call more heavily when the execution plan leans on heavy AI code generation with light
> review, and to say exactly that in the output rather than a bare confidence number.

**Output**: a `debt-delta.json` alongside `waf-scorecard.json` and `sdlc-maturity.json`
(§3), plus one line per workload in MISSION.md — e.g. *"Rehost: cost $X, effort Y weeks,
technical debt unchanged — this migration does not address the missing test coverage or the
three-year-out-of-support runtime."* The point is that a decision-maker can't say afterward
that nobody told them a cheap option was also one that fixed nothing.

---

## 6. CLI, TUI, and report design — the gold standard bar

The bar, stated plainly: **every surface this project produces — the CLI output, the
interactive terminal view, and the generated report — has to look considered, not
default-tool ugly, and every recommendation has to show its pros and cons, never a bare
verdict.** This isn't a polish pass at the end. It's as load-bearing as the rubric itself,
because a tool whose whole pitch is "trust our honesty" loses credibility fast if its output
looks like an afternoon's `fmt.Println` work.

### 6.1 The scan command

A developer opens this tool for a five-minute look and it's still the thing they reach for a
year later. Concretely:

```
$ oporder scan .

  OpOrder v0.1.0 · scanning ./ (git: main @ 4f2a91c)

  ⠋ Reading code................ 142 services detected
  ⠋ Connecting to AWS........... read-only, us-east-1 + 3 more regions
  ⠋ Fanning out assessment...... 142 workers, 3 fresh verifiers each
  ⠋ Pricing (live)............... AWS Price List API, GCP Billing Catalog, Azure Retail Prices
  ⠋ Synthesizing report.........

  Done in 4m12s · $0.38 in LLM spend · full report: ./oporder-report/

  12 retain · 34 rehost · 41 replatform · 28 refactor · 9 rearchitect · 6 repurchase · 12 retire

  Biggest finding: `billing-service` has 4 undocumented dependencies on a deprecated
  internal API. See SITUATION.md#billing-service.

  Next: oporder report --open
```

Principles this enforces:
- **Show real numbers as they happen** — cost, count, elapsed time — never a bare spinner.
  Silence during a long-running scan is the single fastest way to lose trust in a tool that's
  supposed to be about honesty.
- **One command, sensible defaults, deep configurability behind flags** — the same shape as
  `git`, not a wizard.
- **The summary line is the whole point.** Most people will read the terminal output and
  never open the full report — it has to be honest and complete on its own.
- **Every subcommand composes**: `oporder scan`, `oporder report`, `oporder waf`,
  `oporder cost` all work standalone against a previous scan's cache, not just as one
  monolithic run.

### 6.2 The interactive TUI — `oporder browse`

A one-shot terminal log is fine for a CI pipeline; it's the wrong surface for a human
actually deciding what to do with 142 scored workloads. `oporder browse` opens a full
terminal UI against the last scan's cache — built on **Bubble Tea + Lip Gloss** (the Charm
ecosystem), the same Go-native TUI toolkit behind tools like `lazygit` and `gh dash`, chosen
for the same reason Go itself was chosen in §4.1: no new runtime dependency, cross-compiles
cleanly to every platform in §4.4.

The interaction model deliberately mirrors the one already validated in Claude Code's own
`/workflows` progress view (§9 of the Code To Cloud vault's Graph Engineering research):
arrow keys to move between workloads, `Enter` to drill into a recommendation, `Esc` to back
out, `f` to filter by R or by WAF pillar score. That pattern is proven — people already know
how to use it the first time they see it, which matters more here than inventing a novel
interaction model would.

```
┌─ OpOrder · 142 workloads ─────────────────────────────── filter: [all] ─┐
│                                                                          │
│  ▸ billing-service        REFACTOR    debt: ↓ mitigated    conf: high   │
│    payments-api           REPLATFORM  debt: → unchanged    conf: high   │
│    legacy-reports         REHOST      debt: → unchanged    conf: med    │
│    user-uploads           RETIRE      debt: ✓ eliminated   conf: high   │
│    ...                                                                  │
│                                                                          │
├─ billing-service ────────────────────────────────────────────────────── │
│  PROS                          │  CONS                                  │
│  + Removes 3 EOL dependencies  │  − 6-week estimated effort              │
│  + Fixes the untested payment  │  − Team has no prior Go experience      │
│    reconciliation path         │  − Debt reduction depends on review     │
│  + $340/mo hosting saving      │    discipline holding under deadline    │
└──────────────────────────────────────────────────────────────────────── ┘
  ↑↓ select · enter expand · f filter · w waf detail · q quit
```

Every recommendation view shows pros and cons side by side, by construction — not because a
template happens to include both columns, but because §5.3's rubric and §5.8's debt model
already produce evidence for and against each call, and hiding either half would be the same
kind of quiet bias §1 rules out for the scoring itself.

### 6.3 Report visual design

The Markdown report (§3) stays the source of truth — git-diffable, readable with no tooling,
never held hostage behind a renderer. Alongside it, `oporder report --html` generates a
static, self-contained report site: real typography, a cost-comparison chart per workload
across every viable destination, a WAF pillar radar chart, and the debt trajectory from §5.8
rendered as a simple up/flat/down indicator rather than a wall of prose — legible in a
five-minute skim by someone who wasn't in the room for the scan, which is the actual use
case (a decision-maker forwarded the report, not the person who ran the command).

Every design choice here follows the same underlying rule as the palette and accessibility
guidance already established for data visualization work in the Code To Cloud practice:
color communicates meaning (an R with rising debt risk reads differently than one that's
clean), never decoration, and the report is fully legible in both light and dark terminals
and browsers — a tool this proud of showing its own reasoning doesn't get to be unreadable
in half the environments it's opened in.

### 6.4 😈 Devil's advocate: "beat the hyperscalers on output" is a claim, not evidence yet

Worth being honest about what hasn't actually been checked: this spec asserts OpOrder's
output should look and read better than AWS Transform's, Copilot app modernization's, or
watsonx's — but nothing in this project's research so far actually benchmarked their real
report output, only their functional capability. Before that claim survives contact with a
skeptical reviewer, it needs an actual side-by-side: screenshots of what those tools produce
today, compared honestly against an early OpOrder report, not an assumption that enterprise
tooling is automatically uglier. If it turns out one of them already clears this bar, say so
in the open rather than quietly drop the comparison.

### 6.5 What "real value like Alberta" means for report writing specifically

Alberta's own output wasn't impressive because of formatting — it was impressive because
every claim traced to an exact file and line number, across 466 million lines, with nothing
asserted that couldn't be checked (§2.0). That's the actual bar "gorgeous" has to serve, not
compete with: **visual polish earns the report a first look; cited, checkable evidence is
what earns it being believed.** A beautifully designed report full of generic AI prose is a
worse outcome than a plain one with exact citations — if the two ever trade off against each
other during implementation, evidence wins, every time.

---

## 7. Testing strategy

Two different problems, two different testing approaches — conflating them is where most
AI-tool testing goes wrong:

1. **Deterministic logic** (the 5/7-Rs rubric evaluation, WAF pillar mapping, cost
   arithmetic, report rendering): standard unit and golden-file/snapshot tests. No excuse for
   these to be anything less than rigorous — none of this is non-deterministic.
2. **LLM-mediated reasoning** (the verifier agents, the narrative sections of the report):
   an **eval suite**, not unit tests — a fixed set of real-world-shaped fixture repositories
   and mock cloud states with a known-correct answer, run against every model/prompt change,
   scored for drift. This is the same discipline the vault's own Agentic Engineering learning
   path already prescribes ("write 20 eval cases for an LLM-shaped feature") — applied to our
   own product, not just recommended to others.
3. **Provider API mocking**: every MCP server ships a recorded-fixture mode so the full test
   suite runs with zero live cloud credentials and zero cost — a contributor should be able
   to `make test` on a plane.
4. **Adversarial fixtures on purpose**: at least one test repo per provider deliberately
   engineered to trigger every one of the 7 Rs, plus a "genuinely ambiguous, correct answer
   is 'insufficient evidence'" fixture — because a tool that can't admit uncertainty in tests
   won't admit it in production either.

---

## 8. Documentation strategy

- `README.md` — the pitch (already written).
- `SPEC.md` — this document, kept current as the source of truth, not a launch artifact that
  goes stale.
- `docs/rubric.md` — the full 5/7-Rs and WAF scoring logic in plain language, so a skeptical
  reader can audit the reasoning without reading Go source.
- `docs/architecture/` — one page per MCP server / skill / workflow, each with its own
  "what it does, what it doesn't do, what it never sends anywhere."
- `CONTRIBUTING.md` — how to add a fifth cloud provider, written as an actual walkthrough
  with a real PR as the reference example once one exists, not an abstract checklist.
- Every generated report is itself documentation — self-describing enough that someone
  encountering it with zero context on OpOrder can still follow the reasoning.

---

## 9. Website — Omarchy-style

A single, fast, dark-mode-first scrolling page, not a marketing site with a nav bar and six
tabs:
- Above the fold: the one-liner, the install command, nothing else.
- A terminal-recording GIF/asciinema embed of an actual `oporder scan` run — the real UX
  from §6, not a mockup.
- One scroll section per pillar of §1 (vendor-neutral, assess-don't-execute, open scoring,
  self-hosted, fresh-context verification) — each one a claim plus the mechanism that
  enforces it, not just an assertion.
- A live-updating "what it found" wall of anonymized, opt-in example findings from real
  community runs, once that data exists — social proof that's actually true, not testimonial
  copy.
- Footer: license, GitHub link, Code To Cloud attribution, nothing else competing for
  attention.

---

## 10. Roadmap

| Phase | Scope | Architecture | Exit criteria |
|---|---|---|---|
| v0.1 | AWS only. Live inventory + diagram + plain-English SITUATION.md. No Mission/Execution yet. | Plain Go CLI, direct AWS SDK calls — no MCP/skill/workflow split (§4.2) | A stranger can run it against a real AWS account and trust the diagram, and a Go developer can read `main.go` end to end |
| v0.2 | 5/7-Rs MISSION.md, AWS only, rubric fully documented. Technical debt delta (§5.8) ships alongside it — a recommendation with no debt trajectory attached is an incomplete recommendation. `oporder browse` TUI (§6.2) lands here too — the first release with real recommendations to browse is the first release that needs a browsing surface. | Same plain CLI + Bubble Tea/Lip Gloss for the TUI (a display dependency, not an architectural one — doesn't conflict with §4.2) | The rubric survives a public read-through without an obvious hole, no MISSION.md entry ships without a debt-delta line, and every entry shows pros and cons in both the Markdown and the TUI |
| v0.3 | EXECUTION.md — live AWS cost + effort estimate. Eval suite live in CI. `oporder report --html` (§6.3) ships here, once there's a real cost comparison worth charting. | Same plain CLI | A real cost estimate gets checked against a real completed migration, error margin published; the HTML report renders correctly in light and dark, and the §6.4 hyperscaler-output comparison gets actually done, not just asserted |
| v0.4 | GCP + Azure providers added. WAF cross-provider normalization live. | Provider clients still direct, one package per provider — decompose into MCP only if a concrete second agent-host integration need shows up (§4.2) | Same workload, three clouds, one honest comparison |
| v0.5 | Cloudflare added, with the pricing-data maintenance plan from §12 actually running. | | |
| v1.0 | All four providers, full test/eval coverage, docs site, public case study with a real organization's permission. | MCP/skill/workflow split lands here at the earliest, and only if something real needs it by now | Someone outside Code To Cloud ships a PR that adds a capability we didn't think of |

---

## 11. Devil's advocate — stress-testing this spec specifically

> [!danger] "The 7-Rs rubric in §5.3 will produce confidently wrong answers on real, messy codebases."
> Almost certainly true on day one. Real codebases have proprietary dependencies buried three
> layers deep in a config file the static analyzer never opens. **Mitigation, not cure:** the
> "insufficient evidence" outcome has to be exercised constantly in early usage, and every
> wrong call reported by a real user becomes a new eval fixture (§7). A rubric that never
> admits uncertainty in the field is worse than one that admits it too often.

> [!danger] "Cross-provider WAF normalization (§5.4) is quietly the hardest part of this whole spec, and it's one paragraph."
> Fair, and worth saying plainly: mapping three providers' differently-shaped frameworks onto
> one rubric, and keeping that mapping current as AWS/Azure/GCP revise their own frameworks,
> is genuinely ongoing maintenance work, not a one-time engineering task. This is likely the
> single highest-maintenance-cost line item in the whole roadmap and deserves its own tracked
> workstream, not a subsection.

> [!danger] "Nobody will trust cost numbers from a v0.1 tool with zero track record."
> Correct, and that's why §5.6 requires every estimate to ship with its assumptions visible
> and get audited against real outcomes over time — trust in the numbers has to be earned
> the same way Infracost or any estimating tool earned it: slowly, publicly, with errors
> owned rather than hidden.

> [!danger] "A Go CLI plus MCP servers plus a skill plus a workflow is four things to build and keep in sync, not one."
> Yes — this is real architectural complexity, chosen deliberately in §4.1 over a monolith
> specifically so a contributor can extend one piece without touching the others. The
> trade-off is real: more moving parts, in exchange for a codebase where "add GCP support"
> is a scoped PR instead of a rewrite. Worth revisiting if in practice the seams between
> pieces cause more bugs than the isolation saves.

> [!danger] "This spec assumes maintainer bandwidth this project doesn't have yet."
> The single biggest risk to the whole plan, flagged earlier in this project's own history
> and still true here: a roadmap this thorough is worthless without sustained review/triage
> capacity. **The roadmap in §10 should be read as sequential and gated, not parallel** —
> v0.2 doesn't start until v0.1 actually has real users, specifically so scope never outruns
> the hours available to ship it.

---

## 12. Gap analysis — open risks not fully resolved above

> [!warning] The debt-delta model (§5.8) has no validated coefficients, same as the effort estimate.
> "Reduction if executed well, real risk of increase if rushed" for Refactor/Rearchitect is
> directionally right and numerically unproven. Like §5.6's effort estimate, this starts as a
> stated hypothesis and needs to be audited against real completed engagements — track
> predicted vs. actual debt trajectory (via a follow-up scan some months post-migration) from
> the first pilot user onward, not as a someday nice-to-have.

> [!warning] Cloudflare has no public pricing API.
> Every other provider in §4.5 has a live, queryable pricing source. Cloudflare doesn't.
> Options: maintain a manually-updated pricing table with a visible "last verified" date
> (honest but stale-prone), scrape the public pricing page (fragile, breaks silently), or
> scope Cloudflare cost estimates as "directional, verify before committing" in the report
> itself until a better source exists. No option here is fully satisfying — pick one
> explicitly rather than let it default silently.

> [!warning] "Cloudflare Containers" GA status needs verification at build time, not spec time.
> The research behind §4.5 confirmed Workers, D1, R2, and Durable Objects clearly; container
> support on Cloudflare specifically should be re-verified against current docs when v0.5
> actually starts, not assumed from this document.

> [!warning] Usage telemetry for "retire" candidates isn't equally available everywhere.
> AWS, Azure, and GCP expose utilization metrics with varying depth and default retention;
> a workload with no metrics isn't necessarily unused — it might just be un-instrumented.
> The rubric (§5.3) needs a documented fallback for "no telemetry available" that doesn't
> default to a false "retire" signal.

> [!warning] The effort-estimate model (§5.6) has no ground truth yet.
> It's specified as a heuristic — codebase size, complexity, SDLC maturity — with no
> validated coefficients, because none exist yet. This is explicitly an estimate that starts
> rough and gets corrected against real completed engagements. Treat any early number as a
> hypothesis, and say so in the UI, not just in this doc.

> [!warning] Multi-tenant / shared-service workloads break the "one workload, one R" model.
> §5.3 scores per workload, but real infrastructure has shared databases, shared queues, and
> platform services used by dozens of applications at once. The rubric as specified doesn't
> yet say what happens when two workloads that share a dependency get different Rs — this
> needs its own resolved design before v0.3, not an assumption that it'll sort itself out.

> [!warning] Native Windows (non-WSL2) is explicitly out of scope, and that's a real exclusion.
> Reasonable for a first release given the target audience, but worth stating as a conscious
> choice rather than an oversight, in case it turns out to matter more than expected.

> [!warning] Legal/compliance review time for the target buyer isn't modeled anywhere in the roadmap.
> §10's roadmap is all engineering milestones. The earlier research in this project
> established that enterprise adoption of a tool like this is gated by security/compliance
> review, not by feature completeness — that review cycle (SOC2 questions, data-flow
> diagrams) isn't a line item anywhere above and should be, likely starting around v0.3–v0.4
> once real pilot users exist. [SECURITY.md](SECURITY.md) now covers the disclosure process
> and the threat model (credential handling, the prompt-injection risk inherent to reading
> untrusted repos and cloud metadata, MCP scope-minimization) — that part of this gap is
> closed; the compliance-review *timeline* itself still isn't scheduled anywhere.

---

## 13. Governance, versioning, and release practice

The point of this whole project is putting the organization, the engineer, the developer —
not a vendor with a stake in the outcome — in the driving seat, with a genuinely free choice
about what's right for their own situation. That claim is worthless if the project itself is
run casually. A tool asking to be trusted with someone's honest architectural second opinion
has to hold itself to the engineering discipline it implicitly promises the industry it's
critiquing.

### 13.1 Versioning

- **The CLI follows Semantic Versioning.** A breaking change to any flag, output format, or
  default behavior is a major version bump, full stop — no "minor version, but technically
  breaking" exceptions.
- **Every JSON output schema (`waf-scorecard.json`, `sdlc-maturity.json`, `debt-delta.json`)
  is versioned independently of the CLI itself**, with a `schemaVersion` field in every file.
  Downstream tooling — dashboards, CI gates, someone's own script — may parse these directly;
  breaking that silently by coupling schema changes to CLI releases is exactly the kind of
  quiet vendor-style behavior this project exists to be the alternative to.
- **A deprecation is announced at least one minor version before it lands**, with the
  replacement path stated in the deprecation warning itself, not just the changelog — the
  same "write actionable errors, not opaque codes" standard the Code To Cloud vault's Agent
  Design Patterns research already sets for LLM-facing tool output applies here to
  human-facing CLI output too.

### 13.2 Release practice

- **`CHANGELOG.md`** in Keep a Changelog format, updated in the same PR as the change it
  describes — never reconstructed from git log after the fact.
- **Tagged releases with prebuilt binaries** on GitHub Releases for every platform in §4.4,
  the same distribution shape already proven by Infracost, Steampipe, and driftctl.
- **Reproducible builds and a generated SBOM per release**, with SLSA provenance as the
  longer-term bar — this is not new territory for this practice; it's the direct application
  of the Code To Cloud Agentic Engineering vault's own Supply Chain and Governance research
  to its own output, rather than advice given to others and skipped for this project.

### 13.3 Decision process for load-bearing changes

Not every PR needs process. A change to the 5/7-Rs rubric (§5.3), the WAF cross-provider
mapping (§5.4), or the debt-delta model (§5.8) does, because these are the exact surfaces
where a quiet, well-intentioned tweak could drift the tool toward a biased answer without
anyone noticing for months:

- Any such change ships as a short written proposal in the PR description — what evidence
  changes, why, and what real-world case motivated it — not just a diff.
- It ships with a new or updated eval fixture (§7) that would have caught the old behavior
  as wrong, per the standard already set in `CONTRIBUTING.md`.
- It's reviewed by more than one person before merge once the project has more than one
  active maintainer — a rubric this load-bearing shouldn't have a single point of failure
  for "is this still honest," any more than the tool itself should let one worker verify its
  own finding (§5.7).

### 13.4 Community standards

- A `CODE_OF_CONDUCT.md` (Contributor Covenant) from the first public commit, not added
  retroactively once there's a reason to need one.
- Issues and PRs get a first response inside a stated window once the project has real usage
  — an unattended-looking repo is the single fastest way to lose the exact trust this whole
  project depends on, and this vault's own prior research already named maintainer bandwidth
  as the top real risk to community projects; a stated response-time commitment is how that
  risk gets managed rather than just acknowledged.

---

## 🔗 Related

- Code To Cloud Agentic Engineering vault: `20-Graph-Engineering.md` (the diamond pattern
  this workflow architecture is built on), `08-Harness-and-Loops.md`,
  `09-Agent-Design-Patterns.md`
- [README.md](README.md) — the pitch
- [CONTRIBUTING.md](CONTRIBUTING.md) — how to send a real PR after reading one file
- [SECURITY.md](SECURITY.md) — disclosure process and the threat model, including the
  prompt-injection risk this category of tool carries by design
- [LICENSE](LICENSE) — Apache 2.0
