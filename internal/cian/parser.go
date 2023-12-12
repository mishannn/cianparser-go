package cian

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	api2captcha "github.com/2captcha/2captcha-go"
	"golang.org/x/sync/singleflight"

	"github.com/mishannn/cianparser-go/internal/geo"
	"github.com/mishannn/cianparser-go/internal/utils"
)

const cianBaseURL = "https://api.cian.ru"
const cianCaptchaURL = cianBaseURL + "/captcha/"
const cianGetClustersForMapURL = cianBaseURL + "/search-offers-index-map/v1/get-clusters-for-map/"
const cianGetOffersByIDsURL = cianBaseURL + "/search-offers/v1/get-offers-by-ids-desktop/"

const userAgent = "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36"

var captchaKeyRegex = regexp.MustCompile(`'sitekey': '(.*?)'`)

type Parser struct {
	httpClient *http.Client

	geojson                 string
	searchType              string
	searchFilters           map[string]JSONQueryItem
	searchCellSize          float64
	maxWorkersCollectIDs    int
	maxWorkersCollectOffers int
	captchaGroup            singleflight.Group
	captchaClient           *api2captcha.Client
}

func NewParser(httpClient *http.Client, captchaApiKey string, geojson string, searchType string, searchFilters map[string]JSONQueryItem, searchCellSize float64, maxWorkersCollectIDs int, maxWorkersCollectOffers int) *Parser {
	return &Parser{
		httpClient:              httpClient,
		geojson:                 geojson,
		searchType:              searchType,
		searchFilters:           searchFilters,
		searchCellSize:          searchCellSize,
		maxWorkersCollectIDs:    maxWorkersCollectIDs,
		maxWorkersCollectOffers: maxWorkersCollectOffers,
		captchaGroup:            singleflight.Group{},
		captchaClient:           api2captcha.NewClient(captchaApiKey),
	}
}

func (p *Parser) getCaptchaSiteKey() (string, error) {
	req, err := http.NewRequest(http.MethodGet, cianCaptchaURL, nil)
	if err != nil {
		return "", fmt.Errorf("can't create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("can't read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("server sent http error: %d, %s", resp.StatusCode, respBody)
	}

	match := captchaKeyRegex.FindSubmatch(respBody)
	if match == nil {
		return "", errors.New("key not found")
	}

	return string(match[1]), nil
}

func (p *Parser) sendCaptchaCode(code string) error {
	form := url.Values{}
	form.Add("g-recaptcha-response", code)
	form.Add("redirect_url", "")

	req, err := http.NewRequest(http.MethodPost, cianCaptchaURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("can't create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("can't do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read response body: %w", err)
	}

	if resp.StatusCode != 302 {
		return fmt.Errorf("server sent unexpected code: %d (expected 302), %s", resp.StatusCode, respBody)
	}

	return nil
}

func (p *Parser) solveCaptcha() error {
	_, err, _ := p.captchaGroup.Do("captcha", func() (any, error) {
		log.Println("solving captcha...")

		siteKey, err := p.getCaptchaSiteKey()
		if err != nil {
			return nil, fmt.Errorf("can't get captcha sitekey: %w", err)
		}

		cap := api2captcha.ReCaptcha{
			SiteKey: siteKey,
			Url:     cianCaptchaURL,
		}

		code, err := p.captchaClient.Solve(cap.ToRequest())
		if err != nil {
			return nil, fmt.Errorf("can't get solve captcha: %w", err)
		}

		err = p.sendCaptchaCode(code)
		if err != nil {
			return nil, fmt.Errorf("can't send captcha code: %w", err)
		}

		log.Println("captcha solved")
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("can't solve captcha: %w", err)
	}

	return nil
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
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body: %w", err)
	}

	if resp.StatusCode == 302 {
		err := p.solveCaptcha()
		if err != nil {
			return nil, fmt.Errorf("can't solve captcha: %d, %s", resp.StatusCode, respBody)
		}

		return p.getClustersByBounds(bounds)
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
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body: %w", err)
	}

	if resp.StatusCode == 302 {
		err := p.solveCaptcha()
		if err != nil {
			return nil, fmt.Errorf("can't solve captcha: %d, %s", resp.StatusCode, respBody)
		}

		return p.getOffers(ids)
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
