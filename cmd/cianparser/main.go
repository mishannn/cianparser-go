package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mishannn/cianparser-go/internal/cian"
	"github.com/mishannn/cianparser-go/internal/httpclient"
)

func getHeaderWithCookie(cookieFilePath string) (http.Header, error) {
	header := make(http.Header)

	contents, err := os.ReadFile(cookieFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't read file: %w", err)
	}

	header.Set("Cookie", strings.TrimSpace(string(contents)))
	return header, nil
}

func getDistrictString(addresses []cian.Address) string {
	parts := make([]string, 0)

	for _, address := range addresses {
		if address.GeoType == "location" || address.GeoType == "district" {
			parts = append(parts, address.FullName)
		}
	}

	return strings.Join(parts, ", ")
}

func main() {
	polygonFilePath := flag.String("polygon", "polygon.geojson", "polygon for search")
	cookieFilePath := flag.String("cookie", "", "cookie file")
	credentialsFilePath := flag.String("credentials", "credentials.json", "google credentials file")
	spredsheetId := flag.String("spreadsheet", "", "sheet id")
	dataRange := flag.String("datarange", "A:E", "sheets data range")
	flag.Parse()

	var err error

	var header http.Header
	if *cookieFilePath != "" {
		header, err = getHeaderWithCookie(*cookieFilePath)
		if err != nil {
			log.Fatalf("can't read cookie: %s", err)
		}
	} else {
		header = make(http.Header)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: httpclient.NewRoundTripperWithHeaders(transport, header),
	}

	geojson, err := os.ReadFile(*polygonFilePath)
	if err != nil {
		log.Fatalf("can't read polygon file: %s", err)
	}

	searchType := "flatsale"
	searchFilters := map[string]cian.JSONQueryItem{
		"engine_version": {
			Type:  "term",
			Value: 2,
		},
		"demolished_in_moscow_programm": {
			Type:  "term",
			Value: false,
		},
		"only_flat": {
			Type:  "term",
			Value: true,
		},
		"flat_share": {
			Type:  "term",
			Value: 2,
		},
	}

	parser := cian.NewParser(httpClient, string(geojson), searchType, searchFilters, 10000, 1, 10)

	offerIDs, err := parser.GetOfferIDs()
	if err != nil {
		log.Fatalf("can't get offer ids: %s", err)
	}

	offers, err := parser.GetOffers(offerIDs)
	if err != nil {
		log.Fatalf("can't get offers: %s", err)
	}

	flatStat := getFlatStatistic(offers)

	err = saveFlatStatistic(*credentialsFilePath, *spredsheetId, *dataRange, time.Now(), flatStat)
	if err != nil {
		log.Fatalf("can't save statistic: %s", err)
	}
}
