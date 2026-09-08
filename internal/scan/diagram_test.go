package scan

import (
	"strings"
	"testing"

	"github.com/codetocloudorg/oporder/internal/codescan"
	"github.com/codetocloudorg/oporder/internal/correlate"
)

func TestMermaidDiagram_Empty(t *testing.T) {
	if got := mermaidDiagram(Situation{}); got != "" {
		t.Errorf("mermaidDiagram(empty) = %q, want empty string", got)
	}
}

func TestMermaidDiagram_MatchedAndOrphaned(t *testing.T) {
	s := Situation{
		Workloads: []codescan.Workload{
			{Name: "billing", Path: "cmd/billing"},
		},
		Unclassified: []codescan.Unclassified{
			{Path: "scripts", Reason: "no manifest found"},
		},
		Correlation: correlate.Result{
			Matches: []correlate.Match{
				{
					WorkloadPath: "cmd/billing",
					WorkloadName: "billing",
					Resource:     correlate.Resource{Provider: "aws", ID: "i-1", Name: "billing"},
					Confidence:   correlate.ConfidenceTag,
				},
			},
			UnmatchedResources: []correlate.Resource{
				{Provider: "aws", ID: "i-2", Name: "legacy-vm"},
			},
		},
	}

	got := mermaidDiagram(s)
	for _, want := range []string{
		"```mermaid", "graph LR",
		`W0["billing"]`, `U0["? scripts"]`,
		`R0["billing (aws)"]`, `R1["legacy-vm (aws)"]`,
		"W0 -->|tag-match| R0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("diagram missing %q; got:\n%s", want, got)
		}
	}
}

func TestSanitizeLabel(t *testing.T) {
	if got := sanitizeLabel(`say "hi"` + "\nline2"); strings.ContainsAny(got, "\"\n") {
		t.Errorf("sanitizeLabel left unsafe characters: %q", got)
	}
}
