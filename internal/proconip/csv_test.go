package proconip

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoadTemplateFromEmbed(t *testing.T) {
	tpl, err := loadTemplate()
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}
	if len(tpl.sysinfo) == 0 || tpl.sysinfo[0] != "SYSINFO" {
		t.Errorf("sysinfo[0] = %q, want %q", safeIndex(tpl.sysinfo, 0), "SYSINFO")
	}
	if len(tpl.names) != len(tpl.offsets) || len(tpl.offsets) != len(tpl.gains) || len(tpl.gains) != len(tpl.rawValues) {
		t.Errorf("row width mismatch: names=%d offsets=%d gains=%d raw=%d",
			len(tpl.names), len(tpl.offsets), len(tpl.gains), len(tpl.rawValues))
	}
}

func TestRenderGetStateSubstitutesDriftedColumns(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 5, 21, 12, 34, 0, 0, time.UTC) }
	st := New(Config{Clock: clock})

	body, err := RenderGetState(st, time.Date(2026, 5, 21, 12, 34, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RenderGetState: %v", err)
	}

	rows := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(rows) != 6 {
		t.Fatalf("expected 6 rows, got %d:\n%s", len(rows), body)
	}
	if !strings.HasPrefix(rows[0], "SYSINFO,") {
		t.Errorf("row 0 missing SYSINFO prefix: %q", rows[0])
	}
	cols := strings.Split(rows[5], ",")
	wantClock := PackedClockValue(12, 34)
	if got, _ := strconv.Atoi(cols[0]); got != wantClock {
		t.Errorf("Time column = %s, want %d", cols[0], wantClock)
	}
	// pH column should be present and parseable; at t=0 it must equal raw-of-center.
	if _, err := strconv.ParseFloat(cols[colPH], 64); err != nil {
		t.Errorf("pH column %q not parseable: %v", cols[colPH], err)
	}
}

func TestRenderGetStateReflectsRelayState(t *testing.T) {
	st := New(Config{})
	// manual + on for bits 0 and 8 → column 16 (internal) and 28 (external) should read "3"
	if err := st.ApplyENA(0b1_0000_0001, 0b1_0000_0001); err != nil {
		t.Fatalf("ApplyENA: %v", err)
	}
	body, err := RenderGetState(st, time.Now())
	if err != nil {
		t.Fatalf("RenderGetState: %v", err)
	}
	rows := strings.Split(strings.TrimRight(body, "\n"), "\n")
	cols := strings.Split(rows[5], ",")
	if cols[16] != "3" {
		t.Errorf("internal relay column 16 = %q, want %q", cols[16], "3")
	}
	if cols[28] != "3" {
		t.Errorf("external relay column 28 = %q, want %q", cols[28], "3")
	}
}

func TestRenderGetStateConfigOtherEnable(t *testing.T) {
	st := New(Config{ConfigOtherEnable: 260})
	body, err := RenderGetState(st, time.Now())
	if err != nil {
		t.Fatalf("RenderGetState: %v", err)
	}
	rows := strings.Split(body, "\n")
	sysinfo := strings.Split(rows[0], ",")
	if sysinfo[5] != "260" {
		t.Errorf("SYSINFO[5] = %q, want %q", sysinfo[5], "260")
	}
}

func TestRenderGetDMXMatchesState(t *testing.T) {
	st := New(Config{})
	if err := st.ApplyDMX(
		[]int{0, 10, 20, 30, 40, 50, 60, 70},
		[]int{80, 90, 100, 110, 120, 130, 140, 150},
	); err != nil {
		t.Fatalf("ApplyDMX: %v", err)
	}
	got := RenderGetDMX(st)
	want := "0,10,20,30,40,50,60,70,80,90,100,110,120,130,140,150\n"
	if got != want {
		t.Errorf("RenderGetDMX = %q, want %q", got, want)
	}
}

func TestRenderGetDMXMatchesEmbeddedFixture(t *testing.T) {
	st := New(Config{})
	if err := st.ApplyDMX(
		[]int{0, 10, 20, 30, 40, 50, 60, 70},
		[]int{80, 90, 100, 110, 120, 130, 140, 150},
	); err != nil {
		t.Fatalf("ApplyDMX: %v", err)
	}
	got := strings.TrimRight(RenderGetDMX(st), "\n")
	want := strings.TrimRight(rawDMXFixture(), "\n")
	if got != want {
		t.Errorf("DMX render does not match get_dmx.csv fixture\nwant: %s\n got: %s", want, got)
	}
}

func TestFormatFloatIntegerShape(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{-44, "-44"},
		{529, "529"},
		{0.0625, "0.0625"},
		{0.1, "0.1"},
	}
	for _, c := range cases {
		if got := formatFloat(c.in); got != c.want {
			t.Errorf("formatFloat(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderGetStateSysinfoRowMatchesFixture(t *testing.T) {
	st := New(Config{})
	body, err := RenderGetState(st, time.Now())
	if err != nil {
		t.Fatalf("RenderGetState: %v", err)
	}
	rows := strings.Split(body, "\n")
	wantPrefix := "SYSINFO,1.7.3,9559698,1,3,0,257,4,4,5"
	if rows[0] != wantPrefix {
		t.Errorf("sysinfo row = %q, want %q", rows[0], wantPrefix)
	}
	if !strings.Contains(rows[1], "CPU Temp") || !strings.Contains(rows[1], "Redox") || !strings.Contains(rows[1], "pH") {
		t.Errorf("names row missing expected columns: %q", rows[1])
	}
}

func safeIndex(s []string, i int) string {
	if i < 0 || i >= len(s) {
		return ""
	}
	return s[i]
}
