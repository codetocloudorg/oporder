// Command oporder is the entry point for the OpOrder CLI.
//
// See SPEC.md for the full design. This is an early, honest stub: the
// assessment engine (SPEC.md §5) isn't wired up yet, and no subcommand here
// touches a real cloud account or an LLM. It exists so the project has a
// real, buildable binary from commit one, and so every package added under
// internal/ has somewhere to actually get called from.
package main

import (
	"fmt"
	"os"
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
		fmt.Println("oporder scan: not implemented yet — see SPEC.md §5 for the design, and §10 for what's built so far.")
		os.Exit(1)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "oporder: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`oporder — vendor-neutral assessment for cloud migration and application modernization

Usage:
  oporder scan .       Assess the current directory and connected cloud accounts (not implemented yet)
  oporder version      Print the version
  oporder help         Show this message

No binary release exists yet. Follow progress: https://github.com/codetocloudorg/oporder`)
}
