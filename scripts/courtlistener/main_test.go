package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNetrcPassword(t *testing.T) {
	data := "machine api.dnsimple.com password other\nmachine www.courtlistener.com\n  login jp\n  password tok123\ndefault login x password y\n"
	if got, ok := netrcPassword(data, netrcHost); !ok || got != "tok123" {
		t.Fatalf("got %q %v, want tok123", got, ok)
	}
	if _, ok := netrcPassword("machine other password p", netrcHost); ok {
		t.Fatal("found a password for a host not present")
	}
}

func TestReadTokenRefusesLooseNetrc(t *testing.T) {
	p := filepath.Join(t.TempDir(), "netrc")
	if err := os.WriteFile(p, []byte("machine www.courtlistener.com password t"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(p); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("err = %v, want a chmod 600 refusal", err)
	}
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if tok, err := readToken(p); err != nil || tok != "t" {
		t.Fatalf("got %q %v", tok, err)
	}
}

func TestSpendRefusesAtEachWindow(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "requests.log")
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		if err := spend(ledger, now.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatalf("request %d refused: %v", i, err)
		}
	}
	if err := spend(ledger, now.Add(10*time.Second)); err == nil || !strings.Contains(err.Error(), "5 of 5") {
		t.Fatalf("sixth request in a minute: err = %v, want refusal", err)
	}
	if err := spend(ledger, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("after the minute window: %v", err)
	}
}

func TestBuildGet(t *testing.T) {
	req, err := buildGet([]string{"search", "type=o", "q=cites:(1 OR 2)"})
	if err != nil {
		t.Fatal(err)
	}
	if got := req.URL.String(); got != baseURL+"search/?q=cites%3A%281+OR+2%29&type=o" {
		t.Fatalf("url = %s", got)
	}
	for _, bad := range []string{"../x", "https://evil", ""} {
		if _, err := buildGet([]string{bad}); err == nil {
			t.Errorf("endpoint %q accepted", bad)
		}
	}
}

func TestLedgerShrinksWithoutStaleTail(t *testing.T) {
	ledger := filepath.Join(t.TempDir(), "requests.log")
	day := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	for i := range 3 {
		if err := spend(ledger, day.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if err := spend(ledger, day.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(data), "\n"); lines != 1 {
		t.Fatalf("ledger holds %d lines after pruning, want 1:\n%s", lines, data)
	}
}
