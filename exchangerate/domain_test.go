package exchangerate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// These tests are offline (pure string) or use httptest servers.
// Heavy HTTP behaviour is covered in exchangerate_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "exchangerate" {
		t.Errorf("Scheme = %q, want exchangerate", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "exchangerate" {
		t.Errorf("Identity.Binary = %q, want exchangerate", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("USD")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "rate" {
		t.Errorf("type = %q, want rate", typ)
	}
	if id != "USD" {
		t.Errorf("id = %q, want USD", id)
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("rate", "USD")
	want := "https://open.er-api.com/v6/latest/USD"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateBadType(t *testing.T) {
	_, err := Domain{}.Locate("page", "whatever")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}

const domainFakeJSON = `{
  "result": "success",
  "base_code": "USD",
  "time_last_update_utc": "Sun, 14 Jun 2026 00:02:31 +0000",
  "rates": {
    "USD": 1.0,
    "EUR": 0.864531,
    "GBP": 0.746107,
    "JPY": 160.232666,
    "CNY": 6.781714
  }
}`

func newDomainTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClient(cfg)
}

// TestLatestOpNoFilter verifies latestOp emits all rates when --to is empty.
func TestLatestOpNoFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, domainFakeJSON)
	}))
	defer ts.Close()

	var got []Rate
	in := latestInput{Base: "USD", To: "", Client: newDomainTestClient(ts)}
	err := latestOp(context.Background(), in, func(r Rate) error {
		got = append(got, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("no-filter: got %d items, want 5", len(got))
	}
}

// TestLatestOpToFilter verifies --to filters to only the requested codes.
func TestLatestOpToFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, domainFakeJSON)
	}))
	defer ts.Close()

	var got []Rate
	in := latestInput{Base: "USD", To: "EUR,GBP", Client: newDomainTestClient(ts)}
	err := latestOp(context.Background(), in, func(r Rate) error {
		got = append(got, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("--to EUR,GBP: got %d items, want 2", len(got))
	}
	codes := map[string]bool{got[0].Currency: true, got[1].Currency: true}
	if !codes["EUR"] || !codes["GBP"] {
		t.Errorf("--to EUR,GBP: got currencies %v, want EUR and GBP", codes)
	}
}

// TestLatestOpToFilterLowercase verifies codes are normalised to uppercase.
func TestLatestOpToFilterLowercase(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, domainFakeJSON)
	}))
	defer ts.Close()

	var got []Rate
	in := latestInput{Base: "USD", To: "eur,jpy", Client: newDomainTestClient(ts)}
	err := latestOp(context.Background(), in, func(r Rate) error {
		got = append(got, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("--to eur,jpy: got %d items, want 2", len(got))
	}
}
