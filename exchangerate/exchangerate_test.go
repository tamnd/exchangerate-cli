package exchangerate_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/exchangerate-cli/exchangerate"
)

const fakeLatestJSON = `{
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

func newTestClient(ts *httptest.Server) *exchangerate.Client {
	cfg := exchangerate.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	return exchangerate.NewClient(cfg)
}

func TestLatestSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeLatestJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Latest(context.Background(), "USD", 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestLatestParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeLatestJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Latest(context.Background(), "USD", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}
	// sorted alphabetically: CNY EUR GBP JPY USD
	if items[0].Currency != "CNY" {
		t.Errorf("items[0].Currency = %q, want CNY", items[0].Currency)
	}
	if items[0].Rank != 1 {
		t.Errorf("items[0].Rank = %d, want 1", items[0].Rank)
	}
	if items[0].Base != "USD" {
		t.Errorf("items[0].Base = %q, want USD", items[0].Base)
	}
	if items[1].Currency != "EUR" {
		t.Errorf("items[1].Currency = %q, want EUR", items[1].Currency)
	}
	if math.Abs(items[1].Rate-0.864531) > 1e-9 {
		t.Errorf("items[1].Rate = %f, want 0.864531", items[1].Rate)
	}
}

func TestLatestLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeLatestJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Latest(context.Background(), "USD", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(items))
	}
}

func TestLatestRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeLatestJSON)
	}))
	defer ts.Close()

	cfg := exchangerate.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := exchangerate.NewClient(cfg)

	_, err := c.Latest(context.Background(), "USD", 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestConvertCalculatesResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeLatestJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	conv, err := c.Convert(context.Background(), "USD", "EUR", 100.0)
	if err != nil {
		t.Fatal(err)
	}
	if conv.From != "USD" {
		t.Errorf("From = %q, want USD", conv.From)
	}
	if conv.To != "EUR" {
		t.Errorf("To = %q, want EUR", conv.To)
	}
	if math.Abs(conv.Result-86.4531) > 1e-9 {
		t.Errorf("Result = %f, want 86.4531", conv.Result)
	}
	if conv.UpdatedAt == "" {
		t.Error("UpdatedAt is empty")
	}
}
