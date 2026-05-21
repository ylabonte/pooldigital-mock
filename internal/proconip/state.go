// Package proconip implements the in-memory ProCon.IP mock controller.
package proconip

import (
	"fmt"
	"sync"
	"time"
)

const (
	// NumRelayBits is the width of the relay enable/output bitmaps (internal
	// relays in bits 0–7, external relays in bits 8–15).
	NumRelayBits = 16
	// NumDMXChannels is the number of DMX channels the mock tracks.
	NumDMXChannels = 16
)

// Clock returns a monotonic-style time source. tests inject a fake.
type Clock func() time.Time

// State is the mutable runtime state of the mock controller. Reads are
// guarded by a RWMutex; mutations happen behind the write lock so the
// CSV renderer always observes a consistent snapshot.
type State struct {
	mu sync.RWMutex

	relayEnabled [NumRelayBits]bool
	relayOn      [NumRelayBits]bool
	dmx          [NumDMXChannels]uint8

	// ConfigOtherEnable maps to SYSINFO[5] in the CSV. The bit layout (from
	// proconip.definitions): TCP/IP boost, SD logging, DMX, Avatar, relay
	// extension, high-bus, flow sensor, repeated mails, DMX extension.
	configOtherEnable int

	clock Clock
	start time.Time
}

// Config configures a new State.
type Config struct {
	// ConfigOtherEnable seeds SYSINFO[5]; default 0 mirrors the Python mock.
	ConfigOtherEnable int
	// Clock injects a clock for deterministic tests. nil → time.Now.
	Clock Clock
}

// New builds a State with the given config.
func New(cfg Config) *State {
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	return &State{
		configOtherEnable: cfg.ConfigOtherEnable,
		clock:             clock,
		start:             clock(),
	}
}

// ElapsedSeconds returns the time in seconds since the State was created,
// using its injected clock.
func (s *State) ElapsedSeconds() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clock().Sub(s.start).Seconds()
}

// ConfigOtherEnable returns the bitfield rendered into SYSINFO[5].
func (s *State) ConfigOtherEnable() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configOtherEnable
}

// CopyDMX returns a copy of the DMX channels.
func (s *State) CopyDMX() [NumDMXChannels]uint8 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dmx
}

// CSVRelayValue encodes one relay bit as the 0–3 value the controller
// emits: 0=Auto-off, 1=Auto-on, 2=Manual-off, 3=Manual-on.
func (s *State) CSVRelayValue(bit int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if bit < 0 || bit >= NumRelayBits {
		return 0
	}
	v := 0
	if s.relayEnabled[bit] {
		v |= 2
	}
	if s.relayOn[bit] {
		v |= 1
	}
	return v
}

// ApplyENA applies an ENA=enable,on payload from /usrcfg.cgi. Both masks
// are interpreted as unsigned 16-bit values; higher bits are dropped.
func (s *State) ApplyENA(enableMask, onMask int) error {
	if enableMask < 0 || onMask < 0 {
		return fmt.Errorf("ENA masks must be non-negative; got enableMask=%d onMask=%d", enableMask, onMask)
	}
	valid := (1 << NumRelayBits) - 1
	enableMask &= valid
	onMask &= valid

	s.mu.Lock()
	defer s.mu.Unlock()
	for bit := 0; bit < NumRelayBits; bit++ {
		mask := 1 << bit
		s.relayEnabled[bit] = enableMask&mask != 0
		s.relayOn[bit] = onMask&mask != 0
	}
	return nil
}

// ApplyDMX applies an 8+8 DMX payload from /usrcfg.cgi. Values are clamped
// to [0, 255]; len mismatches return an error.
func (s *State) ApplyDMX(low, high []int) error {
	if len(low) != 8 || len(high) != 8 {
		return fmt.Errorf("DMX payload must contain 8+8 values; got %d+%d", len(low), len(high))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, v := range low {
		s.dmx[i] = clampDMX(v)
	}
	for i, v := range high {
		s.dmx[8+i] = clampDMX(v)
	}
	return nil
}

func clampDMX(v int) uint8 {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return uint8(v)
	}
}
