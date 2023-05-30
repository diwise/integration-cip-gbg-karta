package models

import (
	"strings"
	"time"
)

type WaterQualityObserved struct {
	Id           string `json:"id"`
	DateObserved struct {
		Value string `json:"@value"`
	} `json:"dateObserved"`
	Source string  `json:"source"`
	Temp   float64 `json:"temperature"`
}

func (w WaterQualityObserved) Time() time.Time {
	t, err := time.Parse(time.RFC3339, w.DateObserved.Value)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (w WaterQualityObserved) IsCopernicus() bool {
	return strings.Contains(strings.ToLower(w.Source), "smhi")
}

func (w WaterQualityObserved) IsSampleTemp() bool {
	return strings.Contains(strings.ToLower(w.Source), "havochvatten")
}

func (w WaterQualityObserved) IsSensor() bool {
	return !w.IsCopernicus() && !w.IsSampleTemp()
}

type TemperatureObserved struct {
	Value        float64
	DateObserved time.Time
	Source       string
}

func (t TemperatureObserved) IsOlderThan(hours int) bool {
	dur := time.Hour * time.Duration(hours*-1)
	n := time.Now().Add(dur)
	return t.DateObserved.Before(n)
}
