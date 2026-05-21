package proconip

import (
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GetState column indexes (zero-based) that the renderer overrides.
const (
	colTime     = 0
	colCPUTemp  = 5
	colRedox    = 6
	colPH       = 7
	colPumpTemp = 8
)

// Internal relays (bits 0–7) → columns 16–23.
// External relays (bits 8–15) → columns 28–35.
const (
	internalRelayStart = 16
	internalRelayEnd   = 24 // exclusive
	externalRelayStart = 28
	externalRelayEnd   = 36 // exclusive
)

//go:embed get_state.csv
var getStateFixture string

//go:embed get_dmx.csv
var getDMXFixture string

// csvTemplate caches the parsed structural rows from get_state.csv so
// repeated renders don't re-parse the fixture.
type csvTemplate struct {
	sysinfo   []string
	names     []string
	units     []string
	offsets   []float64
	gains     []float64
	rawValues []float64
}

var (
	templateOnce sync.Once
	templateMu   sync.Mutex
	templateVal  csvTemplate
	templateErr  error
)

func loadTemplate() (csvTemplate, error) {
	templateOnce.Do(func() {
		templateMu.Lock()
		defer templateMu.Unlock()
		templateVal, templateErr = parseTemplate(getStateFixture)
	})
	return templateVal, templateErr
}

func parseTemplate(text string) (csvTemplate, error) {
	var t csvTemplate
	var lines []string
	for _, ln := range strings.Split(text, "\n") {
		if strings.TrimSpace(ln) != "" {
			lines = append(lines, ln)
		}
	}
	if len(lines) < 6 {
		return t, fmt.Errorf("get_state.csv fixture: expected 6 non-empty rows, got %d", len(lines))
	}
	t.sysinfo = strings.Split(lines[0], ",")
	t.names = strings.Split(lines[1], ",")
	t.units = strings.Split(lines[2], ",")
	var err error
	if t.offsets, err = parseFloatRow(lines[3]); err != nil {
		return t, fmt.Errorf("offsets row: %w", err)
	}
	if t.gains, err = parseFloatRow(lines[4]); err != nil {
		return t, fmt.Errorf("gains row: %w", err)
	}
	if t.rawValues, err = parseFloatRow(lines[5]); err != nil {
		return t, fmt.Errorf("raw row: %w", err)
	}
	return t, nil
}

func parseFloatRow(row string) ([]float64, error) {
	fields := strings.Split(row, ",")
	out := make([]float64, len(fields))
	for i, f := range fields {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("column %d (%q): %w", i, f, err)
		}
		out[i] = v
	}
	return out, nil
}

// naturalToRaw inverts (raw * gain) + offset so the parser recovers natural.
func naturalToRaw(natural, offset, gain float64) float64 {
	if gain == 0 {
		return 0
	}
	return (natural - offset) / gain
}

// RenderGetState builds the CSV body for GET /GetState.csv.
//
// wallClock is the local time used for column 0; passing the zero value
// uses the host's current time. Drift sensor values are computed from
// state.ElapsedSeconds().
func RenderGetState(state *State, wallClock time.Time) (string, error) {
	tpl, err := loadTemplate()
	if err != nil {
		return "", err
	}
	if wallClock.IsZero() {
		wallClock = time.Now()
	}
	sensors := DriftSensors(state.ElapsedSeconds())

	rawCols := append([]float64(nil), tpl.rawValues...)
	if colTime < len(rawCols) {
		rawCols[colTime] = float64(PackedClockValue(wallClock.Hour(), wallClock.Minute()))
	}
	if colCPUTemp < len(rawCols) {
		rawCols[colCPUTemp] = naturalToRaw(sensors.CPUTempC, tpl.offsets[colCPUTemp], tpl.gains[colCPUTemp])
	}
	if colRedox < len(rawCols) {
		rawCols[colRedox] = naturalToRaw(sensors.RedoxMV, tpl.offsets[colRedox], tpl.gains[colRedox])
	}
	if colPH < len(rawCols) {
		rawCols[colPH] = naturalToRaw(sensors.PH, tpl.offsets[colPH], tpl.gains[colPH])
	}
	if colPumpTemp < len(rawCols) {
		rawCols[colPumpTemp] = naturalToRaw(sensors.PumpTempC, tpl.offsets[colPumpTemp], tpl.gains[colPumpTemp])
	}

	for col := internalRelayStart; col < internalRelayEnd && col < len(rawCols); col++ {
		rawCols[col] = float64(state.CSVRelayValue(col - internalRelayStart))
	}
	for col := externalRelayStart; col < externalRelayEnd && col < len(rawCols); col++ {
		rawCols[col] = float64(state.CSVRelayValue(col - externalRelayStart + 8))
	}

	sysinfo := append([]string(nil), tpl.sysinfo...)
	if len(sysinfo) > 5 {
		sysinfo[5] = strconv.Itoa(state.ConfigOtherEnable())
	}

	var b strings.Builder
	b.WriteString(strings.Join(sysinfo, ","))
	b.WriteByte('\n')
	b.WriteString(strings.Join(tpl.names, ","))
	b.WriteByte('\n')
	b.WriteString(strings.Join(tpl.units, ","))
	b.WriteByte('\n')
	b.WriteString(joinFloatRow(tpl.offsets))
	b.WriteByte('\n')
	b.WriteString(joinFloatRow(tpl.gains))
	b.WriteByte('\n')
	b.WriteString(joinFloatRow(rawCols))
	b.WriteByte('\n')
	return b.String(), nil
}

// RenderGetDMX builds the single-line CSV body for GET /GetDmx.csv.
func RenderGetDMX(state *State) string {
	dmx := state.CopyDMX()
	parts := make([]string, len(dmx))
	for i, v := range dmx {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, ",") + "\n"
}

func joinFloatRow(values []float64) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = formatFloat(v)
	}
	return strings.Join(parts, ",")
}

// formatFloat renders a float in the same shape the Python mock produces:
// integers stay integer-shaped; everything else uses Python's repr-style
// shortest-round-trip representation.
func formatFloat(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	// Match Python's repr(): shortest decimal that round-trips.
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// rawDMXFixture exposes the dmx fixture for golden tests.
func rawDMXFixture() string { return getDMXFixture }
