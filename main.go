package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/lintang-b-s/waze-traffic-scraper/pkg/logger"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/osmparser"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/scraper"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/spatialindex"
)

var (
	bbBottomLon = flag.Float64("bLon", 110.132, "traffic bounding box: bottom longitude")
	bbBottomLat = flag.Float64("bLat", -8.2618, "traffic bounding box: bottom latitude")
	bbTopLon    = flag.Float64("tLon", 110.9221, "traffic bounding box: bottom latitude")
	bbTopLat    = flag.Float64("tLat", -6.888, "traffic bounding box: bottom latitude")
)

func main() {
	flag.Parse()
	logger, err := logger.New()
	if err != nil {
		panic(err)
	}
	osmParser := osmparser.NewOSMParserV2()
	wazeUrl := fmt.Sprintf(`https://www.waze.com/live-map/api/georss?top=%.4f&bottom=%.4f&&left=%.4f&&right=%.4f&&env=row&types=traffic`, *bbTopLat, *bbBottomLat, *bbBottomLon, *bbTopLon)

	arcs := osmParser.Parse("./data/diy_solo_semarang.osm.pbf", logger)
	rt := spatialindex.NewRtree()
	rt.Build(arcs, 0.05, logger)
	scp := scraper.NewScraper(5000*time.Millisecond, 3*time.Millisecond, 81*time.Millisecond, 20*time.Second,
		10*time.Millisecond, 2, wazeUrl, 5, rt, logger)
	err =  scp.ScrapePeriodically("./data/waze_traffic_diy_solo_semarang", "./data/waze_metadata_diy_solo_semarang")
	if err != nil {
		panic(err )
	}
}
