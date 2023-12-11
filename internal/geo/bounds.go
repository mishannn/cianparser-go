package geo

import (
	"fmt"
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/project"
	"github.com/twpayne/go-geos"
)

func geosBoundsToOrbBound(bounds *geos.Bounds) orb.Bound {
	return orb.Bound{
		Min: orb.Point{bounds.MinX, bounds.MinY},
		Max: orb.Point{bounds.MaxX, bounds.MaxY},
	}
}

func orbBoundToGeosBounds(bound orb.Bound) *geos.Bounds {
	return geos.NewBounds(bound.Min[0], bound.Min[1], bound.Max[0], bound.Max[1])
}

func reprojectBounds(bounds *geos.Bounds, projection orb.Projection) *geos.Bounds {
	return orbBoundToGeosBounds(project.Bound(geosBoundsToOrbBound(bounds), projection))
}

func getCellBoundsByGeom(geom *geos.Geom, cellSize float64) []*geos.Bounds {
	bounds4326 := geom.Bounds()
	bounds3857 := reprojectBounds(bounds4326, project.WGS84.ToMercator)

	height := bounds3857.MaxY - bounds3857.MinY
	width := bounds3857.MaxX - bounds3857.MinX

	stepy := height / math.Ceil(height/cellSize)
	stepx := width / math.Ceil(width/cellSize)

	cellBoundsList3857 := make([]*geos.Bounds, 0)

	y1 := bounds3857.MaxY
	x1 := bounds3857.MinX

	for y1 > bounds3857.MinY {
		// Add step to lng
		y2 := y1 - stepy

		// Check if lng smaller than needed
		if y2 < bounds3857.MinY {
			y2 = bounds3857.MinY
		}

		for x1 < bounds3857.MaxX {
			// Add step to lng
			x2 := x1 + stepx

			// Check if lng bigger than needed
			if x2 > bounds3857.MaxX {
				x2 = bounds3857.MaxX
			}

			cellBoundsList3857 = append(cellBoundsList3857, geos.NewBounds(x1, y2, x2, y1))
			x1 = x2
		}

		y1 = y2
		x1 = bounds3857.MinX
	}

	cellBoundsList4326 := make([]*geos.Bounds, 0, len(cellBoundsList3857))

	for _, cellBounds3857 := range cellBoundsList3857 {
		cellBounds4326 := reprojectBounds(cellBounds3857, project.Mercator.ToWGS84)

		if cellBounds4326.Geom().Intersects(geom) {
			cellBoundsList4326 = append(cellBoundsList4326, cellBounds4326)
		}
	}

	return cellBoundsList4326
}

func GetCellBoundsListByGeoJSON(geojson string, cellSize float64) ([]*geos.Bounds, error) {
	geom, err := geos.NewGeomFromGeoJSON(geojson)
	if err != nil {
		return nil, fmt.Errorf("can't parse geojson: %w", err)
	}

	return getCellBoundsByGeom(geom, cellSize), nil
}
