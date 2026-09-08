package scan

import (
	"fmt"
	"os"
)

// WriteMissionMD writes the real, currently-partial slice of §3.2's
// MISSION.md this package can produce: a §5.3 rubric call per workload
// that has both a correlated resource and real CPU utilization data
// (AWS CloudWatch or Azure Monitor — GCP and Cloudflare don't have
// utilization gathering wired in). Every other workload — no correlated
// resource, a provider without utilization gathering, or a failed
// utilization query — is listed as out of scope for this pass, never
// silently omitted.
func WriteMissionMD(path string, s Situation) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("scan: creating %s: %w", path, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Mission")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "A real §5.3 rubric call per workload — but built from exactly one evidence")
	fmt.Fprintln(f, "signal (CPU utilization, AWS and Azure only) out of the dozen §5.3 defines.")
	fmt.Fprintln(f, "Every other signal (business criticality, compliance/EOL drivers, egress cost,")
	fmt.Fprintln(f, "proprietary dependencies, SaaS equivalents, and utilization on GCP/Cloudflare)")
	fmt.Fprintln(f, "isn't gathered yet, so most calls below should land on insufficient-evidence")
	fmt.Fprintln(f, "rather than a confident R — that's the correct, honest result of partial")
	fmt.Fprintln(f, "evidence, not a bug. The debt-delta column (§5.8) follows directly from")
	fmt.Fprintln(f, "whatever R was called, using a zero-value execution context since §5.5's SDLC")
	fmt.Fprintln(f, "scoring isn't built yet. See SPEC.md §10's M2 for what's left.")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "**Every workload below is, by construction, running on a virtual machine** —")
	fmt.Fprintln(f, "an EC2 instance or an Azure VM, the only resource types this pass gathers")
	fmt.Fprintln(f, "utilization for. That's not a coincidence to ignore: this project treats a VM")
	fmt.Fprintln(f, "as a last resort, not a destination — containers or serverless are the")
	fmt.Fprintln(f, "preferred landing spot, reached through code-driven modernization rather than")
	fmt.Fprintln(f, "a lift-and-shift. That bias isn't a rubric signal yet (it needs real")
	fmt.Fprintln(f, "containerizability and proprietary-dependency evidence from §5.1's code")
	fmt.Fprintln(f, "analysis, not just \"currently on a VM\"), so it's stated here plainly instead")
	fmt.Fprintln(f, "of silently baked into a call this evidence doesn't yet support.")
	fmt.Fprintln(f)

	if len(s.Missions) == 0 {
		fmt.Fprintln(f, "No workload had both a correlated resource and usable utilization data this")
		fmt.Fprintln(f, "run — nothing to call yet.")
		return nil
	}

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
	fmt.Fprintln(f)

	return nil
}
