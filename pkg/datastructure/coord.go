package datastructure

type Coordinate struct {
	lat float64
	lon float64
}

func NewCoordinate( lon,lat float64) Coordinate {	
	return Coordinate{
		lat: lat,
		lon: lon,
	}
}

func (c Coordinate) GetLonLat() (float64, float64) {
	return c.lon, c.lat
}