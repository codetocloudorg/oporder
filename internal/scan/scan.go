// Package scan orchestrates the provider connectors, codescan, and
// correlate into the actual `oporder scan` command. This produces the
// first real slice of §3.1's unified SITUATION.md: live inventory across
// whichever providers have credentials configured, workload-boundary
// detection against the local repo (§5.0), and tag/name correlation
// between the two (§5.0 tiers 2-4) — all real, all deterministic, no LLM
// call involved.
//
// What this package deliberately does NOT do yet, because the underlying
// pieces don't exist: §5.0 tier 1's IaC-state-as-ground-truth
// correlation (needs a Terraform-state/CloudFormation parser), §5.1's
// tree-sitter dependency graph and proprietary-SDK detection, the 5/7-Rs
// call (§5.3, needs correlation output plus utilization data this
// package doesn't gather), or cost/effort (§5.6). Claiming more than
// that here would be exactly the kind of overclaiming this whole project
// is built to refuse — see SPEC.md §11's own finding on this.
package scan

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/codescan"
	"github.com/codetocloudorg/oporder/internal/correlate"
	"github.com/codetocloudorg/oporder/internal/provider/aws"
	"github.com/codetocloudorg/oporder/internal/provider/azure"
	"github.com/codetocloudorg/oporder/internal/provider/cloudflare"
	"github.com/codetocloudorg/oporder/internal/provider/gcp"
)

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

	if opts.AzureSubscriptionID != "" {
		pr, res := runAzure(ctx, opts.AzureSubscriptionID)
		s.Providers = append(s.Providers, pr)
		resources = append(resources, res...)
	}
	if opts.AWSRegion != "" {
		pr, res := runAWS(ctx, opts.AWSRegion)
		s.Providers = append(s.Providers, pr)
		resources = append(resources, res...)
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

	return s
}

func runAzure(ctx context.Context, subscriptionID string) (ProviderResult, []correlate.Resource) {
	c, err := azure.NewClient(subscriptionID)
	if err != nil {
		return ProviderResult{Provider: "azure", Detail: fmt.Sprintf("not connected — %v", err)}, nil
	}
	groups, err := c.ListResourceGroups(ctx)
	if err != nil {
		return ProviderResult{Provider: "azure", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}, nil
	}

	var resources []correlate.Resource
	for _, g := range groups {
		resources = append(resources, correlate.Resource{Provider: "azure", ID: g.Name, Name: g.Name, Tags: g.Tags})
		items, err := c.ListResources(ctx, g.Name)
		if err != nil {
			continue // best-effort: the resource-group count above still stands
		}
		for _, r := range items {
			resources = append(resources, correlate.Resource{
				Provider: "azure",
				ID:       g.Name + "/" + r.Name,
				Name:     r.Name,
				Tags:     r.Tags,
			})
		}
	}

	return ProviderResult{
		Provider: "azure",
		Ran:      true,
		Count:    len(groups),
		Detail:   fmt.Sprintf("%d resource group(s)", len(groups)),
	}, resources
}

func runAWS(ctx context.Context, region string) (ProviderResult, []correlate.Resource) {
	c, err := aws.NewClient(ctx, region)
	if err != nil {
		return ProviderResult{Provider: "aws", Detail: fmt.Sprintf("not connected — %v", err)}, nil
	}
	instances, err := c.ListInstances(ctx)
	if err != nil {
		return ProviderResult{Provider: "aws", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}, nil
	}

	var resources []correlate.Resource
	for _, inst := range instances {
		name := inst.Tags["Name"]
		if name == "" {
			name = inst.ID
		}
		resources = append(resources, correlate.Resource{Provider: "aws", ID: inst.ID, Name: name, Tags: inst.Tags})
	}

	return ProviderResult{
		Provider: "aws",
		Ran:      true,
		Count:    len(instances),
		Detail:   fmt.Sprintf("%d EC2 instance(s) in %s", len(instances), region),
	}, resources
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
