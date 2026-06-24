package dynamic

import "math"

func GetRoadTemperature(timeSeconds int, ambientTemperature, solarCoeff, nightCoeff float64) float64 {
	day := float64((timeSeconds%86400+86400)%86400) / 86400
	sun := math.Sin((day - 0.25) * 2 * math.Pi)
	if sun < 0 {
		sun = 0
	}
	// During the day: solar gain heats the road above ambient.
	// At night: clear skies radiate heat away, cooling the road below ambient.
	return ambientTemperature + 13*sun*solarCoeff - 3*(1-sun)*nightCoeff
}
