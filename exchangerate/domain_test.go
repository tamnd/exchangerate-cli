package exchangerate

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions.
// HTTP behaviour is covered in exchangerate_test.go.

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
