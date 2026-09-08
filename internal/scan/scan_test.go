package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codetocloudorg/oporder/internal/codescan"
	"github.com/codetocloudorg/oporder/internal/correlate"
)

func TestWriteSituationMD_NoProviderConfigured(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SITUATION.md")

	if err := WriteSituationMD(path, Situation{}); err != nil {
		t.Fatalf("WriteSituationMD: %v", err)
	}
	got := readFile(t, path)
	if !strings.Contains(got, "No provider connected") {
		t.Errorf("expected the no-provider notice, got:\n%s", got)
	}
	if strings.Contains(got, "## Code") || strings.Contains(got, "## Correlation") {
		t.Errorf("expected no Code/Correlation sections with nothing to report, got:\n%s", got)
	}
}

func TestWriteSituationMD_FullReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SITUATION.md")

	s := Situation{
		Providers: []ProviderResult{
			{Provider: "aws", Ran: true, Count: 2, Detail: "2 EC2 instance(s) in us-east-1"},
			{Provider: "azure", Ran: false, Detail: "not connected — no active az login session"},
		},
		Workloads: []codescan.Workload{
			{Name: "billing", Path: "cmd/billing", Confidence: codescan.ConfidenceMedium, Signal: "go.mod + cmd/billing/main.go"},
		},
		Unclassified: []codescan.Unclassified{
			{Path: "scripts", Reason: "source files present but no manifest or module-boundary file found"},
		},
		Correlation: correlate.Result{
			Matches: []correlate.Match{
				{
					WorkloadPath: "cmd/billing",
					WorkloadName: "billing",
					Resource:     correlate.Resource{Provider: "aws", ID: "i-1", Name: "billing"},
					Confidence:   correlate.ConfidenceTag,
					Reason:       `resource tag "service" matches workload name`,
				},
			},
			UnmatchedResources: []correlate.Resource{
				{Provider: "aws", ID: "i-2", Name: "orphan-vm"},
			},
		},
	}

	if err := WriteSituationMD(path, s); err != nil {
		t.Fatalf("WriteSituationMD: %v", err)
	}
	got := readFile(t, path)

	for _, want := range []string{
		"### aws",
		"2 EC2 instance(s) in us-east-1",
		"### azure — not connected",
		"## Code",
		"billing",
		"cmd/billing/main.go",
		"## Correlation",
		"```mermaid",
		"resource tag \"service\" matches workload name",
		"orphan-vm",
		"scripts",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("SITUATION.md missing %q; got:\n%s", want, got)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}
