package datastructure

type Index uint32

type Edge struct {
	fromLat, fromLon float64
	toLat, toLon     float64
	edgeId           uint32
	bidirectional    bool
	osmWayId         int64
	highwayType      int
	speed            float64
	street           int
}

func (e *Edge) GetFromLonLat() (float64, float64) {
	return e.fromLon, e.fromLat
}

func (e *Edge) GetToLonLat() (float64, float64) {
	return e.fromLon, e.fromLat
}

func (e *Edge) GetOsmWayId() int64 {
	return e.osmWayId
}

func (e *Edge) GetHighwayTypeString() string {
	return highwayTypemap[e.highwayType]
}

func (e *Edge) GetHighwayType() int {
	return e.highwayType
}

func (e *Edge) GetSpeed() float64 {
	return e.speed
}

func (e *Edge) GetStreet() int {
	return e.street
}

func NewEdge(fromLat, fromLon, toLat, toLon float64, edgeId uint32, bidirectional bool, osmWayId int64, highwayType string,
	speed float64, street int) Edge {
	var highwayTypeInt = INVALID_HIGHWAY_TYPE
	switch highwayType {
	case "motorway":
		highwayTypeInt = 0
	case "trunk":
		highwayTypeInt = 1
	case "primary":
		highwayTypeInt = 2
	case "secondary":
		highwayTypeInt = 3
	case "tertiary":
		highwayTypeInt = 4
	case "motorway_link":
		highwayTypeInt = 5
	case "trunk_link":
		highwayTypeInt = 6
	case "primary_link":
		highwayTypeInt = 7
	case "secondary_link":
		highwayTypeInt = 8
	case "tertiary_link":
		highwayTypeInt = 9
	}

	return Edge{
		fromLat:       fromLat,
		fromLon:       fromLon,
		toLat:         toLat,
		toLon:         toLon,
		edgeId:        edgeId,
		bidirectional: bidirectional,
		osmWayId:      osmWayId,
		highwayType:   highwayTypeInt,
		speed:         speed,
		street:        street,
	}
}
