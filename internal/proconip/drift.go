package proconip

import "math"

// Drift constants — `center + amplitude * sin(2π * t / period)`,
// deterministic and stateless. Centers are aligned with the Violet seed
// (internal/violet/seed/getReadings_seed.json) so a single test that hits
// both mocks sees consistent values across the two controllers.
const (
	phCenter        = 7.31 // Violet pH_value_min/max = 7.30/7.32
	phAmplitude     = 0.05
	phPeriodSeconds = 600.0

	redoxCenterMV     = 787.0 // Violet orp_value_min/max = 787.2/787.6
	redoxAmplitudeMV  = 15.0
	redoxPeriodSecond = 300.0

	cpuTempCenterC     = 48.0 // Violet SYSTEM_cpu_temperature = 48.2
	cpuTempAmplitudeC  = 2.0
	cpuTempPeriodSec   = 1800.0
	pumpTempCenterC    = 30.0 // Violet onewire1_value = 30.2 (pool water)
	pumpTempAmplitudeC = 1.0
	pumpTempPeriodSec  = 1800.0
)

// Sensors holds drifted readings in natural units (pH, mV, °C).
type Sensors struct {
	PH        float64
	RedoxMV   float64
	CPUTempC  float64
	PumpTempC float64
}

// DriftSensors returns the drifted sensor values for the given elapsed time.
func DriftSensors(elapsedSeconds float64) Sensors {
	return Sensors{
		PH:        ph(elapsedSeconds),
		RedoxMV:   redoxMV(elapsedSeconds),
		CPUTempC:  cpuTempC(elapsedSeconds),
		PumpTempC: pumpTempC(elapsedSeconds),
	}
}

func phase(elapsed, period float64) float64 {
	return math.Sin(2.0 * math.Pi * elapsed / period)
}

func ph(t float64) float64 {
	return phCenter + phAmplitude*phase(t, phPeriodSeconds)
}

func redoxMV(t float64) float64 {
	return redoxCenterMV + redoxAmplitudeMV*phase(t, redoxPeriodSecond)
}

func cpuTempC(t float64) float64 {
	return cpuTempCenterC + cpuTempAmplitudeC*phase(t, cpuTempPeriodSec)
}

func pumpTempC(t float64) float64 {
	return pumpTempCenterC + pumpTempAmplitudeC*phase(t, pumpTempPeriodSec)
}

// PackedClockValue encodes a wall-clock HH:MM as the controller's hour*256+minute.
func PackedClockValue(hour, minute int) int {
	return hour*256 + minute
}
