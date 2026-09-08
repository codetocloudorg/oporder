// Package gcp implements the read-only Google Cloud inventory connector
// from SPEC.md §5.2. Same discipline as internal/provider/aws and
// internal/provider/azure: only List/AggregatedList-style SDK calls exist
// in this file — no create, modify, or delete operation is implemented,
// by construction, per §1's read-only-by-design principle.
//
// Unverified against a real account: unlike the AWS, Azure, and
// Cloudflare connectors, this package has not been live-tested against a
// real GCP project — no GCP account was available in the environment
// this was built in. It follows the same patterns and the same official
// SDK conventions as the other three, and its pure-logic pieces are unit
// tested, but per SPEC.md §10's M1 exit criteria it isn't "done" until
// someone runs it against a real project and confirms the output.
package gcp

import (
	"context"
	"fmt"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
)

// Client is a thin, read-only wrapper around the official Google Cloud Go
// SDK. Authentication goes through Application Default Credentials —
// whatever's already configured via `gcloud auth application-default
// login`, a service account key in GOOGLE_APPLICATION_CREDENTIALS, or a
// workload identity — the same "reuse what's already there, don't ask for
// a separate credential" approach as the AWS and Azure connectors.
type Client struct {
	projectID string
	instances *compute.InstancesClient
}

// Instance is the minimal shape §5.2's inventory needs from a Compute
// Engine instance — enough to feed §5.0's correlation, not the full,
// much larger SDK response type.
type Instance struct {
	Name   string
	Zone   string
	Status string
	Labels map[string]string
}

// NewClient loads Application Default Credentials for the given GCP
// project and returns a client, or an error immediately if no usable
// credentials are found — same "fail fast and clearly" standard as the
// AWS and Azure connectors' NewClient.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	if projectID == "" {
		return nil, fmt.Errorf("gcp: project ID required (set OPORDER_GCP_PROJECT_ID)")
	}
	c, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: creating instances client (tried Application Default Credentials — run `gcloud auth application-default login`): %w", err)
	}
	return &Client{projectID: projectID, instances: c}, nil
}

// Close releases the underlying REST client's resources. Callers should
// defer this after a successful NewClient, same as any other GCP client.
func (c *Client) Close() error {
	return c.instances.Close()
}

// ListInstances returns every Compute Engine instance across every zone
// in the configured project, using AggregatedList so callers don't need
// to enumerate zones themselves. An empty slice with a nil error is the
// correct result for a project with nothing running, same as the AWS and
// Azure connectors' handling of an empty account.
func (c *Client) ListInstances(ctx context.Context) ([]Instance, error) {
	var out []Instance

	req := &computepb.AggregatedListInstancesRequest{Project: c.projectID}
	it := c.instances.AggregatedList(ctx, req)
	for {
		pair, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gcp: listing instances: %w", err)
		}
		if pair.Value == nil {
			continue
		}
		for _, inst := range pair.Value.Instances {
			out = append(out, Instance{
				Name:   inst.GetName(),
				Zone:   zoneFromURL(inst.GetZone()),
				Status: inst.GetStatus(),
				Labels: inst.GetLabels(),
			})
		}
	}
	return out, nil
}

// zoneFromURL extracts the short zone name (e.g. "us-central1-a") from
// the full resource URL the Compute Engine API returns — the pure-logic
// piece of this file that's actually unit-testable without a live
// account, same role as tagsFrom in the AWS and Azure connectors.
func zoneFromURL(u string) string {
	for i := len(u) - 1; i >= 0; i-- {
		if u[i] == '/' {
			return u[i+1:]
		}
	}
	return u
}
