package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

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

func getDistrictString(addresses []cian.Address) string {
	parts := make([]string, 0)

	for _, address := range addresses {
		if address.GeoType == "location" || address.GeoType == "district" {
			parts = append(parts, address.FullName)
		}
	}

	return strings.Join(parts, ", ")
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

func saveStatistic(db *sql.DB, timestamp time.Time, statistic []flatStatItem) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("can't begin statistic tx: %w", err)
	}
	defer tx.Rollback()

	batch, err := tx.Prepare("INSERT INTO flat_median_price (date_time, location, category, rooms_count, price_per_meter) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("can't prepare statistic SQL: %w", err)
	}

	for _, row := range statistic {
		_, err := batch.Exec(timestamp.UTC(), row.Location, row.Category, row.RoomsCount, row.MedianPrice)
		if err != nil {
			return fmt.Errorf("can't write statistic row: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("can't write statistic data: %w", err)
	}

	return nil
}
