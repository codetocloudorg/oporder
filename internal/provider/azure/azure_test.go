package azure

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
)

func f64(v float64) *float64 { return &v }

// TODO(§7): ListResourceGroups and ListResources themselves are not unit
// tested here — they depend directly on the Azure SDK's concrete pager
// types, which need either a live subscription or an HTTP-fixture
// recording/replay harness to test properly. Verified manually against a
// real, empty subscription via cmd/live-azure-check; the recorded-fixture
// mode §7 already commits to for every provider connector is the honest
// next step, not built here yet.

func s(v string) *string { return &v }

func TestTagsFrom(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]*string
		want map[string]string
	}{
		{
			name: "nil map",
			in:   nil,
			want: map[string]string{},
		},
		{
			name: "empty map",
			in:   map[string]*string{},
			want: map[string]string{},
		},
		{
			name: "normal tags",
			in:   map[string]*string{"service": s("billing"), "env": s("prod")},
			want: map[string]string{"service": "billing", "env": "prod"},
		},
		{
			name: "nil value skipped, matching Azure's actual API shape",
			in:   map[string]*string{"service": s("billing"), "broken": nil},
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

func TestAveragesFrom(t *testing.T) {
	cases := []struct {
		name string
		in   []*armmonitor.Metric
		want []float64
	}{
		{name: "no metrics", in: nil, want: nil},
		{
			name: "nil metric and nil timeseries entries skipped",
			in: []*armmonitor.Metric{
				nil,
				{Timeseries: []*armmonitor.TimeSeriesElement{nil}},
			},
			want: nil,
		},
		{
			name: "single metric, single timeseries",
			in: []*armmonitor.Metric{
				{Timeseries: []*armmonitor.TimeSeriesElement{
					{Data: []*armmonitor.MetricValue{
						{Average: f64(10)},
						{Average: f64(20)},
						{Average: nil}, // skipped, not treated as zero
					}},
				}},
			},
			want: []float64{10, 20},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := averagesFrom(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("averagesFrom(...) = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMeanOf(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{name: "empty", in: nil, want: 0},
		{name: "single value", in: []float64{12.5}, want: 12.5},
		{name: "multiple values", in: []float64{10, 20, 30}, want: 20},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := meanOf(tc.in); got != tc.want {
				t.Errorf("meanOf(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
