package dynamic

import "math"

func GetRoadTemperature(timeSeconds int, ambientTemperature, coefficient float64) float64 {
	day := float64((timeSeconds%86400+86400)%86400) / 86400
	sun := math.Sin((day - 0.25) * 2 * math.Pi)
	if sun < 0 {
		sun = 0
	}
	return ambientTemperature + (3+10*sun)*coefficient
}
