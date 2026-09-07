// Command sample-report generates real output from the actual rubric and
// debt-delta engines against a handful of synthetic, illustrative
// workloads. It exists specifically so the website's "sample report"
// section shows genuinely computed output — real engine, fixture data —
// rather than hand-written marketing copy pretending to be a report.
//
// No live cloud account or LLM is involved. This is exactly the kind of
// fixture-based use §7 designed the recorded-fixture mode for, applied here
// to produce honest sample output instead of a live demo that doesn't exist
// yet.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/debtdelta"
	"github.com/codetocloudorg/oporder/internal/rubric"
)

type workload struct {
	Name     string                     `json:"name"`
	Evidence rubric.Evidence            `json:"-"`
	ExecCtx  debtdelta.ExecutionContext `json:"-"`
}

type workloadResult struct {
	Name            string               `json:"name"`
	R               rubric.R             `json:"r"`
	Confidence      rubric.Confidence    `json:"confidence"`
	Reasoning       string               `json:"reasoning"`
	TieBreakApplied string               `json:"tie_break_applied,omitempty"`
	DebtTrajectory  debtdelta.Trajectory `json:"debt_trajectory"`
	DebtReasoning   string               `json:"debt_reasoning"`
	IsProjection    bool                 `json:"debt_is_projection"`
}

// These five are illustrative, hand-built to show a representative spread
// of the seven Rs — not a real scan, and not claimed to be one anywhere
// this output is used.
func sampleWorkloads() []workload {
	return []workload{
		{
			Name: "billing-service",
			Evidence: rubric.Evidence{
				HasProprietaryManagedDependencies: true,
				HasNoTargetPlatformEquivalent:     true,
				CoreLogicSound:                    true,
				TelemetryAvailable:                true,
			},
			ExecCtx: debtdelta.ExecutionContext{HeavyAIGenerationLightReview: true},
		},
		{
			Name: "payments-api",
			Evidence: rubric.Evidence{
				OneToOneManagedServiceSwapAvailable: true,
				MinimalCodeChangeRequired:           true,
				TelemetryAvailable:                  true,
			},
		},
		{
			Name: "legacy-reports",
			Evidence: rubric.Evidence{
				LowBusinessCriticality: true,
				CostAcceptable:         true,
				Containerizable:        true,
				HasMigrationDeadline:   true,
				TelemetryAvailable:     true,
			},
		},
		{
			Name: "user-uploads",
			Evidence: rubric.Evidence{
				TelemetryAvailable:    true,
				NoOrNegligibleTraffic: true,
			},
		},
		{
			Name: "order-orchestrator",
			Evidence: rubric.Evidence{
				IsMonolith:                 true,
				HasStatedDecompositionGoal: true,
				TelemetryAvailable:         true,
			},
			ExecCtx: debtdelta.ExecutionContext{SDLCMaturityHigh: true},
		},
	}
}

func main() {
	var results []workloadResult
	for _, w := range sampleWorkloads() {
		r := rubric.Evaluate(w.Evidence)
		d := debtdelta.Assess(r.R, w.ExecCtx)
		results = append(results, workloadResult{
			Name:            w.Name,
			R:               r.R,
			Confidence:      r.Confidence,
			Reasoning:       r.Reasoning,
			TieBreakApplied: r.TieBreakApplied,
			DebtTrajectory:  d.Trajectory,
			DebtReasoning:   d.Reasoning,
			IsProjection:    d.IsProjection,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
