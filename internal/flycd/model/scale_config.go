package model

type ScaleState struct {
	Process  string         `json:"Process"`
	Count    int            `json:"Count"`
	CPUKind  string         `json:"CPUKind"`
	CPUs     int            `json:"CPUs"`
	MemoryMB int            `json:"Memory"`
	Regions  map[string]int `json:"Regions"`
}
