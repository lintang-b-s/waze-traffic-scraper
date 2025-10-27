package osmparser

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/lintang-b-s/waze-traffic-scraper/pkg/datastructure"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/geo"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/util"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"go.uber.org/zap"
)

type node struct {
	id    int64
	coord nodeCoord
}

type nodeCoord struct {
	lat float64
	lon float64
}

type restriction struct {
	via             uint32
	to              int64
	turnRestriction TurnRestriction
}

type osmWay struct {
	id     int64
	nodes  []int64
	oneWay bool
}

type OsmParser struct {
	wayNodeMap        map[int64]NodeType
	wayMap            map[int64]datastructure.Way
	relationMemberMap map[int64]struct{}
	acceptedNodeMap   map[int64]nodeCoord
	barrierNodes      map[int64]bool
	nodeTag           map[int64]map[int]int
	tagStringIdMap    *util.IDMap
	nodeIDMap         map[int64]uint32
	nodeToOsmId       map[uint32]int64
	maxNodeID         int64
	restrictions      map[int64][]restriction // wayId -> list of restrictions
	ways              map[int64]osmWay
	streetNameIdMap   *util.IDMap
}

func NewOSMParserV2() *OsmParser {
	return &OsmParser{
		wayNodeMap:        make(map[int64]NodeType),
		relationMemberMap: make(map[int64]struct{}),
		acceptedNodeMap:   make(map[int64]nodeCoord),
		barrierNodes:      make(map[int64]bool),
		nodeTag:           make(map[int64]map[int]int),
		tagStringIdMap:    util.NewIdMap(),
		nodeIDMap:         make(map[int64]uint32),
		nodeToOsmId:       make(map[uint32]int64),
		streetNameIdMap:   util.NewIdMap(),
		wayMap:            make(map[int64]datastructure.Way),
	}
}
func (o *OsmParser) GetTagStringIdMap() *util.IDMap {
	return o.tagStringIdMap
}

func (o *OsmParser) GetStreetIdMap() *util.IDMap {
	return o.streetNameIdMap
}

func (o *OsmParser) GetWayMap() map[int64]datastructure.Way {
	return o.wayMap
}

func (p *OsmParser) Parse(mapFile string, logger *zap.Logger) ([]datastructure.Edge, map[int64]float64) {

	f, err := os.Open(mapFile)

	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	waySpeed := make(map[int64]float64)

	scanner := osmpbf.New(context.Background(), f, 0)
	// must not be parallel
	countWays := 0
	for scanner.Scan() {
		o := scanner.Object()

		tipe := o.ObjectID().Type()

		switch tipe {
		case osm.TypeWay:
			{
				way := o.(*osm.Way)
				if len(way.Nodes) < 2 {
					continue
				}

				if !acceptOsmWay(way) {
					continue
				}

				countWays++

				for i, node := range way.Nodes {
					if _, ok := p.wayNodeMap[int64(node.ID)]; !ok {
						if i == 0 || i == len(way.Nodes)-1 {
							p.wayNodeMap[int64(node.ID)] = END_NODE
						} else {
							p.wayNodeMap[int64(node.ID)] = BETWEEN_NODE
						}
					} else {
						p.wayNodeMap[int64(node.ID)] = JUNCTION_NODE
					}
				}
			}
		case osm.TypeNode:
			{
			}
		}
	}
	scanner.Close()

	edgeSet := make(map[uint32]map[uint32]struct{})

	f.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatal(err)
	}
	scanner = osmpbf.New(context.Background(), f, 0)
	//must not be parallel
	defer scanner.Close()

	scannedEdges := make([]datastructure.Edge, 0)
	p.ways = make(map[int64]osmWay, countWays)

	streetDirection := make(map[string][2]bool)
	countWays = 0
	countNodes := 0
	for scanner.Scan() {
		o := scanner.Object()

		tipe := o.ObjectID().Type()

		switch tipe {
		case osm.TypeWay:
			{
				way := o.(*osm.Way)
				if len(way.Nodes) < 2 {
					continue
				}

				if !acceptOsmWay(way) {
					continue
				}
				if (countWays+1)%100000 == 0 {
					logger.Sugar().Infof("processing openstreetmap ways: %d...", countWays+1)
				}
				countWays++

				p.processWay(way, streetDirection, edgeSet, &scannedEdges)

				wayExtraInfoData := wayExtraInfo{}
				okvf, okmvf, okvb, okmvb := getReversedOneWay(way)
				if val := way.Tags.Find("oneway"); val == "yes" || val == "-1" || okvf || okmvf || okvb || okmvb {
					wayExtraInfoData.oneWay = true
				}

				if way.Tags.Find("oneway") == "-1" || okvf || okmvf {
					// okvf / omvf = restricted/not allowed forward.
					wayExtraInfoData.forward = false

				} else {
					wayExtraInfoData.forward = true
				}
				wNodes := make([]int64, 0, len(way.Nodes))
				for _, node := range way.Nodes {
					nodeId := node.ID
					wNodes = append(wNodes, int64(nodeId))
				}
				p.ways[int64(way.ID)] = osmWay{
					nodes:  wNodes,
					oneWay: wayExtraInfoData.oneWay,
					id:     int64(way.ID),
				}
			}
		case osm.TypeNode:
			{

				if (countNodes+1)%100000 == 0 {
					logger.Sugar().Infof("processing openstreetmap nodes: %d...", countNodes+1)
				}
				countNodes++
				node := o.(*osm.Node)

				p.maxNodeID = max(p.maxNodeID, int64(node.ID))

				if _, ok := p.wayNodeMap[int64(node.ID)]; ok {
					p.acceptedNodeMap[int64(node.ID)] = nodeCoord{
						lat: node.Lat,
						lon: node.Lon,
					}
				}
				accessType := node.Tags.Find("access")
				barrierType := node.Tags.Find("barrier")

				if _, ok := acceptedBarrierType[barrierType]; ok && accessType == "no" && barrierType != "" {
					p.barrierNodes[int64(node.ID)] = true
				}

				for _, tag := range node.Tags {
					if strings.Contains(tag.Key, "created_by") ||
						strings.Contains(tag.Key, "source") ||
						strings.Contains(tag.Key, "note") ||
						strings.Contains(tag.Key, "fixme") {
						continue
					}
					tagID := p.tagStringIdMap.GetID(tag.Key)
					if _, ok := p.nodeTag[int64(node.ID)]; !ok {
						p.nodeTag[int64(node.ID)] = make(map[int]int)
					}
					p.nodeTag[int64(node.ID)][tagID] = p.tagStringIdMap.GetID(tag.Value)
					if strings.Contains(tag.Value, "traffic_signals") {
						p.nodeTag[int64(node.ID)][p.tagStringIdMap.GetID(TRAFFIC_LIGHT)] = 1
					}
				}

			}
		}
	}

	for _, edge := range scannedEdges {
		waySpeed[edge.GetOsmWayId()] = edge.GetSpeed()
	}

	for _, way := range p.ways {
		wCoords := make([]datastructure.Coordinate, 0)
		for _, nodeId := range way.nodes {
			node := p.acceptedNodeMap[int64(nodeId)]
			wCoords = append(wCoords, datastructure.NewCoordinate(node.lon,
				node.lat))
		}
		p.wayMap[way.id] = datastructure.NewWay(way.id, wCoords)
	}

	return scannedEdges, waySpeed
}

type wayExtraInfo struct {
	oneWay      bool
	forward     bool
	highwayType string
}

func newWayExtraInfo(oneWay, forward bool, highwayType string) wayExtraInfo {
	return wayExtraInfo{oneWay, forward, highwayType}
}

func (p *OsmParser) processWay(way *osm.Way,
	streetDirection map[string][2]bool, edgeSet map[uint32]map[uint32]struct{}, scannedEdges *[]datastructure.Edge) error {
	tempMap := make(map[string]string)
	name := way.Tags.Find("name")

	tempMap[STREET_NAME] = name

	refName := way.Tags.Find("ref")
	tempMap[STREET_REF] = refName

	maxSpeed := 0.0
	highwayTypeSpeed := 0.0

	wayExtraInfoData := wayExtraInfo{}
	okvf, okmvf, okvb, okmvb := getReversedOneWay(way)
	if val := way.Tags.Find("oneway"); val == "yes" || val == "-1" || okvf || okmvf || okvb || okmvb {
		wayExtraInfoData.oneWay = true
	}

	if way.Tags.Find("oneway") == "-1" || okvf || okmvf {
		// okvf / omvf = restricted/not allowed forward.
		wayExtraInfoData.forward = false

	} else {
		wayExtraInfoData.forward = true
	}

	if wayExtraInfoData.oneWay {
		if wayExtraInfoData.forward {
			streetDirection[name] = [2]bool{true, false} // {forward, backward}
		} else {
			streetDirection[name] = [2]bool{false, true}
		}
	} else {
		streetDirection[name] = [2]bool{true, true}
	}

	for _, tag := range way.Tags {
		switch tag.Key {
		case "junction":
			{
				tempMap[JUNCTION] = tag.Value
			}
		case "highway":
			{
				highwayTypeSpeed = roadTypeMaxSpeed2(tag.Value)
				wayExtraInfoData.highwayType = tag.Value

				if strings.Contains(tag.Value, "link") {
					tempMap[ROAD_CLASS_LINK] = tag.Value
				} else {
					tempMap[ROAD_CLASS] = tag.Value
				}
			}
		case "lanes":
			{
				tempMap[LANES] = tag.Value
			}
		case "maxspeed":
			{
				if strings.Contains(tag.Value, "mph") {

					currSpeed, err := strconv.ParseFloat(strings.Replace(tag.Value, " mph", "", -1), 64)
					if err != nil {
						return err
					}
					maxSpeed = currSpeed * 1.60934
				} else if strings.Contains(tag.Value, "km/h") {
					currSpeed, err := strconv.ParseFloat(strings.Replace(tag.Value, " km/h", "", -1), 64)
					if err != nil {
						return err
					}
					maxSpeed = currSpeed
				} else if strings.Contains(tag.Value, "knots") {
					currSpeed, err := strconv.ParseFloat(strings.Replace(tag.Value, " knots", "", -1), 64)
					if err != nil {
						return err
					}
					maxSpeed = currSpeed * 1.852
				} else {
					// without unit
					// dont use this
				}
			}

		}

	}

	if maxSpeed == 0 {
		maxSpeed = highwayTypeSpeed
	}
	if maxSpeed == 0 {
		maxSpeed = 30
	}

	waySegment := []node{}
	for _, wayNode := range way.Nodes {
		nodeCoord := p.acceptedNodeMap[int64(wayNode.ID)]
		nodeData := node{
			id:    int64(wayNode.ID),
			coord: nodeCoord,
		}
		if p.isJunctionNode(int64(nodeData.id)) {

			waySegment = append(waySegment, nodeData)
			p.processSegment(waySegment, tempMap, maxSpeed, wayExtraInfoData,
				edgeSet, scannedEdges, int64(way.ID))
			waySegment = []node{}

			waySegment = append(waySegment, nodeData)

		} else {
			waySegment = append(waySegment, nodeData)
		}

	}
	if len(waySegment) > 1 {
		p.processSegment(waySegment, tempMap, maxSpeed, wayExtraInfoData, edgeSet, scannedEdges, int64(way.ID))
	}

	return nil
}

func isRestricted(value string) bool {
	if value == "no" || value == "restricted" {
		return true
	}
	return false
}

func getReversedOneWay(way *osm.Way) (bool, bool, bool, bool) {
	vehicleForward := way.Tags.Find("vehicle:forward")
	motorVehicleForward := way.Tags.Find("motor_vehicle:forward")
	vehicleBackward := way.Tags.Find("vehicle:backward")
	motorVehicleBackward := way.Tags.Find("motor_vehicle:backward")
	return isRestricted(vehicleForward), isRestricted(motorVehicleForward), isRestricted(vehicleBackward), isRestricted(motorVehicleBackward)
}

func (p *OsmParser) processSegment(segment []node, tempMap map[string]string, speed float64,
	wayExtraInfoData wayExtraInfo, edgeSet map[uint32]map[uint32]struct{}, scannedEdges *[]datastructure.Edge, id int64) {

	if len(segment) == 2 && segment[0].id == segment[1].id {
		// skip
		return
	} else if len(segment) > 2 && segment[0].id == segment[len(segment)-1].id {
		// loop
		p.processSegment2(segment[0:len(segment)-1], tempMap, speed, wayExtraInfoData, edgeSet, scannedEdges, id)
		p.processSegment2(segment[len(segment)-2:], tempMap, speed, wayExtraInfoData, edgeSet, scannedEdges, id)
	} else {
		p.processSegment2(segment, tempMap, speed, wayExtraInfoData, edgeSet, scannedEdges, id)
	}
}

func (p *OsmParser) processSegment2(segment []node, tempMap map[string]string, speed float64,
	wayExtraInfoData wayExtraInfo, edgeSet map[uint32]map[uint32]struct{}, scannedEdges *[]datastructure.Edge, id int64) {
	waySegment := []node{}
	for i := 0; i < len(segment); i++ {
		nodeData := segment[i]
		if _, ok := p.barrierNodes[int64(nodeData.id)]; ok {

			if len(waySegment) != 0 {
				// if current node is a barrier
				// add the barrier node and process the segment (add edge)
				waySegment = append(waySegment, nodeData)
				p.addEdge(waySegment, tempMap, speed, wayExtraInfoData, edgeSet, scannedEdges, id)
				waySegment = []node{}
			}
			// copy the barrier node but with different id so that previous edge (with barrier) not connected with the new edge

			nodeData = p.copyNode(nodeData)
			waySegment = append(waySegment, nodeData)

		} else {
			waySegment = append(waySegment, nodeData)
		}
	}
	if len(waySegment) > 1 {
		p.addEdge(waySegment, tempMap, speed, wayExtraInfoData, edgeSet, scannedEdges, id)
	}
}

func (p *OsmParser) copyNode(nodeData node) node {
	// use the same coordinate but different id & and the newID is not used
	newMaxID := p.maxNodeID + 1
	p.acceptedNodeMap[newMaxID] = nodeCoord{
		lat: nodeData.coord.lat,
		lon: nodeData.coord.lon,
	}
	p.maxNodeID++
	return node{
		id: newMaxID,
		coord: nodeCoord{
			lat: nodeData.coord.lat,
			lon: nodeData.coord.lon,
		},
	}
}

func (p *OsmParser) addEdge(segment []node, tempMap map[string]string, speed float64,
	wayExtraInfoData wayExtraInfo, edgeSet map[uint32]map[uint32]struct{}, scannedEdges *[]datastructure.Edge, id int64) {
	from := segment[0]

	to := segment[len(segment)-1]

	if from == to {
		return
	}

	if _, ok := p.nodeIDMap[from.id]; !ok {
		p.nodeIDMap[from.id] = uint32(len(p.nodeIDMap))
		p.nodeToOsmId[p.nodeIDMap[from.id]] = from.id
	}
	if _, ok := p.nodeIDMap[to.id]; !ok {
		p.nodeIDMap[to.id] = uint32(len(p.nodeIDMap))
		p.nodeToOsmId[p.nodeIDMap[to.id]] = to.id
	}

	distance := 0.0
	for i := 0; i < len(segment); i++ {
		if i != 0 && i != len(segment)-1 && p.nodeTag[int64(segment[i].id)][p.tagStringIdMap.GetID(TRAFFIC_LIGHT)] == 1 {

			distToFromNode := geo.CalculateHaversineDistance(from.coord.lat, from.coord.lon, segment[i].coord.lat, segment[i].coord.lon)
			distToToNode := geo.CalculateHaversineDistance(to.coord.lat, to.coord.lon, segment[i].coord.lat, segment[i].coord.lon)
			if distToFromNode < distToToNode {
				if _, ok := p.nodeTag[int64(segment[0].id)]; !ok {
					p.nodeTag[int64(segment[0].id)] = make(map[int]int)
				}
				p.nodeTag[int64(segment[0].id)][p.tagStringIdMap.GetID(TRAFFIC_LIGHT)] = 1
			} else {
				if _, ok := p.nodeTag[int64(segment[len(segment)-1].id)]; !ok {
					p.nodeTag[int64(segment[len(segment)-1].id)] = make(map[int]int)
				}
				p.nodeTag[int64(segment[len(segment)-1].id)][p.tagStringIdMap.GetID(TRAFFIC_LIGHT)] = 1
			}
		}

		if i > 0 {
			distance += geo.CalculateHaversineDistance(segment[i-1].coord.lat, segment[i-1].coord.lon, segment[i].coord.lat, segment[i].coord.lon)
		}
	}

	if _, ok := edgeSet[p.nodeIDMap[from.id]]; !ok {
		edgeSet[p.nodeIDMap[from.id]] = make(map[uint32]struct{})
	}
	if _, ok := edgeSet[p.nodeIDMap[to.id]]; !ok {
		edgeSet[p.nodeIDMap[to.id]] = make(map[uint32]struct{})
	}

	if wayExtraInfoData.oneWay {
		if wayExtraInfoData.forward {

			if _, ok := edgeSet[p.nodeIDMap[from.id]][p.nodeIDMap[to.id]]; ok {
				return
			}

			edgeSet[p.nodeIDMap[from.id]][p.nodeIDMap[to.id]] = struct{}{}

			*scannedEdges = append(*scannedEdges, datastructure.NewEdge(
				from.coord.lat, from.coord.lon,
				to.coord.lat, to.coord.lon,
				uint32(len(*scannedEdges)),
				false,
				id,
				wayExtraInfoData.highwayType,
				speed,
				p.streetNameIdMap.GetID(tempMap[STREET_NAME]),
			))

		} else {

			if _, ok := edgeSet[p.nodeIDMap[to.id]][p.nodeIDMap[from.id]]; ok {
				return
			}
			edgeSet[p.nodeIDMap[to.id]][p.nodeIDMap[from.id]] = struct{}{}

			*scannedEdges = append(*scannedEdges, datastructure.NewEdge(
				from.coord.lat, from.coord.lon,
				to.coord.lat, to.coord.lon,
				uint32(len(*scannedEdges)),
				false,
				id,
				wayExtraInfoData.highwayType,
				speed,
				p.streetNameIdMap.GetID(tempMap[STREET_NAME]),
			))
		}
	} else {
		if _, ok := edgeSet[p.nodeIDMap[from.id]][p.nodeIDMap[to.id]]; ok {
			return
		}
		edgeSet[p.nodeIDMap[from.id]][p.nodeIDMap[to.id]] = struct{}{}
		edgeSet[p.nodeIDMap[to.id]][p.nodeIDMap[from.id]] = struct{}{}

		*scannedEdges = append(*scannedEdges, datastructure.NewEdge(
			from.coord.lat, from.coord.lon,
			to.coord.lat, to.coord.lon,
			uint32(len(*scannedEdges)),
			false,
			id,
			wayExtraInfoData.highwayType,
			speed,
			p.streetNameIdMap.GetID(tempMap[STREET_NAME]),
		))

	}
}

func roadTypeMaxSpeed2(roadType string) float64 {
	switch roadType {
	case "motorway":
		return 100
	case "trunk":
		return 70
	case "primary":
		return 65
	case "secondary":
		return 60
	case "tertiary":
		return 50
	case "unclassified":
		return 40
	case "residential":
		return 30
	case "service":
		return 20
	case "motorway_link":
		return 70
	case "trunk_link":
		return 65
	case "primary_link":
		return 60
	case "secondary_link":
		return 50
	case "tertiary_link":
		return 40
	case "living_street":
		return 5
	case "road":
		return 20
	case "track":
		return 15
	case "motorroad":
		return 90
	default:
		return 30
	}
}

func (p *OsmParser) isJunctionNode(nodeID int64) bool {
	return p.wayNodeMap[int64(nodeID)] == JUNCTION_NODE
}

func acceptOsmWay(way *osm.Way) bool {
	highway := way.Tags.Find("highway")
	junction := way.Tags.Find("junction")
	if highway != "" {
		if _, ok := acceptedHighway[highway]; ok {
			return true
		}
	} else if junction != "" {
		return true
	}
	return false
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
