// Package azure implements a client for the Azure Retail Prices API —
// SPEC.md §5.6's confirmed-live, keyless pricing source for Azure. Standard
// library only: this is a plain, public, unauthenticated REST API, and
// pulling in a full SDK for one GET-with-pagination endpoint would be
// exactly the kind of unearned complexity §1 rules out elsewhere.
//
// This package is intentionally separate from internal/provider/azure,
// which handles live account inventory and does need the full Azure SDK
// and an authenticated session. Pricing needs neither — that distinction is
// real and worth keeping the two packages apart over.
package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const defaultBaseURL = "https://prices.azure.com/api/retail/prices"

// Client queries the Azure Retail Prices API. BaseURL is overridable so
// tests can point it at a local httptest server instead of the real
// endpoint — see azure_test.go.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a client pointed at the real, public Azure Retail
// Prices API with a sane default HTTP client.
func NewClient() *Client {
	return &Client{
		BaseURL:    defaultBaseURL,
		HTTPClient: http.DefaultClient,
	}
}

// Query filters the price list — all fields optional, matching the API's
// own OData filter semantics. Left empty, ServiceName and ArmRegionName
// have to be provided by the caller in practice, since an unfiltered query
// against the full retail price list is enormous.
type Query struct {
	ServiceName   string // e.g. "Virtual Machines"
	ArmRegionName string // e.g. "eastus"
	ArmSkuName    string // e.g. "Standard_D2s_v5" — optional, narrows to one SKU
}

// Price is the subset of the API's response fields §5.6's cost engine
// actually needs — not the full response shape, which carries fields this
// project has no use for.
type Price struct {
	RetailPrice   float64 `json:"retailPrice"`
	CurrencyCode  string  `json:"currencyCode"`
	UnitOfMeasure string  `json:"unitOfMeasure"`
	ArmRegionName string  `json:"armRegionName"`
	ArmSkuName    string  `json:"armSkuName"`
	ProductName   string  `json:"productName"`
	ServiceName   string  `json:"serviceName"`
	Type          string  `json:"type"`
}

type apiResponse struct {
	Items        []Price `json:"Items"`
	NextPageLink string  `json:"NextPageLink"`
}

// List returns every price matching the query, following pagination via
// the API's own NextPageLink until it's empty. §5.6 requires prices to be
// queried live at report time, never baked in — this method has no cache
// and makes a real HTTP call every time, by design.
func (c *Client) List(ctx context.Context, q Query) ([]Price, error) {
	var out []Price

	next := c.BaseURL + "?" + buildFilter(q).Encode()
	for next != "" {
		var resp apiResponse
		if err := c.getJSON(ctx, next, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Items...)
		next = resp.NextPageLink
	}
	return out, nil
}

func buildFilter(q Query) url.Values {
	var clauses []string
	if q.ServiceName != "" {
		clauses = append(clauses, fmt.Sprintf("serviceName eq '%s'", q.ServiceName))
	}
	if q.ArmRegionName != "" {
		clauses = append(clauses, fmt.Sprintf("armRegionName eq '%s'", q.ArmRegionName))
	}
	if q.ArmSkuName != "" {
		clauses = append(clauses, fmt.Sprintf("armSkuName eq '%s'", q.ArmSkuName))
	}

	filter := ""
	for i, c := range clauses {
		if i > 0 {
			filter += " and "
		}
		filter += c
	}

	v := url.Values{}
	if filter != "" {
		v.Set("$filter", filter)
	}
	return v
}

func (c *Client) getJSON(ctx context.Context, target string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("azure pricing: building request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("azure pricing: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("azure pricing: unexpected status %d from %s", resp.StatusCode, target)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("azure pricing: decoding response: %w", err)
	}
	return nil
}
