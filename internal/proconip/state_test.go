package proconip

import (
	"testing"
	"time"
)

func newFixedClockState(t *testing.T) (*State, func(d time.Duration)) {
	t.Helper()
	now := time.Unix(1_700_000_000, 0).UTC()
	st := New(Config{Clock: func() time.Time { return now }})
	advance := func(d time.Duration) {
		now = now.Add(d)
	}
	return st, advance
}

func TestNewElapsedSecondsStartsAtZero(t *testing.T) {
	st, _ := newFixedClockState(t)
	if got := st.ElapsedSeconds(); got != 0 {
		t.Errorf("ElapsedSeconds at t0 = %v, want 0", got)
	}
}

func TestElapsedSecondsAdvances(t *testing.T) {
	st, advance := newFixedClockState(t)
	advance(90 * time.Second)
	if got := st.ElapsedSeconds(); got != 90 {
		t.Errorf("ElapsedSeconds = %v, want 90", got)
	}
}

func TestApplyENAEncodesRelayValues(t *testing.T) {
	st := New(Config{})
	// enable bits 0 & 3, all on
	if err := st.ApplyENA(0b1001, 0b1111); err != nil {
		t.Fatalf("ApplyENA: %v", err)
	}
	cases := []struct {
		bit  int
		want int
	}{
		{0, 3}, // manual on
		{1, 1}, // auto on
		{2, 1}, // auto on
		{3, 3}, // manual on
		{4, 0},
	}
	for _, c := range cases {
		if got := st.CSVRelayValue(c.bit); got != c.want {
			t.Errorf("CSVRelayValue(%d) = %d, want %d", c.bit, got, c.want)
		}
	}
}

func TestApplyENARejectsNegative(t *testing.T) {
	st := New(Config{})
	if err := st.ApplyENA(-1, 0); err == nil {
		t.Error("expected error on negative enable mask")
	}
	if err := st.ApplyENA(0, -1); err == nil {
		t.Error("expected error on negative on mask")
	}
}

func TestApplyENADropsBitsAbove16(t *testing.T) {
	st := New(Config{})
	// Bits 0–15 + bit 16 (should be dropped)
	if err := st.ApplyENA(0x1FFFF, 0x1FFFF); err != nil {
		t.Fatalf("ApplyENA: %v", err)
	}
	// All 16 representable bits should be manual+on (value 3)
	for bit := 0; bit < NumRelayBits; bit++ {
		if got := st.CSVRelayValue(bit); got != 3 {
			t.Errorf("CSVRelayValue(%d) = %d, want 3", bit, got)
		}
	}
}

func TestApplyDMXClampsAndAssigns(t *testing.T) {
	st := New(Config{})
	low := []int{0, 50, 100, 200, 255, 300, -10, 7}
	high := []int{1, 2, 3, 4, 5, 6, 7, 8}
	if err := st.ApplyDMX(low, high); err != nil {
		t.Fatalf("ApplyDMX: %v", err)
	}
	got := st.CopyDMX()
	want := [NumDMXChannels]uint8{0, 50, 100, 200, 255, 255, 0, 7, 1, 2, 3, 4, 5, 6, 7, 8}
	if got != want {
		t.Errorf("dmx = %v, want %v", got, want)
	}
}

func TestApplyDMXLenMismatch(t *testing.T) {
	st := New(Config{})
	if err := st.ApplyDMX([]int{1, 2}, []int{1, 2, 3, 4, 5, 6, 7, 8}); err == nil {
		t.Error("expected error on short low slice")
	}
}

func TestCSVRelayValueOutOfRange(t *testing.T) {
	st := New(Config{})
	if got := st.CSVRelayValue(-1); got != 0 {
		t.Errorf("negative bit should return 0, got %d", got)
	}
	if got := st.CSVRelayValue(NumRelayBits); got != 0 {
		t.Errorf("bit at width should return 0, got %d", got)
	}
}
