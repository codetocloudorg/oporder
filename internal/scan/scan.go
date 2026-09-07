// Package scan orchestrates the provider connectors into the actual
// `oporder scan` command. This is honestly scoped to what's real today:
// live inventory across whichever providers have credentials configured,
// written out as SITUATION.md's inventory section (SPEC.md §3.1).
//
// What this package deliberately does NOT do yet, because the underlying
// pieces don't exist: §5.0's code-to-infra correlation (needs §5.1's code
// analysis, not built), the 5/7-Rs call (§5.3, needs correlation first),
// cost/effort (§5.6), or anything else past raw inventory. Claiming more
// than that here would be exactly the kind of overclaiming this whole
// project is built to refuse — see SPEC.md §11's own finding on this.
package scan

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/provider/aws"
	"github.com/codetocloudorg/oporder/internal/provider/azure"
	"github.com/codetocloudorg/oporder/internal/provider/cloudflare"
)

// Options controls which providers a scan attempts to reach. Each field
// left empty means "skip this provider" — a scan against whatever's
// actually configured, not a hard requirement to have all three ready.
type Options struct {
	AzureSubscriptionID string
	AWSRegion           string // empty disables AWS
	CloudflareTokenFile string // empty disables Cloudflare
}

// ProviderResult is one provider's outcome — either real counts or a
// specific, actionable reason it didn't run, never a silent skip.
type ProviderResult struct {
	Provider string
	Ran      bool
	Count    int
	Detail   string // e.g. "3 resource groups" or the actionable error
}

// Situation is everything a scan actually produced — the honest subset of
// §3.1's SITUATION.md this package can currently generate.
type Situation struct {
	Providers []ProviderResult
}

// Run attempts every configured provider independently — one provider
// failing (missing credentials, a permissions error) never stops the
// others from running. Every provider gets a ProviderResult either way,
// per §5.7's fan-in guard principle: a partial scan is reported as
// partial, never silently presented as complete.
func Run(ctx context.Context, opts Options) Situation {
	var results []ProviderResult

	if opts.AzureSubscriptionID != "" {
		results = append(results, runAzure(ctx, opts.AzureSubscriptionID))
	}
	if opts.AWSRegion != "" {
		results = append(results, runAWS(ctx, opts.AWSRegion))
	}
	if opts.CloudflareTokenFile != "" {
		results = append(results, runCloudflare(opts.CloudflareTokenFile))
	}

	return Situation{Providers: results}
}

func runAzure(ctx context.Context, subscriptionID string) ProviderResult {
	c, err := azure.NewClient(subscriptionID)
	if err != nil {
		return ProviderResult{Provider: "azure", Detail: fmt.Sprintf("not connected — %v", err)}
	}
	groups, err := c.ListResourceGroups(ctx)
	if err != nil {
		return ProviderResult{Provider: "azure", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}
	}
	return ProviderResult{
		Provider: "azure",
		Ran:      true,
		Count:    len(groups),
		Detail:   fmt.Sprintf("%d resource group(s)", len(groups)),
	}
}

func runAWS(ctx context.Context, region string) ProviderResult {
	c, err := aws.NewClient(ctx, region)
	if err != nil {
		return ProviderResult{Provider: "aws", Detail: fmt.Sprintf("not connected — %v", err)}
	}
	instances, err := c.ListInstances(ctx)
	if err != nil {
		return ProviderResult{Provider: "aws", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}
	}
	return ProviderResult{
		Provider: "aws",
		Ran:      true,
		Count:    len(instances),
		Detail:   fmt.Sprintf("%d EC2 instance(s) in %s", len(instances), region),
	}
}

func runCloudflare(tokenFile string) ProviderResult {
	token, err := cloudflare.TokenFromFile(tokenFile)
	if err != nil {
		return ProviderResult{Provider: "cloudflare", Detail: fmt.Sprintf("not connected — %v", err)}
	}
	c := cloudflare.NewClient(token)
	zones, err := c.ListZones(context.Background())
	if err != nil {
		return ProviderResult{Provider: "cloudflare", Detail: fmt.Sprintf("connected, but listing failed — %v", err)}
	}
	return ProviderResult{
		Provider: "cloudflare",
		Ran:      true,
		Count:    len(zones),
		Detail:   fmt.Sprintf("%d zone(s)", len(zones)),
	}
}

// WriteSituationMD writes the honest, currently-real subset of §3.1's
// SITUATION.md — inventory counts per provider, nothing claimed that
// wasn't actually gathered.
func WriteSituationMD(path string, s Situation) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("scan: creating %s: %w", path, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Situation")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "Live inventory only — code analysis and correlation (§5.0/§5.1) aren't wired in")
	fmt.Fprintln(f, "yet, so this is not the unified architecture picture SPEC.md §3.1 describes as the")
	fmt.Fprintln(f, "eventual goal. What's below is real and live-queried; nothing here is a placeholder.")
	fmt.Fprintln(f)

	ranAny := false
	for _, p := range s.Providers {
		if p.Ran {
			ranAny = true
			fmt.Fprintf(f, "## %s\n\n%s\n\n", p.Provider, p.Detail)
		} else {
			fmt.Fprintf(f, "## %s — not connected\n\n%s\n\n", p.Provider, p.Detail)
		}
	}
	if !ranAny {
		fmt.Fprintln(f, "No provider connected. See SPEC.md §4.6a for how to set up credentials.")
	}
	return nil
}
