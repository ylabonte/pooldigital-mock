package proconip

import "math"

// Drift constants — ported from
// proconip-pypi/tools/proconip_mock/drift.py. The model is
// `center + amplitude * sin(2π * t / period)`, deterministic and stateless.
const (
	phCenter        = 7.40
	phAmplitude     = 0.10
	phPeriodSeconds = 600.0

	redoxCenterMV     = 700.0
	redoxAmplitudeMV  = 25.0
	redoxPeriodSecond = 300.0

	cpuTempCenterC     = 30.0
	cpuTempAmplitudeC  = 2.0
	cpuTempPeriodSec   = 1800.0
	pumpTempCenterC    = 27.0
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
