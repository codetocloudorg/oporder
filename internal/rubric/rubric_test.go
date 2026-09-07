package rubric

import "testing"

// Each single-trigger case below asserts that exactly one R matches and it's
// the one the evidence was built for — these correspond one-to-one with the
// eight rows of §5.3's trigger table.

func TestEvaluate_SingleTriggerCases(t *testing.T) {
	cases := []struct {
		name string
		e    Evidence
		want R
	}{
		{
			name: "retain: low criticality, acceptable cost, no forcing driver",
			e: Evidence{
				LowBusinessCriticality: true,
				CostAcceptable:         true,
				TelemetryAvailable:     true,
			},
			want: Retain,
		},
		{
			name: "repatriate: steady load, high egress, cloud premium over owned TCO",
			e: Evidence{
				SteadyStatePredictableLoad:  true,
				HighEgressCost:              true,
				CloudPremiumExceedsOwnedTCO: true,
				TelemetryAvailable:          true,
			},
			want: Repatriate,
		},
		{
			name: "rehost: containerizable, no proprietary deps, deadline-driven",
			e: Evidence{
				Containerizable:      true,
				HasMigrationDeadline: true,
				TelemetryAvailable:   true,
			},
			want: Rehost,
		},
		{
			name: "replatform: 1:1 managed swap, minimal code change",
			e: Evidence{
				OneToOneManagedServiceSwapAvailable: true,
				MinimalCodeChangeRequired:           true,
				TelemetryAvailable:                  true,
			},
			want: Replatform,
		},
		{
			name: "refactor: proprietary deps with no equivalent, sound core logic",
			e: Evidence{
				HasProprietaryManagedDependencies: true,
				HasNoTargetPlatformEquivalent:     true,
				CoreLogicSound:                    true,
				TelemetryAvailable:                true,
			},
			want: Refactor,
		},
		{
			name: "rearchitect: monolith with a stated decomposition goal",
			e: Evidence{
				IsMonolith:                 true,
				HasStatedDecompositionGoal: true,
				TelemetryAvailable:         true,
			},
			want: Rearchitect,
		},
		{
			name: "repurchase: equivalent SaaS is cheaper",
			e: Evidence{
				EquivalentSaaSCheaper: true,
				TelemetryAvailable:    true,
			},
			want: Repurchase,
		},
		{
			name: "retire: telemetry available, shows negligible traffic",
			e: Evidence{
				TelemetryAvailable:    true,
				NoOrNegligibleTraffic: true,
			},
			want: Retire,
		},
		{
			name: "retire: redundant with another system",
			e: Evidence{
				TelemetryAvailable:         true,
				RedundantWithAnotherSystem: true,
			},
			want: Retire,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.e)
			if got.R != tc.want {
				t.Fatalf("Evaluate() = %q, want %q (matched: %v, reasoning: %s)",
					got.R, tc.want, got.MatchedTriggers, got.Reasoning)
			}
			if got.Confidence != ConfidenceHigh {
				t.Errorf("single-trigger match should be ConfidenceHigh, got %q", got.Confidence)
			}
			if got.TieBreakApplied != "" {
				t.Errorf("single-trigger match should not apply a tie-break, got %q", got.TieBreakApplied)
			}
		})
	}
}

// TestEvaluate_RetainWinsAbsentForcingDriver is the exact example given in
// SPEC.md §5.3's own prose: "a service with no compliance driver that's also
// cleanly containerizable is a completely normal, common case ... Retain
// wins over any migration path unless a specific forcing driver is present."
func TestEvaluate_RetainWinsAbsentForcingDriver(t *testing.T) {
	e := Evidence{
		// Retain's trigger
		LowBusinessCriticality: true,
		CostAcceptable:         true,
		// Rehost's trigger, matched at the same time
		Containerizable:      true,
		HasMigrationDeadline: true,
		TelemetryAvailable:   true,
	}

	got := Evaluate(e)

	if len(got.MatchedTriggers) < 2 {
		t.Fatalf("expected both Retain and Rehost to match, got %v", got.MatchedTriggers)
	}
	if got.R != Retain {
		t.Fatalf("Evaluate() = %q, want %q — Retain must win absent a forcing driver (§5.3)", got.R, Retain)
	}
	if got.TieBreakApplied == "" {
		t.Fatal("expected a stated tie-break reason, got none — §5.3 requires the tie-break to be shown, never silent")
	}
	if got.Confidence != ConfidenceMedium {
		t.Errorf("a tie-break resolution should report ConfidenceMedium, got %q", got.Confidence)
	}
}

// TestEvaluate_ForcingDriverOverridesRetain checks the other side of the same
// rule: if a forcing driver *is* present, Retain must not silently win just
// because it also matched.
func TestEvaluate_ForcingDriverOverridesRetain(t *testing.T) {
	e := Evidence{
		// Retain's trigger technically can't match once HasEOLDriver is true
		// (triggerRetain requires !HasEOLDriver) — this case instead checks
		// that a forcing driver present alongside a Rehost/Rearchitect tie
		// resolves to the lowest-effort R, not an unexplained default.
		Containerizable:            true,
		HasMigrationDeadline:       true,
		IsMonolith:                 true,
		HasStatedDecompositionGoal: true,
		TelemetryAvailable:         true,
	}

	got := Evaluate(e)

	if got.R != Rehost {
		t.Fatalf("Evaluate() = %q, want %q — lowest-effort R among a Rehost/Rearchitect tie (§5.3)", got.R, Rehost)
	}
	if got.TieBreakApplied == "" {
		t.Fatal("expected a stated tie-break reason for the Rehost/Rearchitect tie")
	}
}

// TestEvaluate_RetireWinsOverGeneralTie checks that Retire, once its own
// trigger is satisfied, wins over the general effort-ordered Rs — per §5.8's
// framing of Retire as "the only unambiguous R."
func TestEvaluate_RetireWinsOverGeneralTie(t *testing.T) {
	e := Evidence{
		TelemetryAvailable:    true,
		NoOrNegligibleTraffic: true,
		Containerizable:       true,
		HasMigrationDeadline:  true,
	}

	got := Evaluate(e)

	if got.R != Retire {
		t.Fatalf("Evaluate() = %q, want %q — Retire should win over a Rehost tie once its own trigger matches", got.R, Retire)
	}
}

// TestEvaluate_NoTelemetryIsNotRetire is the specific regression case named
// in the gap analysis (§12): a workload with no telemetry at all must never
// be silently treated as a Retire candidate.
func TestEvaluate_NoTelemetryIsNotRetire(t *testing.T) {
	e := Evidence{
		TelemetryAvailable:    false,
		NoOrNegligibleTraffic: false, // zero value — no evidence, not "false traffic"
	}

	got := Evaluate(e)

	if got.R == Retire {
		t.Fatal("a workload with no telemetry must never resolve to Retire by default — §12")
	}
	if got.Confidence != Insufficient {
		t.Errorf("no telemetry and no other evidence should report Insufficient confidence, got %q", got.Confidence)
	}
}

// TestEvaluate_NoMatchIsInsufficientNotGuessed covers §5.3's own standard:
// "a stated 'insufficient evidence, here's what would resolve it' beats a
// wrong confident one."
func TestEvaluate_NoMatchIsInsufficientNotGuessed(t *testing.T) {
	e := Evidence{
		TelemetryAvailable: true,
		// Every other field left at zero value — no trigger can match.
	}

	got := Evaluate(e)

	if got.Confidence != Insufficient {
		t.Fatalf("Confidence = %q, want %q when no trigger matches", got.Confidence, Insufficient)
	}
	if got.R != "" {
		t.Errorf("R should be empty on insufficient evidence, got %q", got.R)
	}
	if got.Reasoning == "" {
		t.Error("insufficient-evidence result must still state why, not return silently")
	}
}
