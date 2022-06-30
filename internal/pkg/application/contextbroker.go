package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/domain"
)

type ContextBrokerClient interface {
	GetBeaches(ctx context.Context) []domain.Beach
}

type contextBrokerClient struct {
	defaultContextURL string
	contextBrokerUrl  string
	maxDistance       string
}

func (c contextBrokerClient) GetBeaches(ctx context.Context) []domain.Beach {
	beaches := c.getBeaches(ctx)
	for idx, b := range beaches {
		lat, lon := b.AsPoint()
		wqo := c.getWaterQualityObserved(ctx, lat, lon)
		beaches[idx].WaterQualityObserved = wqo
	}
	return beaches
}

func NewContextBrokerClient(contextBrokerClientUrl string) ContextBrokerClient {
	return &contextBrokerClient{
		contextBrokerUrl:  contextBrokerClientUrl,
		maxDistance:       "1000",
		defaultContextURL: "https://raw.githubusercontent.com/diwise/context-broker/main/assets/jsonldcontexts/default-context.jsonld",
	}
}

func (c contextBrokerClient) getBeaches(ctx context.Context) []domain.Beach {
	params := url.Values{}
	params.Add("type", "Beach")

	r, _ := q[domain.Beach](ctx, c, params)
	return r
}

func (c contextBrokerClient) getWaterQualityObserved(ctx context.Context, latitude, longitude float64) []domain.WaterQualityObserved {
	params := url.Values{}
	params.Add("type", "WaterQualityObserved")
	params.Add("geoproperty", "location")
	params.Add("georel", fmt.Sprintf("near;maxDistance==%s", c.maxDistance))
	params.Add("geometry", "Point")
	params.Add("coordinates", fmt.Sprintf("[%g,%g]", latitude, longitude))

	r, _ := q[domain.WaterQualityObserved](ctx, c, params)
	return r
}

func q[T any](ctx context.Context, cb contextBrokerClient, params url.Values) ([]T, error) {
	var err error

	params.Add("options", "keyValues")

	reqUrl := fmt.Sprintf("%s/%s?%s", cb.contextBrokerUrl, "ngsi-ld/v1/entities", params.Encode())

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
		fmt.Printf("failed to retrieve data from context-broker: %s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("failed to retrieve data from context-broker, expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
		return nil, fmt.Errorf("expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("failed to read response body: %s", err.Error())
		return nil, err
	}

	result := []T{}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}