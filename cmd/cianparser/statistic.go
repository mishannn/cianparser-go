package main

import (
	"log"
	"sort"
	"strconv"

	"github.com/mishannn/cianparser-go/internal/cian"
	"gonum.org/v1/gonum/stat"
)

type flatKey struct {
	Category   string
	Location   string
	RoomsCount int
}

type flatStatItem struct {
	Location    string  `json:"location"`
	Category    string  `json:"category"`
	RoomsCount  int     `json:"rooms_count"`
	MedianPrice float64 `json:"median_price"`
}

func getFlatStatistic(offers []cian.Offer) []flatStatItem {
	groupedOffers := make(map[flatKey][]float64)
	for _, offer := range offers {
		totalArea, err := strconv.ParseFloat(offer.TotalArea, 64)
		if err != nil {
			log.Printf("can't parse flat area '%s': %s", offer.TotalArea, err)
			continue
		}

		key := flatKey{
			Location:   getDistrictString(offer.Geo.Address),
			RoomsCount: int(offer.RoomsCount),
			Category:   offer.Category,
		}

		groupedOffers[key] = append(groupedOffers[key], offer.BargainTerms.PriceRur/totalArea)
	}

	offersWithMedianPricePerMeter := make([]flatStatItem, 0, len(groupedOffers))
	for key, value := range groupedOffers {
		sort.Float64s(value)

		offersWithMedianPricePerMeter = append(offersWithMedianPricePerMeter, flatStatItem{
			Location:    key.Location,
			Category:    key.Category,
			RoomsCount:  key.RoomsCount,
			MedianPrice: stat.Quantile(0.5, stat.Empirical, value, nil),
		})
	}

	return offersWithMedianPricePerMeter
}
