package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/mishannn/cianparser-go/internal/cian"
)

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
	polygonFilePath := flag.String("polygon", "", "polygon for search")
	credentialsFilePath := flag.String("credentials", "", "google credentials file")
	spredsheetId := flag.String("spreadsheet", "", "sheet id")
	dataRange := flag.String("datarange", "", "sheets data range")
	rucaptchaKey := flag.String("rucaptchakey", "", "rucapctha key")
	flag.Parse()

	if *spredsheetId == "" {
		log.Fatalf("spreadsheet flag not set")
	}

	if *dataRange == "" {
		log.Fatalf("datarange flag not set")
	}

	if *rucaptchaKey == "" {
		log.Fatalf("rucaptchakey flag not set")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("can't create cookie jar: %s", err)
	}

	credentials, err := os.ReadFile(*credentialsFilePath)
	if err != nil {
		log.Fatalf("can't read credentials file: %s", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar:       jar,
		Transport: transport,
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

	parser := cian.NewParser(httpClient, *rucaptchaKey, string(geojson), searchType, searchFilters, 10000, 1, 6)

	offerIDs, err := parser.GetOfferIDs()
	if err != nil {
		log.Fatalf("can't get offer ids: %s", err)
	}

	offers, err := parser.GetOffers(offerIDs)
	if err != nil {
		log.Fatalf("can't get offers: %s", err)
	}

	flatStat := getFlatStatistic(offers)

	err = saveFlatStatistic(credentials, *spredsheetId, *dataRange, time.Now(), flatStat)
	if err != nil {
		log.Fatalf("can't save statistic: %s", err)
	}
}
