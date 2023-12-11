package cian

import "github.com/twpayne/go-geos"

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Bounds struct {
	BottomRight Coordinates `json:"bottomRight"`
	TopLeft     Coordinates `json:"topLeft"`
}

type Cluster struct {
	Coordinates        Coordinates `json:"coordinates"`
	Bbox               Bounds      `json:"bbox"`
	Geohash            string      `json:"geohash"`
	Count              int         `json:"count"`
	MinPrice           float64     `json:"minPrice"`
	MaxPrice           float64     `json:"maxPrice"`
	HasNewobject       bool        `json:"hasNewobject"`
	FavoriteIds        []int64     `json:"favoriteIds"`
	Subdomain          string      `json:"subdomain"`
	ClusterOfferIds    []int64     `json:"clusterOfferIds"`
	IsViewed           bool        `json:"isViewed"`
	IsAnyFromDeveloper bool        `json:"isAnyFromDeveloper"`
}

type GetClustersRequestBody struct {
	Zoom      int            `json:"zoom"`
	Bbox      []Bounds       `json:"bbox"`
	JSONQuery map[string]any `json:"jsonQuery"`
}

type GetClustersResponseBody struct {
	JSONQuery            map[string]any `json:"jsonQuery"`
	QueryString          string         `json:"queryString"`
	NonGeoQueryString    string         `json:"nonGeoQueryString"`
	ExtendedJSONQuery    map[string]any `json:"extendedJsonQuery"`
	ExtendedQueryString  string         `json:"extendedQueryString"`
	IsNewobject          bool           `json:"isNewobject"`
	NewbuildingsPolygons []any          `json:"newbuildingsPolygons"`
	Bbox                 Bounds         `json:"bbox"`
	Precision            int            `json:"precision"`
	Extended             []any          `json:"extended"`
	Filtered             []Cluster      `json:"filtered"`
	OffersCount          int            `json:"offersCount"`
}

func GeosBoundsToCianBounds(bounds *geos.Bounds) Bounds {
	return Bounds{
		TopLeft:     Coordinates{Lat: bounds.MaxY, Lng: bounds.MinX},
		BottomRight: Coordinates{Lat: bounds.MinY, Lng: bounds.MaxX},
	}
}
