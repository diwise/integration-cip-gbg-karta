package models

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

func (b Beach) AsPoint() (float64, float64) {
	return b.Location.Coordinates[0][0][0][0], b.Location.Coordinates[0][0][0][1]
}
