package worker

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestLocalStorageSeedScriptIsDeterministicAndOnlyWritesAbsentKeys(t *testing.T) {
	seed := map[string]string{"b.key": "2", "a.key": "1"}
	first, err := localStorageSeedScript(seed)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		again, _ := localStorageSeedScript(seed)
		if again != first {
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
}

func TestLocalStorageSeedScriptQuotesHostileKeysAndValues(t *testing.T) {
	hostile := map[string]string{
		`k"); alert(1); ("`: `v"); alert(2); ("` + "\n</script> ",
	}
	script, err := localStorageSeedScript(hostile)
	if err != nil {
		t.Fatal(err)
	}
	// The key and value must come back out of the script exactly, as JSON string literals.
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
	if strings.Contains(script, "alert(1); (") && !strings.Contains(script, `\"); alert(1)`) {
		t.Fatal("hostile text escaped its string literal")
	}
}

func TestInjectLocalStorageSeedIsANoOpWithoutASeed(t *testing.T) {
	// A nil context is never touched when there is nothing to seed.
	if err := injectLocalStorageSeed(nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := injectLocalStorageSeed(nil, map[string]string{}); err != nil {
		t.Fatal(err)
	}
}
