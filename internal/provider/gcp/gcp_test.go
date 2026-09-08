package gcp

import "testing"

func TestZoneFromURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "full compute API URL",
			in:   "https://www.googleapis.com/compute/v1/projects/example/zones/us-central1-a",
			want: "us-central1-a",
		},
		{
			name: "already-short zone",
			in:   "us-east1-b",
			want: "us-east1-b",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zoneFromURL(tt.in); got != tt.want {
				t.Errorf("zoneFromURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
