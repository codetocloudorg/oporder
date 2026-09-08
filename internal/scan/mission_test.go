package scan

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/codetocloudorg/oporder/internal/correlate"
	"github.com/codetocloudorg/oporder/internal/debtdelta"
	"github.com/codetocloudorg/oporder/internal/narrative"
	"github.com/codetocloudorg/oporder/internal/rubric"
)

func TestWriteMissionMD_NoMissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MISSION.md")

	if err := WriteMissionMD(path, Situation{}); err != nil {
		t.Fatalf("WriteMissionMD: %v", err)
	}
	got := readFile(t, path)
	if !strings.Contains(got, "nothing to call yet") {
		t.Errorf("expected the no-missions notice, got:\n%s", got)
	}
}

func TestWriteMissionMD_RealRetireCall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MISSION.md")

	retireCall := rubric.Evaluate(rubric.Evidence{TelemetryAvailable: true, NoOrNegligibleTraffic: true})
	insufficientCall := rubric.Evaluate(rubric.Evidence{})

	s := Situation{
		Missions: []Mission{
			{
				WorkloadName: "idle-worker",
				WorkloadPath: "cmd/idle-worker",
				Resource:     correlate.Resource{Provider: "aws", ID: "i-1", Name: "idle-worker"},
				Result:       retireCall,
				DebtDelta:    debtdelta.Assess(retireCall.R, debtdelta.ExecutionContext{}),
			},
			{
				WorkloadName: "unknown-workload",
				WorkloadPath: "cmd/unknown",
				Resource:     correlate.Resource{Provider: "aws", ID: "i-2", Name: "unknown-workload"},
				Result:       insufficientCall,
				DebtDelta:    debtdelta.Assess(insufficientCall.R, debtdelta.ExecutionContext{}),
			},
		},
	}

	if err := WriteMissionMD(path, s); err != nil {
		t.Fatalf("WriteMissionMD: %v", err)
	}
	got := readFile(t, path)

	if s.Missions[0].Result.R != rubric.Retire {
		t.Fatalf("sanity check failed: expected rubric.Evaluate to call Retire from telemetry alone, got %+v", s.Missions[0].Result)
	}
	if s.Missions[0].DebtDelta.Trajectory != debtdelta.FullElimination {
		t.Errorf("sanity check failed: expected Retire's debt trajectory to be full-elimination, got %+v", s.Missions[0].DebtDelta)
	}
	for _, want := range []string{"idle-worker", "retire", "full-elimination", "unknown-workload", "insufficient-evidence"} {
		if !strings.Contains(got, want) {
			t.Errorf("MISSION.md missing %q; got:\n%s", want, got)
		}
	}
}

func TestWriteMissionMD_NarrativesRenderEvenWithNoMissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MISSION.md")

	// A code-only scan (no cloud account, so no Missions at all) should
	// still produce a real "In plain English" section — narratives cover
	// every detected workload, not just correlated ones.
	s := Situation{
		Narratives: []narrative.Paragraph{
			{WorkloadName: "worker", WorkloadPath: "cmd/worker", Text: "worker is probably a deployable workload."},
		},
	}

	if err := WriteMissionMD(path, s); err != nil {
		t.Fatalf("WriteMissionMD: %v", err)
	}
	got := readFile(t, path)

	if !strings.Contains(got, "nothing to call yet") {
		t.Errorf("expected the no-missions notice to still appear, got:\n%s", got)
	}
	if !strings.Contains(got, "## In plain English") {
		t.Errorf("expected the plain-English section, got:\n%s", got)
	}
	if !strings.Contains(got, "worker is probably a deployable workload.") {
		t.Errorf("expected the narrative paragraph, got:\n%s", got)
	}
}
