// Package scan orchestrates the provider connectors, codescan, correlate,
// and rubric into the actual `oporder scan` command. This produces the
// first real slice of §3.1's unified output model: live inventory,
// workload-boundary detection against the local repo (§5.0), tag/name
// correlation between the two (§5.0 tiers 2-4), and — for AWS and Azure
// VM-hosted workloads, on real CloudWatch/Azure Monitor utilization data
// — a first genuine 5/7-Rs call (§5.3), written to MISSION.md. All of it
// is deterministic; the only non-deterministic step is the rubric's own
// trigger table, which is itself plain code, not an LLM call.
//
// Deliberately not a direction to keep pushing: gathering utilization
// for GCP and Cloudflare too would extend provider parity, but it's the
// same kind of infrastructure-monitoring work AWS Migration Hub and
// Azure Migrate already do — not this project's actual differentiator.
// The next real gap to close is on the code side (§5.1's proprietary-SDK
// and containerizability detection), which is what would let a VM-hosted
// workload's Mission entry say something a vendor migration tool
// structurally can't: not just "here's its utilization" but "here's
// whether it's actually a good candidate to leave the VM behind for a
// container or serverless target," per this project's own modernization
// bias (see MISSION.md's own note on this).
//
// What this package deliberately does NOT do yet, because the underlying
// pieces don't exist: §5.0 tier 1's IaC-state-as-ground-truth
// correlation (needs a Terraform-state/CloudFormation parser), §5.1's
// tree-sitter dependency graph and proprietary-SDK detection, or
// cost/effort (§5.6). The rubric call itself also uses exactly one
// evidence signal (telemetry) out of the dozen §5.3 defines — every other
// field is correctly left at zero ("no evidence gathered," not "false"),
// so most real workloads will land on Insufficient confidence until more
// evidence-gathering exists, which is the honest result, not a bug.
// Claiming more than that here would be exactly the kind of overclaiming
// this whole project is built to refuse — see SPEC.md §11's own finding
// on this.
package scan

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/codetocloudorg/oporder/internal/codescan"
	"github.com/codetocloudorg/oporder/internal/correlate"
	"github.com/codetocloudorg/oporder/internal/debtdelta"
	"github.com/codetocloudorg/oporder/internal/provider/aws"
	"github.com/codetocloudorg/oporder/internal/provider/azure"
	"github.com/codetocloudorg/oporder/internal/provider/cloudflare"
	"github.com/codetocloudorg/oporder/internal/provider/gcp"
	"github.com/codetocloudorg/oporder/internal/rubric"
)

// retireCPUThresholdPercent is the average-CPU cutoff below which a
// workload's traffic is treated as negligible for §5.3's Retire trigger.
// An unvalidated heuristic, same status as every other coefficient in
// SPEC.md §12's gap analysis — stated as a constant here specifically so
// it's easy to find and revise once real outcomes can check it.
const retireCPUThresholdPercent = 5.0

// utilizationLookback is how far back a provider's CPUUtilization call
// looks — CloudWatch for AWS, Azure Monitor for Azure.
const utilizationLookback = 7 * 24 * time.Hour

// utilizationSample is the common shape this package needs from any
// provider's utilization type (aws.Utilization, azure.Utilization) —
// just enough to build rubric.Evidence, regardless of which provider's
// metrics API produced it.
type utilizationSample struct {
	HasTelemetry      bool
	AverageCPUPercent float64
}

// utilizationKey combines provider and resource ID so utilization
// lookups can't collide across providers even though AWS instance IDs
// and Azure ARM resource IDs happen to look nothing alike in practice.
func utilizationKey(provider, id string) string {
	return provider + "|" + id
}

// Options controls which providers a scan attempts to reach, and whether
// it also analyzes local code. Each field left empty means "skip this
// part" — a scan against whatever's actually configured, not a hard
// requirement to have everything ready.
type Options struct {
	RepoPath            string // "" disables code analysis and correlation
	AzureSubscriptionID string
	AWSRegion           string // empty disables AWS
	CloudflareTokenFile string // empty disables Cloudflare
	GCPProjectID        string // empty disables GCP
}

// ProviderResult is one provider's outcome — either real counts or a
// specific, actionable reason it didn't run, never a silent skip.
type ProviderResult struct {
	Provider string
	Ran      bool
	Count    int
	Detail   string // e.g. "3 resource groups" or the actionable error
}

// Situation is everything a scan actually produced.
type Situation struct {
	Providers    []ProviderResult
	Workloads    []codescan.Workload
	Unclassified []codescan.Unclassified
	Correlation  correlate.Result
	Missions     []Mission
}

// Mission is one workload's real §5.3 rubric call, built from whatever
// evidence this pass could actually gather — currently AWS CloudWatch
// utilization only. See the package doc for exactly how partial that
// evidence is.
type Mission struct {
	WorkloadName string
	WorkloadPath string
	Resource     correlate.Resource
	Result       rubric.Result
	DebtDelta    debtdelta.Result
}

// Run attempts every configured provider independently — one provider
// failing (missing credentials, a permissions error) never stops the
// others from running — then runs code analysis and correlation if a
// repo path was given. Every provider gets a ProviderResult either way,
// per §5.7's fan-in guard principle: a partial scan is reported as
// partial, never silently presented as complete.
func Run(ctx context.Context, opts Options) Situation {
	var s Situation
	var resources []correlate.Resource
	utilization := map[string]utilizationSample{}

	if opts.AzureSubscriptionID != "" {
		pr, res, util := runAzure(ctx, opts.AzureSubscriptionID)
		s.Providers = append(s.Providers, pr)
		resources = append(resources, res...)
		for id, u := range util {
			utilization[utilizationKey("azure", id)] = utilizationSample{
				HasTelemetry:      u.HasTelemetry,
				AverageCPUPercent: u.AverageCPUPercent,
			}
		}
	}
	if opts.AWSRegion != "" {
		pr, res, util := runAWS(ctx, opts.AWSRegion)
		s.Providers = append(s.Providers, pr)
		resources = append(resources, res...)
		for id, u := range util {
			utilization[utilizationKey("aws", id)] = utilizationSample{
				HasTelemetry:      u.HasTelemetry,
				AverageCPUPercent: u.AverageCPUPercent,
			}
		}
	}
	if opts.GCPProjectID != "" {
		pr, res := runGCP(ctx, opts.GCPProjectID)
		s.Providers = append(s.Providers, pr)
		resources = append(resources, res...)
	}
	if opts.CloudflareTokenFile != "" {
		pr, res := runCloudflare(opts.CloudflareTokenFile)
		s.Providers = append(s.Providers, pr)
		resources = append(resources, res...)
	}

	if opts.RepoPath != "" {
		result, err := codescan.Analyze(opts.RepoPath)
		if err != nil {
			s.Providers = append(s.Providers, ProviderResult{
				Provider: "code",
				Detail:   fmt.Sprintf("code analysis failed — %v", err),
			})
		} else {
			s.Workloads = result.Workloads
			s.Unclassified = result.Unclassified
		}
	}

	if len(s.Workloads) > 0 {
		cw := make([]correlate.Workload, len(s.Workloads))
		for i, w := range s.Workloads {
			cw[i] = correlate.Workload{Name: w.Name, Path: w.Path}
		}
		s.Correlation = correlate.Correlate(cw, resources)
	}

	for _, m := range s.Correlation.Matches {
		u, ok := utilization[utilizationKey(m.Resource.Provider, m.Resource.ID)]
		if !ok {
			continue
		}
		evidence := rubric.Evidence{
			TelemetryAvailable:    u.HasTelemetry,
			NoOrNegligibleTraffic: u.HasTelemetry && u.AverageCPUPercent < retireCPUThresholdPercent,
		}
		call := rubric.Evaluate(evidence)
		s.Missions = append(s.Missions, Mission{
			WorkloadName: m.WorkloadName,
			WorkloadPath: m.WorkloadPath,
			Resource:     m.Resource,
			Result:       call,
			// ExecutionContext is left at its zero value: §5.5's SDLC
			// scoring, which would populate SDLCMaturityHigh, isn't built
			// yet. That zero value still produces a real, honestly-labeled
			// projection (assessRefactorOrRearchitect's default case) for
			// Refactor/Rearchitect, and every other R's trajectory doesn't
			// depend on execution context at all.
			DebtDelta: debtdelta.Assess(call.R, debtdelta.ExecutionContext{}),
		})
	}

	return s
}

func runAzure(ctx context.Context, subscriptionID string) (ProviderResult, []correlate.Resource, map[string]azure.Utilization) {
	c, err := azure.NewClient(subscriptionID)
	if err != nil {
		return ProviderResult{Provider: "azure", Detail: fmt.Sprintf("not connected — %v", err)}, nil, nil
	}
	groups, err := c.ListResourceGroups(ctx)
	if err != nil {
		return ProviderResult{Provider: "azure", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}, nil, nil
	}

	var resources []correlate.Resource
	utilization := map[string]azure.Utilization{}
	for _, g := range groups {
		// Resource groups are correlation targets in their own right
		// (e.g. a group tagged service:billing), keyed by name since a
		// group has no ARM resource ID of its own the way a resource does.
		resources = append(resources, correlate.Resource{Provider: "azure", ID: g.Name, Name: g.Name, Tags: g.Tags})

		items, err := c.ListResources(ctx, g.Name)
		if err != nil {
			continue // best-effort: the resource-group count above still stands
		}
		for _, r := range items {
			resources = append(resources, correlate.Resource{
				Provider: "azure",
				ID:       r.ID,
				Name:     r.Name,
				Tags:     r.Tags,
			})

			if r.Type != azure.VirtualMachineResourceType || r.ID == "" {
				continue
			}
			if u, err := c.CPUUtilization(ctx, r.ID, utilizationLookback); err == nil {
				utilization[r.ID] = u
			}
			// A failed utilization query for one VM never stops the scan
			// — same fan-in guard principle as the AWS connector.
		}
	}

	return ProviderResult{
		Provider: "azure",
		Ran:      true,
		Count:    len(groups),
		Detail:   fmt.Sprintf("%d resource group(s)", len(groups)),
	}, resources, utilization
}

func runAWS(ctx context.Context, region string) (ProviderResult, []correlate.Resource, map[string]aws.Utilization) {
	c, err := aws.NewClient(ctx, region)
	if err != nil {
		return ProviderResult{Provider: "aws", Detail: fmt.Sprintf("not connected — %v", err)}, nil, nil
	}
	instances, err := c.ListInstances(ctx)
	if err != nil {
		return ProviderResult{Provider: "aws", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}, nil, nil
	}

	var resources []correlate.Resource
	utilization := map[string]aws.Utilization{}
	for _, inst := range instances {
		name := inst.Tags["Name"]
		if name == "" {
			name = inst.ID
		}
		resources = append(resources, correlate.Resource{Provider: "aws", ID: inst.ID, Name: name, Tags: inst.Tags})

		if u, err := c.CPUUtilization(ctx, inst.ID, utilizationLookback); err == nil {
			utilization[inst.ID] = u
		}
		// A failed utilization query for one instance never stops the
		// scan — the instance still shows up in inventory and
		// correlation, just without a Mission call.
	}

	return ProviderResult{
		Provider: "aws",
		Ran:      true,
		Count:    len(instances),
		Detail:   fmt.Sprintf("%d EC2 instance(s) in %s", len(instances), region),
	}, resources, utilization
}

func runGCP(ctx context.Context, projectID string) (ProviderResult, []correlate.Resource) {
	c, err := gcp.NewClient(ctx, projectID)
	if err != nil {
		return ProviderResult{Provider: "gcp", Detail: fmt.Sprintf("not connected — %v", err)}, nil
	}
	defer c.Close()

	instances, err := c.ListInstances(ctx)
	if err != nil {
		return ProviderResult{Provider: "gcp", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}, nil
	}

	var resources []correlate.Resource
	for _, inst := range instances {
		resources = append(resources, correlate.Resource{Provider: "gcp", ID: inst.Name, Name: inst.Name, Tags: inst.Labels})
	}

	return ProviderResult{
		Provider: "gcp",
		Ran:      true,
		Count:    len(instances),
		Detail:   fmt.Sprintf("%d Compute Engine instance(s)", len(instances)),
	}, resources
}

func runCloudflare(tokenFile string) (ProviderResult, []correlate.Resource) {
	token, err := cloudflare.TokenFromFile(tokenFile)
	if err != nil {
		return ProviderResult{Provider: "cloudflare", Detail: fmt.Sprintf("not connected — %v", err)}, nil
	}
	c := cloudflare.NewClient(token)
	zones, err := c.ListZones(context.Background())
	if err != nil {
		return ProviderResult{Provider: "cloudflare", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}, nil
	}

	var resources []correlate.Resource
	for _, z := range zones {
		resources = append(resources, correlate.Resource{Provider: "cloudflare", ID: z.ID, Name: z.Name})
	}

	return ProviderResult{
		Provider: "cloudflare",
		Ran:      true,
		Count:    len(zones),
		Detail:   fmt.Sprintf("%d zone(s)", len(zones)),
	}, resources
}

// WriteSituationMD writes the real subset of §3.1's SITUATION.md this
// package can currently produce: per-provider inventory, detected code
// workloads, and their correlation — nothing claimed that wasn't
// actually gathered or computed.
func WriteSituationMD(path string, s Situation) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("scan: creating %s: %w", path, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Situation")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "Live inventory, code workload-boundary detection (§5.0), and tag/name")
	fmt.Fprintln(f, "correlation between them — real and computed, not placeholders. All of it")
	fmt.Fprintln(f, "is deterministic string matching and file-layout heuristics; no LLM call")
	fmt.Fprintln(f, "is involved, per §5.0's own stated cost decision. IaC-state correlation")
	fmt.Fprintln(f, "(§5.0 tier 1) and §5.1's tree-sitter dependency graph aren't built yet —")
	fmt.Fprintln(f, "see SPEC.md §10 for what's left before this is the full picture §3.1")
	fmt.Fprintln(f, "describes.")
	fmt.Fprintln(f)

	fmt.Fprintln(f, "## Infrastructure")
	fmt.Fprintln(f)
	ranAny := false
	for _, p := range s.Providers {
		if p.Ran {
			ranAny = true
			fmt.Fprintf(f, "### %s\n\n%s\n\n", p.Provider, p.Detail)
		} else {
			fmt.Fprintf(f, "### %s — not connected\n\n%s\n\n", p.Provider, p.Detail)
		}
	}
	if !ranAny {
		fmt.Fprintln(f, "No provider connected. See SPEC.md §4.6a for how to set up credentials.")
		fmt.Fprintln(f)
	}

	if len(s.Workloads) > 0 || len(s.Unclassified) > 0 {
		fmt.Fprintln(f, "## Code")
		fmt.Fprintln(f)
		if len(s.Workloads) > 0 {
			fmt.Fprintln(f, "| Workload | Path | Confidence | Signal |")
			fmt.Fprintln(f, "|---|---|---|---|")
			for _, w := range s.Workloads {
				fmt.Fprintf(f, "| %s | `%s` | %s | %s |\n", w.Name, w.Path, w.Confidence, w.Signal)
			}
			fmt.Fprintln(f)
		}
		if len(s.Unclassified) > 0 {
			fmt.Fprintln(f, "**Unclassified — needs manual review, never silently assigned a guessed boundary (§5.0 tier 3):**")
			fmt.Fprintln(f)
			for _, u := range s.Unclassified {
				fmt.Fprintf(f, "- `%s` — %s\n", u.Path, u.Reason)
			}
			fmt.Fprintln(f)
		}
	}

	if len(s.Correlation.Matches) > 0 || len(s.Correlation.UnmatchedWorkloads) > 0 || len(s.Correlation.UnmatchedResources) > 0 {
		fmt.Fprintln(f, "## Correlation")
		fmt.Fprintln(f)
		if diagram := mermaidDiagram(s); diagram != "" {
			fmt.Fprint(f, diagram)
			fmt.Fprintln(f)
		}
		if len(s.Correlation.Matches) > 0 {
			fmt.Fprintln(f, "| Workload | Resource | Provider | Confidence | Reason |")
			fmt.Fprintln(f, "|---|---|---|---|---|")
			for _, m := range s.Correlation.Matches {
				fmt.Fprintf(f, "| %s | %s | %s | %s | %s |\n", m.WorkloadName, m.Resource.Name, m.Resource.Provider, m.Confidence, m.Reason)
			}
			fmt.Fprintln(f)
		}
		if len(s.Correlation.UnmatchedWorkloads) > 0 {
			fmt.Fprintln(f, "**Code with no matching live resource** — not yet deployed, or deployed under a name/tag this pass couldn't match:")
			fmt.Fprintln(f)
			for _, w := range s.Correlation.UnmatchedWorkloads {
				fmt.Fprintf(f, "- `%s` (%s)\n", w.Path, w.Name)
			}
			fmt.Fprintln(f)
		}
		if len(s.Correlation.UnmatchedResources) > 0 {
			fmt.Fprintln(f, "**Live resources with no matching code** — shadow IT, abandoned infrastructure, and the clearest Retire candidates all show up here first (§5.0):")
			fmt.Fprintln(f)
			for _, r := range s.Correlation.UnmatchedResources {
				fmt.Fprintf(f, "- %s (%s)\n", r.Name, r.Provider)
			}
			fmt.Fprintln(f)
		}
	}

	return nil
}
