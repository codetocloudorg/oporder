// Package aws implements the read-only AWS inventory connector from
// SPEC.md §5.2. Same discipline as internal/provider/azure: only GET/
// Describe-style SDK calls exist in this file — no create, modify, or
// terminate operation is implemented, by construction, per §1's
// read-only-by-design principle.
package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
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
	cwClient  *cloudwatch.Client
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
		cwClient:  cloudwatch.NewFromConfig(cfg),
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

// Utilization is one instance's CPU utilization over a lookback window —
// SPEC.md §5.2's "the primary signal for flagging retire candidates."
// HasTelemetry is deliberately separate from AverageCPUPercent: an
// instance CloudWatch has no datapoints for isn't necessarily idle, it
// might just be un-instrumented, and §12's gap analysis requires the
// rubric never conflate the two into a false "retire" signal.
type Utilization struct {
	InstanceID        string
	AverageCPUPercent float64
	HasTelemetry      bool
}

// CPUUtilization queries CloudWatch's AWS/EC2 CPUUtilization metric for
// one instance over the given lookback window, using 1-hour datapoints.
// Read-only, same as every other method in this package: GetMetricStatistics
// is the CloudWatch equivalent of DescribeInstances, not a mutating call.
func (c *Client) CPUUtilization(ctx context.Context, instanceID string, lookback time.Duration) (Utilization, error) {
	end := time.Now()
	start := end.Add(-lookback)

	out, err := c.cwClient.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/EC2"),
		MetricName: aws.String("CPUUtilization"),
		Dimensions: []cwtypes.Dimension{
			{Name: aws.String("InstanceId"), Value: aws.String(instanceID)},
		},
		StartTime:  &start,
		EndTime:    &end,
		Period:     aws.Int32(3600),
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage},
	})
	if err != nil {
		return Utilization{}, fmt.Errorf("aws: getting CPU utilization for %s: %w", instanceID, err)
	}

	if len(out.Datapoints) == 0 {
		return Utilization{InstanceID: instanceID, HasTelemetry: false}, nil
	}
	return Utilization{
		InstanceID:        instanceID,
		AverageCPUPercent: averageOf(out.Datapoints),
		HasTelemetry:      true,
	}, nil
}

// averageOf is the pure-logic piece of CPUUtilization that's actually
// unit-testable without a live account — same role as tagsFrom and
// stateName in this file.
func averageOf(datapoints []cwtypes.Datapoint) float64 {
	var sum float64
	var count int
	for _, d := range datapoints {
		if d.Average != nil {
			sum += *d.Average
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
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
