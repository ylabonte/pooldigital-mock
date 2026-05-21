package violet

import (
	"errors"
	"testing"
	"time"
)

func newTestState(t *testing.T) *State {
	t.Helper()
	seed, err := LoadSeed()
	if err != nil {
		t.Fatalf("LoadSeed: %v", err)
	}
	fixed := time.Unix(1_700_000_000, 0).UTC()
	return New(Config{Seed: seed, Clock: func() time.Time { return fixed }})
}

func TestSnapshotIsACopy(t *testing.T) {
	st := newTestState(t)
	a := st.Snapshot()
	a["pH_value"] = 999.0
	b := st.Snapshot()
	if b["pH_value"] == 999.0 {
		t.Error("Snapshot should return an independent copy")
	}
}

func TestApplySetFunctionOnRelay(t *testing.T) {
	st := newTestState(t)
	// PUMP isn't necessarily in the seed but matches a valid prefix.
	if err := st.ApplySetFunction("PUMP", "ON", 0, 0); err != nil {
		t.Fatalf("ApplySetFunction: %v", err)
	}
	got := st.Snapshot()
	if got["PUMP"] != int(OutputManualOn) {
		t.Errorf("PUMP = %v, want %d", got["PUMP"], OutputManualOn)
	}
	if _, ok := got["PUMP_LAST_ON"]; !ok {
		t.Error("PUMP_LAST_ON should be recorded on turn-on")
	}
}

func TestApplySetFunctionOffRecordsLastOff(t *testing.T) {
	st := newTestState(t)
	if err := st.ApplySetFunction("HEATER", "OFF", 0, 0); err != nil {
		t.Fatalf("ApplySetFunction: %v", err)
	}
	got := st.Snapshot()
	if got["HEATER"] != int(OutputManualOff) {
		t.Errorf("HEATER = %v, want %d", got["HEATER"], OutputManualOff)
	}
	if _, ok := got["HEATER_LAST_OFF"]; !ok {
		t.Error("HEATER_LAST_OFF should be recorded on turn-off")
	}
}

func TestApplySetFunctionAuto(t *testing.T) {
	st := newTestState(t)
	if err := st.ApplySetFunction("LIGHT", "AUTO", 0, 0); err != nil {
		t.Fatalf("ApplySetFunction: %v", err)
	}
	got := st.Snapshot()
	if got["LIGHT"] != int(OutputAutoOff) {
		t.Errorf("LIGHT = %v, want %d", got["LIGHT"], OutputAutoOff)
	}
}

func TestApplySetFunctionCover(t *testing.T) {
	st := newTestState(t)
	cases := []struct {
		key      string
		expected string
	}{
		{CoverOpenKey, CoverOpening},
		{CoverCloseKey, CoverClosing},
		{CoverStopKey, CoverStopped},
	}
	for _, c := range cases {
		if err := st.ApplySetFunction(c.key, "", 0, 0); err != nil {
			t.Fatalf("ApplySetFunction(%s): %v", c.key, err)
		}
		if got := st.Snapshot()["COVER_STATE"]; got != c.expected {
			t.Errorf("after %s, COVER_STATE = %v, want %q", c.key, got, c.expected)
		}
	}
}

func TestApplySetFunctionDMXScene(t *testing.T) {
	st := newTestState(t)
	cases := []struct {
		action string
		want   int
	}{
		{"ON", int(OutputManualOn)},
		{"OFF", int(OutputManualOff)},
		{"AUTO", int(OutputAutoOff)},
	}
	for _, c := range cases {
		if err := st.ApplySetFunction("DMX_SCENE1", c.action, 0, 0); err != nil {
			t.Fatalf("ApplySetFunction: %v", err)
		}
		if got := st.Snapshot()["DMX_SCENE1"]; got != c.want {
			t.Errorf("DMX_SCENE1 after %s = %v, want %d", c.action, got, c.want)
		}
	}
}

func TestApplySetFunctionUnknownKey(t *testing.T) {
	st := newTestState(t)
	err := st.ApplySetFunction("RANDOM_THING", "ON", 0, 0)
	if !errors.Is(err, ErrUnknownControlKey) {
		t.Errorf("expected ErrUnknownControlKey, got %v", err)
	}
}

func TestApplySetFunctionUnhandledAction(t *testing.T) {
	st := newTestState(t)
	before := st.Snapshot()["LIGHT"]
	if err := st.ApplySetFunction("LIGHT", "COLOR", 0, 0); err != nil {
		t.Fatalf("ApplySetFunction: %v", err)
	}
	if got := st.Snapshot()["LIGHT"]; got != before {
		t.Errorf("LIGHT changed by unhandled COLOR action: before=%v after=%v", before, got)
	}
}

func TestApplySetTarget(t *testing.T) {
	st := newTestState(t)
	st.ApplySetTarget("pH", 7.2)
	if got := st.Snapshot()["pH_target"]; got != 7.2 {
		t.Errorf("pH_target = %v, want 7.2", got)
	}
}

func TestApplySetConfigMerges(t *testing.T) {
	st := newTestState(t)
	st.ApplySetConfig(map[string]any{"a": 1, "b": "two"})
	st.ApplySetConfig(map[string]any{"a": 99})
	got := st.ConfigSnapshot()
	if got["a"] != 99 || got["b"] != "two" {
		t.Errorf("merge wrong: %v", got)
	}
}

func TestApplySetDosingParametersMerges(t *testing.T) {
	st := newTestState(t)
	st.ApplySetDosingParameters(map[string]any{"x": 1.5})
	if got := st.DosingParametersSnapshot()["x"]; got != 1.5 {
		t.Errorf("dosing x = %v, want 1.5", got)
	}
}

func TestOutputStateIsOn(t *testing.T) {
	cases := []struct {
		s    OutputState
		want bool
	}{
		{OutputAutoOff, false},
		{OutputAutoOn, true},
		{OutputAutoPrioOff, false},
		{OutputAutoPrioOn, true},
		{OutputManualOn, true},
		{OutputEmergencyOff, false},
		{OutputManualOff, false},
	}
	for _, c := range cases {
		if got := c.s.IsOn(); got != c.want {
			t.Errorf("OutputState(%d).IsOn() = %v, want %v", c.s, got, c.want)
		}
	}
}
