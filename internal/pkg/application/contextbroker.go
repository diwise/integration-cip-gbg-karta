package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/domain"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type ContextBrokerClient interface {
	GetBeaches(ctx context.Context) ([]domain.Beach, error)
	GetGreenspaceRecords(ctx context.Context) ([]domain.GreenspaceRecord, error)
}

type contextBrokerClient struct {
	defaultContextURL string
	contextBrokerUrl  string
	maxDistance       string
}

func (c contextBrokerClient) GetBeaches(ctx context.Context) ([]domain.Beach, error) {
	beaches, err := c.getBeaches(ctx)
	if err != nil {
		return nil, err
	}

	for idx, b := range beaches {
		lon, lat := b.AsPoint()
		if wqo, err := c.getWaterQualityObserved(ctx, lat, lon); err == nil {
			beaches[idx].WaterQualityObserved = wqo
			log := logging.GetFromContext(ctx)
			log.Info().Msgf("found %d wqos near beach %s", len(wqo), b.Name)
		}
	}

	return beaches, nil
}

func (c contextBrokerClient) GetGreenspaceRecords(ctx context.Context) ([]domain.GreenspaceRecord, error) {
	gr, err := c.getGreenspaceRecords(ctx)
	if err != nil {
		return nil, err
	} else {
		log := logging.GetFromContext(ctx)
		log.Info().Msgf("found %d GreenspaceRecords", len(gr))
		return gr, nil
	}
}

func NewContextBrokerClient(contextBrokerClientUrl string) ContextBrokerClient {
	return &contextBrokerClient{
		contextBrokerUrl:  contextBrokerClientUrl,
		maxDistance:       "500",
		defaultContextURL: entities.DefaultContextURL,
	}
}

func (c contextBrokerClient) getBeaches(ctx context.Context) ([]domain.Beach, error) {
	params := url.Values{}
	params.Add("type", "Beach")
	params.Add("limit", "50")

	r, err := q[domain.Beach](ctx, c, params)
	return r, err
}

func (c contextBrokerClient) getWaterQualityObserved(ctx context.Context, latitude, longitude float64) ([]domain.WaterQualityObserved, error) {
	params := url.Values{}
	params.Add("type", "WaterQualityObserved")
	params.Add("geoproperty", "location")
	params.Add("georel", fmt.Sprintf("near;maxDistance==%s", c.maxDistance))
	params.Add("geometry", "Point")
	params.Add("coordinates", fmt.Sprintf("[%g,%g]", longitude, latitude))
	params.Add("limit", "1000")

	r, err := q[domain.WaterQualityObserved](ctx, c, params)
	return r, err
}

func (c contextBrokerClient) getGreenspaceRecords(ctx context.Context) ([]domain.GreenspaceRecord, error) {
	params := url.Values{}
	params.Add("type", "GreenspaceRecord")
	params.Add("limit", "1000")

	r, err := q[domain.GreenspaceRecord](ctx, c, params)
	return r, err
}

func q[T any](ctx context.Context, cb contextBrokerClient, params url.Values) ([]T, error) {
	var err error

	params.Add("options", "keyValues")

	reqUrl := fmt.Sprintf("%s/%s?%s", cb.contextBrokerUrl, "ngsi-ld/v1/entities", params.Encode())

	log := logging.GetFromContext(ctx)
	log.Debug().Msgf("calling %s", reqUrl)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header = map[string][]string{
		"Accept": {"application/ld+json"},
		"Link":   {"<" + cb.defaultContextURL + ">; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\""},
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
