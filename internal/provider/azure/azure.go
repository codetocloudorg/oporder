// Package azure implements the read-only Azure inventory connector from
// SPEC.md §5.2. Every method here issues GET/LIST calls only — no create,
// update, or delete operation exists in this package, by design, per §1's
// "nothing leaves the user's boundary they didn't approve" and "read-only
// by design" principles. That's not a runtime check bolted on afterward; it's
// enforced by which SDK methods this file simply never calls.
package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// Client is a thin, read-only wrapper around the official Azure SDK for Go.
// Authentication goes through azidentity's Azure CLI credential, which
// reuses whatever `az login` session is already active rather than asking
// for or storing separate service-principal credentials — one less
// credential for this tool to be responsible for.
type Client struct {
	subscriptionID string
	groupsClient   *armresources.ResourceGroupsClient
	resClient      *armresources.Client
}

// ResourceGroup is the minimal shape §5.2's inventory needs — enough to
// build the correlation input for §5.0 without carrying the full,
// much-larger SDK response type through the rest of the pipeline.
type ResourceGroup struct {
	Name     string
	Location string
	Tags     map[string]string
}

// Resource is one resource within a group — Tags matters directly for
// §5.0's tier-2 correlation signal (service:/app:/team: tags).
type Resource struct {
	Name     string
	Type     string
	Location string
	Tags     map[string]string
}

// NewClient authenticates via the Azure CLI's existing session (`az login`)
// against the given subscription ID. Returns an error immediately if no
// CLI session is active, rather than a confusing failure on first real call.
func NewClient(subscriptionID string) (*Client, error) {
	cred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure: no active `az login` session found: %w", err)
	}

	groupsClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: creating resource groups client: %w", err)
	}
	resClient, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: creating resources client: %w", err)
	}

	return &Client{
		subscriptionID: subscriptionID,
		groupsClient:   groupsClient,
		resClient:      resClient,
	}, nil
}

// ListResourceGroups returns every resource group in the subscription.
// An empty slice with a nil error is the correct, valid result for a
// subscription with nothing deployed yet — not treated as an error case,
// since a brand-new or sandboxed account is a completely ordinary input.
func (c *Client) ListResourceGroups(ctx context.Context) ([]ResourceGroup, error) {
	var out []ResourceGroup

	pager := c.groupsClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure: listing resource groups: %w", err)
		}
		for _, g := range page.Value {
			if g == nil || g.Name == nil {
				continue
			}
			rg := ResourceGroup{Name: *g.Name, Tags: tagsFrom(g.Tags)}
			if g.Location != nil {
				rg.Location = *g.Location
			}
			out = append(out, rg)
		}
	}
	return out, nil
}

// ListResources returns every resource within a resource group. Same
// empty-is-valid handling as ListResourceGroups.
func (c *Client) ListResources(ctx context.Context, resourceGroup string) ([]Resource, error) {
	var out []Resource

	filter := fmt.Sprintf("resourceGroup eq '%s'", resourceGroup)
	pager := c.resClient.NewListPager(&armresources.ClientListOptions{Filter: &filter})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("azure: listing resources in %q: %w", resourceGroup, err)
		}
		for _, r := range page.Value {
			if r == nil || r.Name == nil {
				continue
			}
			res := Resource{Name: *r.Name, Tags: tagsFrom(r.Tags)}
			if r.Type != nil {
				res.Type = *r.Type
			}
			if r.Location != nil {
				res.Location = *r.Location
			}
			out = append(out, res)
		}
	}
	return out, nil
}

// tagsFrom converts the SDK's map[string]*string (nil-able values, matching
// Azure's actual API shape) into a plain map[string]string — the one piece
// of this file that's pure logic rather than a live API call, and the one
// piece that's actually unit-testable without a real subscription or an
// HTTP-fixture recording harness (a real, tracked next step, not faked here
// — see the package-level TODO in azure_test.go).
func tagsFrom(src map[string]*string) map[string]string {
	out := map[string]string{}
	for k, v := range src {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}
