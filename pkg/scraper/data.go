package scraper

import "github.com/lintang-b-s/waze-traffic-scraper/pkg/datastructure"

type wazeResponse struct {
	EndTimeMillis   int64     `json:"endTimeMillis"`
	StartTimeMillis int64     `json:"startTimeMillis"`
	StartTime       string    `json:"startTime"`
	EndTime         string    `json:"endTime"`
	Jams            []wazeJam `json:"jams"`
}

type wazeJam struct {
	Country           string         `json:"country"`
	City              string         `json:"city"`
	Line              []wazePoint    `json:"line"`
	SpeedKMH          float64        `json:"speedKMH"`
	Type              string         `json:"type"`
	BlockingAlertID   int64          `json:"blockingAlertID"`
	BlockExpiration   int64          `json:"blockExpiration"`
	UUID              int64          `json:"uuid"`
	EndNode           string         `json:"endNode"`
	Speed             float64        `json:"speed"`
	Segments          []wazeSegment  `json:"segments"`
	Street            string         `json:"street"`
	ID                int64          `json:"id"`
	BlockStartTime    int64          `json:"blockStartTime"`
	BlockUpdate       int64          `json:"blockUpdate"`
	Severity          int            `json:"severity"`
	Level             int            `json:"level"`
	BlockType         string         `json:"blockType"`
	Length            int            `json:"length"`
	TurnType          string         `json:"turnType"`
	BlockingAlertUuid string         `json:"blockingAlertUuid"`
	RoadType          int            `json:"roadType"`
	Delay             int            `json:"delay"`
	BlockDescription  string         `json:"blockDescription"`
	UpdateMillis      int64          `json:"updateMillis"`
	CauseAlert        wazeCauseAlert `json:"causeAlert"`
	PubMillis         int64          `json:"pubMillis"`
}

type wazeSegment struct {
	FromNode  int  `json:"fromNode"`
	ID        int  `json:"ID"`
	ToNode    int  `json:"toNode"`
	IsForward bool `json:"isForward"`
}

type wazePoint struct {
	Longitude float64 `json:"x"`
	Latitude  float64 `json:"y"`
}

type wazeCauseAlert struct {
	Country                  string    `json:"country"`
	City                     string    `json:"city"`
	ReportRating             int       `json:"reportRating"`
	ReportByMunicipalityUser string    `json:"reportByMunicipalityUser"`
	Reliability              int       `json:"reliability"`
	Type                     string    `json:"type"`
	FromNodeID               int       `json:"fromNodeId"`
	UUID                     string    `json:"uuid"`
	Speed                    int       `json:"speed"`
	ReportMood               int       `json:"reportMood"`
	Subtype                  string    `json:"subtype"`
	Provider                 string    `json:"provider"`
	Street                   string    `json:"street"`
	ProviderID               string    `json:"providerId"`
	AdditionalInfo           string    `json:"additionalInfo"`
	ToNodeID                 int       `json:"toNodeId"`
	ID                       string    `json:"id"`
	ReportBy                 string    `json:"reportBy"`
	Inscale                  bool      `json:"inscale"`
	Confidence               int       `json:"confidence"`
	RoadType                 int       `json:"roadType"`
	Magvar                   int       `json:"magvar"`
	WazeData                 string    `json:"wazeData"`
	ReportDescription        string    `json:"reportDescription"`
	Location                 wazePoint `json:"location"`
	PubMillis                int64     `json:"pubMillis"`
}

type edgesWithDistance struct {
	edge datastructure.Edge
	dist float64
}

func (e edgesWithDistance) getDist() float64 {
	return e.dist
}

func (e edgesWithDistance) getEdge() datastructure.Edge {
	return e.edge
}

func NewEdgesWithDistance(edge datastructure.Edge, dist float64) edgesWithDistance {
	return edgesWithDistance{edge, dist}
}

type osmwayTrafficData struct {
	id       int64
	speedKMH float64
	street   string
	city     string
	endNode  string
}

func (o osmwayTrafficData) getSpeed() float64 {
	return o.speedKMH
}

func (o osmwayTrafficData) getStreet() string {
	return o.street
}

func (o osmwayTrafficData) getCity() string {
	return o.city
}

func (o osmwayTrafficData) getEndNode() string {
	return o.endNode
}

func NewOsmWayTrafficData(id int64, speedKMH float64,
	street string, city string, endNode string) osmwayTrafficData {
	return osmwayTrafficData{id, speedKMH, street, city, endNode}
}
