// Package aws implements the read-only AWS inventory connector from
// SPEC.md §5.2. Same discipline as internal/provider/azure: only GET/
// Describe-style SDK calls exist in this file — no create, modify, or
// terminate operation is implemented, by construction, per §1's
// read-only-by-design principle.
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Client is a thin, read-only wrapper around the official AWS SDK for Go
// v2. Authentication goes through the SDK's default credential chain —
// whatever's already configured via `aws configure`, environment variables,
// or an IAM role — the same "reuse what's already there, don't ask for a
// separate credential" approach as the Azure connector.
type Client struct {
	ec2Client *ec2.Client
	stsClient *sts.Client
	region    string
}

// Instance is the minimal shape §5.2's inventory needs from an EC2
// instance — enough to feed §5.0's correlation, not the full, much larger
// SDK response type.
type Instance struct {
	ID    string
	Type  string
	State string
	Tags  map[string]string
}

// NewClient loads the AWS SDK's default credential chain for the given
// region and returns a client, or an error immediately if no usable
// credentials are found — same "fail fast and clearly" standard as the
// Azure connector's NewClient.
func NewClient(ctx context.Context, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("aws: loading default config: %w", err)
	}
	return &Client{
		ec2Client: ec2.NewFromConfig(cfg),
		stsClient: sts.NewFromConfig(cfg),
		region:    region,
	}, nil
}

// WhoAmI confirms the configured credentials actually work and reports back
// only what's needed to prove connectivity — the account ID and ARN, not
// anything more identifying than that. Callers of this package should log
// or display these only in ways that don't end up in a public commit,
// per SECURITY.md's rule on real account identifiers.
func (c *Client) WhoAmI(ctx context.Context) (accountID, arn string, err error) {
	out, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", "", fmt.Errorf("aws: get caller identity: %w", err)
	}
	return aws.ToString(out.Account), aws.ToString(out.Arn), nil
}

// ListInstances returns every EC2 instance in the configured region across
// all reservations. An empty slice with a nil error is the correct result
// for a region with nothing running, same as the Azure connector's handling
// of an empty subscription — not treated as an error case.
func (c *Client) ListInstances(ctx context.Context) ([]Instance, error) {
	var out []Instance

	paginator := ec2.NewDescribeInstancesPaginator(c.ec2Client, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("aws: describing instances: %w", err)
		}
		for _, reservation := range page.Reservations {
			for _, inst := range reservation.Instances {
				out = append(out, Instance{
					ID:    aws.ToString(inst.InstanceId),
					Type:  string(inst.InstanceType),
					State: stateName(inst.State),
					Tags:  tagsFrom(inst.Tags),
				})
			}
		}
	}
	return out, nil
}

func stateName(s *types.InstanceState) string {
	if s == nil {
		return ""
	}
	return string(s.Name)
}

// tagsFrom converts the SDK's []types.Tag (a slice of Key/Value pointer
// pairs, matching AWS's actual API shape) into a plain map[string]string —
// the pure-logic piece of this file that's actually unit-testable without a
// live account. Same pattern as the Azure connector's tagsFrom, adapted to
// AWS's slice-of-pairs shape rather than Azure's map-of-pointers shape.
func tagsFrom(src []types.Tag) map[string]string {
	out := map[string]string{}
	for _, t := range src {
		if t.Key != nil && t.Value != nil {
			out[*t.Key] = *t.Value
		}
	}
	return out
}
