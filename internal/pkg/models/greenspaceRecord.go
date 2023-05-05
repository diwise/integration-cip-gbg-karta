package models

import (
	"time"
)

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
