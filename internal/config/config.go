package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type PluginSettings struct {
	ListeningPort     int
	RemotePort        int
	RemoteHostName    string
	Capacity          int
	LogServerRequests bool
}
type ExternalPluginInfo struct {
	ListeningPort  int
	RemoteHostname string
	RemotePort     int
}
type LiveConditionParams struct {
	UseV2                                                                                          bool
	ApiKey                                                                                         string
	RainEnable                                                                                     bool
	TrackLatitude, TrackLongitude, TrackLengthKm                                                   float64
	TimezoneID                                                                                     string
	UseRealConditions                                                                              bool
	TimeOffset                                                                                     time.Duration
	UseFixedStartingTime                                                                           bool
	FixedStartingTimeValue                                                                         int
	FixedStartingDateValue                                                                         time.Time
	TimeMultiplier, TemperatureOffset                                                              float64
	UseFixedAirTemperature                                                                         bool
	FixedAirTemperature                                                                            float64
	WeatherTypeChangePeriod                                                                        time.Duration
	WeatherTypeChangeToNeighboursOnly                                                              bool
	WeatherRainChance, WeatherThunderChance                                                        float64
	TrackGripStartingValue, TrackGripIncreasePerLap, TrackGripTransfer                             float64
	RainTimeMultiplier                                                                             float64
	RainWetnessIncreaseTime, RainWetnessDecreaseTime, RainWaterIncreaseTime, RainWaterDecreaseTime time.Duration
}
type Params struct {
	Plugin          PluginSettings
	ExternalPlugins []ExternalPluginInfo
	Weather         LiveConditionParams
}

func Defaults() Params {
	return Params{
		Plugin: PluginSettings{
			RemoteHostName: "127.0.0.1",
		},
		Weather: LiveConditionParams{
			RainEnable:                        true,
			FixedStartingTimeValue:            43200,
			FixedStartingDateValue:            time.Now(),
			TimeMultiplier:                    1,
			FixedAirTemperature:               25,
			WeatherTypeChangePeriod:           5 * time.Minute,
			WeatherTypeChangeToNeighboursOnly: true,
			WeatherRainChance:                 0.05,
			WeatherThunderChance:              0.005,
			TrackGripStartingValue:            99,
			TrackGripIncreasePerLap:           0.05,
			TrackGripTransfer:                 80,
			RainTimeMultiplier:                1,
			RainWetnessIncreaseTime:           3 * time.Minute,
			RainWetnessDecreaseTime:           15 * time.Minute,
			RainWaterIncreaseTime:             30 * time.Minute,
			RainWaterDecreaseTime:             120 * time.Minute,
		},
	}
}
func ReadFile(path string) (Params, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Params{}, err
	}
	return Parse(strings.Split(string(data), "\n"))
}
func Parse(lines []string) (Params, error) {
	p := Defaults()
	for _, line := range lines {
		line = strings.TrimSpace(strings.Split(line, "#")[0])
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if k == "message" {
			fmt.Println(v)
			continue
		}
		if err := apply(&p, k, v); err != nil {
			return p, fmt.Errorf("failed to parse %s: %w", k, err)
		}
	}
	return p, nil
}
func apply(p *Params, k, v string) error {
	f := func() (float64, error) { return strconv.ParseFloat(v, 64) }
	i := func() (int, error) { return strconv.Atoi(v) }
	b := func() bool { return v != "0" }
	d := func() (time.Duration, error) { return parseDuration(v) }
	switch k {
	case "plugin.listeningPort":
		x, e := i()
		p.Plugin.ListeningPort = x
		return e
	case "plugin.remotePort":
		x, e := i()
		p.Plugin.RemotePort = x
		return e
	case "plugin.remoteHostName":
		p.Plugin.RemoteHostName = v
		return nil
	case "plugin.serverCapacity":
		x, e := i()
		p.Plugin.Capacity = x
		return e
	case "plugin.externalPlugin":
		parts := strings.Split(v, ",")
		if len(parts) != 2 {
			return fmt.Errorf("expected <listeningPort>,<host>:<port>")
		}
		lp, e := strconv.Atoi(strings.TrimSpace(parts[0]))
		if e != nil {
			return e
		}
		hp := strings.Split(strings.TrimSpace(parts[1]), ":")
		if len(hp) != 2 {
			return fmt.Errorf("expected host:port")
		}
		rp, e := strconv.Atoi(hp[1])
		if e != nil {
			return e
		}
		p.ExternalPlugins = append(p.ExternalPlugins, ExternalPluginInfo{lp, hp[0], rp})
		return nil
	case "weather.useV2":
		p.Weather.UseV2 = b()
	case "weather.apiKey":
		p.Weather.ApiKey = v
	case "weather.rainEnable":
		p.Weather.RainEnable = b()
	case "weather.trackLatitude":
		x, e := f()
		p.Weather.TrackLatitude = x
		return e
	case "weather.trackLongitude":
		x, e := f()
		p.Weather.TrackLongitude = x
		return e
	case "weather.trackLengthKm":
		x, e := f()
		p.Weather.TrackLengthKm = x
		return e
	case "weather.trackTimezoneId":
		p.Weather.TimezoneID = v
	case "weather.useRealConditions":
		p.Weather.UseRealConditions = b()
	case "weather.timeOffset":
		x, e := d()
		p.Weather.TimeOffset = x
		return e
	case "weather.useFixedStartingTime":
		p.Weather.UseFixedStartingTime = b()
	case "weather.fixedStartingTimeValue":
		x, e := i()
		p.Weather.FixedStartingTimeValue = x
		return e
	case "weather.fixedStartingDateValue":
		t, e := parseDate(v)
		p.Weather.FixedStartingDateValue = t
		return e
	case "weather.timeMultiplier":
		x, e := f()
		p.Weather.TimeMultiplier = x
		return e
	case "weather.temperatureOffset":
		x, e := f()
		p.Weather.TemperatureOffset = x
		return e
	case "weather.useFixedAirTemperature":
		p.Weather.UseFixedAirTemperature = b()
	case "weather.fixedAirTemperature":
		x, e := f()
		p.Weather.FixedAirTemperature = x
		return e
	case "weather.weatherTypeChangePeriod":
		x, e := d()
		p.Weather.WeatherTypeChangePeriod = x
		return e
	case "weather.weatherTypeChangeToNeighboursOnly":
		p.Weather.WeatherTypeChangeToNeighboursOnly = b()
	case "weather.weatherRainChance":
		x, e := f()
		p.Weather.WeatherRainChance = x
		return e
	case "weather.weatherThunderChance":
		x, e := f()
		p.Weather.WeatherThunderChance = x
		return e
	case "weather.trackGripStartingValue":
		x, e := f()
		p.Weather.TrackGripStartingValue = x
		return e
	case "weather.trackGripIncreasePerLap":
		x, e := f()
		p.Weather.TrackGripIncreasePerLap = x
		return e
	case "weather.trackGripTransfer":
		x, e := f()
		p.Weather.TrackGripTransfer = x
		return e
	case "weather.rainTimeMultiplier":
		x, e := f()
		p.Weather.RainTimeMultiplier = x
		return e
	case "weather.rainWetnessIncreaseTime":
		x, e := d()
		p.Weather.RainWetnessIncreaseTime = x
		return e
	case "weather.rainWetnessDecreaseTime":
		x, e := d()
		p.Weather.RainWetnessDecreaseTime = x
		return e
	case "weather.rainWaterIncreaseTime":
		x, e := d()
		p.Weather.RainWaterIncreaseTime = x
		return e
	case "weather.rainWaterDecreaseTime":
		x, e := d()
		p.Weather.RainWaterDecreaseTime = x
		return e
	}
	return nil
}
func parseDuration(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return time.ParseDuration(s)
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	sec, _ := strconv.ParseFloat(parts[2], 64)
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec*float64(time.Second)), nil
}
func parseDate(s string) (time.Time, error) {
	layouts := []string{"01/02/2006 15:04:05", "02/01/2006 15:04:05", "2006-01-02 15:04:05"}
	var last error
	for _, l := range layouts {
		t, e := time.ParseInLocation(l, s, time.Local)
		if e == nil {
			return t, nil
		}
		last = e
	}
	return time.Time{}, last
}
