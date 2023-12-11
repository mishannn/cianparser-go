package cian

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/mishannn/cianparser-go/internal/geo"
	"github.com/mishannn/cianparser-go/internal/utils"
)

const cianGetClustersForMapURL = "https://api.cian.ru/search-offers-index-map/v1/get-clusters-for-map/"
const cianGetOffersByIDsURL = "https://api.cian.ru/search-offers/v1/get-offers-by-ids-desktop/"

type Parser struct {
	httpClient *http.Client

	geojson                 string
	searchType              string
	searchFilters           map[string]JSONQueryItem
	searchCellSize          float64
	maxWorkersCollectIDs    int
	maxWorkersCollectOffers int
}

func NewParser(httpClient *http.Client, geojson string, searchType string, searchFilters map[string]JSONQueryItem, searchCellSize float64, maxWorkersCollectIDs int, maxWorkersCollectOffers int) *Parser {
	return &Parser{
		httpClient:              httpClient,
		geojson:                 geojson,
		searchType:              searchType,
		searchFilters:           searchFilters,
		searchCellSize:          searchCellSize,
		maxWorkersCollectIDs:    maxWorkersCollectIDs,
		maxWorkersCollectOffers: maxWorkersCollectOffers,
	}
}

func (p *Parser) getJSONQuery() map[string]any {
	jsonQuery := map[string]any{}

	for key, value := range p.searchFilters {
		jsonQuery[key] = value
	}

	jsonQuery["_type"] = p.searchType

	return jsonQuery
}

func (p *Parser) getClustersByBounds(bounds Bounds) ([]Cluster, error) {
	reqBody := GetClustersRequestBody{
		Zoom:      15,
		Bbox:      []Bounds{bounds},
		JSONQuery: p.getJSONQuery(),
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("can't marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cianGetClustersForMapURL, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("can't create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server sent http error: %d, %s", resp.StatusCode, respBody)
	}

	var clustersResponseBody GetClustersResponseBody
	err = json.Unmarshal(respBody, &clustersResponseBody)
	if err != nil {
		return nil, fmt.Errorf("can't parse response body: %w, %s", err, respBody)
	}

	return clustersResponseBody.Filtered, nil
}

func (p *Parser) getOffers(ids []int64) ([]Offer, error) {
	reqBody := GetOffersByIDsRequestBody{
		CianOfferIDS: ids,
		JSONQuery:    p.getJSONQuery(),
	}

	reqBodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("can't marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cianGetOffersByIDsURL, bytes.NewReader(reqBodyJSON))
	if err != nil {
		return nil, fmt.Errorf("can't create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("server sent http error: %d, %s", resp.StatusCode, respBody)
	}

	var offersResponseBody GetOffersByIDsResponseBody
	err = json.Unmarshal(respBody, &offersResponseBody)
	if err != nil {
		return nil, fmt.Errorf("can't parse response body: %w, %s", err, respBody)
	}

	return offersResponseBody.OffersSerialized, nil
}

func (p *Parser) GetOfferIDs() ([]int64, error) {
	boundsList, err := geo.GetCellBoundsListByGeoJSON(p.geojson, p.searchCellSize)
	if err != nil {
		return nil, fmt.Errorf("can't get cell bounds list: %s", err)
	}

	cianBoundsList := make([]Bounds, len(boundsList))
	for i := 0; i < len(boundsList); i++ {
		cianBoundsList[i] = GeosBoundsToCianBounds(boundsList[i])
	}

	workerPool := utils.NewWorkerPool(p.getClustersByBounds, p.maxWorkersCollectIDs)
	workerPool.OnProgress(func(current, total int) {
		log.Printf("get clusters progress: %d%%\n", (current * 100 / total))
	})

	clustersList, err := workerPool.Map(context.Background(), cianBoundsList)
	if err != nil {
		return nil, fmt.Errorf("can't get clusters: %w", err)
	}

	offerIDs := make([]int64, 0)
	for _, clusters := range clustersList {
		for _, cluster := range clusters {
			offerIDs = append(offerIDs, cluster.ClusterOfferIds...)
		}
	}

	return utils.RemoveDuplicateInt64(offerIDs), nil
}

func (p *Parser) GetOffers(ids []int64) ([]Offer, error) {
	chunks := utils.Chunks(ids, 28)

	workerPool := utils.NewWorkerPool(p.getOffers, p.maxWorkersCollectOffers)
	workerPool.OnProgress(func(current, total int) {
		log.Printf("get offers progress: %d%%\n", (current * 100 / total))
	})

	offersList, err := workerPool.Map(context.Background(), chunks)
	if err != nil {
		return nil, fmt.Errorf("can't get offers: %w", err)
	}

	offers := make([]Offer, 0)
	for _, tmpOffers := range offersList {
		offers = append(offers, tmpOffers...)
	}

	return offers, nil
}
