package dynamic

import (
	"fmt"
	"math"
	"time"

	"plugin-dynamic-conditions/internal/acproto"
	"plugin-dynamic-conditions/internal/config"
	"plugin-dynamic-conditions/internal/csp"
	"plugin-dynamic-conditions/internal/util"
)

type Manager interface {
	IsConnected() bool
	BroadcastCSPCommand(string)
	SendCSPCommand(byte, string)
	DriverSpeeds() []float64
}

type Plugin struct {
	params                                                       config.LiveConditionParams
	manager                                                      Manager
	updatePeriod, broadcastPeriod                                time.Duration
	drivenLapsEstimate, lapLengthKm                              float64
	disposed                                                     chan struct{}
	startingDate                                                 time.Time
	startMono, weatherTransitionStart                            time.Time
	weatherCurrent, weatherNext                                  *WeatherDescription
	rainCurrent, rainNext, rainIntensity, rainWetness, rainWater float64
	rainCursor                                                   int
	rainLag                                                      [120]float64
}

func NewPlugin(p config.LiveConditionParams) *Plugin {
	if p.TrackLengthKm < 1 {
		p.TrackLengthKm = 1
	}
	bp := time.Minute
	if !p.UseRealConditions {
		bp = time.Duration(util.Clamp(p.WeatherTypeChangePeriod.Minutes()*0.2, 0.1, 2) * float64(time.Minute))
	}
	st := time.Now().Add(p.TimeOffset)
	if p.UseFixedStartingTime {
		st = time.Date(p.FixedStartingDateValue.Year(), p.FixedStartingDateValue.Month(), p.FixedStartingDateValue.Day(), 0, 0, 0, 0, time.Local).Add(time.Duration(p.FixedStartingTimeValue) * time.Second)
	}
	return &Plugin{params: p, lapLengthKm: p.TrackLengthKm, broadcastPeriod: bp, disposed: make(chan struct{}), startingDate: st, startMono: time.Now(), weatherTransitionStart: time.Now()}
}
func (p *Plugin) OnInit(m *acproto.Manager) {
	p.manager = m
	go p.syncWeather()
	go p.updateState()
}
func (p *Plugin) OnConnected() {
	go p.broadcastLoop()
}
func (p *Plugin) OnClientLoaded(msg acproto.MsgClientLoaded) {
	if p.weatherCurrent != nil && p.weatherNext != nil {
		p.manager.SendCSPCommand(msg.CarID, p.serializedCommand(*p.weatherCurrent, *p.weatherNext))
	}
}
func (p *Plugin) OnNewSession(acproto.MsgSessionInfo) {
	p.drivenLapsEstimate *= util.Saturate(p.params.TrackGripTransfer / 100)
}
func (p *Plugin) OnCarUpdate(acproto.MsgCarUpdate) {}
func (p *Plugin) Dispose() {
	select {
	case <-p.disposed:
	default:
		close(p.disposed)
	}
}
func (p *Plugin) transition() float64 {
	if p.updatePeriod <= 0 {
		return 1
	}
	return util.SmoothStep(time.Since(p.weatherTransitionStart).Seconds() / p.updatePeriod.Seconds())
}
func (p *Plugin) currentDate() time.Time {
	return p.startingDate.Add(time.Duration(time.Since(p.startMono).Seconds()*p.params.TimeMultiplier) * time.Second)
}
func (p *Plugin) serializedCommand(cur, next WeatherDescription) string {
	tr := p.transition()
	date := p.currentDate()
	ambient := util.Lerp(tr, cur.Temperature, next.Temperature) + p.params.TemperatureOffset
	if p.params.UseFixedAirTemperature {
		ambient = p.params.FixedAirTemperature
	}

	road := GetRoadTemperature(
		util.RoundToInt(float64(date.Hour()*3600+date.Minute()*60+date.Second())),
		ambient,
		util.Lerp(tr, roadTemperatureCoefficient(cur.Type), roadTemperatureCoefficient(next.Type)),
		util.Lerp(tr, nightCoolingCoefficient(cur.Type), nightCoolingCoefficient(next.Type)),
	)

	wind := util.Lerp(tr, cur.WindSpeed, next.WindSpeed) * 3.6
	grip := util.Clamp((p.params.TrackGripStartingValue+p.params.TrackGripIncreasePerLap*p.drivenLapsEstimate)/100, 0.6, 1)

	if p.params.UseV2 {
		return csp.SerializeWeatherV2(
			uint64(date.Unix()),
			byte(cur.Type-1),
			byte(next.Type-1),
			uint16(65535*tr),
			p.broadcastPeriod.Seconds(),
			ambient,
			road,
			grip,
			byte(util.Clamp(util.Lerp(tr, cur.Humidity, next.Humidity)*255, 0, 255)),
			util.Lerp(tr, cur.WindDirection, next.WindDirection), wind,
			util.Lerp(tr, cur.Pressure, next.Pressure),
			p.rainIntensity,
			p.rainWetness,
			p.rainLag[p.rainCursor],
		)
	}

	return csp.SerializeWeatherV1(uint64(date.Unix()), byte(cur.Type-1), byte(next.Type-1), float32(tr), float32(p.broadcastPeriod.Seconds()))
}

func (p *Plugin) broadcastLoop() {
	ticker := time.NewTicker(p.broadcastPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-p.disposed:
			return
		case <-ticker.C:
			if p.weatherCurrent != nil && p.weatherNext != nil && p.manager != nil && p.manager.IsConnected() {
				p.manager.BroadcastCSPCommand(p.serializedCommand(*p.weatherCurrent, *p.weatherNext))
			}
		}
	}
}

func (p *Plugin) updateState() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-p.disposed:
			return
		case <-ticker.C:
			p.updateRainAndGrip(500 * time.Millisecond)
		}
	}
}

func (p *Plugin) updateRainAndGrip(dt time.Duration) {
	tr := p.transition()
	missing := p.weatherCurrent == nil || p.weatherNext == nil
	if missing {
		p.rainIntensity = 0
	} else {
		p.rainIntensity = util.Lerp(tr, p.rainCurrent, p.rainNext)
	}
	road := 25.0
	if !missing {
		ambient := util.Lerp(tr, p.weatherCurrent.Temperature, p.weatherNext.Temperature) + p.params.TemperatureOffset
		if p.params.UseFixedAirTemperature {
			ambient = p.params.FixedAirTemperature
		}
		road = GetRoadTemperature(
			util.RoundToInt(float64(p.currentDate().Hour()*3600+p.currentDate().Minute()*60+p.currentDate().Second())),
			ambient,
			util.Lerp(tr, roadTemperatureCoefficient(p.weatherCurrent.Type), roadTemperatureCoefficient(p.weatherNext.Type)),
			util.Lerp(tr, nightCoolingCoefficient(p.weatherCurrent.Type), nightCoolingCoefficient(p.weatherNext.Type)),
		)
	}
	if p.rainIntensity > 0 {
		p.rainWetness = math.Min(1, p.rainWetness+util.Lerp(p.rainIntensity, 0.3, 1.7)*dt.Seconds()/math.Max(1, p.params.RainWetnessIncreaseTime.Seconds()*p.params.RainTimeMultiplier))
	} else {
		p.rainWetness = math.Max(0, p.rainWetness-util.Lerp(util.LerpInvSat(road, 10, 35), 0.3, 1.7)*dt.Seconds()/math.Max(1, p.params.RainWetnessDecreaseTime.Seconds()*p.params.RainTimeMultiplier))
	}
	if p.rainWater < p.rainIntensity {
		p.rainWater = math.Min(p.rainIntensity, p.rainWater+util.Lerp(p.rainIntensity, 0.3, 1.7)*dt.Seconds()/math.Max(1, p.params.RainWaterIncreaseTime.Seconds()*p.params.RainTimeMultiplier))
	} else {
		p.rainWater = math.Max(p.rainIntensity, p.rainWater-util.Lerp(util.LerpInvSat(road, 10, 35), 0.3, 1.7)*dt.Seconds()/math.Max(1, p.params.RainWaterDecreaseTime.Seconds()*p.params.RainTimeMultiplier))
	}
	if p.manager != nil {
		for _, speed := range p.manager.DriverSpeeds() {
			p.drivenLapsEstimate += speed * dt.Hours() / p.lapLengthKm
		}
	}
	p.rainLag[p.rainCursor] = p.rainWater
	p.rainCursor = (p.rainCursor + 1) % len(p.rainLag)
}

func (p *Plugin) syncWeatherByForecast() {
	// 3-hour forecast slots mapped to real time based on how fast the game clock runs
	slotReal := time.Duration(3 * 3600 / p.params.TimeMultiplier * float64(time.Second))
	p.updatePeriod = slotReal

	var cache []ForecastPoint
	var cacheExpiry time.Time

	for {
		if cache == nil || time.Now().After(cacheExpiry) {
			if entries, err := getOpenWeatherForecast(p.params.ApiKey, p.params.TrackLatitude, p.params.TrackLongitude); err == nil {
				cache = entries
				cacheExpiry = time.Now().Add(time.Hour)
			} else {
				fmt.Println("OpenWeather forecast warning:", err, "— falling back to random weather")
			}
		}
		if cache != nil {
			gameTime := p.currentDate().UTC()
			best := cache[0]
			bestDiff := absDuration(best.At.Sub(gameTime))
			for _, e := range cache[1:] {
				if d := absDuration(e.At.Sub(gameTime)); d < bestDiff {
					best = e
					bestDiff = d
				}
			}
			p.applyWeather(best.WeatherDescription)
		} else {
			current := WeatherNone
			if p.weatherNext != nil {
				current = p.weatherNext.Type
			}
			p.applyWeather(generateRandomDescription(current, p.params))
		}
		select {
		case <-p.disposed:
			return
		case <-time.After(slotReal):
		}
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func (p *Plugin) applyWeather(w WeatherDescription) {
	if !p.params.RainEnable && rainIntensity(w.Type) > 0 {
		w.Type = OvercastClouds
	}
	fmt.Printf("Applying weather: %s, Temp: %.1f°C, Wind: %.1f km/h at %.0f°, Humidity: %.0f%%, Pressure: %.0f hPa\n", w.Type.String(), w.Temperature, w.WindSpeed*3.6, w.WindDirection, w.Humidity*100, w.Pressure)

	rain := rainIntensity(w.Type) * util.RandomRange(0.8, 1.2)
	p.weatherTransitionStart = time.Now()
	if p.weatherNext != nil {
		p.rainCurrent = p.rainNext
		p.weatherCurrent = p.weatherNext
	} else {
		p.rainCurrent = rain
		p.weatherCurrent = &w
	}
	p.rainNext = rain
	p.weatherNext = &w
}

func (p *Plugin) syncWeather() {
	if p.params.UseRealConditions {
		if p.params.TimeMultiplier > 2 {
			p.syncWeatherByForecast()
			return
		}
		p.updatePeriod = 10 * time.Minute
		for {
			select {
			case <-p.disposed:
				return
			default:
				if w, err := getOpenWeather(p.params.ApiKey, p.params.TrackLatitude, p.params.TrackLongitude); err == nil {
					p.applyWeather(w)
				} else {
					fmt.Println("OpenWeather warning:", err, "— falling back to random weather")
					current := WeatherNone
					if p.weatherNext != nil {
						current = p.weatherNext.Type
					}
					p.applyWeather(generateRandomDescription(current, p.params))
				}
				time.Sleep(p.updatePeriod)
			}
		}
	} else {
		for {
			p.updatePeriod = time.Duration(p.params.WeatherTypeChangePeriod.Seconds()*util.RandomRange(0.5, 1.5)) * time.Second
			current := WeatherNone
			if p.weatherNext != nil {
				current = p.weatherNext.Type
			}
			p.applyWeather(generateRandomDescription(current, p.params))
			select {
			case <-p.disposed:
				return
			case <-time.After(p.updatePeriod):
			}
		}
	}
}
