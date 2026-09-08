// Command live-azure-check is a throwaway manual verification tool — not
// part of the product, not covered by CI, deleted once §5.2's real
// integration test suite exists to replace it. It exists only to confirm
// internal/provider/azure actually authenticates and lists against a real
// subscription, read-only, right now.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/codetocloudorg/oporder/internal/provider/azure"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: live-azure-check <subscription-id>")
		os.Exit(1)
	}

	c, err := azure.NewClient(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	groups, err := c.ListResourceGroups(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list resource groups failed:", err)
		os.Exit(1)
	}

	fmt.Printf("connected OK — %d resource group(s) found\n", len(groups))
	for _, g := range groups {
		fmt.Printf("  - %s (%s) tags=%v\n", g.Name, g.Location, g.Tags)
		resources, err := c.ListResources(ctx, g.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    list resources in %s failed: %v\n", g.Name, err)
			continue
		}
		for _, r := range resources {
			fmt.Printf("      · %s (%s) tags=%v\n", r.Name, r.Type, r.Tags)
			if r.Type != azure.VirtualMachineResourceType || r.ID == "" {
				continue
			}
			u, err := c.CPUUtilization(ctx, r.ID, 7*24*time.Hour)
			if err != nil {
				fmt.Printf("        utilization query failed: %v\n", err)
				continue
			}
			if !u.HasTelemetry {
				fmt.Println("        utilization: no Azure Monitor datapoints in the last 7 days (un-instrumented, not necessarily idle)")
				continue
			}
			fmt.Printf("        utilization: avg %.1f%% CPU over the last 7 days\n", u.AverageCPUPercent)
		}
	}
}
