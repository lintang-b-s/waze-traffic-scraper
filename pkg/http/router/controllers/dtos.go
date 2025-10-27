package controllers

import "github.com/lintang-b-s/waze-traffic-scraper/pkg/datastructure"

type trafficResponse struct {
	Traffics []TrafficData `json:"traffics"`
}

type TrafficData struct {
	Way   Way     `json:"way"`
	Speed float64 `json:"speed"`
}

type Way struct {
	Id          int64        `json:"id"`
	Coordinates []Coordinate `json:"coordinates"`
}

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func NewTrafficResponse(traffics []datastructure.WayTraffic) trafficResponse {
	var response trafficResponse

	for _, wt := range traffics {
		way := wt.GetWay()
		wayResp := Way{
			Id:          way.GetID(),
			Coordinates: []Coordinate{},
		}
		for _, coord := range way.GetCoordinates() {
			lon, lat := coord.GetLonLat()
			wayResp.Coordinates = append(wayResp.Coordinates, Coordinate{
				Lon: lon,
				Lat: lat,
			})
		}
		response.Traffics = append(response.Traffics, TrafficData{
			Way:   wayResp,
			Speed: wt.GetSpeed(),
		})
	}

	return response
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
