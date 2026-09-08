// Package codescan detects workload boundaries in a source tree, per
// SPEC.md §5.0's three-tier method: an explicit deploy manifest (tier 1,
// high confidence), a module boundary with its own entry point (tier 2,
// medium confidence), or ambiguous source code with neither (tier 3,
// listed explicitly as unclassified rather than silently guessed).
//
// This is a file-layout heuristic — it reads directory structure and
// well-known marker files, nothing more. It is deliberately NOT §5.1's
// tree-sitter dependency graph, proprietary-SDK detection, or SDLC-
// maturity signals; those need a real per-language parser and aren't
// built yet. §5.0 treats workload-boundary detection as its own
// prerequisite phase before any of that, and this package answers
// exactly that one question: where are the boundaries.
package codescan

import (
	"os"
	"path/filepath"
	"sort"
)

// Confidence mirrors §5.0's boundary-detection tiers. There is no "low"
// value here — tier 3 (ambiguous) is represented by Unclassified, not by
// a low-confidence Workload, per §5.0's explicit instruction never to
// silently assign a guessed boundary.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
)

// Workload is one detected deployable unit.
type Workload struct {
	Name       string
	Path       string
	Confidence Confidence
	Signal     string // which file(s) triggered detection
}

// Unclassified is source code found with neither a deploy manifest nor a
// module-boundary file — §5.0 tier 3, surfaced explicitly rather than
// dropped or guessed at.
type Unclassified struct {
	Path   string
	Reason string
}

// Result is everything Analyze found in one repo.
type Result struct {
	Workloads    []Workload
	Unclassified []Unclassified
}

var skipDirNames = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".venv": true,
	"venv": true, "__pycache__": true, "dist": true, "build": true,
	"target": true, ".terraform": true, ".idea": true, ".vscode": true,
	".github": true,
}

// manifestMarkers are §5.0 tier 1: an explicit deploy manifest, one
// workload per manifest, high confidence.
var manifestMarkers = []string{
	"Dockerfile", "docker-compose.yml", "docker-compose.yaml",
	"serverless.yml", "serverless.yaml", "template.yaml", "template.yml",
}

// moduleMarkers are §5.0 tier 2 for non-Go ecosystems: a clear module
// boundary, medium confidence. go.mod is handled separately by
// goModWorkloads since a single Go module commonly builds several
// independently deployable binaries under cmd/.
var moduleMarkers = []string{
	"package.json", "pyproject.toml", "requirements.txt",
	"Cargo.toml", "pom.xml", "build.gradle",
}

var sourceExt = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".rb": true,
	".java": true, ".rs": true,
}

// Analyze walks root and returns every workload boundary and every
// unclassified source directory it finds. A directory that resolves to a
// workload (tier 1 or 2) has its subtree treated as belonging to that
// workload — everything below it is not separately classified, so a
// go.mod's internal packages don't each get flagged as their own
// ambiguous workload.
func Analyze(root string) (Result, error) {
	var res Result

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = ""
		}
		if rel != "" && skipDirNames[d.Name()] {
			return filepath.SkipDir
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		names := map[string]bool{}
		for _, e := range entries {
			names[e.Name()] = true
		}

		if marker, ok := firstPresent(names, manifestMarkers); ok {
			res.Workloads = append(res.Workloads, Workload{
				Name:       workloadName(rel),
				Path:       displayPath(rel),
				Confidence: ConfidenceHigh,
				Signal:     marker,
			})
			return filepath.SkipDir
		}

		if names["go.mod"] {
			res.Workloads = append(res.Workloads, goModWorkloads(rel, path)...)
			return filepath.SkipDir
		}

		if marker, ok := firstPresent(names, moduleMarkers); ok {
			res.Workloads = append(res.Workloads, Workload{
				Name:       workloadName(rel),
				Path:       displayPath(rel),
				Confidence: ConfidenceMedium,
				Signal:     marker,
			})
			return filepath.SkipDir
		}

		if hasSourceFiles(entries) {
			res.Unclassified = append(res.Unclassified, Unclassified{
				Path:   displayPath(rel),
				Reason: "source files present but no manifest or module-boundary file found",
			})
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	sort.Slice(res.Workloads, func(i, j int) bool { return res.Workloads[i].Path < res.Workloads[j].Path })
	sort.Slice(res.Unclassified, func(i, j int) bool { return res.Unclassified[i].Path < res.Unclassified[j].Path })
	return res, nil
}

// goModWorkloads handles a Go module root specially: if it has one or
// more cmd/<name>/main.go entry points, each is its own workload —
// §5.0 tier 2 (module boundary with its own entry point) applied once
// per binary, since one Go module commonly builds several independently
// deployable binaries. A module with no cmd/ entry point is reported as
// a single library workload instead of silently vanishing.
func goModWorkloads(rel, absPath string) []Workload {
	cmdDir := filepath.Join(absPath, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return []Workload{libraryWorkload(rel)}
	}

	var out []Workload
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mainPath := filepath.Join(cmdDir, e.Name(), "main.go")
		if _, err := os.Stat(mainPath); err != nil {
			continue
		}
		out = append(out, Workload{
			Name:       e.Name(),
			Path:       displayPath(filepath.Join(rel, "cmd", e.Name())),
			Confidence: ConfidenceMedium,
			Signal:     "go.mod + cmd/" + e.Name() + "/main.go",
		})
	}
	if len(out) == 0 {
		return []Workload{libraryWorkload(rel)}
	}
	return out
}

func libraryWorkload(rel string) Workload {
	return Workload{
		Name:       workloadName(rel),
		Path:       displayPath(rel),
		Confidence: ConfidenceMedium,
		Signal:     "go.mod (library, no cmd/ entry point found)",
	}
}

func firstPresent(names map[string]bool, candidates []string) (string, bool) {
	for _, c := range candidates {
		if names[c] {
			return c, true
		}
	}
	return "", false
}

func hasSourceFiles(entries []os.DirEntry) bool {
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if sourceExt[filepath.Ext(e.Name())] {
			return true
		}
	}
	return false
}

func workloadName(rel string) string {
	if rel == "" {
		return "root"
	}
	return filepath.Base(rel)
}

func displayPath(rel string) string {
	if rel == "" {
		return "."
	}
	return rel
}
