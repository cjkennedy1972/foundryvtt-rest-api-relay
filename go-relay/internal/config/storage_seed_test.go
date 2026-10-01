package config

import "testing"

func TestParseLocalStorageSeed(t *testing.T) {
	if seed, err := ParseLocalStorageSeed("  "); err != nil || seed != nil {
		t.Fatalf("blank must mean no seed, got %v, %v", seed, err)
	}

	seed, err := ParseLocalStorageSeed(`{"http://foundry:30000":{"mod.clientId":"\"abc\"","mod.credentials":"{\"pairingId\":\"p\"}"}}`)
	if err != nil {
		t.Fatal(err)
	}
	entries := seed["http://foundry:30000"]
	if entries["mod.clientId"] != `"abc"` || entries["mod.credentials"] != `{"pairingId":"p"}` {
		t.Fatalf("values must be kept verbatim, got %#v", seed)
	}
}

func TestParseLocalStorageSeedRejectsMalformedInput(t *testing.T) {
	bad := map[string]string{
		"not json":                           `not json`,
		"flat map (the old, unscoped shape)": `{"mod.key":"value"}`,
		"array":                              `["a"]`,
		"non-string value":                   `{"http://foundry:30000":{"k":1}}`,
		"null value":                         `{"http://foundry:30000":{"k":null}}`,
		"null entries":                       `{"http://foundry:30000":null}`,
		"origin without scheme":              `{"foundry:30000":{"k":"v"}}`,
		"origin with a path":                 `{"http://foundry:30000/game":{"k":"v"}}`,
		"origin with uppercase host":         `{"http://Foundry:30000":{"k":"v"}}`,
		"non-http origin":                    `{"ftp://foundry":{"k":"v"}}`,
		"origin with userinfo":               `{"http://u@foundry:30000":{"k":"v"}}`,
		"trailing garbage":                   `{"http://foundry:30000":{"k":"v"}}x`,
	}
	for name, raw := range bad {
		if _, err := ParseLocalStorageSeed(raw); err == nil {
			t.Errorf("%s must be rejected: %s", name, raw)
		}
	}
}

func TestOriginOf(t *testing.T) {
	good := map[string]string{
		"http://foundry:30000":            "http://foundry:30000",
		"HTTP://Foundry:30000/game/join":  "http://foundry:30000",
		"https://vtt.example.com":         "https://vtt.example.com",
		" http://127.0.0.1:30000/?x=1#y ": "http://127.0.0.1:30000",
	}
	for in, want := range good {
		if got, err := OriginOf(in); err != nil || got != want {
			t.Errorf("OriginOf(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "foundry:30000", "ftp://x", "http://", "javascript:alert(1)"} {
		if _, err := OriginOf(in); err == nil {
			t.Errorf("OriginOf(%q) must fail", in)
		}
	}
	// The host is what follows the userinfo, so this URL's origin is evil.example, never foundry.
	if got, _ := OriginOf("http://foundry:30000@evil.example/"); got != "http://evil.example" {
		t.Errorf("userinfo must not be mistaken for the host, got %q", got)
	}
}
