// Command live-gcp-check is a throwaway manual verification tool — not
// part of the product, not covered by CI, deleted once §5.2's real
// integration test suite exists to replace it. It exists to confirm
// internal/provider/gcp actually authenticates and lists against a real
// project, read-only, right now. Unlike its AWS/Azure/Cloudflare
// counterparts, this hasn't been run against a real project yet — no GCP
// account was available when it was written.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/provider/gcp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: live-gcp-check <project-id>")
		os.Exit(1)
	}

	ctx := context.Background()
	c, err := gcp.NewClient(ctx, os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}
	defer c.Close()

	instances, err := c.ListInstances(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list instances failed:", err)
		os.Exit(1)
	}

	fmt.Printf("connected OK — %d instance(s) found\n", len(instances))
	for _, inst := range instances {
		fmt.Printf("  - %s (%s) status=%s labels=%v\n", inst.Name, inst.Zone, inst.Status, inst.Labels)
	}
}
