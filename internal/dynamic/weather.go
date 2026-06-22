package dynamic

import (
	"plugin-dynamic-conditions/internal/config"
	"plugin-dynamic-conditions/internal/util"
)

type chance struct {
	w float64
	t WeatherType
}

var regular = []chance{{0.4, Clear}, {0.2, FewClouds}, {0.1, ScatteredClouds}, {0.1, BrokenClouds}, {0.2, OvercastClouds}, {0.1, Fog}, {0.1, Mist}, {0.2, Windy}}
var rain = []chance{{0.1, LightDrizzle}, {0.2, Drizzle}, {0.2, HeavyDrizzle}, {0.2, LightRain}, {0.3, Rain}, {0.1, HeavyRain}}
var thunder = []chance{{0.6, LightThunderstorm}, {0.3, Thunderstorm}, {0.1, HeavyThunderstorm}}
var neighbours = map[WeatherType][]WeatherType{}

func init() {
	add := func(a, b WeatherType) {
		neighbours[a] = append(neighbours[a], b)
		neighbours[b] = append(neighbours[b], a)
	}
	add(Clear, FewClouds)
	add(Clear, ScatteredClouds)
	add(Clear, Mist)
	add(FewClouds, ScatteredClouds)
	add(FewClouds, BrokenClouds)
	add(FewClouds, Mist)
	add(FewClouds, Windy)
	add(FewClouds, LightDrizzle)
	add(ScatteredClouds, BrokenClouds)
	add(ScatteredClouds, OvercastClouds)
	add(ScatteredClouds, Mist)
	add(ScatteredClouds, Windy)
	add(ScatteredClouds, Drizzle)
	add(ScatteredClouds, LightRain)
	add(BrokenClouds, OvercastClouds)
	add(BrokenClouds, Mist)
	add(BrokenClouds, Windy)
	add(BrokenClouds, Drizzle)
	add(BrokenClouds, Rain)
	add(OvercastClouds, Mist)
	add(OvercastClouds, Fog)
	add(OvercastClouds, Windy)
	add(OvercastClouds, HeavyDrizzle)
	add(OvercastClouds, Rain)
	add(Fog, Mist)
	add(Fog, BrokenClouds)
	add(Fog, ScatteredClouds)
	add(Fog, Rain)
	add(Mist, LightDrizzle)
	add(Mist, LightRain)
	add(Mist, Windy)
	add(Windy, Drizzle)
	add(Windy, LightRain)
	add(Windy, Rain)
	add(LightDrizzle, Drizzle)
	add(LightDrizzle, HeavyDrizzle)
	add(LightDrizzle, LightRain)
	add(LightDrizzle, Rain)
	add(Drizzle, HeavyDrizzle)
	add(Drizzle, LightRain)
	add(Drizzle, Rain)
	add(HeavyDrizzle, LightRain)
	add(HeavyDrizzle, Rain)
	add(HeavyDrizzle, HeavyRain)
	add(HeavyDrizzle, LightThunderstorm)
	add(LightRain, Rain)
	add(LightRain, LightThunderstorm)
	add(Rain, HeavyRain)
	add(Rain, Thunderstorm)
	add(HeavyRain, Thunderstorm)
	add(HeavyRain, HeavyThunderstorm)
	add(LightThunderstorm, Thunderstorm)
	add(Thunderstorm, HeavyThunderstorm)
	for k := range neighbours {
		neighbours[k] = append(neighbours[k], k)
	}
}
func contains(a []WeatherType, t WeatherType) bool {
	for _, x := range a {
		if x == t {
			return true
		}
	}
	return false
}
func pick(table []chance, allowed []WeatherType) (WeatherType, bool) {
	sum := 0.0
	filtered := make([]chance, 0, len(table))
	for _, x := range table {
		if allowed == nil || contains(allowed, x.t) {
			filtered = append(filtered, x)
			sum += x.w
		}
	}
	if len(filtered) == 0 {
		return Clear, false
	}
	c := util.Random() * sum
	for _, x := range filtered {
		if c < x.w {
			return x.t, true
		}
		c -= x.w
	}
	return filtered[0].t, true
}
func generateRandomType(current WeatherType, p config.LiveConditionParams) WeatherType {
	has := current != WeatherNone
	if !p.WeatherTypeChangeToNeighboursOnly && has && util.Random() < 0.2 {
		return current
	}
	var allowed []WeatherType
	if has && p.WeatherTypeChangeToNeighboursOnly {
		allowed = neighbours[current]
	}
	chance := util.Random()*max(1, p.WeatherThunderChance+p.WeatherRainChance) - p.WeatherThunderChance
	if chance < 0 || util.Random() > 0.2 && inTable(thunder, current) {
		if t, ok := pick(thunder, allowed); ok {
			return t
		}
	}
	if chance < p.WeatherRainChance || util.Random() > 0.2 && inTable(rain, current) {
		if t, ok := pick(rain, allowed); ok {
			return t
		}
	}
	if t, ok := pick(regular, allowed); ok {
		return t
	}
	return Clear
}
func inTable(table []chance, t WeatherType) bool {
	for _, x := range table {
		if x.t == t {
			return true
		}
	}
	return false
}
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func estimate(t WeatherType) (float64, float64, float64) {
	switch t {
	case Clear:
		return 26, 1, 0.6
	case FewClouds:
		return 25, 2, 0.6
	case ScatteredClouds:
		return 25, 3, 0.6
	case BrokenClouds:
		return 25, 4, 0.6
	case OvercastClouds:
		return 24, 5, 0.6
	case Fog:
		return 24, 0, 0.9
	case Mist:
		return 24, 0, 0.8
	case Windy:
		return 24, 10, 0.4
	case LightDrizzle:
		return 25, 2, 0.7
	case Drizzle:
		return 24, 3, 0.7
	case HeavyDrizzle:
		return 23, 4, 0.7
	case LightRain:
		return 24, 4, 0.8
	case Rain:
		return 23, 6, 0.8
	case HeavyRain:
		return 23, 10, 0.8
	case LightThunderstorm:
		return 23, 12, 0.9
	case Thunderstorm:
		return 22, 13, 0.9
	case HeavyThunderstorm:
		return 22, 14, 0.9
	default:
		return 24, 1, 0.6
	}
}
func generateRandomDescription(current WeatherType, p config.LiveConditionParams) WeatherDescription {
	t := generateRandomType(current, p)
	temp, wind, hum := estimate(t)
	return WeatherDescription{Type: t, Temperature: temp * util.RandomRange(0.95, 1.05), WindSpeed: wind * util.RandomRange(0.6, 1.2), WindDirection: util.Random() * 360, Humidity: hum * util.RandomRange(0.6, 1.2), Pressure: 1030}
}
func rainIntensity(t WeatherType) float64 {
	switch t {
	case LightThunderstorm:
		return 0.5
	case Thunderstorm:
		return 0.6
	case HeavyThunderstorm:
		return 0.7
	case LightDrizzle:
		return 0.1
	case Drizzle:
		return 0.2
	case HeavyDrizzle, LightRain:
		return 0.3
	case Rain:
		return 0.4
	case HeavyRain, Tornado:
		return 0.5
	case LightSnow:
		return 0.2
	case Snow:
		return 0.3
	case HeavySnow, Sleet:
		return 0.4
	case LightSleet:
		return 0.3
	case HeavySleet, Hurricane, Squalls:
		return 0.5
	default:
		return 0
	}
}
func roadTemperatureCoefficient(t WeatherType) float64 {
	switch t {
	case LightThunderstorm:
		return 0.7
	case Thunderstorm:
		return 0.2
	case HeavyThunderstorm:
		return -0.2
	case LightDrizzle:
		return 0.1
	case Drizzle:
		return -0.1
	case HeavyDrizzle:
		return -0.3
	case LightRain:
		return 0.01
	case Rain:
		return -0.2
	case HeavyRain:
		return -0.5
	case LightSnow:
		return -0.7
	case Snow, Cold:
		return -0.8
	case HeavySnow:
		return -0.9
	case LightSleet, Sleet, HeavySleet, Hail:
		return -1
	case Squalls:
		return -0.5
	case Tornado:
		return -0.3
	case Hurricane:
		return -0.6
	case Clear, FewClouds, Sand, Dust, Hot:
		return 1
	case ScatteredClouds:
		return 0.8
	case BrokenClouds:
		return 0.1
	case OvercastClouds:
		return 0.01
	case Fog:
		return -0.3
	case Mist, Smoke:
		return -0.2
	case Haze:
		return 0.9
	case Windy:
		return 0.3
	default:
		return 0
	}
}
