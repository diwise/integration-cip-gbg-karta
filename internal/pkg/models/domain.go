package models

func GetLatestTemperatureObserved(b Beach, filter func(w WaterQualityObserved) bool) (TemperatureObserved, bool) {
	for _, wqo := range b.WaterQualityObserved {
		if filter(wqo) {
			t := wqo.Temp
			d := wqo.Time()
			to := TemperatureObserved{
				Value:        t,
				DateObserved: d,
				Source:       wqo.Source,
			}
			return to, true
		}
	}
	return TemperatureObserved{}, false
}

func CalcLastTemperatureObserved(b Beach) (*TemperatureObserved, bool) {
	if sensorTemp, ok := GetLatestTemperatureObserved(b, func(w WaterQualityObserved) bool { return w.IsSensor() }); ok {
		if !sensorTemp.IsOlderThan(4) {
			if sensorTemp.Source == "" {
				sensorTemp.Source = "Göteborgs Stad"
			}
			return &sensorTemp, true
		}
	}

	if copernicus, ok := GetLatestTemperatureObserved(b, func(w WaterQualityObserved) bool { return w.IsCopernicus() }); ok {
		if !copernicus.IsOlderThan(12) {
			return &copernicus, true
		}
	}

	if sampleTemp, ok := GetLatestTemperatureObserved(b, func(w WaterQualityObserved) bool { return w.IsSampleTemp() }); ok {
		if !sampleTemp.IsOlderThan(24) {
			return &sampleTemp, true
		}
	}

	return nil, false
}
