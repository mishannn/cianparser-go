package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/mishannn/cianparser-go/internal/cian"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func upMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("clickhouse"); err != nil {
		return fmt.Errorf("can't set dialect for migrations: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("can't up migrations: %w", err)
	}

	return nil
}

func newHttpClient() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("can't create cookie jar: %s", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar:       jar,
		Transport: transport,
	}
}

func main() {
	var configFilePath string
	flag.StringVar(&configFilePath, "c", "config.yaml", "config file path")

	var geojsonFilePath string
	flag.StringVar(&geojsonFilePath, "f", "polygon.geojson", "geojson file path")

	flag.Parse()

	cfg, err := newConfig(configFilePath)
	if err != nil {
		log.Fatalf("can't read config: %s", err)
	}

	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{cfg.Database.Address},
		Auth: clickhouse.Auth{
			Database: cfg.Database.Database,
			Username: cfg.Database.Username,
			Password: cfg.Database.Password,
		},
	})
	err = upMigrations(db)
	if err != nil {
		log.Fatalf("can't up migrations: %s", err)
	}

	httpClient := newHttpClient()

	geojson, err := os.ReadFile(geojsonFilePath)
	if err != nil {
		log.Fatalf("can't read polygon file: %s", err)
	}

	parser := cian.NewParser(httpClient, cfg.Rucaptcha.APIKey, string(geojson), cfg.Cian.SearchType, cfg.Cian.SearchQuery, cfg.Cian.MaxCellSizeMeters, cfg.Cian.MaxWorkersCollectIds, cfg.Cian.MaxWorkersCollectOffers)

	offerIDs, err := parser.GetOfferIDs()
	if err != nil {
		log.Fatalf("can't get offer ids: %s", err)
	}

	offers, err := parser.GetOffers(offerIDs)
	if err != nil {
		log.Fatalf("can't get offers: %s", err)
	}

	flatStat := getFlatStatistic(offers)

	err = saveStatistic(db, time.Now(), flatStat)
	if err != nil {
		log.Fatalf("can't save statistic: %s", err)
	}

	log.Println("statistic collected and saved")
}
