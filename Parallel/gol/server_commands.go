package gol

import "sync"

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
	World   [][]uint8
	Turns   int
	Threads int
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

type PingReq struct{}
type PingRes struct{ Ping bool }

type NomBrokerReq struct{}
type NomBrokerRes struct{ ID int }
