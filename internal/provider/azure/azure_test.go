package azure

import (
	"reflect"
	"testing"
)

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
