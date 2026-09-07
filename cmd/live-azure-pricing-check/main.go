// Command live-azure-pricing-check is a throwaway manual verification tool
// confirming internal/pricing/azure actually reaches the real, public Azure
// Retail Prices API. Not part of the product, not in CI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/codetocloudorg/oporder/internal/pricing/azure"
)

func main() {
	c := azure.NewClient()
	prices, err := c.List(context.Background(), azure.Query{
		ServiceName:   "Virtual Machines",
		ArmRegionName: "eastus",
		ArmSkuName:    "Standard_D2s_v5",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "query failed:", err)
		os.Exit(1)
	}
	fmt.Printf("connected OK — %d price entries found for Standard_D2s_v5 in eastus\n", len(prices))
	for _, p := range prices {
		fmt.Printf("  %s: %.4f %s per %s\n", p.ProductName, p.RetailPrice, p.CurrencyCode, p.UnitOfMeasure)
	}
}
