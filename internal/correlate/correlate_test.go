package correlate

import "testing"

func TestCorrelate_TagMatchTakesPriorityOverName(t *testing.T) {
	workloads := []Workload{{Name: "billing-service", Path: "services/billing"}}
	resources := []Resource{
		{Provider: "aws", ID: "i-1", Name: "billing-service-old", Tags: nil}, // would name-match, but shouldn't win
		{Provider: "aws", ID: "i-2", Name: "prod-web-07", Tags: map[string]string{"service": "billing-service"}},
	}

	got := Correlate(workloads, resources)
	if len(got.Matches) != 1 {
		t.Fatalf("matches = %+v, want exactly 1", got.Matches)
	}
	m := got.Matches[0]
	if m.Confidence != ConfidenceTag || m.Resource.ID != "i-2" {
		t.Errorf("match = %+v, want tag-match against i-2", m)
	}
	if len(got.UnmatchedResources) != 1 || got.UnmatchedResources[0].ID != "i-1" {
		t.Errorf("unmatched resources = %+v, want exactly i-1 left over", got.UnmatchedResources)
	}
	if len(got.UnmatchedWorkloads) != 0 {
		t.Errorf("unmatched workloads = %+v, want none", got.UnmatchedWorkloads)
	}
}

func TestCorrelate_NameMatchFallback(t *testing.T) {
	workloads := []Workload{{Name: "billing_service", Path: "services/billing"}}
	resources := []Resource{{Provider: "gcp", ID: "vm-1", Name: "Billing-Service-01"}}

	got := Correlate(workloads, resources)
	if len(got.Matches) != 1 {
		t.Fatalf("matches = %+v, want exactly 1", got.Matches)
	}
	if got.Matches[0].Confidence != ConfidenceName {
		t.Errorf("confidence = %s, want name-match", got.Matches[0].Confidence)
	}
}

func TestCorrelate_UnmatchedBothSides(t *testing.T) {
	workloads := []Workload{{Name: "orphan-code", Path: "services/orphan"}}
	resources := []Resource{{Provider: "azure", ID: "r1", Name: "legacy-vm-nobody-remembers"}}

	got := Correlate(workloads, resources)
	if len(got.Matches) != 0 {
		t.Fatalf("matches = %+v, want none", got.Matches)
	}
	if len(got.UnmatchedWorkloads) != 1 || got.UnmatchedWorkloads[0].Name != "orphan-code" {
		t.Errorf("unmatched workloads = %+v", got.UnmatchedWorkloads)
	}
	if len(got.UnmatchedResources) != 1 || got.UnmatchedResources[0].ID != "r1" {
		t.Errorf("unmatched resources = %+v", got.UnmatchedResources)
	}
}

func TestCorrelate_OneResourceNotDoubleMatched(t *testing.T) {
	// Two workloads could both plausibly name-match one resource; only
	// one match should be produced, and the loser stays unmatched rather
	// than the resource being attached twice.
	workloads := []Workload{
		{Name: "api", Path: "services/api"},
		{Name: "api-gateway", Path: "services/api-gateway"},
	}
	resources := []Resource{{Provider: "aws", ID: "i-1", Name: "api"}}

	got := Correlate(workloads, resources)
	if len(got.Matches) != 1 {
		t.Fatalf("matches = %+v, want exactly 1 (resource can't match twice)", got.Matches)
	}
	if len(got.UnmatchedWorkloads) != 1 {
		t.Fatalf("unmatched workloads = %+v, want exactly 1 left over", got.UnmatchedWorkloads)
	}
}

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"billing-service":  "billingservice",
		"billing_service":  "billingservice",
		"BillingService":   "billingservice",
		"Billing Service!": "billingservice",
		"":                 "",
	}
	for in, want := range tests {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
