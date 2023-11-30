package gol

import (
	"fmt"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

func liveCellsReportMemShare(ticker *time.Ticker, c distributorChannels, done chan int) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			c.events <- AliveCellsCount{
				CompletedTurns: currentTurn,
				CellsCount:     util.CalcAliveCellsCountUint16(currentWorld),
			}
		}
	}
}

func updateFlippedCellsMemShare(done chan int, c distributorChannels, oldWorld [][]uint16) {
	localTurn := 0
	for {
		select {
		case <-done:
			return
		default:
			if localTurn != currentTurn {
				copy := currentWorld
				localTurn = currentTurn
				flipped := calculateCellsFlipped(copy, oldWorld)
				for _, cell := range flipped {
					c.events <- CellFlipped{
						Cell:           cell,
						CompletedTurns: localTurn,
					}
				}
				oldWorld = copy
				c.events <- TurnComplete{
					CompletedTurns: localTurn,
				}
			}
		}
	}
}

func bitSimulateGolMemShare(p Params, c distributorChannels, keyPresses <-chan rune, world [][]uint8) {
	stopChannels := make(map[string]chan int)
	// util.PrintUint8World(world)
	uint16World := util.ConvertToUint16(world)
	// util.PrintUint16World(uint16World)
	currentWorld = uint16World
	currentTurn = 0

	stopChannels["liveCells"] = make(chan int)
	ticker := time.NewTicker(2000 * time.Millisecond)
	go liveCellsReportMemShare(ticker, c, stopChannels["liveCells"])

	stopChannels["updateWorld"] = make(chan int)
	go updateFlippedCellsMemShare(stopChannels["updateWorld"], c, uint16World)

	stopChannels["simulator"] = make(chan int, 2)
	go util.SimulateSliceMemSharing(&currentWorld, stopChannels["simulator"], p.Turns, p.Threads, &currentTurn)
	for {
		if currentTurn == p.Turns {
			break
		}
		if len(keyPresses) > 0 {
			key := <-keyPresses
			if key == 's' {
				go writePGMBit(c, currentTurn, currentWorld)
			} else if key == 'q' {
				go writePGMBit(c, currentTurn, currentWorld)
				break
			} else if key == 'p' {
				stopChannels["simulator"] <- 1
				fmt.Println("Current Turn:", currentTurn)
				for {
					key := <-keyPresses
					if key == 'p' {
						fmt.Println("Continuing...")
						go util.SimulateSliceMemSharing(&currentWorld, stopChannels["simulator"], p.Turns-currentTurn, p.Threads, &currentTurn)
						break
					}
				}
			}
		}
	}

	ticker.Stop()
	for _, channel := range stopChannels {
		// fmt.Println("Stopping", name)
		channel <- 1
	}
	// fmt.Println("Stopped all goroutines")

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          util.CalcAliveCellsUint16(currentWorld),
	}
	uint8World := util.ConvertToUint8(currentWorld)
	writePGM(c, p, makeImmutableMatrix(uint8World))
}
