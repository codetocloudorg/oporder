// Command oporder is the entry point for the OpOrder CLI.
//
// See SPEC.md for the full design. `scan` is honestly scoped to what's
// real today: live inventory across whichever cloud providers have
// credentials configured, workload-boundary detection and proprietary-
// dependency scanning against the current directory, tag/name
// correlation between code and infrastructure, and a partial-evidence
// 5/7-Rs call where enough real evidence exists (internal/scan), written
// to SITUATION.md and MISSION.md — see internal/scan's own package doc
// for exactly what's real so far and what isn't.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/scan"
)

// version is overridden at build time via -ldflags "-X main.version=vX.Y.Z"
// by .goreleaser.yaml — "dev" is the correct value for a `go run`/`go
// build` invocation with no injected version, not a placeholder to fix.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println("oporder " + version)
	case "scan":
		runScan()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "oporder: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runScan() {
	repoPath, err := os.Getwd()
	if err != nil {
		repoPath = "."
	}

	opts := scan.Options{
		RepoPath:            repoPath,
		AzureSubscriptionID: os.Getenv("OPORDER_AZURE_SUBSCRIPTION_ID"),
		AWSRegion:           os.Getenv("AWS_REGION"),
		CloudflareTokenFile: cloudflareTokenPath(),
		GCPProjectID:        os.Getenv("OPORDER_GCP_PROJECT_ID"),
	}

	if opts.AzureSubscriptionID == "" && opts.AWSRegion == "" && opts.CloudflareTokenFile == "" && opts.GCPProjectID == "" {
		fmt.Println(`oporder scan: no cloud provider configured — running code analysis only.

Set at least one of these to also scan live infrastructure and correlate it against
the code found here:
  OPORDER_AZURE_SUBSCRIPTION_ID   an Azure subscription ID (uses your existing 'az login' session)
  AWS_REGION                       an AWS region, e.g. us-east-1 (uses your existing AWS credentials)
  OPORDER_GCP_PROJECT_ID           a GCP project ID (uses Application Default Credentials)
  ~/.cloudflare_token              a file containing a Cloudflare API token (see SPEC.md §4.6a)

The 5/7-Rs recommendation isn't wired in yet. See SPEC.md §10 for what's built so far.`)
	}

	fmt.Println("oporder scan: checking configured providers and analyzing code...")
	situation := scan.Run(context.Background(), opts)

	const situationPath = "SITUATION.md"
	if err := scan.WriteSituationMD(situationPath, situation); err != nil {
		fmt.Fprintln(os.Stderr, "oporder scan: writing report failed:", err)
		os.Exit(1)
	}

	anyRan := false
	for _, p := range situation.Providers {
		fmt.Printf("  %s: %s\n", p.Provider, p.Detail)
		if p.Ran {
			anyRan = true
		}
	}
	if len(situation.Workloads) > 0 {
		fmt.Printf("  code: %d workload(s) detected\n", len(situation.Workloads))
	}
	fmt.Printf("\nWrote %s\n", situationPath)

	if len(situation.Missions) > 0 {
		const missionPath = "MISSION.md"
		if err := scan.WriteMissionMD(missionPath, situation); err != nil {
			fmt.Fprintln(os.Stderr, "oporder scan: writing mission report failed:", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s (%d workload call(s), partial evidence — see SPEC.md §10 M2)\n", missionPath, len(situation.Missions))
	}

	if !anyRan && len(situation.Workloads) == 0 {
		os.Exit(1)
	}
}

// cloudflareTokenPath returns ~/.cloudflare_token if it exists, empty
// otherwise — Cloudflare stays optional and silently skipped rather than
// an error when nobody's configured it, same as the other two providers
// being controlled by whether their env vars are set.
func cloudflareTokenPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := home + "/.cloudflare_token"
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

func printUsage() {
	fmt.Println(`oporder — vendor-neutral assessment for cloud migration and application modernization

Usage:
  oporder scan          Analyzes the current directory's code, plus any configured cloud
                        providers below, and correlates the two. Writes SITUATION.md.
  oporder version       Print the version
  oporder help          Show this message

Cloud provider setup (all optional — 'scan' always analyzes code even with none set):
  export OPORDER_AZURE_SUBSCRIPTION_ID=<id>   uses your existing 'az login' session
  export AWS_REGION=us-east-1                 uses your existing AWS credentials
  export OPORDER_GCP_PROJECT_ID=<id>          uses Application Default Credentials
  echo "<token>" > ~/.cloudflare_token         generate a token at https://dash.cloudflare.com/profile/api-tokens`)
	printReleaseFooter()
}

// printReleaseFooter states honestly whether this exact binary is a
// real tagged release (version injected via .goreleaser.yaml's ldflags)
// or a local dev build (version left at its "dev" default) — the
// message a released binary ships with can't hardcode "no release
// exists yet" the way the source once did, since that binary would
// itself be the release.
func printReleaseFooter() {
	if version == "dev" {
		fmt.Println("No binary release exists yet. Follow progress: https://github.com/codetocloudorg/oporder")
		return
	}
	fmt.Printf("oporder %s — https://github.com/codetocloudorg/oporder\n", version)
}
