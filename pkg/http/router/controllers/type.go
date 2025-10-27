package controllers

import "github.com/lintang-b-s/waze-traffic-scraper/pkg/datastructure"

type TrafficService interface {
	GetRealtimeTraffic() ([]datastructure.WayTraffic, error)
}
