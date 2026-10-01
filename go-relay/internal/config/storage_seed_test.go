package config

import "testing"

func TestParseLocalStorageSeed(t *testing.T) {
	if seed, err := ParseLocalStorageSeed("  "); err != nil || seed != nil {
		t.Fatalf("blank must mean no seed, got %v, %v", seed, err)
	}

	seed, err := ParseLocalStorageSeed(`{"mod.clientId":"\"abc\"","mod.credentials":"{\"pairingId\":\"p\"}"}`)
	if err != nil {
		t.Fatal(err)
	}
	if seed["mod.clientId"] != `"abc"` || seed["mod.credentials"] != `{"pairingId":"p"}` {
		t.Fatalf("values must be kept verbatim, got %#v", seed)
	}

	for _, bad := range []string{`not json`, `["a"]`, `{"k": 1}`, `{"k": {"nested": "x"}}`, `{"k": null}x`} {
		if _, err := ParseLocalStorageSeed(bad); err == nil {
			t.Errorf("%q must be rejected: values have to be strings", bad)
		}
	}
}
