package dynamic

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type response struct {
	Temperature struct {
		Value string `xml:"value,attr"`
	} `xml:"temperature"`
	Weather struct {
		Number string `xml:"number,attr"`
	} `xml:"weather"`
	Wind struct {
		Speed struct {
			Value string `xml:"value,attr"`
		} `xml:"speed"`
		Direction struct {
			Value string `xml:"value,attr"`
		} `xml:"direction"`
	} `xml:"wind"`
	Humidity struct {
		Value string `xml:"value,attr"`
	} `xml:"humidity"`
	Pressure struct {
		Value string `xml:"value,attr"`
	} `xml:"pressure"`
}

func getOpenWeather(apiKey string, lat, lon float64) (WeatherDescription, error) {
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?lat=%v&lon=%v&APPID=%s&mode=xml&units=metric", lat, lon, apiKey)
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return WeatherDescription{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WeatherDescription{}, fmt.Errorf("OpenWeather status %s", resp.Status)
	}
	var r response
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		return WeatherDescription{}, err
	}
	num, _ := strconv.Atoi(r.Weather.Number)
	temp := parse(r.Temperature.Value, 25)
	wind := parse(r.Wind.Speed.Value, 0)
	dir := parse(r.Wind.Direction.Value, 0)
	hum := parse(r.Humidity.Value, 70) / 100
	press := parse(r.Pressure.Value, 1000)
	weatherType := mapType(num)
	return WeatherDescription{Type: weatherType, Temperature: temp, WindSpeed: wind, WindDirection: dir, Humidity: hum, Pressure: press}, nil
}
func parse(s string, def float64) float64 {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return def
}
func mapType(t int) WeatherType {
	switch {
	case t == 800:
		return Clear
	case t == 801:
		return FewClouds
	case t == 802:
		return ScatteredClouds
	case t == 803:
		return BrokenClouds
	case t == 804:
		return OvercastClouds
	case t == 701:
		return Mist
	case t == 741:
		return Fog
	case t == 721:
		return Haze
	case t == 731:
		return Dust
	case t == 751:
		return Sand
	case t == 711:
		return Smoke
	case t == 300:
		return LightDrizzle
	case t == 301:
		return Drizzle
	case t >= 302 && t <= 321:
		return HeavyDrizzle
	case t == 500:
		return LightRain
	case t == 501:
		return Rain
	case t >= 502 && t <= 531:
		return HeavyRain
	case t == 200 || t == 210 || t == 230:
		return LightThunderstorm
	case t == 201 || t == 211 || t == 231:
		return Thunderstorm
	case t == 202 || t == 212 || t == 221 || t == 232:
		return HeavyThunderstorm
	case t == 771:
		return Squalls
	case t == 781:
		return Tornado
	case t == 600 || t == 620:
		return LightSnow
	case t == 601 || t == 621:
		return Snow
	case t == 602 || t == 622:
		return HeavySnow
	case t == 611 || t == 615:
		return LightSleet
	case t == 612 || t == 616:
		return Sleet
	case t == 613:
		return HeavySleet
	default:
		return Clear
	}
}
