package models

import "encoding/json"

type Beach struct {
	Id           string   `json:"id"`
	AreaServed   string   `json:"areaServed"`
	BeachType    []string `json:"beachType"`
	BusinessId   string   `json:"businessId"`
	DataProvider string   `json:"dataProvider"`
	Description  string   `json:"description"`
	Location     struct {
		Coordinates [][][][]float64 `json:"coordinates"`
	} `json:"location"`
	Name                 string   `json:"name"`
	SeeAlso              []string `json:"seeAlso"`
	Source               string   `json:"source"`
	Type                 string   `json:"type"`
	WaterQualityObserved []WaterQualityObserved
}

type BeachRaw struct {
	Id           string          `json:"id"`
	AreaServed   string          `json:"areaServed"`
	BeachType    json.RawMessage `json:"beachType"`
	BusinessId   string          `json:"businessId"`
	DataProvider string          `json:"dataProvider"`
	Description  string          `json:"description"`
	Location     struct {
		Coordinates [][][][]float64 `json:"coordinates"`
	} `json:"location"`
	Name                 string          `json:"name"`
	SeeAlso              json.RawMessage `json:"seeAlso"`
	Source               string          `json:"source"`
	Type                 string          `json:"type"`
	WaterQualityObserved []WaterQualityObserved
}

func (br BeachRaw) ToModel() Beach {
	return Beach{
		Id: br.Id,
		AreaServed: br.AreaServed,
		BeachType: rawToStrings(br.BeachType),
		BusinessId: br.BusinessId,
		DataProvider: br.DataProvider,
		Description: br.Description,
		Location: br.Location,
		Name: br.Name,
		SeeAlso: rawToStrings(br.SeeAlso),
		Source: br.Source,
		Type: br.Type,
		WaterQualityObserved: br.WaterQualityObserved,
	}
}

func rawToStrings(p json.RawMessage) []string {
	var strs []string
	err := json.Unmarshal(p, &strs)
	if err == nil {
		return strs
	}

	var s string
	err = json.Unmarshal(p, &s)
	if err == nil {
		return []string{s}
	}

	return []string{}
}

func (b Beach) AsPoint() (float64, float64) {
	return b.Location.Coordinates[0][0][0][0], b.Location.Coordinates[0][0][0][1]
}
