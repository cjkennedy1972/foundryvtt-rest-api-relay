package worker

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

const foundryOrigin = "http://foundry:30000"

func TestLocalStorageSeedScriptIsDeterministicGuardedAndOriginScoped(t *testing.T) {
	seed := map[string]string{"b.key": "2", "a.key": "1"}
	first, err := localStorageSeedScript(foundryOrigin, seed)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if again, _ := localStorageSeedScript(foundryOrigin, seed); again != first {
			t.Fatal("script must not depend on map iteration order")
		}
	}
	if strings.Index(first, `"a.key"`) > strings.Index(first, `"b.key"`) {
		t.Fatal("keys must be written in sorted order")
	}
	// A rotated credential (re-pair) must survive a reload, so the seed only fills a missing key.
	if got := strings.Count(first, "getItem("); got != 2 {
		t.Fatalf("every entry must be guarded by an absence check, got %d guards", got)
	}
	// The script runs on every document the tab loads; it must not write anywhere but its own origin.
	if !strings.Contains(first, `window.location.origin === "http://foundry:30000"`) {
		t.Fatalf("script must check the page origin before writing:\n%s", first)
	}
	if strings.Index(first, "location.origin") > strings.Index(first, "setItem(") {
		t.Fatal("the origin check must come before any write")
	}
}

func TestLocalStorageSeedScriptQuotesHostileKeysAndValues(t *testing.T) {
	hostile := map[string]string{`k"); alert(1); ("`: `v"); alert(2); ("` + "\n</script> "}
	script, err := localStorageSeedScript(foundryOrigin, hostile)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`setItem\(("(?:[^"\\]|\\.)*"), ("(?:[^"\\]|\\.)*")\)`).FindStringSubmatch(script)
	if m == nil {
		t.Fatalf("no setItem call found in:\n%s", script)
	}
	var key, val string
	if json.Unmarshal([]byte(m[1]), &key) != nil || json.Unmarshal([]byte(m[2]), &val) != nil {
		t.Fatal("key and value must be valid JSON string literals")
	}
	for k, v := range hostile {
		if key != k || val != v {
			t.Fatalf("round trip lost data: %q=%q", key, val)
		}
	}
}

// The seed is a secret and the Foundry URL of a session is supplied by the caller, so only the
// operator-listed origin may ever receive it.
func TestSeedForOnlyReturnsEntriesForTheConfiguredOrigin(t *testing.T) {
	m := &HeadlessManager{storageSeed: map[string]map[string]string{
		foundryOrigin: {"mod.credentials": "secret"},
	}}

	if origin, entries := m.seedFor("http://foundry:30000/game/join"); origin != foundryOrigin || entries["mod.credentials"] != "secret" {
		t.Fatalf("the configured origin must be seeded, got %q %v", origin, entries)
	}
	if _, entries := m.seedFor("HTTP://FOUNDRY:30000"); len(entries) != 1 {
		t.Fatal("scheme and host case must not matter")
	}

	attacker := []string{
		"http://evil.example:30000",
		"http://foundry:30001",                // same host, other port
		"https://foundry:30000",               // same host, other scheme
		"http://foundry:30000@evil.example/",  // userinfo trick: the host is evil.example
		"http://evil.example/#@foundry:30000", // fragment trick
		"http://foundry.evil.example:30000",   // look-alike subdomain
		"not a url",
		"",
		"javascript:alert(1)",
	}
	for _, u := range attacker {
		if _, entries := m.seedFor(u); len(entries) != 0 {
			t.Errorf("%q must receive no seed, got %v", u, entries)
		}
	}
	if _, entries := (&HeadlessManager{}).seedFor("http://foundry:30000"); len(entries) != 0 {
		t.Fatal("no configured seed means nothing to inject")
	}
}

func TestInjectLocalStorageSeedIsANoOpWithoutEntries(t *testing.T) {
	// A nil context is never touched when there is nothing to seed.
	if err := injectLocalStorageSeed(nil, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := injectLocalStorageSeed(nil, foundryOrigin, map[string]string{}); err != nil {
		t.Fatal(err)
	}
}
