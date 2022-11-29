package domain

import (
	"context"
	"strings"
	"time"
)

type Beach struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Source   string `json:"source"`
	Location struct {
		Coordinates [][][][]float64 `json:"coordinates"`
	} `json:"location"`
	WaterQualityObserved []WaterQualityObserved
}

func (b Beach) AsPoint() (float64, float64) {
	return b.Location.Coordinates[0][0][0][0], b.Location.Coordinates[0][0][0][1]
}

func (b Beach) LatestTO(filter func(w WaterQualityObserved) bool) (TemperatureObserved, bool) {
	ts := b.getTemperatures(filter)
	if len(ts) == 1 {
		return ts[0], true
	} else if len(ts) > 1 {
		d := ts[0].DateObserved
		t := ts[0]

		for idx, x := range ts {
			if x.DateObserved.After(d) {
				t = ts[idx]
				d = ts[idx].DateObserved
			}
		}

		return t, true
	}
	return TemperatureObserved{}, false
}

func (b Beach) GetLatestTemperature(ctx context.Context) (*TemperatureObserved, bool) {

	if sensorTemp, ok := b.LatestTO(func(w WaterQualityObserved) bool { return w.IsSensor() }); ok {
		if !sensorTemp.IsOlderThan(4) {
			if sensorTemp.Source == "" {
				sensorTemp.Source = "Göteborgs Stad"
			}
			return &sensorTemp, true
		}
	}

	if copernicus, ok := b.LatestTO(func(w WaterQualityObserved) bool { return w.IsCopernicus() }); ok {
		if !copernicus.IsOlderThan(12) {
			return &copernicus, true
		}
	}

	if sampleTemp, ok := b.LatestTO(func(w WaterQualityObserved) bool { return w.IsSampleTemp() }); ok {
		if !sampleTemp.IsOlderThan(24) {
			return &sampleTemp, true
		}
	}

	return nil, false
}

func (b Beach) getTemperatures(filter func(w WaterQualityObserved) bool) []TemperatureObserved {
	var temperatures []TemperatureObserved
	for _, w := range b.WaterQualityObserved {
		if filter(w) {
			t := w.Temp
			d := w.Time()
			to := &TemperatureObserved{
				Value:        t,
				DateObserved: d,
				Source:       w.Source,
			}
			temperatures = append(temperatures, *to)
		}
	}
	return temperatures
}

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
	return strings.HasPrefix(w.Source, "https://www.smhi.se")
}

func (w WaterQualityObserved) IsSampleTemp() bool {
	return strings.HasPrefix(w.Source, "https://badplatsen.havochvatten.se")
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

type GreenspaceRecord struct {
	Id           string `json:"id"`
	DateObserved struct {
		Value string `json:"@value"`
	} `json:"dateObserved"`
	Location struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"location"`
	SoilMoisturePressure int    `json:"soilMoisturePressure"`
	Source               string `json:"source"`
}

func (g GreenspaceRecord) Time() time.Time {
	t, err := time.Parse(time.RFC3339, g.DateObserved.Value)
	if err != nil {
		return time.Time{}
	}
	return t
}
