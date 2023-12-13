package main

import (
	"fmt"
	"os"

	"github.com/mishannn/cianparser-go/internal/cian"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Cian struct {
		SearchType              string                        `yaml:"search_type"`
		SearchQuery             map[string]cian.JSONQueryItem `yaml:"search_query"`
		MaxCellSizeMeters       float64                       `yaml:"max_cell_size_meters"`
		MaxWorkersCollectIds    int                           `yaml:"max_workers_collect_ids"`
		MaxWorkersCollectOffers int                           `yaml:"max_workers_collect_offers"`
	} `yaml:"cian"`
	Rucaptcha struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"rucaptcha"`
	Database struct {
		Address  string `yaml:"address"`
		Database string `yaml:"database"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"database"`
}

func newConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't open config file: %w", err)
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&config); err != nil {
		return nil, fmt.Errorf("can't parse config file: %w", err)
	}

	return config, nil
}
