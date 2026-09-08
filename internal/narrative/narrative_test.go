package narrative

import (
	"strings"
	"testing"

	"github.com/codetocloudorg/oporder/internal/codescan"
	"github.com/codetocloudorg/oporder/internal/correlate"
	"github.com/codetocloudorg/oporder/internal/debtdelta"
	"github.com/codetocloudorg/oporder/internal/depscan"
	"github.com/codetocloudorg/oporder/internal/rubric"
)

func TestExplain_NoEvidenceAtAll(t *testing.T) {
	in := Input{
		Workload: codescan.Workload{
			Name: "worker", Path: "cmd/worker", Confidence: codescan.ConfidenceMedium,
			Signal: "go.mod + cmd/worker/main.go",
		},
	}
	got := Explain([]Input{in})
	if len(got) != 1 {
		t.Fatalf("got %d paragraphs, want 1", len(got))
	}
	text := got[0].Text
	for _, want := range []string{
		"worker", "cmd/worker", "standalone Go binary",
		"No known cloud-vendor SDK dependencies",
		"isn't yet enough evidence gathered",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("paragraph missing %q; got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "No matching live resource") {
		t.Errorf("should not mention correlation when correlation wasn't attempted; got:\n%s", text)
	}
}

func TestExplain_ProprietaryDependencyRulesOutRehost(t *testing.T) {
	in := Input{
		Workload: codescan.Workload{
			Name: "billing", Path: "cmd/billing", Confidence: codescan.ConfidenceMedium,
			Signal: "go.mod + cmd/billing/main.go",
		},
		Dependencies: depscan.Result{Findings: []depscan.Finding{
			{Provider: depscan.ProviderAWS, Package: "github.com/aws/aws-sdk-go", Source: "go.mod"},
		}},
		Unmatched: true,
	}
	got := Explain([]Input{in})[0].Text
	for _, want := range []string{
		"AWS SDK",
		"rules out a plain rehost",
		"No matching live resource",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("paragraph missing %q; got:\n%s", want, got)
		}
	}
}

func TestExplain_RealRetireCall(t *testing.T) {
	call := rubric.Evaluate(rubric.Evidence{TelemetryAvailable: true, NoOrNegligibleTraffic: true})
	in := Input{
		Workload: codescan.Workload{
			Name: "idle-worker", Path: "cmd/idle-worker", Confidence: codescan.ConfidenceMedium,
			Signal: "go.mod + cmd/idle-worker/main.go",
		},
		Match: &correlate.Match{
			WorkloadPath: "cmd/idle-worker", WorkloadName: "idle-worker",
			Resource:   correlate.Resource{Provider: "aws", ID: "i-1", Name: "idle-worker"},
			Confidence: correlate.ConfidenceTag,
		},
		Mission: &MissionEvidence{
			Result:    call,
			DebtDelta: debtdelta.Assess(call.R, debtdelta.ExecutionContext{}),
		},
	}
	got := Explain([]Input{in})[0].Text
	for _, want := range []string{
		`running today as "idle-worker" on aws`,
		"recommendation is to retire",
		"full elimination",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("paragraph missing %q; got:\n%s", want, got)
		}
	}
}

func TestExplain_DockerfileWorkload(t *testing.T) {
	in := Input{
		Workload: codescan.Workload{
			Name: "api", Path: "services/api", Confidence: codescan.ConfidenceHigh,
			Signal: "Dockerfile",
		},
	}
	got := Explain([]Input{in})[0].Text
	if !strings.Contains(got, "already containerized") {
		t.Errorf("expected Dockerfile-based workload to mention containerization; got:\n%s", got)
	}
	if !strings.Contains(got, "clearly a deployable workload") {
		t.Errorf("expected high confidence to say 'clearly'; got:\n%s", got)
	}
}

func TestExplain_PreservesOrderAndCount(t *testing.T) {
	inputs := []Input{
		{Workload: codescan.Workload{Name: "a", Path: "a", Signal: "Dockerfile"}},
		{Workload: codescan.Workload{Name: "b", Path: "b", Signal: "Dockerfile"}},
		{Workload: codescan.Workload{Name: "c", Path: "c", Signal: "Dockerfile"}},
	}
	got := Explain(inputs)
	if len(got) != 3 {
		t.Fatalf("got %d paragraphs, want 3", len(got))
	}
	for i, name := range []string{"a", "b", "c"} {
		if got[i].WorkloadName != name {
			t.Errorf("paragraph %d name = %q, want %q", i, got[i].WorkloadName, name)
		}
	}
}
