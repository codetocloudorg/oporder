package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TODO(§7): ListInstances and WhoAmI are not unit tested here — same reason
// as the Azure connector's ListResourceGroups/ListResources: they depend
// directly on the AWS SDK's concrete client types, which need either a live
// account or an HTTP-fixture recording/replay harness to test properly.
// Verified manually against a real account via cmd/live-aws-check.

func TestTagsFrom(t *testing.T) {
	cases := []struct {
		name string
		in   []types.Tag
		want map[string]string
	}{
		{
			name: "nil slice",
			in:   nil,
			want: map[string]string{},
		},
		{
			name: "empty slice",
			in:   []types.Tag{},
			want: map[string]string{},
		},
		{
			name: "normal tags",
			in: []types.Tag{
				{Key: aws.String("service"), Value: aws.String("billing")},
				{Key: aws.String("env"), Value: aws.String("prod")},
			},
			want: map[string]string{"service": "billing", "env": "prod"},
		},
		{
			name: "nil key or value skipped, matching AWS's actual API shape",
			in: []types.Tag{
				{Key: aws.String("service"), Value: aws.String("billing")},
				{Key: nil, Value: aws.String("orphan-value")},
				{Key: aws.String("orphan-key"), Value: nil},
			},
			want: map[string]string{"service": "billing"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tagsFrom(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tagsFrom(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestStateName(t *testing.T) {
	if got := stateName(nil); got != "" {
		t.Errorf("stateName(nil) = %q, want empty string", got)
	}
	running := types.InstanceStateNameRunning
	s := &types.InstanceState{Name: running}
	if got := stateName(s); got != "running" {
		t.Errorf("stateName(running) = %q, want %q", got, "running")
	}
}

func TestAverageOf(t *testing.T) {
	cases := []struct {
		name string
		in   []cwtypes.Datapoint
		want float64
	}{
		{name: "no datapoints", in: nil, want: 0},
		{
			name: "single datapoint",
			in:   []cwtypes.Datapoint{{Average: aws.Float64(12.5)}},
			want: 12.5,
		},
		{
			name: "multiple datapoints",
			in: []cwtypes.Datapoint{
				{Average: aws.Float64(10)},
				{Average: aws.Float64(20)},
				{Average: aws.Float64(30)},
			},
			want: 20,
		},
		{
			name: "nil averages skipped, not treated as zero",
			in: []cwtypes.Datapoint{
				{Average: aws.Float64(10)},
				{Average: nil},
				{Average: aws.Float64(30)},
			},
			want: 20,
		},
		{
			name: "all nil",
			in: []cwtypes.Datapoint{
				{Average: nil},
			},
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := averageOf(tc.in); got != tc.want {
				t.Errorf("averageOf(%+v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
