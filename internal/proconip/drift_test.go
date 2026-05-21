package proconip

import (
	"math"
	"testing"
)

func TestDriftAtZeroEqualsCenter(t *testing.T) {
	s := DriftSensors(0)
	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"pH", s.PH, phCenter},
		{"redox", s.RedoxMV, redoxCenterMV},
		{"cpu", s.CPUTempC, cpuTempCenterC},
		{"pump", s.PumpTempC, pumpTempCenterC},
	}
	for _, c := range cases {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Errorf("%s at t=0 = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestDriftQuarterPeriodHitsAmplitudePeak(t *testing.T) {
	cases := []struct {
		name   string
		period float64
		amp    float64
		center float64
		got    func(float64) float64
	}{
		{"pH", phPeriodSeconds, phAmplitude, phCenter, func(t float64) float64 { return DriftSensors(t).PH }},
		{"redox", redoxPeriodSecond, redoxAmplitudeMV, redoxCenterMV, func(t float64) float64 { return DriftSensors(t).RedoxMV }},
		{"cpu", cpuTempPeriodSec, cpuTempAmplitudeC, cpuTempCenterC, func(t float64) float64 { return DriftSensors(t).CPUTempC }},
		{"pump", pumpTempPeriodSec, pumpTempAmplitudeC, pumpTempCenterC, func(t float64) float64 { return DriftSensors(t).PumpTempC }},
	}
	for _, c := range cases {
		// sin(2π * (period/4) / period) = sin(π/2) = 1 → value = center + amp
		got := c.got(c.period / 4)
		want := c.center + c.amp
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("%s at T/4 = %v, want %v", c.name, got, want)
		}
	}
}

func TestDriftDeterministic(t *testing.T) {
	a := DriftSensors(123.456)
	b := DriftSensors(123.456)
	if a != b {
		t.Errorf("drift is not deterministic: %+v vs %+v", a, b)
	}
}

func TestPackedClockValue(t *testing.T) {
	cases := []struct {
		hour, minute, want int
	}{
		{0, 0, 0},
		{1, 0, 256},
		{12, 34, 12*256 + 34},
		{23, 59, 23*256 + 59},
	}
	for _, c := range cases {
		if got := PackedClockValue(c.hour, c.minute); got != c.want {
			t.Errorf("PackedClockValue(%d,%d) = %d, want %d", c.hour, c.minute, got, c.want)
		}
	}
}
