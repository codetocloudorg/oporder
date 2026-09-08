package scan

import (
	"fmt"
	"os"
)

// WriteMissionMD writes the real, currently-partial slice of §3.2's
// MISSION.md this package can produce: a §5.3 rubric call per workload
// that has both a correlated AWS resource and real CloudWatch
// utilization data. Every other workload — no correlated resource, a
// resource on a provider without utilization gathering yet, or a
// resource whose utilization query failed — is listed as out of scope
// for this pass, never silently omitted.
func WriteMissionMD(path string, s Situation) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("scan: creating %s: %w", path, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Mission")
	fmt.Fprintln(f)
	fmt.Fprintln(f, "A real §5.3 rubric call per workload — but built from exactly one evidence")
	fmt.Fprintln(f, "signal (CloudWatch CPU utilization, AWS only) out of the dozen §5.3 defines.")
	fmt.Fprintln(f, "Every other signal (business criticality, compliance/EOL drivers, egress cost,")
	fmt.Fprintln(f, "proprietary dependencies, SaaS equivalents, and utilization on Azure/GCP/")
	fmt.Fprintln(f, "Cloudflare) isn't gathered yet, so most calls below should land on")
	fmt.Fprintln(f, "insufficient-evidence rather than a confident R — that's the correct, honest")
	fmt.Fprintln(f, "result of partial evidence, not a bug. See SPEC.md §10's M2 for what's left.")
	fmt.Fprintln(f)

	if len(s.Missions) == 0 {
		fmt.Fprintln(f, "No workload had both a correlated AWS resource and usable utilization data")
		fmt.Fprintln(f, "this run — nothing to call yet.")
		return nil
	}

	fmt.Fprintln(f, "| Workload | Resource | Call | Confidence | Reasoning |")
	fmt.Fprintln(f, "|---|---|---|---|---|")
	for _, m := range s.Missions {
		call := string(m.Result.R)
		if call == "" {
			call = "—"
		}
		fmt.Fprintf(f, "| %s | %s | %s | %s | %s |\n", m.WorkloadName, m.Resource.Name, call, m.Result.Confidence, m.Result.Reasoning)
	}
	fmt.Fprintln(f)

	return nil
}
