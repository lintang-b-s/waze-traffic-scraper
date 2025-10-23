package datastructure

var (
	highwayTypemap = map[int]string{
		0: "motorway",
		1: "trunk",
		2: "primary",
		3: "secondary",
		4: "tertiary",
		5: "motorway_link",
		6: "trunk_link",
		7: "primary_link",
		8: "secondary_link",
		9: "tertiary_link",
	}
)

const (
	INVALID_HIGHWAY_TYPE = -1
)
