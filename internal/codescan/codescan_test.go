package codescan

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates path (and its parent directories) under root with
// the given content, failing the test on any error.
func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func TestAnalyze_SingleDockerfileRepo(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Dockerfile", "FROM scratch\n")
	writeFile(t, root, "main.py", "print('hi')\n")

	got, err := Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(got.Workloads) != 1 {
		t.Fatalf("workloads = %+v, want exactly 1", got.Workloads)
	}
	w := got.Workloads[0]
	if w.Confidence != ConfidenceHigh || w.Signal != "Dockerfile" || w.Path != "." {
		t.Errorf("workload = %+v, want high confidence Dockerfile at path .", w)
	}
	if len(got.Unclassified) != 0 {
		t.Errorf("unclassified = %+v, want none (Dockerfile claims the whole root)", got.Unclassified)
	}
}

func TestAnalyze_GoModuleWithMultipleBinaries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/thing\n")
	writeFile(t, root, "internal/foo/foo.go", "package foo\n")
	writeFile(t, root, "cmd/api/main.go", "package main\nfunc main() {}\n")
	writeFile(t, root, "cmd/worker/main.go", "package main\nfunc main() {}\n")
	writeFile(t, root, "cmd/notabinary/helper.go", "package notabinary\n") // no main.go here

	got, err := Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(got.Workloads) != 2 {
		t.Fatalf("workloads = %+v, want exactly 2 (api and worker)", got.Workloads)
	}
	names := map[string]bool{}
	for _, w := range got.Workloads {
		names[w.Name] = true
		if w.Confidence != ConfidenceMedium {
			t.Errorf("workload %s: confidence = %s, want medium", w.Name, w.Confidence)
		}
	}
	if !names["api"] || !names["worker"] {
		t.Errorf("names = %v, want api and worker", names)
	}
	if len(got.Unclassified) != 0 {
		t.Errorf("unclassified = %+v, want none — internal/ and cmd/notabinary belong to the go.mod workload", got.Unclassified)
	}
}

func TestAnalyze_GoLibraryNoEntryPoint(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/lib\n")
	writeFile(t, root, "lib.go", "package lib\n")

	got, err := Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(got.Workloads) != 1 {
		t.Fatalf("workloads = %+v, want exactly 1", got.Workloads)
	}
	if got.Workloads[0].Confidence != ConfidenceMedium {
		t.Errorf("confidence = %s, want medium", got.Workloads[0].Confidence)
	}
	if got.Workloads[0].Name != "root" || got.Workloads[0].Path != "." {
		t.Errorf("workload = %+v, want name=root path=.", got.Workloads[0])
	}
}

func TestAnalyze_MonorepoWithUnrelatedScript(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "services/api/Dockerfile", "FROM scratch\n")
	writeFile(t, root, "services/api/main.go", "package main\n")
	writeFile(t, root, "services/worker/package.json", `{"name":"worker"}`)
	writeFile(t, root, "scripts/deploy.py", "print('deploying')\n") // stray, no manifest/module marker

	got, err := Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(got.Workloads) != 2 {
		t.Fatalf("workloads = %+v, want exactly 2", got.Workloads)
	}
	byPath := map[string]Workload{}
	for _, w := range got.Workloads {
		byPath[w.Path] = w
	}
	if w, ok := byPath[filepath.Join("services", "api")]; !ok || w.Confidence != ConfidenceHigh {
		t.Errorf("services/api workload = %+v, ok=%v, want high confidence", w, ok)
	}
	if w, ok := byPath[filepath.Join("services", "worker")]; !ok || w.Confidence != ConfidenceMedium {
		t.Errorf("services/worker workload = %+v, ok=%v, want medium confidence", w, ok)
	}

	if len(got.Unclassified) != 1 || got.Unclassified[0].Path != "scripts" {
		t.Errorf("unclassified = %+v, want exactly one entry for scripts/", got.Unclassified)
	}
}

func TestAnalyze_SkipsNoiseDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/thing\n")
	writeFile(t, root, "vendor/github.com/x/y/y.go", "package y\n")
	writeFile(t, root, "node_modules/leftpad/index.js", "module.exports = {}\n")

	got, err := Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(got.Workloads) != 1 {
		t.Fatalf("workloads = %+v, want exactly 1 (root go.mod claims everything, vendor/node_modules never walked)", got.Workloads)
	}
	if len(got.Unclassified) != 0 {
		t.Errorf("unclassified = %+v, want none", got.Unclassified)
	}
}
