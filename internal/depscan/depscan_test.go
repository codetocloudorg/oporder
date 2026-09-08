package depscan

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func TestAnalyze_GoModWithAWSSDK(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", `module example.com/thing

go 1.27

require (
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.0.0
)
`)

	got, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !got.HasProprietaryDependencies() {
		t.Fatalf("HasProprietaryDependencies() = false, want true for %+v", got.Findings)
	}
	if providers := got.Providers(); len(providers) != 1 || providers[0] != ProviderAWS {
		t.Errorf("Providers() = %v, want [aws]", providers)
	}
}

func TestAnalyze_PackageJSONWithAzureSDK(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
  "name": "thing",
  "dependencies": {
    "express": "^4.18.0",
    "@azure/storage-blob": "^12.0.0"
  }
}`)

	got, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !got.HasProprietaryDependencies() {
		t.Fatalf("expected a match on @azure/storage-blob, got %+v", got.Findings)
	}
	if providers := got.Providers(); len(providers) != 1 || providers[0] != ProviderAzure {
		t.Errorf("Providers() = %v, want [azure]", providers)
	}
	// "express" must never match — a generic dependency isn't a
	// proprietary managed-service SDK.
	for _, f := range got.Findings {
		if f.Package == "express" {
			t.Errorf("express incorrectly matched as a proprietary SDK")
		}
	}
}

func TestAnalyze_RequirementsTxtWithGCP(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "flask==2.3.0\ngoogle-cloud-storage==2.10.0  # comment\n# a full-line comment\n")

	got, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !got.HasProprietaryDependencies() {
		t.Fatalf("expected a match on google-cloud-storage, got %+v", got.Findings)
	}
	if providers := got.Providers(); len(providers) != 1 || providers[0] != ProviderGCP {
		t.Errorf("Providers() = %v, want [gcp]", providers)
	}
}

func TestAnalyze_NoProprietaryDependencies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/thing\n\ngo 1.27\n\nrequire github.com/spf13/cobra v1.8.0\n")

	got, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.HasProprietaryDependencies() {
		t.Errorf("expected no findings, got %+v", got.Findings)
	}
}

func TestAnalyze_NoManifestsPresent(t *testing.T) {
	dir := t.TempDir()

	got, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.HasProprietaryDependencies() {
		t.Errorf("expected no findings for an empty directory, got %+v", got.Findings)
	}
}

func TestRequirementName(t *testing.T) {
	tests := map[string]string{
		"boto3==1.26.0":              "boto3",
		"  flask >= 2.0  ":           "flask",
		"# a comment":                "",
		"":                           "",
		"requests[security]==2.28.0": "requests[security]",
		"boto3==1.26.0  # pinned":    "boto3",
	}
	for in, want := range tests {
		if got := requirementName(in); got != want {
			t.Errorf("requirementName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPackageJSON_InvalidJSONIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", "not valid json{{{")

	got, err := Analyze(dir)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got.HasProprietaryDependencies() {
		t.Errorf("expected no findings from invalid JSON, got %+v", got.Findings)
	}
}
