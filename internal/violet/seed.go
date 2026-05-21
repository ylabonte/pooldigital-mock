// Package violet implements the in-memory Violet mock controller.
package violet

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed seed/getReadings_seed.json
var seedJSON []byte

// LoadSeed returns a fresh copy of the embedded /getReadings snapshot.
// Each call decodes the JSON anew so callers can mutate freely.
func LoadSeed() (map[string]any, error) {
	var out map[string]any
	dec := json.NewDecoder(newBytesReader(seedJSON))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("decoding embedded seed: %w", err)
	}
	return out, nil
}

// SeedBytes returns the raw embedded JSON. Used by tests to assert
// embedding worked without re-decoding.
func SeedBytes() []byte { return seedJSON }
