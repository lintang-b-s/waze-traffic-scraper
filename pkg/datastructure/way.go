package datastructure

type Way struct {
	id int64 
	coordinates []Coordinate
}

func NewWay(id int64, coords []Coordinate) Way {
	return Way{
		id: id,
		coordinates: coords,
	}
}

func (w Way) GetID() int64 {
	return w.id
}

func (w Way) GetCoordinates() []Coordinate {
	return w.coordinates
}


