package cip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/models"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

func GetBeaches(ctx context.Context, contextBrokerUrl string) ([]models.Beach, error) {
	params := url.Values{}
	params.Add("type", "Beach")
	params.Add("limit", "50")

	result, err := queryEntities[models.BeachRaw](ctx, contextBrokerUrl, params)
	if err != nil {
		return nil, err
	}

	var beaches []models.Beach
	for _, b := range result {
		beaches = append(beaches, b.ToModel())
	}

	return beaches, nil
}

func GetBeachesWithTemp(ctx context.Context, contextBrokerUrl, maxDistance string) ([]models.Beach, error) {
	log := logging.GetFromContext(ctx)

	beaches, err := GetBeaches(ctx, contextBrokerUrl)
	if err != nil {
		return nil, err
	}

	var errs []error

	for i, b := range beaches {
		lon, lat := b.AsPoint()

		wqo, err := GetWaterQualityObserved(ctx, contextBrokerUrl, maxDistance, lat, lon)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		log.Debug().Msgf("found %d wqos near beach %s", len(wqo), b.Name)
		beaches[i].WaterQualityObserved = wqo
	}

	return beaches, errors.Join(errs...)
}

func GetWaterQualityObserved(ctx context.Context, contextBrokerUrl, maxDistance string, latitude, longitude float64) ([]models.WaterQualityObserved, error) {
	params := url.Values{}
	params.Add("type", "WaterQualityObserved")
	params.Add("geoproperty", "location")
	params.Add("georel", fmt.Sprintf("near;maxDistance==%s", maxDistance))
	params.Add("geometry", "Point")
	params.Add("coordinates", fmt.Sprintf("[%g,%g]", longitude, latitude))
	params.Add("limit", "1000")

	r, err := queryEntities[models.WaterQualityObserved](ctx, contextBrokerUrl, params)

	sort.Slice(r, func(i, j int) bool {
		d1, err := time.Parse(time.RFC3339, r[i].DateObserved.Value)
		if err != nil {
			return false
		}
		d2, err := time.Parse(time.RFC3339, r[j].DateObserved.Value)
		if err != nil {
			return true
		}

		return d1.Unix() > d2.Unix()
	})

	return r, err
}

func GetGreenspaceRecords(ctx context.Context, contextBrokerUrl string) ([]models.GreenspaceRecord, error) {
	params := url.Values{}
	params.Add("type", "GreenspaceRecord")
	params.Add("limit", "1000")

	r, err := queryEntities[models.GreenspaceRecord](ctx, contextBrokerUrl, params)
	return r, err
}

func queryEntities[T models.BeachRaw | models.GreenspaceRecord | models.WaterQualityObserved](ctx context.Context, contextBrokerUrl string, params url.Values) ([]T, error) {
	var err error

	params.Add("options", "keyValues")

	reqUrl := fmt.Sprintf("%s/%s?%s", contextBrokerUrl, "ngsi-ld/v1/entities", params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header = map[string][]string{
		"Accept": {"application/ld+json"},
		"Link":   {entities.LinkHeader},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from context-broker: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	result := []T{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
