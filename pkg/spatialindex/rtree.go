package spatialindex

import (
	"math"

	"github.com/lintang-b-s/waze-traffic-scraper/pkg/datastructure"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/geo"
	"github.com/tidwall/rtree"
	"go.uber.org/zap"
)

type Rtree struct {
	tr *rtree.RTreeG[datastructure.Edge]
}

func NewRtree() *Rtree {
	var tr rtree.RTreeG[datastructure.Edge]
	return &Rtree{
		tr: &tr,
	}
}
func (rt *Rtree) Build(edges []datastructure.Edge, boundingBoxRadius float64, log *zap.Logger) {
	for i, edge := range edges {
		percentage := ((i + 1) * 100) / len(edges)
		if percentage%10 == 0 && percentage > 0 {
			log.Info("Building R-tree spatial index...", zap.Int("progress", percentage))
		}
		if edge.GetHighwayType() == datastructure.INVALID_HIGHWAY_TYPE {
			// skip certain road segment type (yang gak ada data traffic)
			continue
		}

		fromLon, fromLat := edge.GetFromLonLat()
		toLon, toLat := edge.GetToLonLat()
		lowerFromLat, lowerFromLon := geo.GetDestinationPoint(fromLat, fromLon, 225, boundingBoxRadius)
		upperFromLat, upperFromLon := geo.GetDestinationPoint(fromLat, fromLon, 45, boundingBoxRadius)

		lowerToLat, lowerToLon := geo.GetDestinationPoint(toLat, toLon, 225, boundingBoxRadius)
		upperToLat, upperToLon := geo.GetDestinationPoint(toLat, toLon, 45, boundingBoxRadius)

		minLat := math.Min(lowerFromLat, lowerToLat)
		minLon := math.Min(lowerFromLon, lowerToLon)
		maxLat := math.Max(upperFromLat, upperToLat)
		maxLon := math.Max(upperFromLon, upperToLon)

		rt.tr.Insert([2]float64{minLon, minLat}, [2]float64{maxLon, maxLat},
			edge)

	}
	log.Info("R-tree spatial index built.")
}

// SearchWithinRadius search for all arc endpoints within radius (in km) from the query point (qLat, qLon)
func (rt *Rtree) SearchWithinRadius(qLon, qLat, radius float64) []datastructure.Edge {
	lowerLat, lowerLon := geo.GetDestinationPoint(qLat, qLon, 225, radius)
	upperLat, upperLon := geo.GetDestinationPoint(qLat, qLon, 45, radius)

	results := make([]datastructure.Edge, 0, 10)
	rt.tr.Search([2]float64{lowerLon, lowerLat}, [2]float64{upperLon, upperLat},
		func(min, max [2]float64, data datastructure.Edge) bool {
			results = append(results, data)
			if len(results) >= 20 {
				return false
			}
			return true
		})
	return results
}
