package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestList_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Items: []Price{
				{RetailPrice: 0.096, CurrencyCode: "USD", ArmSkuName: "Standard_D2s_v5", ServiceName: "Virtual Machines"},
			},
			NextPageLink: "",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	prices, err := c.List(context.Background(), Query{ServiceName: "Virtual Machines", ArmRegionName: "eastus"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("len(prices) = %d, want 1", len(prices))
	}
	if prices[0].RetailPrice != 0.096 {
		t.Errorf("RetailPrice = %v, want 0.096", prices[0].RetailPrice)
	}
}

// TestList_FollowsPagination is the specific behavior §5.6 depends on for
// a complete price list — a query with more results than one page returns
// must not silently stop at page one.
func TestList_FollowsPagination(t *testing.T) {
	var callCount int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp apiResponse
		if callCount == 1 {
			resp = apiResponse{
				Items:        []Price{{ArmSkuName: "page-1-item"}},
				NextPageLink: srv.URL + "/page2",
			}
		} else {
			resp = apiResponse{
				Items:        []Price{{ArmSkuName: "page-2-item"}},
				NextPageLink: "",
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	prices, err := c.List(context.Background(), Query{ServiceName: "Virtual Machines"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("len(prices) = %d, want 2 (one per page) — pagination did not follow NextPageLink", len(prices))
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestList_NonOKStatusIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	_, err := c.List(context.Background(), Query{ServiceName: "Virtual Machines"})
	if err == nil {
		t.Fatal("expected an error on a non-200 response, got nil")
	}
}

func TestBuildFilter(t *testing.T) {
	cases := []struct {
		name string
		q    Query
		want string
	}{
		{
			name: "empty query has no filter",
			q:    Query{},
			want: "",
		},
		{
			name: "single field",
			q:    Query{ServiceName: "Virtual Machines"},
			want: "serviceName eq 'Virtual Machines'",
		},
		{
			name: "multiple fields joined with and",
			q:    Query{ServiceName: "Virtual Machines", ArmRegionName: "eastus"},
			want: "serviceName eq 'Virtual Machines' and armRegionName eq 'eastus'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildFilter(tc.q).Get("$filter")
			if got != tc.want {
				t.Errorf("buildFilter(%+v) filter = %q, want %q", tc.q, got, tc.want)
			}
		})
	}
}
