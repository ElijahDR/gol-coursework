package main

import (
	"sync"

	"uk.ac.bris.cs/gameoflife/util"
)

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type GolCommands struct {
	params     Params
	world      [][]uint8
	mu         sync.Mutex
	turn       int
	alive      int
	keyPresses chan rune
	finished   chan bool
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

type GolBrokerResponse struct {
	Turn  int
	World [][]uint8
}

type GolBrokerRequest struct {
	Params util.Params
	World  [][]uint8
}

type GolWorkerRequest struct {
	ID    int
	Slice [][]uint16
}

type GolWorkerResponse struct {
	Slice [][]uint16
}
