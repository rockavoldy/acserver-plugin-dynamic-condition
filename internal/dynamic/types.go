package dynamic

type WeatherType int

const (
	WeatherNone       WeatherType = -1
	LightThunderstorm WeatherType = iota
	Thunderstorm
	HeavyThunderstorm
	LightDrizzle
	Drizzle
	HeavyDrizzle
	LightRain
	Rain
	HeavyRain
	LightSnow
	Snow
	HeavySnow
	LightSleet
	Sleet
	HeavySleet
	Clear
	FewClouds
	ScatteredClouds
	BrokenClouds
	OvercastClouds
	Fog
	Mist
	Smoke
	Haze
	Sand
	Dust
	Squalls
	Tornado
	Hurricane
	Cold
	Hot
	Windy
	Hail
)

type WeatherDescription struct {
	Type          WeatherType
	Temperature   float64
	WindSpeed     float64
	WindDirection float64
	Humidity      float64
	Pressure      float64
}

func (t WeatherType) String() string {
	switch t {
	case LightThunderstorm:
		return "Light Thunderstorm"
	case Thunderstorm:
		return "Thunderstorm"
	case HeavyThunderstorm:
		return "Heavy Thunderstorm"
	case LightDrizzle:
		return "Light Drizzle"
	case Drizzle:
		return "Drizzle"
	case HeavyDrizzle:
		return "Heavy Drizzle"
	case LightRain:
		return "Light Rain"
	case Rain:
		return "Rain"
	case HeavyRain:
		return "Heavy Rain"
	case LightSnow:
		return "Light Snow"
	case Snow:
		return "Snow"
	case HeavySnow:
		return "Heavy Snow"
	case LightSleet:
		return "Light Sleet"
	case Sleet:
		return "Sleet"
	case HeavySleet:
		return "Heavy Sleet"
	case Clear:
		return "CLear"
	case FewClouds:
		return "Few Clouds"
	case ScatteredClouds:
		return "Scattered Clouds"
	case BrokenClouds:
		return "Broken Clouds"
	case OvercastClouds:
		return "Overcast"
	case Fog:
		return "Fog"
	case Mist:
		return "Mist"
	case Smoke:
		return "Smoke"
	case Haze:
		return "Haze"
	case Sand:
		return "Sand"
	case Dust:
		return "Dust"
	case Squalls:
		return "Squalls"
	case Tornado:
		return "Tornado"
	case Hurricane:
		return "Hurricane"
	case Windy:
		return "Windy"
	case Hail:
		return "Hail"
	default:
		return "unknown"
	}
}

type DriverInfo struct {
	CarID        byte
	IsConnected  bool
	CurrentSpeed float64
}
