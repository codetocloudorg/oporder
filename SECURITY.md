# Security Policy

OpOrder is designed to be run against a real codebase and a real cloud account. If it can't
be trusted with that, nothing else in this project matters — this document is held to the
same evidence standard as the rest of the repo, not boilerplate.

## Supported versions

| Version | Supported |
|---|---|
| `main` / latest tagged release | Yes |
| Anything pre-1.0 | Security fixes land on `main` and the most recent tag only — there is no long-term support branch yet. This will be revisited honestly once a 1.0 exists with real users depending on older releases, not before. |

## Reporting a vulnerability

**Do not open a public GitHub issue for a security vulnerability.** Use
[GitHub Security Advisories](https://github.com/codetocloudorg/oporder/security/advisories/new)
for this repository, which creates a private disclosure thread. If that's not available to
you for any reason, email **security@codetocloud.io**.

Include what you'd want to receive yourself: the affected version, a minimal repro, and the
actual impact — what an attacker gains, not just that something looks off.

**Response commitment**: acknowledgment within 5 business days. This project does not yet
have the maintainer bandwidth to promise faster, and a false promise here would be worse than
an honest one — see this spec's own repeated position on not committing to process the
project can't actually sustain yet.

**Disclosure**: coordinated. We'll work with you on a fix and a timeline before any public
detail, and credit you in the advisory and the changelog unless you'd rather stay anonymous.

## What's in scope

- Anything that leaks credentials, cloud account data, or source code somewhere it wasn't
  explicitly sent by the user.
- Anything that lets a scanned repository or cloud account cause OpOrder to take an action,
  make a network call, or alter its own output beyond what the user asked for — see the
  prompt-injection threat model below, which is the sharpest edge case this category of tool
  has.
- Privilege escalation beyond what the configured cloud credentials should allow.
- Supply-chain issues in a released binary (see §"Supply chain" below).

**Out of scope**: a wrong 5/7-Rs recommendation, an inaccurate cost estimate, or a debt-delta
call you disagree with. Those are quality bugs — file them as normal GitHub issues per
[CONTRIBUTING.md](CONTRIBUTING.md), not here. A wrong answer isn't a vulnerability; a leaked
credential is.

## The threat model, stated plainly

### 1. Credentials never leave the machine they're configured on

OpOrder's cloud connectors (§4, §4.5 of [SPEC.md](SPEC.md)) are read-only by design and are
never logged, never written into any report file, and never included in what gets sent to an
LLM. Only derived findings — "this bucket has public read access," not the credential that
discovered it — cross that boundary, and only when the user has configured an LLM call at
all. Self-hosted is the default; there is no telemetry, and no report or finding is sent
anywhere OpOrder wasn't explicitly told to send it.

### 2. Prompt injection is the real, open threat for this category of tool — not a hypothetical

OpOrder reads two kinds of content it does not control: arbitrary source code in the
repository being assessed, and arbitrary metadata in the cloud account being scanned
(resource names, tags, descriptions). Both are attacker-reachable in a real scenario — a
malicious dependency, a compromised contributor's commit, or a resource tag set by someone
with limited account access could all contain text crafted to manipulate an LLM into a
false or dangerous conclusion. This is close to exactly the shape of
[Simon Willison's "lethal trifecta"](https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/)
already cited as Tier 0 material in the Code To Cloud vault's own security research: private
data (the account being scanned), untrusted content (arbitrary code/metadata), and a channel
out (the LLM call, and anything downstream that trusts its output).

**Mitigations already designed into the architecture**, and their honest limits:
- The fresh-context verifier discipline (§5.7) means no single compromised finding reaches
  the report without an independent check — but a verifier reading the *same* poisoned
  content can, in principle, be fooled the same way the worker was. This is a partial
  mitigation, not a solved problem, and is tracked as an open item, not a closed one.
- The rubric (§5.3–§5.5) is evidence-based and auditable specifically so a manipulated
  conclusion is more likely to look wrong against the visible rubric than it would coming
  from an opaque model call with no rubric to check it against.
- **What this doesn't yet have**: a dedicated, tested defense specifically against adversarial
  content planted in scanned repos or cloud metadata. Treat this as a known gap until it has
  one, and report anything that demonstrates it concretely as a vulnerability, not a feature
  request.

### 3. Supply chain

Every release ships with an SBOM and works toward SLSA provenance (SPEC.md §13.2). Until that
pipeline exists, verify a release's checksum against the GitHub release page before running
an unfamiliar binary, the same caution you'd apply to any infrastructure tool that touches
live cloud credentials.

### 4. MCP servers are the narrowest possible surface, on purpose

Per SPEC.md §4.1, cloud and pricing API access is isolated behind MCP servers specifically so
the blast radius of a bug in that code is bounded to "reads cloud state," never "runs
arbitrary commands" or "writes to the account." Any MCP server requesting write access,
broader-than-read-only scopes, or credentials beyond what a specific check needs is treated
as a design defect, not a convenience worth keeping.

## A vulnerability found here becomes a permanent fixture, not just a patch

Same discipline as a wrong recommendation (CONTRIBUTING.md): a confirmed security issue gets
a regression test that would have caught it, committed alongside the fix, so the same class
of issue can't silently reappear in a future refactor.
