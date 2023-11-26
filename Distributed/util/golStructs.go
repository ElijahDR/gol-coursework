package util

import "sync"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type GolCommands struct {
	Params     Params
	World      [][]uint8
	MU         sync.Mutex
	Turn       int
	Alive      int
	KeyPresses chan rune
	Finished   chan bool
}

type SingleThreadGolRequest struct {
	Params Params
	World  [][]uint8
}

type SingleThreadGolResponse struct {
	World [][]uint8
	Turns int
}

type AliveCellsCountRequest struct {
}

type AliveCellsCountResponse struct {
	Count int
	Turn  int
}

type KeyPressRequest struct {
	Key rune
}

type KeyPressResponse struct {
	Turn  int
	World [][]uint8
}
