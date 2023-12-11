package cian

type GetOffersByIDsRequestBody struct {
	CianOfferIDS []int64        `json:"cianOfferIds"`
	JSONQuery    map[string]any `json:"jsonQuery"`
}

type GetOffersByIDsResponseBody struct {
	OffersSerialized []Offer `json:"offersSerialized"`
}

// Offer has only important values
type Offer struct {
	Geo          Geo          `json:"geo"`          // need
	Category     string       `json:"category"`     // need
	RoomsCount   int          `json:"roomsCount"`   // need
	TotalArea    string       `json:"totalArea"`    // need
	BargainTerms BargainTerms `json:"bargainTerms"` // need
}

// BargainTerms has only important values
type BargainTerms struct {
	PriceRur float64 `json:"priceRur"`
}

// Geo has only important values
type Geo struct {
	// Coordinates     Coordinates   `json:"coordinates"`
	Address []Address `json:"address"`
}

type Address struct {
	FullName string `json:"fullName"`
	// Type     string `json:"type"`
	GeoType string `json:"geoType"`
}
