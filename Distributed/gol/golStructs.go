package gol

import "sync"

type GolCommands struct {
	params Params
	world  [][]uint8
	mu     sync.Mutex
	turn   int
	alive  int
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
