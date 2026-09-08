package scan

import (
	"fmt"
	"strings"

	"github.com/codetocloudorg/oporder/internal/correlate"
)

// mermaidDiagram renders a Situation's code and infrastructure as a
// Mermaid graph — matched workloads connected to their resource, with
// unclassified code and unmatched resources shown as unconnected nodes
// so an orphan is visually obvious rather than only listed in a table.
// Both GitHub and Obsidian render Mermaid natively, no extra tooling
// needed to view it.
func mermaidDiagram(s Situation) string {
	if len(s.Workloads) == 0 && len(s.Unclassified) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("```mermaid\ngraph LR\n")

	workloadNode := map[string]string{}
	b.WriteString("  subgraph Code\n")
	for i, w := range s.Workloads {
		id := fmt.Sprintf("W%d", i)
		workloadNode[w.Path] = id
		fmt.Fprintf(&b, "    %s[\"%s\"]\n", id, sanitizeLabel(w.Name))
	}
	for i, u := range s.Unclassified {
		id := fmt.Sprintf("U%d", i)
		fmt.Fprintf(&b, "    %s[\"? %s\"]\n", id, sanitizeLabel(u.Path))
	}
	b.WriteString("  end\n")

	if len(s.Correlation.Matches) > 0 || len(s.Correlation.UnmatchedResources) > 0 {
		resourceNode := map[string]string{}
		next := 0
		nodeFor := func(r correlate.Resource) string {
			if id, ok := resourceNode[r.ID]; ok {
				return id
			}
			id := fmt.Sprintf("R%d", next)
			next++
			resourceNode[r.ID] = id
			return id
		}

		b.WriteString("  subgraph Infrastructure\n")
		for _, m := range s.Correlation.Matches {
			id := nodeFor(m.Resource)
			fmt.Fprintf(&b, "    %s[\"%s (%s)\"]\n", id, sanitizeLabel(m.Resource.Name), m.Resource.Provider)
		}
		for _, r := range s.Correlation.UnmatchedResources {
			id := nodeFor(r)
			fmt.Fprintf(&b, "    %s[\"%s (%s)\"]\n", id, sanitizeLabel(r.Name), r.Provider)
		}
		b.WriteString("  end\n")

		for _, m := range s.Correlation.Matches {
			wid, ok := workloadNode[m.WorkloadPath]
			if !ok {
				continue
			}
			fmt.Fprintf(&b, "  %s -->|%s| %s\n", wid, m.Confidence, nodeFor(m.Resource))
		}
	}

	b.WriteString("```\n")
	return b.String()
}

// sanitizeLabel strips characters that would break Mermaid's quoted
// node-label syntax.
func sanitizeLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "'")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
