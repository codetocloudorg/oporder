// Command oporder is the entry point for the OpOrder CLI.
//
// See SPEC.md for the full design. `scan` is honestly scoped to what's
// real today: live inventory across whichever cloud providers have
// credentials configured (internal/scan), written to SITUATION.md. It does
// not yet do code analysis, correlation, or produce a 5/7-Rs recommendation
// — see internal/scan's own package doc for exactly what's real so far.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/scan"
)

const version = "0.0.0-dev"

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
	opts := scan.Options{
		AzureSubscriptionID: os.Getenv("OPORDER_AZURE_SUBSCRIPTION_ID"),
		AWSRegion:           os.Getenv("AWS_REGION"),
		CloudflareTokenFile: cloudflareTokenPath(),
	}

	if opts.AzureSubscriptionID == "" && opts.AWSRegion == "" && opts.CloudflareTokenFile == "" {
		fmt.Println(`oporder scan: no provider configured — nothing to do yet.

Set at least one of these before running again:
  OPORDER_AZURE_SUBSCRIPTION_ID   an Azure subscription ID (uses your existing 'az login' session)
  AWS_REGION                       an AWS region, e.g. us-east-1 (uses your existing AWS credentials)
  ~/.cloudflare_token              a file containing a Cloudflare API token (see SPEC.md §4.6a)

This only lists live inventory right now — code analysis, correlation, and the 5/7-Rs
recommendation aren't wired in yet. See SPEC.md §10 for what's built so far.`)
		os.Exit(1)
	}

	fmt.Println("oporder scan: checking configured providers...")
	situation := scan.Run(context.Background(), opts)

	const outPath = "SITUATION.md"
	if err := scan.WriteSituationMD(outPath, situation); err != nil {
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
	fmt.Printf("\nWrote %s\n", outPath)
	if !anyRan {
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
  oporder scan          Live inventory across configured providers (see below), writes SITUATION.md
  oporder version       Print the version
  oporder help          Show this message

Provider setup (at least one required for 'scan'):
  export OPORDER_AZURE_SUBSCRIPTION_ID=<id>   uses your existing 'az login' session
  export AWS_REGION=us-east-1                 uses your existing AWS credentials
  echo "<token>" > ~/.cloudflare_token         generate a token at https://dash.cloudflare.com/profile/api-tokens

No binary release exists yet. Follow progress: https://github.com/codetocloudorg/oporder`)
}
