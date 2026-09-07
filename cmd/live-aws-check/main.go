// Command live-aws-check is a throwaway manual verification tool — not part
// of the product, not covered by CI, deleted once §5.2's real integration
// test suite exists to replace it. It exists only to confirm
// internal/provider/aws actually authenticates and lists against a real
// account, read-only, right now.
//
// Deliberately prints nothing that identifies the account per SECURITY.md's
// rule on real account identifiers — only counts and whether the call
// succeeded, never the account ID, ARN, or instance IDs themselves.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/provider/aws"
)

func main() {
	region := "us-east-1"
	if len(os.Args) > 1 {
		region = os.Args[1]
	}

	ctx := context.Background()
	c, err := aws.NewClient(ctx, region)
	if err != nil {
		fmt.Fprintln(os.Stderr, "client setup failed:", err)
		os.Exit(1)
	}

	_, _, err = c.WhoAmI(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}
	fmt.Println("auth OK — credentials valid")

	instances, err := c.ListInstances(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list instances failed:", err)
		os.Exit(1)
	}
	fmt.Printf("connected OK — region %s — %d instance(s) found\n", region, len(instances))

	states := map[string]int{}
	tagged := 0
	for _, i := range instances {
		states[i.State]++
		if len(i.Tags) > 0 {
			tagged++
		}
	}
	fmt.Printf("  by state: %v\n", states)
	fmt.Printf("  %d of %d instances have at least one tag\n", tagged, len(instances))
}
