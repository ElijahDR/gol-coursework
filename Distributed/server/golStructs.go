package main

import "sync"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type GolCommands struct {
	params Params
	world  [][]uint8
	mu     sync.Mutex
	turn   int
}

type SingleThreadGolRequest struct {
	Params Params
	World  [][]uint8
}

type SingleThreadGolResponse struct {
	World [][]uint8
	Turns int
}
