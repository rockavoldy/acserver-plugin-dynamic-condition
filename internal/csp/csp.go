package csp

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"math"
)

func SerializeWeatherV1(timestamp uint64, current, next byte, transition, timeToApply float32) string {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(1000))
	binary.Write(&b, binary.LittleEndian, timestamp)
	b.WriteByte(current)
	b.WriteByte(next)
	binary.Write(&b, binary.LittleEndian, transition)
	binary.Write(&b, binary.LittleEndian, timeToApply)
	return "\t\t\t\t$CSP0:" + trimPadding(base64.StdEncoding.EncodeToString(b.Bytes()))
}

func SerializeWeatherV2(timestamp uint64, current, next byte, transition uint16, timeToApply, ambient, road float64, trackGrip float64, humidity byte, windDir, windSpeed, pressure, rainIntensity, rainWetness, rainWater float64) string {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(1001))
	binary.Write(&b, binary.LittleEndian, timestamp)
	b.WriteByte(current)
	b.WriteByte(next)
	binary.Write(&b, binary.LittleEndian, transition)
	writeHalf(&b, timeToApply)
	writeHalf(&b, ambient)
	writeHalf(&b, road)
	b.WriteByte(byte(math.Round(clamp((trackGrip-0.6)/(1-0.6), 0, 1) * 255)))
	b.WriteByte(humidity)
	writeHalf(&b, windDir)
	writeHalf(&b, windSpeed)
	writeHalf(&b, pressure)
	writeHalf(&b, rainIntensity)
	writeHalf(&b, rainWetness)
	writeHalf(&b, rainWater)
	return "\t\t\t\t$CSP0:" + trimPadding(base64.StdEncoding.EncodeToString(b.Bytes()))
}

func trimPadding(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}
func clamp(v, a, b float64) float64 {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}

func writeHalf(b *bytes.Buffer, f float64) {
	binary.Write(b, binary.LittleEndian, float32ToHalf(float32(f)))
}
func float32ToHalf(f float32) uint16 {
	bits := math.Float32bits(f)
	sign := uint16((bits >> 16) & 0x8000)
	exp := int((bits>>23)&0xff) - 127 + 15
	mant := bits & 0x7fffff
	if exp <= 0 {
		if exp < -10 {
			return sign
		}
		mant = (mant | 0x800000) >> uint(1-exp)
		return sign | uint16((mant+0x1000)>>13)
	}
	if exp >= 31 {
		if mant == 0 {
			return sign | 0x7c00
		}
		return sign | 0x7c00 | uint16(mant>>13)
	}
	return sign | uint16(exp<<10) | uint16((mant+0x1000)>>13)
}
