package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/lintang-b-s/waze-traffic-scraper/pkg/logger"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/osmparser"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/scraper"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/spatialindex"
)

var (
	bbBottomLon    = flag.Float64("bLon", 110.132, "traffic bounding box: bottom longitude")
	bbBottomLat    = flag.Float64("bLat", -8.2618, "traffic bounding box: bottom latitude")
	bbTopLon       = flag.Float64("tLon", 110.9221, "traffic bounding box: top longitude")
	bbTopLat       = flag.Float64("tLat", -6.888, "traffic bounding box: top latitude")
	osmFile        = flag.String("osm", "./data/diy_solo_semarang.osm.pbf", "path to osm pbf file")
	outputFileName = flag.String("out", "diy_solo_semarang", "traffic output file name")
)

func main() {
	flag.Parse()
	logger, err := logger.New()
	if err != nil {
		panic(err)
	}
	osmParser := osmparser.NewOSMParserV2()
	wazeUrl := fmt.Sprintf(`https://www.waze.com/live-map/api/georss?top=%.4f&bottom=%.4f&&left=%.4f&&right=%.4f&&env=row&types=traffic`, *bbTopLat, *bbBottomLat, *bbBottomLon, *bbTopLon)

	arcs, waySpeed := osmParser.Parse(*osmFile, logger)
	rt := spatialindex.NewRtree()
	rt.Build(arcs, 0.03, logger)

	// --scraper--
	scp := scraper.NewScraper(4000*time.Millisecond, 3*time.Millisecond, 81*time.Millisecond, 20*time.Second,
		10*time.Millisecond, 2, wazeUrl, 5, rt, logger, waySpeed, osmParser.GetStreetIdMap(), osmParser.GetWayMap())
	err = scp.ScrapePeriodically(fmt.Sprintf("./data/waze_traffic_%s.csv", *outputFileName),
		fmt.Sprintf("./data/waze_metadata_%s.csv", *outputFileName))
	if err != nil {
		panic(err)
	}

	// --server--
	// scp := scraper.NewScraper(4000*time.Millisecond, 3*time.Millisecond, 81*time.Millisecond, 20*time.Second,
	// 	10*time.Millisecond, 2, wazeUrl, 5, rt, logger, waySpeed, osmParser.GetStreetIdMap(), osmParser.GetWayMap())
	// api := http.NewServer(logger)
	// trafficService := usecases.NewTrafficService(logger, scp)
	// ctx, cleanup, err := NewContext()
	// if err != nil {
	// 	panic(err)
	// }
	// api.Use(ctx,
	// 	logger, false, trafficService)

	// signal := http.GracefulShutdown()

	// logger.Info("Navigatorx Routing Engine Server Stopped", zap.String("signal", signal.String()))
	// cleanup()

}

func NewContext() (context.Context, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	cb := func() {
		cancel()
	}

	return ctx, cb, nil
}
