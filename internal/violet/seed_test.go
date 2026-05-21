package violet

import (
	"encoding/json"
	"testing"
)

func TestLoadSeedDecodes(t *testing.T) {
	seed, err := LoadSeed()
	if err != nil {
		t.Fatalf("LoadSeed: %v", err)
	}
	if len(seed) == 0 {
		t.Fatal("seed is empty")
	}
	for _, key := range []string{"pH_value_min", "orp_value_min", "fw", "CONFIGCHANGEMARKER"} {
		if _, ok := seed[key]; !ok {
			t.Errorf("seed missing expected key %q", key)
		}
	}
}

func TestSeedBytesNonEmpty(t *testing.T) {
	if len(SeedBytes()) == 0 {
		t.Error("SeedBytes() is empty")
	}
	// quick well-formed JSON check
	var anything any
	if err := json.Unmarshal(SeedBytes(), &anything); err != nil {
		t.Errorf("embedded seed is not valid JSON: %v", err)
	}
}

func TestLoadSeedIndependentCopies(t *testing.T) {
	a, err := LoadSeed()
	if err != nil {
		t.Fatalf("LoadSeed (a): %v", err)
	}
	b, err := LoadSeed()
	if err != nil {
		t.Fatalf("LoadSeed (b): %v", err)
	}
	a["fw"] = "modified"
	if b["fw"] == "modified" {
		t.Error("LoadSeed should return independent maps")
	}
}
