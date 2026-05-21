package violet

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestApplyDriftAtZeroEqualsRoundedBaseline(t *testing.T) {
	snap := map[string]any{
		"pH_value":  7.30,
		"orp_value": 700.0,
		"pot_value": 0.30,
		"CPU_TEMP":  48.0,
		"other":     42.0,
	}
	ApplyDrift(snap, 0)
	if snap["pH_value"] != 7.30 {
		t.Errorf("pH at t=0 = %v, want 7.30", snap["pH_value"])
	}
	if snap["other"] != 42.0 {
		t.Errorf("non-sensor key should be untouched, got %v", snap["other"])
	}
}

func TestApplyDriftQuarterPeriodHitsPeak(t *testing.T) {
	snap := map[string]any{"pH_value": 7.30}
	// pH period = 300, amp = 0.05 → at t=75, sin(π/2)=1 → 7.30 + 0.05 = 7.35
	ApplyDrift(snap, 75)
	got := snap["pH_value"].(float64)
	if math.Abs(got-7.35) > 1e-9 {
		t.Errorf("pH at T/4 = %v, want ~7.35", got)
	}
}

func TestApplyDriftHandlesJSONNumber(t *testing.T) {
	var raw any
	dec := json.NewDecoder(strings.NewReader(`{"pH_value": 7.30}`))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	snap := raw.(map[string]any)
	ApplyDrift(snap, 0)
	got := snap["pH_value"].(float64)
	if math.Abs(got-7.30) > 1e-9 {
		t.Errorf("pH (json.Number path) at t=0 = %v, want 7.30", got)
	}
}

func TestApplyDriftSkipsMissingKeys(t *testing.T) {
	snap := map[string]any{"only_other": 1.0}
	ApplyDrift(snap, 100)
	if _, ok := snap["pH_value"]; ok {
		t.Error("ApplyDrift should not add keys that weren't present")
	}
}

func TestRoundTo3(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{0, 0},
		{7.3001, 7.3},
		{-0.123456, -0.123},
	}
	for _, c := range cases {
		got := roundTo3(c.in)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("roundTo3(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNumericToFloatTypes(t *testing.T) {
	if v, ok := numericToFloat(int(7)); !ok || v != 7 {
		t.Errorf("int: %v %v", v, ok)
	}
	if v, ok := numericToFloat(int64(8)); !ok || v != 8 {
		t.Errorf("int64: %v %v", v, ok)
	}
	if v, ok := numericToFloat(float32(1.5)); !ok || math.Abs(v-1.5) > 1e-6 {
		t.Errorf("float32: %v %v", v, ok)
	}
	if _, ok := numericToFloat("nope"); ok {
		t.Error("string should not be numeric")
	}
}
