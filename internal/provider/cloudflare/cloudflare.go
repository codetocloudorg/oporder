// Package cloudflare implements the read-only Cloudflare inventory
// connector from SPEC.md §5.2. Same discipline as the AWS and Azure
// connectors: only List/Get-style SDK calls exist in this file.
//
// Cloudflare has no public pricing API (§5.6, §12) — this package covers
// inventory only. Any cost figure derived from Cloudflare resources stays
// labeled "directional, verify before committing" per the decision already
// made in §12, never silently upgraded to look as live-priced as the other
// three providers.
package cloudflare

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

// Client is a thin, read-only wrapper around the official Cloudflare Go
// SDK v4.
type Client struct {
	cf *cloudflare.Client
}

// Zone is the minimal shape §5.2's inventory needs from a Cloudflare zone —
// the closest Cloudflare concept to an AWS/Azure "resource group."
type Zone struct {
	ID     string
	Name   string
	Status string
}

// NewClient builds a client from an API token. TokenFromFile reads the
// token the way this project actually stores it locally (see
// SECURITY.md and CONTRIBUTING.md) — never as a value typed into a commit,
// a log line, or a chat session.
func NewClient(apiToken string) *Client {
	return &Client{
		cf: cloudflare.NewClient(option.WithAPIToken(apiToken)),
	}
}

// TokenFromFile reads a Cloudflare API token from a local file — the
// pattern this project actually used to onboard a real token during
// development: saved to a file with 0600 permissions, never pasted into
// chat or committed. Returns an error naming the expected location rather
// than a bare "not found," per §13.1's actionable-error standard.
func TokenFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cloudflare: reading token from %s: %w (generate one at https://dash.cloudflare.com/profile/api-tokens and save it there)", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ListZones returns every zone visible to the token's permissions. An empty
// slice with a nil error is a valid result for an account with no zones —
// same "empty is not an error" handling as the AWS and Azure connectors.
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	var out []Zone

	page, err := c.cf.Zones.List(ctx, zones.ZoneListParams{})
	if err != nil {
		return nil, fmt.Errorf("cloudflare: listing zones: %w", err)
	}
	for page != nil {
		for _, z := range page.Result {
			out = append(out, Zone{
				ID:     z.ID,
				Name:   z.Name,
				Status: string(z.Status),
			})
		}
		page, err = page.GetNextPage()
		if err != nil {
			return nil, fmt.Errorf("cloudflare: paginating zones: %w", err)
		}
	}
	return out, nil
}
