package main

import "sync"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
	Type        string
}

type ServerArgs struct {
	ip   string
	port string
}

type ServerCommands struct {
	id          int
	params      Params
	slice       [][]uint16
	mu          sync.Mutex
	currentTurn int
	keyPresses  chan rune
	finished    chan bool
	haloRegions map[int][][]uint16
	haloLock    sync.Mutex
}

type GolRequest struct {
	World [][]uint8
	Turns int
}

type GolResponse struct {
	World       [][]uint8
	CurrentTurn int
}

type CheckAliveReq struct {
	RequestID int
}

type CheckAliveRes struct {
	ResponseID int
}

type HaloExchangeReq struct {
	Slice [][]uint16
	Turns int
}

type HaloExchangeRes struct {
	Slice       [][]uint16
	CurrentTurn int
}

type HaloRegionReq struct {
	Region      []uint16
	CurrentTurn int
	Type        int
}

type HaloRegionRes struct {
}

type IterateSliceReq struct {
	Slice [][]uint16
}

type IterateSliceRes struct {
	Slice [][]uint16
}
