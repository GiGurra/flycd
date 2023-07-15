package model

type ScaleState struct {
	Process  string         `json:"Process"`
	Count    int            `json:"Count"`
	CPUKind  string         `json:"CPUKind"`
	CPUs     int            `json:"CPUs"`
	MemoryMB int            `json:"Memory"`
	Regions  map[string]int `json:"Regions"`
}

func (s ScaleState) IncludesRegion(region string) bool {
	_, ok := s.Regions[region]
	return ok
}

func (s ScaleState) CountInRegion(region string) int {
	return s.Regions[region]
}
