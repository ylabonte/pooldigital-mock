package violet

import (
	"encoding/json"
	"math"
)

// driftProfile mirrors myviolet/tools/myviolet_mock/drift.py.
var driftProfile = []struct {
	key       string
	amplitude float64
	period    float64
}{
	{"pH_value", 0.05, 300.0},
	{"orp_value", 1.5, 240.0},
	{"pot_value", 0.02, 180.0},
	{"CPU_TEMP", 0.5, 600.0},
}

// ApplyDrift returns snapshot with sine-wave drift applied to known
// oscillating sensor fields. Non-sensor keys are left untouched. The map
// is mutated in place and returned.
func ApplyDrift(snapshot map[string]any, elapsedSeconds float64) map[string]any {
	for _, p := range driftProfile {
		v, ok := snapshot[p.key]
		if !ok {
			continue
		}
		baseline, ok := numericToFloat(v)
		if !ok {
			continue
		}
		delta := p.amplitude * math.Sin(2*math.Pi*elapsedSeconds/p.period)
		snapshot[p.key] = roundTo3(baseline + delta)
	}
	return snapshot
}

// numericToFloat coerces JSON-decoded numbers (json.Number when UseNumber
// is set, or float64 by default) into float64.
func numericToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// roundTo3 rounds to 3 decimal places — matches Python's round(x, 3).
func roundTo3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
