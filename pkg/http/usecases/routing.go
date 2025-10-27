package usecases

import (
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/datastructure"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/scraper"
	"go.uber.org/zap"
)

type TrafficService struct {
	log     *zap.Logger
	scraper *scraper.Scraper
}

func NewTrafficService(log *zap.Logger, scraper *scraper.Scraper) *TrafficService {
	return &TrafficService{
		log:     log,
		scraper: scraper,
	}
}

func (rs *TrafficService) GetRealtimeTraffic() ([]datastructure.WayTraffic, error) {
	return rs.scraper.Scrape()
}
