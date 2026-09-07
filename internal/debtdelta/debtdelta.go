// Package debtdelta implements the projected technical debt trajectory from
// SPEC.md §5.8 — the question cost and effort (§5.6) don't answer: does the
// organization's debt go down, stay flat, or go up as a result of the
// recommended R?
//
// This package covers only the "projected trajectory" half of §5.8. The
// "baseline debt fingerprint" half (cyclomatic complexity, duplication,
// dependency staleness, measured directly from a real codebase) depends on
// §5.1's code analysis layer, which doesn't exist yet — implementing a
// baseline fingerprint against nothing to measure would be exactly the kind
// of unearned precision this spec repeatedly rules out elsewhere.
package debtdelta

import "github.com/codetocloudorg/oporder/internal/rubric"

// Trajectory is the projected direction of an organization's technical debt
// if the recommended R is carried out.
type Trajectory string

const (
	Unchanged               Trajectory = "unchanged"
	UsuallyUnchanged        Trajectory = "usually-unchanged" // Rehost: "usually unchanged, sometimes worse"
	ModestReduction         Trajectory = "modest-reduction"
	Reduction               Trajectory = "reduction"
	FullElimination         Trajectory = "full-elimination"
	ReductionIfExecutedWell Trajectory = "reduction-if-executed-well"
	RiskOfIncrease          Trajectory = "risk-of-increase"
)

// ExecutionContext captures what's known about how a Refactor or Rearchitect
// would actually be carried out — the only two Rs whose trajectory §5.8
// says has "the widest variance of any R," and the only two where this
// context changes the answer.
type ExecutionContext struct {
	// HeavyAIGenerationLightReview is true when the stated execution plan
	// leans on substantial AI-generated code with light human review. This
	// is the specific, named risk factor from §5.8's own citation: GitClear
	// found duplicated code blocks rose roughly 8x in frequency industry-
	// wide as AI-generated code volume grew, while "moved lines" (their
	// refactoring proxy) fell from ~25% to under 10% of changed lines over
	// the same period. A mechanism, not a guarantee — see Result.Reasoning
	// for how this gets surfaced as context, never as a bare confidence
	// number, per §5.8's explicit instruction.
	HeavyAIGenerationLightReview bool

	// SDLCMaturityHigh reflects §5.5's scoring — a team with strong test
	// coverage, gated CI, and disciplined review practice is materially
	// less likely to have a Refactor/Rearchitect regress silently.
	SDLCMaturityHigh bool
}

// Result is one workload's projected debt trajectory, with the reasoning
// that produced it — same auditability standard as rubric.Result.
type Result struct {
	Trajectory Trajectory
	Reasoning  string
	// IsProjection is true when Trajectory describes a future state that
	// hasn't happened yet (Refactor/Rearchitect), as opposed to a trajectory
	// derived from what the R structurally does or doesn't touch. §5.8 is
	// explicit that baseline debt is measurable today and introduced debt
	// is an estimate — this field is how that distinction survives into the
	// output rather than getting flattened into one undifferentiated field.
	IsProjection bool
}

// Assess returns the projected debt trajectory for a workload given its
// chosen R (from rubric.Evaluate) and, for Refactor/Rearchitect only, the
// execution context that changes the answer.
func Assess(r rubric.R, ctx ExecutionContext) Result {
	switch r {
	case rubric.Retain:
		return Result{
			Trajectory: Unchanged,
			Reasoning:  "nothing moves — by definition the debt profile is unchanged",
		}

	case rubric.Rehost:
		return Result{
			Trajectory: UsuallyUnchanged,
			Reasoning:  "lift-and-shift relocates the codebase without touching it — this migration does not by itself address any existing debt, and the report says so plainly rather than letting \"we migrated\" imply \"we fixed something\"",
		}

	case rubric.Replatform:
		return Result{
			Trajectory: ModestReduction,
			Reasoning:  "swapping to a managed service typically removes the operational debt of self-managing that piece, without touching application-level debt",
		}

	case rubric.Refactor, rubric.Rearchitect:
		return assessRefactorOrRearchitect(ctx)

	case rubric.Repurchase:
		return Result{
			Trajectory: Reduction,
			Reasoning:  "retiring in-house code for a maintained product removes that code's debt entirely, at the cost of new integration debt — both sides get shown, not just the reduction",
		}

	case rubric.Retire:
		return Result{
			Trajectory: FullElimination,
			Reasoning:  "the only R that's unambiguous — the debt is eliminated along with the workload",
		}

	case rubric.Repatriate:
		// Not in §5.8's original table — Repatriate moves infrastructure,
		// not code, so its debt trajectory follows the same reasoning as
		// Rehost: nothing about the application itself changes.
		return Result{
			Trajectory: UsuallyUnchanged,
			Reasoning:  "repatriation moves where the workload runs, not what it is — application-level debt is untouched, same reasoning as Rehost",
		}
	}

	return Result{
		Trajectory: Unchanged,
		Reasoning:  "no debt-trajectory rule matched this R — flagged for review, not silently defaulted",
	}
}

// assessRefactorOrRearchitect is where §5.8's "widest variance of any R"
// caution actually gets applied. Both outcomes are stated as projections,
// never as a measurement, per the explicit distinction in §5.8.
func assessRefactorOrRearchitect(ctx ExecutionContext) Result {
	if ctx.HeavyAIGenerationLightReview {
		return Result{
			Trajectory:   RiskOfIncrease,
			IsProjection: true,
			Reasoning:    "the stated execution plan leans on heavy AI code generation with light review — GitClear's industry data found duplicated code blocks rose roughly 8x in frequency and refactoring activity fell from ~25% to under 10% of changed lines as AI-generated code volume grew industry-wide. That's a mechanism, not a guarantee: it's a reason to weight the introduced-debt risk more heavily here, not a prediction that this specific execution will fail.",
		}
	}
	if ctx.SDLCMaturityHigh {
		return Result{
			Trajectory:   ReductionIfExecutedWell,
			IsProjection: true,
			Reasoning:    "strong existing test coverage and disciplined CI/review practice (§5.5) make a clean Refactor/Rearchitect materially more likely than a silent regression",
		}
	}
	return Result{
		Trajectory:   ReductionIfExecutedWell,
		IsProjection: true,
		Reasoning:    "no execution-context evidence available either way — this is the widest-variance R in the table, and without SDLC maturity signal or a stated execution plan, the projection carries lower confidence than the trajectory alone conveys",
	}
}
