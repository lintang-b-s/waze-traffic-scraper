package scraper

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"math/rand"

	"github.com/gojek/heimdall/v7"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/geo"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/spatialindex"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/util"
	"go.uber.org/zap"
)

type Scraper struct {
	initialTimeout        time.Duration
	maxTimeout            time.Duration
	requestTimeout        time.Duration
	exponentFactor        float64
	maximumJitterInterval time.Duration
	period                time.Duration
	url                   string
	retryCount            int
	rt                    *spatialindex.Rtree
	log                   *zap.Logger
	osmWayDefaultSpeed    map[int64]float64
	streetIdMap           *util.IDMap
}

func NewScraper(requestTimeout, initialTimeout, maxTimeout, period, maximumJitterInterval time.Duration,
	exponentFactor float64, url string, retryCount int, rt *spatialindex.Rtree, log *zap.Logger, waySpeed map[int64]float64,
	streetIdMap *util.IDMap) *Scraper {
	return &Scraper{
		initialTimeout:        initialTimeout,
		maxTimeout:            maxTimeout,
		requestTimeout:        requestTimeout,
		maximumJitterInterval: maximumJitterInterval,
		exponentFactor:        exponentFactor,
		url:                   url,
		retryCount:            retryCount,
		rt:                    rt,
		log:                   log,
		osmWayDefaultSpeed:    waySpeed,
		period:                period,
		streetIdMap:           streetIdMap,
	}
}

func (sc *Scraper) getInitialTimeout() time.Duration {
	return sc.initialTimeout
}

func (sc *Scraper) getRequestTimeout() time.Duration {
	return sc.requestTimeout
}

func (sc *Scraper) getMaxTimeout() time.Duration {
	return sc.maxTimeout
}

func (sc *Scraper) getMaximumJitterInterval() time.Duration {
	return sc.maximumJitterInterval
}

func (sc *Scraper) getPeriod() time.Duration {
	return sc.period
}

func (sc *Scraper) getExponentFactor() float64 {
	return sc.exponentFactor
}

func (sc *Scraper) getURL() string {
	return sc.url
}

func (sc *Scraper) getRetryCount() int {
	return sc.retryCount
}

func (sc *Scraper) scrape() (wazeResponse, error) {

	backoff := heimdall.NewExponentialBackoff(sc.getInitialTimeout(), sc.getMaxTimeout(), sc.getExponentFactor(), sc.getMaximumJitterInterval())
	retrier := heimdall.NewRetrier(backoff)

	client := httpclient.NewClient(
		httpclient.WithHTTPTimeout(sc.getRequestTimeout()),
		httpclient.WithRetrier(retrier),
		httpclient.WithRetryCount(sc.getRetryCount()),
	)

	httpHeaders := make(http.Header)
	httpHeaders.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	httpHeaders.Set("Accept", "application/json")
	resp, err := client.Get(sc.getURL(), httpHeaders)
	if err != nil {
		return wazeResponse{}, errors.New(fmt.Sprintf("failed to receive response from api after %d times retry: %s",
			sc.getRetryCount(), err.Error()))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return wazeResponse{}, errors.New(fmt.Sprintf("failed read response data: %s", err.Error()))

	}
	var data wazeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return wazeResponse{}, errors.New(fmt.Sprintf("failed parsing waze response data: %s", err.Error()))
	}

	return data, nil
}

// scrapePeriodically. scrape every period seconds + rand(0,maxJitterInterval)
func (sc *Scraper) ScrapePeriodically(trafficCsvFilePath, metadataCsvFilePath string) error {
	for {
		jitter := time.Duration(rand.Int63n(int64(sc.getMaximumJitterInterval())))
		sleepDuration := sc.getPeriod() + jitter
		time.Sleep(sleepDuration)

		data, err := sc.scrape()
		if err != nil {
			return err
		}
		err = sc.writeTrafficDataToCSV(data, trafficCsvFilePath, metadataCsvFilePath)
		if err != nil {
			return err
		}
		sc.log.Info("scraping waze traffic...", zap.Time("timestamp", time.Now()))
	}
}

func (sc *Scraper) writeTrafficDataToCSV(data wazeResponse, trafficCsvFilePath, metadataCsvFilePath string) error {
	affectedWays := make(map[int64]osmwayTrafficData)
	for _, jam := range data.Jams {
		if jam.CauseAlert.Type != "" || jam.BlockType != "" { // skip road segment block event
			continue
		}
		for _, coord := range jam.Line {
			edges := sc.rt.SearchWithinRadius(coord.Longitude, coord.Latitude,
				0.15) // 150 meters radius
			if len(edges) == 0 {
				continue
			}

			edgeDists := make([]edgesWithDistance, 0)
			for _, edge := range edges {
				fromLon, fromLat := edge.GetFromLonLat()
				toLon, toLat := edge.GetToLonLat()
				midLon, midLat := geo.MidPoint(fromLon, fromLat, toLon, toLat)
				edgeDists = append(edgeDists, NewEdgesWithDistance(edge,
					geo.CalculateHaversineDistance(coord.Longitude, coord.Latitude,
						midLon, midLat)))
			}
			util.QuickSortGIdx(edgeDists, func(j, pivotIdx int) bool {
				return edgeDists[j].getDist() < edgeDists[pivotIdx].getDist()
			})

			nearestEdge := edgeDists[0].getEdge()
			affectedWays[nearestEdge.GetOsmWayId()] = NewOsmWayTrafficData(
				nearestEdge.GetOsmWayId(), float64(jam.SpeedKMH),
				jam.Street, jam.City, jam.EndNode, sc.streetIdMap.GetStr(nearestEdge.GetStreet()),
			)
		}
	}

	// traffic speed data
	err := sc.writeTrafficSpeedDataToCSV(affectedWays, trafficCsvFilePath)
	if err != nil {
		return err
	}
	// metadata
	return sc.writeMetadataToCSV(affectedWays, metadataCsvFilePath)
}

func (sc *Scraper) writeTrafficSpeedDataToCSV(affectedWays map[int64]osmwayTrafficData, trafficCsvFilePath string) error {
	var headers []string
	var records [][]string

	fileExists := false
	if _, err := os.Stat(trafficCsvFilePath); err == nil {
		fileExists = true
	}

	if fileExists {
		f, err := os.Open(trafficCsvFilePath)
		if err != nil {
			return err
		}
		r := csv.NewReader(f)
		records, err = r.ReadAll()
		f.Close()
		if err != nil {
			return err
		}
		headers = records[0]
	} else {
		headers = []string{"timestamp"}
	}

	for osmWayId := range affectedWays {
		found := false
		for _, h := range headers {
			if h == strconv.FormatInt(osmWayId, 10) {
				found = true
				break
			}
		}
		if !found {
			headers = append(headers, strconv.FormatInt(osmWayId, 10))
		}
	}

	row := make([]string, len(headers))
	row[0] = time.Now().Format(time.RFC3339)
	for i, h := range headers {
		if h == "timestamp" {
			continue
		}
		osmWayId, _ := strconv.ParseInt(h, 10, 64)
		if speed, ok := affectedWays[osmWayId]; ok {
			row[i] = fmt.Sprintf("%.2f", speed.getSpeed())
		} else {
			row[i] = fmt.Sprintf("%.2f", sc.osmWayDefaultSpeed[osmWayId])
		}
	}

	records = append(records, row)
	f, err := os.Create(trafficCsvFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(headers); err != nil {
		return err
	}
	startId := 0
	if len(records) > 1 {
		startId = 1
	}
	for _, rec := range records[startId:] {
		if len(row) > len(rec) {
			oldId := len(rec)
			rec = append(rec, make([]string, len(row)-len(rec))...)

			for i := oldId; i < len(row); i++ {
				// isi pakai default speed value...
				h := headers[i]
				osmWayId, _ := strconv.ParseInt(h, 10, 64)
				rec[i] = fmt.Sprintf("%.2f", sc.osmWayDefaultSpeed[osmWayId])
			}

		}

		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (sc *Scraper) writeMetadataToCSV(affectedWays map[int64]osmwayTrafficData, csvPath string) error {
	existing := make(map[int64]bool)
	fileExists := false
	if _, err := os.Stat(csvPath); err == nil {
		fileExists = true
	}

	var existingRecords [][]string
	if fileExists {
		f, err := os.Open(csvPath)
		if err != nil {
			return err
		}
		defer f.Close()

		r := csv.NewReader(f)
		existingRecords, err = r.ReadAll()
		if err != nil {
			return err
		}

		for _, rec := range existingRecords[1:] {
			if len(rec) > 0 {
				if id, err := strconv.ParseInt(rec[0], 10, 64); err == nil {
					existing[id] = true
				}
			}
		}
	}

	f, err := os.OpenFile(csvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if !fileExists {
		if err := w.Write([]string{"osm_way_id", "street", "city", "end_node", "osm_way_street_name"}); err != nil {
			return err
		}
	}

	for id, info := range affectedWays {
		if _, exists := existing[id]; exists {
			continue
		}
		rec := []string{
			strconv.FormatInt(id, 10),
			info.getStreet(),
			info.getCity(),
			info.getEndNode(),
			info.getOsmStreet(),
		}
		if err := w.Write(rec); err != nil {
			return err
		}
		existing[id] = true
	}

	return nil
}
