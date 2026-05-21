package violet

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// OutputState mirrors myviolet.enums.OutputState (IntEnum).
// AUTO_OFF=0, AUTO_ON=1, AUTO_PRIO_OFF=2, AUTO_PRIO_ON=3, MANUAL_ON=4,
// EMERGENCY_OFF=5, MANUAL_OFF=6.
type OutputState int

// OutputState constants — see OutputState type doc for semantics.
const (
	OutputAutoOff      OutputState = 0
	OutputAutoOn       OutputState = 1
	OutputAutoPrioOff  OutputState = 2
	OutputAutoPrioOn   OutputState = 3
	OutputManualOn     OutputState = 4
	OutputEmergencyOff OutputState = 5
	OutputManualOff    OutputState = 6
)

// IsOn reports whether an output is energised regardless of cause.
func (s OutputState) IsOn() bool {
	return s == OutputAutoOn || s == OutputAutoPrioOn || s == OutputManualOn
}

// Cover state strings — match myviolet.enums.CoverState.
const (
	CoverOpening = "OPENING" // COVER_STATE while opening.
	CoverClosing = "CLOSING" // COVER_STATE while closing.
	CoverStopped = "STOPPED" // COVER_STATE after a manual stop.
)

// Control-key constants from myviolet.constants.
const (
	CoverOpenKey  = "COVER_OPEN"  // /setFunctionManually key to open the cover.
	CoverCloseKey = "COVER_CLOSE" // /setFunctionManually key to close the cover.
	CoverStopKey  = "COVER_STOP"  // /setFunctionManually key to stop the cover.
)

// validOutputPrefixes mirrors the Python mock's allow-list of control keys
// it accepts even when not present in the seed snapshot.
var validOutputPrefixes = []string{
	"PUMP", "HEATER", "SOLAR", "LIGHT", "REFILL", "ECO",
	"BACKWASH", "EXT1_", "EXT2_", "DOS_", "PVSURPLUS",
}

// ErrUnknownControlKey is returned when /setFunctionManually targets a key
// that is neither in the snapshot nor on the allow-list.
var ErrUnknownControlKey = errors.New("unknown control key")

// State is the violet mock's mutable state. snapshot is the readings map;
// config and dosingParameters are merged from /setConfig and
// /setDosingParameters writes. All maps are guarded by mu.
type State struct {
	mu               sync.RWMutex
	snapshot         map[string]any
	config           map[string]any
	dosingParameters map[string]any
	clock            func() time.Time
	epochStart       time.Time
}

// Config configures a new violet State.
type Config struct {
	// Seed is the readings snapshot; LoadSeed() is the canonical source.
	Seed map[string]any
	// Clock injects a time source for deterministic tests. nil → time.Now.
	Clock func() time.Time
}

// New builds a State from a seed snapshot.
func New(cfg Config) *State {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &State{
		snapshot:         dupMap(cfg.Seed),
		config:           map[string]any{},
		dosingParameters: map[string]any{},
		clock:            clock,
		epochStart:       clock(),
	}
}

// Snapshot returns a shallow copy of the current readings.
func (s *State) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dupMap(s.snapshot)
}

// ConfigSnapshot returns a shallow copy of the merged config.
func (s *State) ConfigSnapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dupMap(s.config)
}

// DosingParametersSnapshot returns a shallow copy of the merged dosing
// parameters.
func (s *State) DosingParametersSnapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return dupMap(s.dosingParameters)
}

// ElapsedSeconds returns time since New() was called using the injected clock.
func (s *State) ElapsedSeconds() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clock().Sub(s.epochStart).Seconds()
}

// ApplySetFunction maps a /setFunctionManually call onto state mutations,
// matching the Python mock's semantics in
// myviolet/tools/myviolet_mock/state.py.
func (s *State) ApplySetFunction(key, action string, duration, value int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch key {
	case CoverOpenKey:
		s.snapshot["COVER_STATE"] = CoverOpening
		return nil
	case CoverCloseKey:
		s.snapshot["COVER_STATE"] = CoverClosing
		return nil
	case CoverStopKey:
		s.snapshot["COVER_STATE"] = CoverStopped
		return nil
	}

	if strings.HasPrefix(key, "DMX_SCENE") {
		switch action {
		case "ON":
			s.snapshot[key] = int(OutputManualOn)
		case "OFF":
			s.snapshot[key] = int(OutputManualOff)
		case "AUTO":
			s.snapshot[key] = int(OutputAutoOff)
		}
		return nil
	}

	if _, ok := s.snapshot[key]; !ok && !hasValidPrefix(key) {
		return fmt.Errorf("%w: %s", ErrUnknownControlKey, key)
	}

	var newState OutputState
	switch action {
	case "ON":
		newState = OutputManualOn
	case "OFF":
		newState = OutputManualOff
	case "AUTO":
		newState = OutputAutoOff
	default:
		// Unhandled action — leave state untouched (PUSH, COLOR, etc.).
		return nil
	}

	now := s.clock().Unix()
	s.snapshot[key] = int(newState)
	if newState.IsOn() {
		s.snapshot[key+"_LAST_ON"] = now
	} else {
		s.snapshot[key+"_LAST_OFF"] = now
	}
	_ = duration
	_ = value
	return nil
}

// ApplySetTarget records a /setTargetValues call under "<target>_target".
func (s *State) ApplySetTarget(target string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot[target+"_target"] = value
}

// ApplySetConfig merges payload into the config map.
func (s *State) ApplySetConfig(payload map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range payload {
		s.config[k] = v
	}
}

// ApplySetDosingParameters merges payload into the dosing-parameter map.
func (s *State) ApplySetDosingParameters(payload map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range payload {
		s.dosingParameters[k] = v
	}
}

func hasValidPrefix(key string) bool {
	for _, p := range validOutputPrefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func dupMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
