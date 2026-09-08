// Package narrative turns the evidence already gathered elsewhere in the
// pipeline — codescan's boundary detection, depscan's dependency
// findings, correlate's matches, and rubric/debtdelta's calls — into
// plain-English paragraphs a non-technical stakeholder can read without
// knowing what any of those words mean.
//
// This is composition, not inference: every sentence traces back to a
// specific field on Input, nothing here invents a fact the rest of the
// pipeline didn't already establish. No LLM call is involved — same
// "the logic already did the work, this just states it in words"
// principle SPEC.md §5.0 applies to correlation, applied here to the
// write-up instead.
package narrative

import (
	"fmt"
	"strings"

	"github.com/codetocloudorg/oporder/internal/codescan"
	"github.com/codetocloudorg/oporder/internal/correlate"
	"github.com/codetocloudorg/oporder/internal/debtdelta"
	"github.com/codetocloudorg/oporder/internal/depscan"
	"github.com/codetocloudorg/oporder/internal/rubric"
)

// MissionEvidence is the rubric call and debt projection for one
// workload, when enough evidence existed to make one at all.
type MissionEvidence struct {
	Result    rubric.Result
	DebtDelta debtdelta.Result
}

// Input is everything Explain needs about one workload — the caller
// (internal/scan) already has all of this gathered; this package only
// composes sentences from it.
type Input struct {
	Workload     codescan.Workload
	Dependencies depscan.Result   // zero value: none found
	Match        *correlate.Match // nil: no correlated resource
	Unmatched    bool             // true if correlation ran and found nothing for this workload
	Mission      *MissionEvidence // nil: no rubric call was made
}

// Paragraph is one workload's plain-English write-up.
type Paragraph struct {
	WorkloadName string
	WorkloadPath string
	Text         string
}

// Explain composes one paragraph per input, in the order given.
func Explain(inputs []Input) []Paragraph {
	out := make([]Paragraph, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, Paragraph{
			WorkloadName: in.Workload.Name,
			WorkloadPath: in.Workload.Path,
			Text:         in.paragraph(),
		})
	}
	return out
}

func (in Input) paragraph() string {
	w := in.Workload
	var b strings.Builder

	fmt.Fprintf(&b, "%s is %s a deployable workload, found at `%s` — %s.",
		w.Name, confidenceWord(w.Confidence), w.Path, describeSignal(w))

	if s := dependencySentence(in.Dependencies); s != "" {
		b.WriteString(" ")
		b.WriteString(s)
	}
	if s := deploymentSentence(in); s != "" {
		b.WriteString(" ")
		b.WriteString(s)
	}
	if s := recommendationSentence(in); s != "" {
		b.WriteString(" ")
		b.WriteString(s)
	}

	return b.String()
}

func confidenceWord(c codescan.Confidence) string {
	if c == codescan.ConfidenceHigh {
		return "clearly"
	}
	return "probably"
}

func describeSignal(w codescan.Workload) string {
	switch {
	case strings.Contains(w.Signal, "Dockerfile"):
		return "it already has a Dockerfile, so it's already containerized"
	case strings.Contains(w.Signal, "docker-compose"):
		return "it's declared as a service in a docker-compose file"
	case strings.Contains(w.Signal, "library, no cmd/"):
		return "it's a Go library with no standalone entry point of its own — likely shared code rather than its own deployable service"
	case strings.Contains(w.Signal, "main.go"):
		return "it's a standalone Go binary with its own entry point"
	case strings.Contains(w.Signal, "package.json"):
		return "it's a Node.js package"
	case strings.Contains(w.Signal, "requirements.txt"), strings.Contains(w.Signal, "pyproject.toml"):
		return "it's a Python project"
	case strings.Contains(w.Signal, "Cargo.toml"):
		return "it's a Rust project"
	case strings.Contains(w.Signal, "pom.xml"), strings.Contains(w.Signal, "build.gradle"):
		return "it's a JVM project"
	case strings.Contains(w.Signal, "serverless."):
		return "it's declared as a serverless function"
	case strings.Contains(w.Signal, "template.y"):
		return "it's declared as a SAM/CloudFormation-style deployment"
	default:
		return "it was detected from " + w.Signal
	}
}

var providerLabel = map[depscan.Provider]string{
	depscan.ProviderAWS:   "AWS",
	depscan.ProviderAzure: "Azure",
	depscan.ProviderGCP:   "GCP",
}

func dependencySentence(dep depscan.Result) string {
	if !dep.HasProprietaryDependencies() {
		return "No known cloud-vendor SDK dependencies were found in its manifest — that doesn't guarantee it's portable, since this check only reads declared dependencies, not actual code."
	}
	providers := dep.Providers()
	names := make([]string, len(providers))
	for i, p := range providers {
		if label, ok := providerLabel[p]; ok {
			names[i] = label
		} else {
			names[i] = string(p)
		}
	}
	return fmt.Sprintf(
		"It directly depends on the %s SDK, which means a straightforward lift-and-shift wouldn't remove that dependency — it would just relocate it. If this workload needs to move, treat replacing or abstracting that dependency as part of the work, not an afterthought.",
		strings.Join(names, "/"),
	)
}

func deploymentSentence(in Input) string {
	if in.Match != nil {
		return fmt.Sprintf("It appears to be running today as %q on %s (matched by %s).",
			in.Match.Resource.Name, in.Match.Resource.Provider, in.Match.Confidence)
	}
	if in.Unmatched {
		return "No matching live resource was found for it in the infrastructure that was scanned — it may not be deployed yet, or it's deployed under a name this pass didn't recognize."
	}
	return ""
}

func recommendationSentence(in Input) string {
	if in.Mission == nil {
		if in.Dependencies.HasProprietaryDependencies() {
			return "There isn't yet enough evidence for a confident recommendation, but the dependency alone rules out a plain rehost — a refactor toward removing it, ideally onto a container or serverless target rather than another VM, is the safer default direction."
		}
		return "There isn't yet enough evidence gathered to make a confident recommendation for this workload — connecting a cloud account so utilization data can be gathered is the next concrete step."
	}
	r := in.Mission.Result
	if r.R == "" {
		return "Even with what's been gathered, no clear recommendation could be made yet: " + r.Reasoning + "."
	}
	trajectory := strings.ReplaceAll(string(in.Mission.DebtDelta.Trajectory), "-", " ")
	return fmt.Sprintf("The recommendation is to %s: %s. If carried out, the technical debt this workload carries is expected to see a %s.",
		strings.ToLower(string(r.R)), r.Reasoning, trajectory)
}
