package util

import (
	"math"
	"math/rand"
	"time"
)

func Clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
func Saturate(v float64) float64         { return Clamp(v, 0, 1) }
func Lerp(t, a, b float64) float64       { return (1-t)*a + t*b }
func LerpInvSat(t, a, b float64) float64 { return Saturate((t - a) / (b - a)) }
func SmoothStep(x float64) float64       { x = Saturate(x); return x * x * (3 - 2*x) }
func RoundToInt(v float64) int           { return int(math.Round(v)) }
func Random() float64                    { return rand.Float64() }
func RandomRange(a, b float64) float64   { return rand.Float64()*(b-a) + a }
func UnixTimestamp(t time.Time) int64    { return t.UTC().Unix() }
