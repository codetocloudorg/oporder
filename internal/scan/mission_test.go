package scan

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/codetocloudorg/oporder/internal/correlate"
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

	s := Situation{
		Missions: []Mission{
			{
				WorkloadName: "idle-worker",
				WorkloadPath: "cmd/idle-worker",
				Resource:     correlate.Resource{Provider: "aws", ID: "i-1", Name: "idle-worker"},
				Result:       rubric.Evaluate(rubric.Evidence{TelemetryAvailable: true, NoOrNegligibleTraffic: true}),
			},
			{
				WorkloadName: "unknown-workload",
				WorkloadPath: "cmd/unknown",
				Resource:     correlate.Resource{Provider: "aws", ID: "i-2", Name: "unknown-workload"},
				Result:       rubric.Evaluate(rubric.Evidence{}),
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
	for _, want := range []string{"idle-worker", "retire", "unknown-workload", "insufficient-evidence"} {
		if !strings.Contains(got, want) {
			t.Errorf("MISSION.md missing %q; got:\n%s", want, got)
		}
	}
}
