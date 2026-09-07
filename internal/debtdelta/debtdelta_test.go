package debtdelta

import (
	"testing"

	"github.com/codetocloudorg/oporder/internal/rubric"
)

func TestAssess_SimpleRs(t *testing.T) {
	cases := []struct {
		r    rubric.R
		want Trajectory
	}{
		{rubric.Retain, Unchanged},
		{rubric.Rehost, UsuallyUnchanged},
		{rubric.Repatriate, UsuallyUnchanged},
		{rubric.Replatform, ModestReduction},
		{rubric.Repurchase, Reduction},
		{rubric.Retire, FullElimination},
	}

	for _, tc := range cases {
		t.Run(string(tc.r), func(t *testing.T) {
			got := Assess(tc.r, ExecutionContext{})
			if got.Trajectory != tc.want {
				t.Fatalf("Assess(%s) = %q, want %q", tc.r, got.Trajectory, tc.want)
			}
			if got.Reasoning == "" {
				t.Error("every result must state its reasoning, per §1's auditability principle")
			}
		})
	}
}

// TestAssess_RehostNeverImpliesFixed is the specific regression this
// package exists to prevent: §5.8 is explicit that "we migrated" must never
// be allowed to imply "we fixed something" for a Rehost.
func TestAssess_RehostNeverImpliesFixed(t *testing.T) {
	got := Assess(rubric.Rehost, ExecutionContext{})
	if got.Trajectory == Reduction || got.Trajectory == FullElimination {
		t.Fatalf("Rehost must never report a reduction trajectory, got %q", got.Trajectory)
	}
}

func TestAssess_RefactorHeavyAIGenerationRisksIncrease(t *testing.T) {
	got := Assess(rubric.Refactor, ExecutionContext{HeavyAIGenerationLightReview: true})

	if got.Trajectory != RiskOfIncrease {
		t.Fatalf("Trajectory = %q, want %q when execution leans on heavy AI generation with light review", got.Trajectory, RiskOfIncrease)
	}
	if !got.IsProjection {
		t.Error("Refactor/Rearchitect trajectories must be marked IsProjection — §5.8 requires the projection/measurement distinction to survive into the output")
	}
	if got.Reasoning == "" {
		t.Error("the GitClear-derived caution must be surfaced as reasoning, not a bare confidence number")
	}
}

func TestAssess_RearchitectHighSDLCMaturityFavorsReduction(t *testing.T) {
	got := Assess(rubric.Rearchitect, ExecutionContext{SDLCMaturityHigh: true})

	if got.Trajectory != ReductionIfExecutedWell {
		t.Fatalf("Trajectory = %q, want %q with high SDLC maturity and no AI-heavy flag", got.Trajectory, ReductionIfExecutedWell)
	}
	if !got.IsProjection {
		t.Error("Rearchitect must be marked IsProjection — it's a future state, not a measurement")
	}
}

func TestAssess_RefactorWithNoContextStillProjectsNotMeasures(t *testing.T) {
	got := Assess(rubric.Refactor, ExecutionContext{})

	if !got.IsProjection {
		t.Fatal("a Refactor call with zero execution context must still be marked as a projection, never treated as a measured fact")
	}
	if got.Trajectory != ReductionIfExecutedWell {
		t.Fatalf("Trajectory = %q, want %q as the default absent contrary evidence", got.Trajectory, ReductionIfExecutedWell)
	}
}

// TestAssess_NonRefactorRsAreNeverMarkedAsProjections checks the other side
// of the projection/measurement distinction: only Refactor/Rearchitect
// describe a future state. Retain, Rehost, Replatform, Repurchase, Retire,
// and Repatriate all describe what the R structurally does or doesn't touch
// — that's knowable now, not a forecast.
func TestAssess_NonRefactorRsAreNeverMarkedAsProjections(t *testing.T) {
	for _, r := range []rubric.R{
		rubric.Retain, rubric.Rehost, rubric.Repatriate,
		rubric.Replatform, rubric.Repurchase, rubric.Retire,
	} {
		got := Assess(r, ExecutionContext{})
		if got.IsProjection {
			t.Errorf("Assess(%s).IsProjection = true, want false — only Refactor/Rearchitect are projections per §5.8", r)
		}
	}
}
