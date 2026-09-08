package scan

import (
	"fmt"
	"os"
)

// WriteMissionMD writes the real, currently-partial slice of §3.2's
// MISSION.md this package can produce: a §5.3 rubric call for every
// correlated workload that has at least one of two real evidence
// signals — CPU utilization (AWS CloudWatch or Azure Monitor, VM-hosted
// workloads only) or proprietary-managed-dependency findings (any
// workload with a matching manifest, any provider). A workload with
// neither signal, or no correlated resource at all, is out of scope for
// the table, never silently omitted — it still gets a plain-English
// paragraph in the "In plain English" section below, which covers every
// detected workload regardless of how much evidence exists for it.
func WriteMissionMD(path string, s Situation) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("scan: creating %s: %w", path, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Mission")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "A real §5.3 rubric call per workload — but built from exactly two evidence")
	fmt.Fprintln(f, "signals (CPU utilization on AWS/Azure VMs; proprietary managed-dependency")
	fmt.Fprintln(f, "detection on any provider) out of the dozen §5.3 defines. Every other signal")
	fmt.Fprintln(f, "(business criticality, compliance/EOL drivers, egress cost, SaaS equivalents,")
	fmt.Fprintln(f, "containerizability, and utilization on GCP/Cloudflare) isn't gathered yet, so")
	fmt.Fprintln(f, "most calls below should land on insufficient-evidence rather than a confident")
	fmt.Fprintln(f, "R — that's the correct, honest result of partial evidence, not a bug. Finding")
	fmt.Fprintln(f, "a proprietary dependency is still real signal even alone: it rules out Rehost")
	fmt.Fprintln(f, "(§5.3's trigger requires *no* proprietary managed-service dependency), which is")
	fmt.Fprintln(f, "exactly the \"rehost quietly becomes refactor\" case §5.1 names. The debt-delta")
	fmt.Fprintln(f, "column (§5.8) follows directly from whatever R was called, using a zero-value")
	fmt.Fprintln(f, "execution context since §5.5's SDLC scoring isn't built yet. See SPEC.md §10's")
	fmt.Fprintln(f, "M2 for what's left.")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "This project treats a VM as a last resort, not a destination — containers or")
	fmt.Fprintln(f, "serverless are the preferred landing spot, reached through code-driven")
	fmt.Fprintln(f, "modernization rather than a lift-and-shift. That bias isn't a rubric signal")
	fmt.Fprintln(f, "yet (it needs real containerizability evidence §5.1 doesn't gather), so it's")
	fmt.Fprintln(f, "stated here plainly instead of silently baked into a call the evidence doesn't")
	fmt.Fprintln(f, "yet support.")
	fmt.Fprintln(f)

	if len(s.Missions) == 0 {
		fmt.Fprintln(f, "No correlated workload had usable utilization or dependency evidence this")
		fmt.Fprintln(f, "run — nothing to call yet.")
	} else {
		fmt.Fprintln(f, "| Workload | Resource | Call | Confidence | Debt trajectory | Reasoning |")
		fmt.Fprintln(f, "|---|---|---|---|---|---|")
		for _, m := range s.Missions {
			call := string(m.Result.R)
			if call == "" {
				call = "—"
			}
			fmt.Fprintf(f, "| %s | %s | %s | %s | %s | %s |\n",
				m.WorkloadName, m.Resource.Name, call, m.Result.Confidence, m.DebtDelta.Trajectory, m.Result.Reasoning)
		}
	}
	fmt.Fprintln(f)

	if len(s.Narratives) > 0 {
		fmt.Fprintln(f, "## In plain English")
		fmt.Fprintln(f)
		fmt.Fprintln(f, "The same evidence above, restated in plain language — every detected workload")
		fmt.Fprintln(f, "gets a paragraph here, not just the ones with enough evidence for the table.")
		fmt.Fprintln(f, "Nothing below states a fact the sections above didn't already establish; this")
		fmt.Fprintln(f, "just says it in words instead of a table cell.")
		fmt.Fprintln(f)
		for _, p := range s.Narratives {
			fmt.Fprintf(f, "**%s** (`%s`)\n\n%s\n\n", p.WorkloadName, p.WorkloadPath, p.Text)
		}
	}

	return nil
}
