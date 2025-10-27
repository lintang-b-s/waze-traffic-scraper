package datastructure

type WayTraffic struct {
	way   Way
	speed float64
}

func NewWayTraffic(way Way, speed float64) WayTraffic {
	return WayTraffic{
		way:   way,
		speed: speed,
	}
}

func (wt WayTraffic) GetWay() Way{
	return wt.way
}

func (wt WayTraffic) GetSpeed() float64 {
	return wt.speed
}