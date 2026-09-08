// Package depscan detects direct use of vendor-specific managed-service
// SDKs within one workload's source tree — SPEC.md §5.1's
// "proprietary-SDK detection... this is where 'rehost' quietly becomes
// 'refactor' and most naive migration estimates go wrong."
//
// This is real, but simpler than §5.1's eventual tree-sitter dependency
// graph: it matches package-manifest dependencies (go.mod requires,
// package.json dependencies, requirements.txt/pyproject.toml entries)
// against a known list of vendor-specific SDK package prefixes, rather
// than parsing an AST. It answers "does this workload depend on a
// proprietary managed-service SDK," not "which specific API calls does
// it make" — the latter needs a real parser this project doesn't have
// yet, and claiming otherwise would overclaim exactly the way SPEC.md
// §11 rules out.
//
// This is also, deliberately, the code-side work this project picked up
// specifically to stop drifting toward re-implementing infrastructure
// monitoring (see SPEC.md §11's note on the AWS/Azure-utilization
// course-correction) — it's evidence no vendor migration tool can
// produce, because it requires reading the code, not the account.
package depscan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Provider is which cloud vendor a matched SDK belongs to.
type Provider string

const (
	ProviderAWS   Provider = "aws"
	ProviderAzure Provider = "azure"
	ProviderGCP   Provider = "gcp"
)

// Finding is one matched proprietary dependency.
type Finding struct {
	Provider Provider
	Package  string // the known SDK prefix that matched
	Source   string // which manifest file it came from, e.g. "go.mod"
}

// Result is everything Analyze found for one workload.
type Result struct {
	Findings []Finding
}

// HasProprietaryDependencies is the direct signal
// rubric.Evidence.HasProprietaryManagedDependencies needs.
func (r Result) HasProprietaryDependencies() bool {
	return len(r.Findings) > 0
}

// Providers returns the distinct set of providers found, in a stable
// order — useful for reporting without repeating a provider per finding.
func (r Result) Providers() []Provider {
	seen := map[Provider]bool{}
	var out []Provider
	for _, f := range r.Findings {
		if !seen[f.Provider] {
			seen[f.Provider] = true
			out = append(out, f.Provider)
		}
	}
	return out
}

type sdkPattern struct {
	Provider Provider
	Prefix   string
}

// knownSDKs is deliberately a starting set, not exhaustive — each entry
// is a real, currently-maintained SDK package prefix, easy to extend as
// gaps are found rather than a one-time list frozen at write time.
var knownSDKs = []sdkPattern{
	{ProviderAWS, "github.com/aws/aws-sdk-go"},
	{ProviderAWS, "@aws-sdk/"},
	{ProviderAWS, "aws-sdk"},
	{ProviderAWS, "boto3"},
	{ProviderAWS, "botocore"},

	{ProviderAzure, "github.com/Azure/azure-sdk-for-go"},
	{ProviderAzure, "@azure/"},
	{ProviderAzure, "azure-mgmt-"},
	{ProviderAzure, "azure-storage-"},
	{ProviderAzure, "azure-identity"},
	{ProviderAzure, "msrest"},

	{ProviderGCP, "cloud.google.com/go"},
	{ProviderGCP, "@google-cloud/"},
	{ProviderGCP, "google-cloud-"},
}

// Analyze scans one workload's directory (codescan.Workload.Path,
// resolved to an absolute path — not the whole repo) for known
// vendor-SDK package references in its manifest files. A manifest that
// doesn't exist is skipped, not an error — most workloads only have one
// of these.
func Analyze(workloadPath string) (Result, error) {
	var findings []Finding

	if content, err := os.ReadFile(filepath.Join(workloadPath, "go.mod")); err == nil {
		findings = append(findings, matchContent(string(content), "go.mod")...)
	}
	if content, err := os.ReadFile(filepath.Join(workloadPath, "package.json")); err == nil {
		findings = append(findings, matchPackageJSON(content)...)
	}
	if content, err := os.ReadFile(filepath.Join(workloadPath, "requirements.txt")); err == nil {
		findings = append(findings, matchRequirementsTxt(string(content))...)
	}
	if content, err := os.ReadFile(filepath.Join(workloadPath, "pyproject.toml")); err == nil {
		findings = append(findings, matchContent(string(content), "pyproject.toml")...)
	}

	return Result{Findings: findings}, nil
}

// matchContent checks raw file content for each known SDK prefix — safe
// for go.mod and pyproject.toml, whose dependency names appear as plain
// substrings with no risk of a false match from surrounding syntax.
func matchContent(content, source string) []Finding {
	var out []Finding
	for _, sdk := range knownSDKs {
		if strings.Contains(content, sdk.Prefix) {
			out = append(out, Finding{Provider: sdk.Provider, Package: sdk.Prefix, Source: source})
		}
	}
	return out
}

func matchRequirementsTxt(content string) []Finding {
	var out []Finding
	for _, line := range strings.Split(content, "\n") {
		name := requirementName(line)
		if name == "" {
			continue
		}
		for _, sdk := range knownSDKs {
			if strings.HasPrefix(name, sdk.Prefix) {
				out = append(out, Finding{Provider: sdk.Provider, Package: sdk.Prefix, Source: "requirements.txt"})
			}
		}
	}
	return out
}

// requirementName strips a requirements.txt line down to the bare
// package name — everything before the first version specifier or
// environment marker, and before any inline comment.
func requirementName(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	if i := strings.IndexAny(line, "#"); i >= 0 {
		line = line[:i]
	}
	cut := strings.IndexAny(line, "=<>!~; ")
	if cut >= 0 {
		line = line[:cut]
	}
	return strings.TrimSpace(line)
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func matchPackageJSON(content []byte) []Finding {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil
	}

	var out []Finding
	for name := range pkg.Dependencies {
		out = append(out, matchDependencyName(name, "package.json")...)
	}
	for name := range pkg.DevDependencies {
		out = append(out, matchDependencyName(name, "package.json")...)
	}
	return out
}

func matchDependencyName(name, source string) []Finding {
	var out []Finding
	for _, sdk := range knownSDKs {
		if strings.HasPrefix(name, sdk.Prefix) {
			out = append(out, Finding{Provider: sdk.Provider, Package: sdk.Prefix, Source: source})
		}
	}
	return out
}
