// Package correlate joins codescan's detected workloads against live
// cloud resources, per SPEC.md §5.0's code↔infrastructure correlation
// tiers. Tier 1 (IaC state as ground truth — Terraform state,
// CloudFormation stack resources) isn't implemented yet; that needs a
// state-file parser this package doesn't have. What's here is tier 2
// (resource tags), tier 3 (name-matching heuristics), and tier 4
// (unmatched resources on either side, listed explicitly rather than
// dropped) — all deterministic string matching, no LLM call, exactly as
// §5.0 specifies.
package correlate

import (
	"regexp"
	"sort"
	"strings"
)

// Confidence mirrors the tier that produced a match. Tag matches are
// never presented as equivalent to a future IaC-state match, and name
// matches are never presented as equivalent to a tag match — per §5.0's
// explicit instruction that lower tiers always show their tier.
type Confidence string

const (
	ConfidenceTag  Confidence = "tag-match"  // §5.0 tier 2
	ConfidenceName Confidence = "name-match" // §5.0 tier 3
)

// tagKeys are the resource tag/label keys checked for tier 2, in the
// order §5.0 lists them.
var tagKeys = []string{"service", "app", "team"}

// Resource is the minimal shape correlate needs from a live cloud
// resource — provider connectors already return richer types
// (aws.Instance, azure.Resource, gcp.Instance); callers adapt to this.
type Resource struct {
	Provider string
	ID       string
	Name     string
	Tags     map[string]string
}

// Workload is the minimal shape correlate needs from codescan.Workload.
type Workload struct {
	Name string
	Path string
}

// Match is one confirmed workload↔resource pair.
type Match struct {
	WorkloadPath string
	WorkloadName string
	Resource     Resource
	Confidence   Confidence
	Reason       string
}

// Result is everything one correlation pass found, including both
// unmatched sides — §5.0 tier 4, never silently dropped.
type Result struct {
	Matches            []Match
	UnmatchedWorkloads []Workload
	UnmatchedResources []Resource
}

// Correlate matches every workload against every resource. Tag matches
// are found first and take priority; a resource already matched by tag
// is not also offered as a name match, and a workload already matched by
// tag is not re-checked against name-matching.
func Correlate(workloads []Workload, resources []Resource) Result {
	var res Result

	matchedWorkload := make([]bool, len(workloads))
	matchedResource := make([]bool, len(resources))

	// Tier 2: tag match.
	for wi, w := range workloads {
		for ri, r := range resources {
			if matchedResource[ri] {
				continue
			}
			if key, ok := tagMatches(w, r); ok {
				res.Matches = append(res.Matches, Match{
					WorkloadPath: w.Path,
					WorkloadName: w.Name,
					Resource:     r,
					Confidence:   ConfidenceTag,
					Reason:       "resource tag \"" + key + "\" matches workload name",
				})
				matchedWorkload[wi] = true
				matchedResource[ri] = true
				break
			}
		}
	}

	// Tier 3: name match, on whatever's left.
	for wi, w := range workloads {
		if matchedWorkload[wi] {
			continue
		}
		for ri, r := range resources {
			if matchedResource[ri] {
				continue
			}
			if nameMatches(w, r) {
				res.Matches = append(res.Matches, Match{
					WorkloadPath: w.Path,
					WorkloadName: w.Name,
					Resource:     r,
					Confidence:   ConfidenceName,
					Reason:       "resource name resembles workload name",
				})
				matchedWorkload[wi] = true
				matchedResource[ri] = true
				break
			}
		}
	}

	// Tier 4: what's left on either side, explicitly.
	for wi, w := range workloads {
		if !matchedWorkload[wi] {
			res.UnmatchedWorkloads = append(res.UnmatchedWorkloads, w)
		}
	}
	for ri, r := range resources {
		if !matchedResource[ri] {
			res.UnmatchedResources = append(res.UnmatchedResources, r)
		}
	}

	sort.Slice(res.Matches, func(i, j int) bool { return res.Matches[i].WorkloadPath < res.Matches[j].WorkloadPath })
	sort.Slice(res.UnmatchedWorkloads, func(i, j int) bool { return res.UnmatchedWorkloads[i].Path < res.UnmatchedWorkloads[j].Path })
	sort.Slice(res.UnmatchedResources, func(i, j int) bool { return res.UnmatchedResources[i].Name < res.UnmatchedResources[j].Name })
	return res
}

func tagMatches(w Workload, r Resource) (string, bool) {
	target := normalize(w.Name)
	if target == "" {
		return "", false
	}
	for _, key := range tagKeys {
		for tk, tv := range r.Tags {
			if strings.EqualFold(tk, key) && normalize(tv) == target {
				return key, true
			}
		}
	}
	return "", false
}

func nameMatches(w Workload, r Resource) bool {
	wn := normalize(w.Name)
	rn := normalize(r.Name)
	if wn == "" || rn == "" {
		return false
	}
	return wn == rn || strings.Contains(rn, wn) || strings.Contains(wn, rn)
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// normalize strips separators and casing so "billing-service",
// "billing_service", and "BillingService" all compare equal — the fuzzy
// matching §5.0 tier 3 calls for, kept as simple, auditable string
// normalization rather than a distance metric that would need its own
// threshold to justify.
func normalize(s string) string {
	return nonAlnum.ReplaceAllString(strings.ToLower(s), "")
}
