// Command live-cloudflare-check is a throwaway manual verification tool —
// not part of the product, not covered by CI, deleted once §5.2's real
// integration test suite exists to replace it. Prints only counts and
// status, never zone names or IDs, per SECURITY.md's rule on real account
// identifiers.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/provider/cloudflare"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not resolve home dir:", err)
		os.Exit(1)
	}

	token, err := cloudflare.TokenFromFile(home + "/.cloudflare_token")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c := cloudflare.NewClient(token)
	zones, err := c.ListZones(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "list zones failed:", err)
		os.Exit(1)
	}

	statuses := map[string]int{}
	for _, z := range zones {
		statuses[z.Status]++
	}
	fmt.Printf("connected OK — %d zone(s) found, by status: %v\n", len(zones), statuses)
}
