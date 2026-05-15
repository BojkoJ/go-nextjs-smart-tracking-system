package main

import (
	"math"
	"testing"
	"time"
)

// --- haversine ---

func TestHaversine_SamePoint_ReturnsZero(t *testing.T) {
	p := Waypoint{Latitude: 35.10, Longitude: 129.04}
	if dist := haversine(p, p); dist != 0.0 {
		t.Errorf("same point should return 0 km, got %v", dist)
	}
}

func TestHaversine_QuarterEquator_CorrectDistance(t *testing.T) {
	// (0°N,0°E) to (0°N,90°E) = quarter of Earth's equatorial circumference = π/2 * R
	a := Waypoint{Latitude: 0, Longitude: 0}
	b := Waypoint{Latitude: 0, Longitude: 90}
	got := haversine(a, b)
	want := math.Pi / 2.0 * 6371.0 // ≈ 10007.5 km
	if math.Abs(got-want) > 1.0 {
		t.Errorf("quarter equator: want ~%.1f km, got %.1f km", want, got)
	}
}

func TestHaversine_EquatorToNorthPole(t *testing.T) {
	equator := Waypoint{Latitude: 0, Longitude: 0}
	pole := Waypoint{Latitude: 90, Longitude: 0}
	got := haversine(equator, pole)
	want := math.Pi / 2.0 * 6371.0 // same as quarter equator — great circle property
	if math.Abs(got-want) > 1.0 {
		t.Errorf("equator to north pole: want ~%.1f km, got %.1f km", want, got)
	}
}

func TestHaversine_IsSymmetric(t *testing.T) {
	a := Waypoint{Latitude: 35.10, Longitude: 129.04}
	b := Waypoint{Latitude: 51.98, Longitude: 4.05}
	if haversine(a, b) != haversine(b, a) {
		t.Error("haversine must be symmetric: d(a,b) == d(b,a)")
	}
}

func TestHaversine_ReturnsPositiveDistance(t *testing.T) {
	a := Waypoint{Latitude: 35.10, Longitude: 129.04}
	b := Waypoint{Latitude: 51.98, Longitude: 4.05}
	if dist := haversine(a, b); dist <= 0 {
		t.Errorf("expected positive distance, got %v", dist)
	}
}

func TestHaversine_BusanRotterdam_WithinGreatCircleRange(t *testing.T) {
	// Great-circle (not sea route): expected ~9100–9500 km
	busan := Waypoint{Latitude: 35.10, Longitude: 129.04}
	rotterdam := Waypoint{Latitude: 51.98, Longitude: 4.05}
	dist := haversine(busan, rotterdam)
	if dist < 8000 || dist > 11000 {
		t.Errorf("Busan–Rotterdam great-circle out of expected range: %.1f km", dist)
	}
}

func TestHaversine_InitComputedTotalRouteLength_IsPositive(t *testing.T) {
	// init() is called automatically; verify it populated the globals
	if totalRouteLength <= 0 {
		t.Errorf("totalRouteLength should be > 0, got %v", totalRouteLength)
	}
	if len(segmentLengths) != len(routeWaypoints)-1 {
		t.Errorf("expected %d segment lengths, got %d", len(routeWaypoints)-1, len(segmentLengths))
	}
}

func TestHaversine_SegmentLengthsArePositive(t *testing.T) {
	for i, l := range segmentLengths {
		if l <= 0 {
			t.Errorf("segment %d length must be > 0, got %v", i, l)
		}
	}
}

// --- computeShipPosition ---

func TestComputeShipPosition_AtStart_NearFirstWaypoint(t *testing.T) {
	lat, lon := computeShipPosition(time.Now())
	first := routeWaypoints[0] // Busan: 35.10, 129.04
	if math.Abs(lat-first.Latitude) > 0.5 || math.Abs(lon-first.Longitude) > 0.5 {
		t.Errorf("at t≈0 expected near (%v,%v), got (%v,%v)", first.Latitude, first.Longitude, lat, lon)
	}
}

func TestComputeShipPosition_FarFuture_AtLastWaypoint(t *testing.T) {
	// 1 year ago → progress clamped to 1.0 → Rotterdam
	lat, lon := computeShipPosition(time.Now().Add(-365 * 24 * time.Hour))
	last := routeWaypoints[len(routeWaypoints)-1] // Rotterdam: 51.98, 4.05
	if math.Abs(lat-last.Latitude) > 0.01 || math.Abs(lon-last.Longitude) > 0.01 {
		t.Errorf("at far future expected (%v,%v), got (%v,%v)", last.Latitude, last.Longitude, lat, lon)
	}
}

func TestComputeShipPosition_AlwaysWithinGeographicBounds(t *testing.T) {
	offsets := []time.Duration{0, 1 * time.Hour, 12 * time.Hour, 72 * time.Hour, 1000 * 24 * time.Hour}
	for _, offset := range offsets {
		lat, lon := computeShipPosition(time.Now().Add(-offset))
		if lat < -90 || lat > 90 {
			t.Errorf("latitude %v out of [-90, 90]", lat)
		}
		if lon < -180 || lon > 180 {
			t.Errorf("longitude %v out of [-180, 180]", lon)
		}
	}
}

func TestComputeShipPosition_ProgressesForwardInTime(t *testing.T) {
	// Route goes south first (Busan→South China Sea), so latitude is NOT a monotonic proxy.
	// Instead verify: position at t=2h is farther from Busan than position at t=1h.
	lat1, lon1 := computeShipPosition(time.Now().Add(-1 * time.Hour))
	lat2, lon2 := computeShipPosition(time.Now().Add(-2 * time.Hour))
	busan := routeWaypoints[0]
	dist1 := haversine(Waypoint{Latitude: lat1, Longitude: lon1}, busan)
	dist2 := haversine(Waypoint{Latitude: lat2, Longitude: lon2}, busan)
	if dist2 <= dist1 {
		t.Errorf("position at 2h should be farther from Busan than at 1h: dist1=%.1fkm dist2=%.1fkm", dist1, dist2)
	}
}

// --- computeTemperature ---

func TestComputeTemperature_WithinAbsoluteTheoreticalBounds(t *testing.T) {
	// base ∈ [14, 26], noise ∈ [-0.8, +0.8], offset = 0 → result ∈ [13.2, 26.8]
	// use loose tolerance to be time-of-day independent
	for i := 0; i < 1000; i++ {
		temp := computeTemperature(0.0)
		if temp < 13.0 || temp > 27.0 {
			t.Errorf("temperature %v out of theoretical bounds [13.0, 27.0]", temp)
		}
	}
}

func TestComputeTemperature_PositiveOffsetIncreasesAverage(t *testing.T) {
	const n = 2000
	var sumHigh, sumLow float64
	for i := 0; i < n; i++ {
		sumHigh += computeTemperature(+2.0)
		sumLow += computeTemperature(-2.0)
	}
	if sumHigh <= sumLow {
		t.Errorf("positive offset should raise average temperature: sumHigh=%v sumLow=%v", sumHigh, sumLow)
	}
}

func TestComputeTemperature_WithMaxOffset_StillWithinBounds(t *testing.T) {
	// max runtime offset from ContainerState is +2.0
	for i := 0; i < 1000; i++ {
		temp := computeTemperature(+2.0)
		if temp < 11.0 || temp > 29.0 {
			t.Errorf("temperature %v out of bounds with +2.0 offset", temp)
		}
	}
}

// --- computeHumidity ---

func TestComputeHumidity_AlwaysWithinClampedRange(t *testing.T) {
	startTime := time.Now()
	// offsets well beyond the runtime range to stress-test clamping
	for _, offset := range []float64{-20.0, -3.0, 0.0, +3.0, +20.0} {
		for i := 0; i < 500; i++ {
			h := computeHumidity(startTime, offset)
			if h < 20.0 || h > 60.0 {
				t.Errorf("humidity %v out of clamped range [20, 60] with offset %v", h, offset)
			}
		}
	}
}

func TestComputeHumidity_PositiveOffsetIncreasesAverage(t *testing.T) {
	startTime := time.Now()
	const n = 2000
	var sumHigh, sumLow float64
	for i := 0; i < n; i++ {
		sumHigh += computeHumidity(startTime, +3.0)
		sumLow += computeHumidity(startTime, -3.0)
	}
	if sumHigh <= sumLow {
		t.Errorf("positive offset should raise average humidity: sumHigh=%v sumLow=%v", sumHigh, sumLow)
	}
}

func TestComputeHumidity_BaseOscillatesWithTime(t *testing.T) {
	// elapsed=3h → sin(2π*3/12) = sin(π/2) = +1 → base = 40
	// elapsed=9h → sin(2π*9/12) = sin(3π/2) = -1 → base = 30
	// 6h apart gives sin(0)=sin(π)=0 — identical, so we use 3h vs 9h.
	now := time.Now()
	startAt3h := now.Add(-3 * time.Hour)
	startAt9h := now.Add(-9 * time.Hour)
	const n = 500
	var sum3h, sum9h float64
	for i := 0; i < n; i++ {
		sum3h += computeHumidity(startAt3h, 0.0)
		sum9h += computeHumidity(startAt9h, 0.0)
	}
	// avg at 3h ≈ 40, avg at 9h ≈ 30 — difference must exceed noise
	if math.Abs(sum3h-sum9h)/n < 1.0 {
		t.Errorf("humidity averages at 3h and 9h elapsed should differ by >1%%: avg3h=%.2f avg9h=%.2f", sum3h/n, sum9h/n)
	}
}
