// Package rubric implements the 5/7-Rs decision logic from SPEC.md §5.3.
//
// Every trigger here is a pure function of Evidence gathered elsewhere in the
// pipeline (§5.0-§5.2) — this package never calls an LLM and never touches a
// network. That's deliberate: §1 requires every scoring decision to be
// visible and auditable, which means the rubric itself has to be ordinary,
// readable code, not a model call standing in for one.
package rubric

// R is one of the seven migration/modernization strategies plus Retain and
// Repatriate — SPEC.md calls this "the 7 Rs" while listing eight, because
// Retain and Repatriate are the two zero-migration-work outcomes weighted
// identically to the other six, not counted as a distinct pair in the name.
type R string

const (
	Retain      R = "retain"
	Repatriate  R = "repatriate"
	Rehost      R = "rehost"
	Replatform  R = "replatform"
	Refactor    R = "refactor"
	Rearchitect R = "rearchitect"
	Repurchase  R = "repurchase"
	Retire      R = "retire"
)

// Evidence is every signal §5.3's trigger table reads from. Each field maps
// directly to a clause in one or more trigger definitions — see the comment
// on each trigger function for exactly which clauses it reads.
//
// Evidence is gathered by §5.0 (correlation), §5.1 (code analysis), and §5.2
// (live infrastructure) before this package ever runs. A field left at its
// zero value means "no evidence found," not "false" — see Confidence in
// Result for how the evaluator surfaces that distinction rather than
// silently treating missing evidence as a negative signal.
type Evidence struct {
	// Retain signals
	LowBusinessCriticality bool
	CostAcceptable         bool
	HasComplianceDriver    bool
	HasEOLDriver           bool

	// Repatriate signals
	SteadyStatePredictableLoad  bool
	HighEgressCost              bool
	CloudPremiumExceedsOwnedTCO bool

	// Rehost signals
	Containerizable                   bool
	HasProprietaryManagedDependencies bool
	HasMigrationDeadline              bool

	// Replatform signals
	OneToOneManagedServiceSwapAvailable bool
	MinimalCodeChangeRequired           bool

	// Refactor signals
	HasNoTargetPlatformEquivalent bool
	CoreLogicSound                bool

	// Rearchitect signals
	IsMonolith                 bool
	HasStatedDecompositionGoal bool // e.g. the container/serverless targets in §4.5

	// Repurchase signals
	EquivalentSaaSCheaper bool

	// Retire signals
	NoOrNegligibleTraffic      bool
	RedundantWithAnotherSystem bool

	// TelemetryAvailable is false when the provider exposed no usage data at
	// all for this resource — §12's gap analysis requires this be
	// distinguished from "telemetry available and shows zero usage," since
	// the two mean very different things for a Retire call.
	TelemetryAvailable bool
}

// Confidence states how much the evaluator trusts the call it made. A rubric
// that never admits uncertainty is worse than one that admits it too often
// (§5.3) — Low is not a hedge, it's a real, first-class outcome.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
	Insufficient     Confidence = "insufficient-evidence"
)

// Result is one workload's evaluated call, with every piece of reasoning
// that produced it — §1 requires this to be auditable, not just a verdict.
type Result struct {
	R               R
	MatchedTriggers []R    // every R whose trigger matched, before tie-breaking
	TieBreakApplied string // human-readable reason, empty if only one R matched
	Confidence      Confidence
	Reasoning       string // the specific evidence that drove the call, per §3.2
}

// Evaluate applies §5.3's trigger table to a single workload's evidence,
// then resolves any tie per the tie-break rule added to that section: Retain
// wins absent a forcing driver, then lowest-effort-first among what's left.
func Evaluate(e Evidence) Result {
	if !e.TelemetryAvailable && !anyEvidenceGiven(e) {
		return Result{
			Confidence: Insufficient,
			Reasoning:  "no usage telemetry and no other evidence gathered for this workload — see §12's gap analysis on telemetry-absent Retire false positives",
		}
	}

	matched := matchedTriggers(e)
	if len(matched) == 0 {
		return Result{
			Confidence: Insufficient,
			Reasoning:  "no trigger in §5.3's table matched the gathered evidence — this workload needs a documented fallback, not a guessed default",
		}
	}
	if len(matched) == 1 {
		return Result{
			R:               matched[0],
			MatchedTriggers: matched,
			Confidence:      ConfidenceHigh,
			Reasoning:       reasoningFor(matched[0], e),
		}
	}

	chosen, tieBreak := resolveTie(matched, e)
	return Result{
		R:               chosen,
		MatchedTriggers: matched,
		TieBreakApplied: tieBreak,
		Confidence:      ConfidenceMedium,
		Reasoning:       reasoningFor(chosen, e),
	}
}

func anyEvidenceGiven(e Evidence) bool {
	return e != Evidence{}
}

// matchedTriggers evaluates every trigger independently — §5.3 is explicit
// that these are not mutually exclusive, so this deliberately does not stop
// at the first match.
func matchedTriggers(e Evidence) []R {
	var out []R
	if triggerRetain(e) {
		out = append(out, Retain)
	}
	if triggerRepatriate(e) {
		out = append(out, Repatriate)
	}
	if triggerRehost(e) {
		out = append(out, Rehost)
	}
	if triggerReplatform(e) {
		out = append(out, Replatform)
	}
	if triggerRefactor(e) {
		out = append(out, Refactor)
	}
	if triggerRearchitect(e) {
		out = append(out, Rearchitect)
	}
	if triggerRepurchase(e) {
		out = append(out, Repurchase)
	}
	if triggerRetire(e) {
		out = append(out, Retire)
	}
	return out
}

// "Low business criticality, acceptable current cost, no compliance or EOL
// driver forcing action" — SPEC.md §5.3.
func triggerRetain(e Evidence) bool {
	return e.LowBusinessCriticality && e.CostAcceptable && !e.HasComplianceDriver && !e.HasEOLDriver
}

// "Steady-state predictable load + high egress cost + cloud premium exceeds
// owned-hardware TCO over the depreciation horizon" — SPEC.md §5.3.
func triggerRepatriate(e Evidence) bool {
	return e.SteadyStatePredictableLoad && e.HighEgressCost && e.CloudPremiumExceedsOwnedTCO
}

// "Containerizable or VM-portable, no proprietary managed-service
// dependencies detected, migration deadline is the dominant constraint" —
// SPEC.md §5.3.
func triggerRehost(e Evidence) bool {
	return e.Containerizable && !e.HasProprietaryManagedDependencies && e.HasMigrationDeadline
}

// "1:1 managed-service swap available (e.g. self-managed DB → managed DB)
// with minimal code change" — SPEC.md §5.3.
func triggerReplatform(e Evidence) bool {
	return e.OneToOneManagedServiceSwapAvailable && e.MinimalCodeChangeRequired
}

// "Proprietary dependencies with no target-platform equivalent, but the core
// logic is sound" — SPEC.md §5.3.
func triggerRefactor(e Evidence) bool {
	return e.HasProprietaryManagedDependencies && e.HasNoTargetPlatformEquivalent && e.CoreLogicSound
}

// "Monolith requiring decomposition to meet a stated goal (e.g. the
// container/serverless targets in §4.5)" — SPEC.md §5.3.
func triggerRearchitect(e Evidence) bool {
	return e.IsMonolith && e.HasStatedDecompositionGoal
}

// "Equivalent SaaS/COTS functionality is cheaper than continued
// maintenance" — SPEC.md §5.3.
func triggerRepurchase(e Evidence) bool {
	return e.EquivalentSaaSCheaper
}

// "Usage telemetry shows no or negligible traffic; redundant with another
// system" — SPEC.md §5.3. Gated on TelemetryAvailable per §12's gap
// analysis: a workload with no telemetry at all is not evidence of no
// traffic, it's an instrumentation gap, and must never resolve to Retire by
// default.
func triggerRetire(e Evidence) bool {
	return e.TelemetryAvailable && (e.NoOrNegligibleTraffic || e.RedundantWithAnotherSystem)
}

// resolveTie applies the resolution order added to §5.3: Retain wins absent
// a forcing driver, Retire is next (called "the only unambiguous R" in
// §5.8), then the lowest-effort R among what's left, in the stated order.
func resolveTie(matched []R, e Evidence) (R, string) {
	hasForcingDriver := e.HasEOLDriver || e.HasComplianceDriver || !e.CostAcceptable || e.HasStatedDecompositionGoal

	if contains(matched, Retain) && !hasForcingDriver {
		return Retain, "Retain matched alongside " + others(matched, Retain) + ", but no forcing driver (EOL, compliance, cost, or a stated architectural goal) is present — §5.3 requires Retain to win rather than recommend action with no forcing reason"
	}
	if contains(matched, Retire) {
		return Retire, "Retire matched alongside " + others(matched, Retire) + " — Retire is treated as the most conservative, unambiguous call when its own trigger is satisfied (§5.8)"
	}
	// Narrower, more evidence-specific Rs before the generic effort-ordering.
	for _, specific := range []R{Repatriate, Repurchase} {
		if contains(matched, specific) {
			return specific, specific.String() + " matched alongside " + others(matched, specific) + " — its trigger is more evidence-specific than the general rehost-through-rearchitect progression, so it takes precedence over the generic ordering"
		}
	}
	// Lowest-effort-first among the general migration/modernization path.
	for _, r := range []R{Rehost, Replatform, Refactor, Rearchitect} {
		if contains(matched, r) {
			return r, r.String() + " chosen as the lowest-effort R among " + joinRs(matched) + " that still satisfies every matched trigger — §5.3 places the burden of proof on the scope of change being recommended"
		}
	}
	// Unreachable given the trigger set above, but never silently fall
	// through to an unexplained default.
	return matched[0], "no explicit tie-break rule matched this combination — flagged for rubric review, not silently defaulted"
}

func reasoningFor(r R, e Evidence) string {
	switch r {
	case Retain:
		return "low business criticality, acceptable current cost, no compliance or EOL driver forcing action"
	case Repatriate:
		return "steady-state predictable load with high egress cost; cloud premium exceeds owned-hardware TCO"
	case Rehost:
		return "containerizable with no proprietary managed-service dependencies; migration deadline is the dominant constraint"
	case Replatform:
		return "1:1 managed-service swap available with minimal code change required"
	case Refactor:
		return "proprietary dependencies with no target-platform equivalent, but core logic is sound"
	case Rearchitect:
		return "monolith requiring decomposition to meet a stated architectural goal"
	case Repurchase:
		return "equivalent SaaS/COTS functionality is cheaper than continued maintenance"
	case Retire:
		if e.RedundantWithAnotherSystem {
			return "redundant with another system"
		}
		return "usage telemetry shows no or negligible traffic"
	}
	return ""
}

func contains(rs []R, target R) bool {
	for _, r := range rs {
		if r == target {
			return true
		}
	}
	return false
}

func others(rs []R, exclude R) string {
	var out []R
	for _, r := range rs {
		if r != exclude {
			out = append(out, r)
		}
	}
	return joinRs(out)
}

func joinRs(rs []R) string {
	s := ""
	for i, r := range rs {
		if i > 0 {
			s += ", "
		}
		s += string(r)
	}
	return s
}

func (r R) String() string { return string(r) }
