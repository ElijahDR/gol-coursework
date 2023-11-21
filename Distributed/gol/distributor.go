package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func makeCall(client *rpc.Client, params Params, world [][]uint8) [][]uint8 {
	fmt.Println(params)
	request := SingleThreadGolRequest{
		Params: params,
		World:  world,
	}
	response := new(SingleThreadGolResponse)
	client.Call("GolCommands.SingleThreadGOL", request, response)
	// fmt.Println(response.World)

	return response.World
}

func liveCellsReport(client *rpc.Client, ticker *time.Ticker, c distributorChannels, done chan bool) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			request := AliveCellsCountRequest{}
			response := new(AliveCellsCountResponse)
			client.Call("GolCommands.AliveCellsCount", request, response)
			c.events <- AliveCellsCount{
				CompletedTurns: response.Turn,
				CellsCount:     response.Count,
			}
		}
	}
}

func handleKeyPresses(keyPresses <-chan rune, c distributorChannels, p Params, world func(x, y int) uint8, i int) bool {
	if len(keyPresses) > 0 {
		key := <-keyPresses
		if key == 's' {
			go writePGM(c, p, world)
		} else if key == 'q' {
			writePGM(c, p, world)
			return false
		} else if key == 'p' {
			fmt.Println("Current Turn: ", i)
			for {
				key := <-keyPresses
				if key == 'p' {
					break
				}
			}
		}
	}
	return true
}

func writePGM(c distributorChannels, p Params, world func(x, y int) uint8) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprint(p.ImageWidth, "x", p.ImageHeight, "x", p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world(y, x)
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

	// TODO: Create a 2D slice to store the world.
	var world [][]uint8

	turn := 0

	// fmt.Println("Flag")
	// for i, b := range c.ioInput {
	// 	board[int32(math.Floor(float64(int(i)/(p.ImageWidth))))][int(int(i) % p.ImageWidth)] = b

	// }
	// var inp := c.ioInput

	c.ioCommand <- ioInput
	filename := fmt.Sprint(p.ImageWidth, "x", p.ImageHeight)
	// fmt.Println(filename)
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		line := make([]uint8, 0)
		for x := 0; x < p.ImageWidth; x++ {
			value := <-c.ioInput
			line = append(line, value)
			if value == 255 {
				reportCells(c, 0, []int{x, y})
			}
		}
		world = append(world, line)
	}

	// TODO: Execute all turns of the Game of Life.

	// server := "23.22.135.15:8030"
	server := "127.0.0.1:8030"
	flag.Parse()
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	done := make(chan bool)
	ticker := time.NewTicker(2000 * time.Millisecond)

	go liveCellsReport(client, ticker, c, done)

	world = makeCall(client, p, world)

	immutableWorld := makeImmutableMatrix(world)

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calcAliveCells(p, immutableWorld),
	}

	// writePGM(c, p, immutableWorld)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

func reportCells(c distributorChannels, turns int, pos []int) {
	c.events <- CellFlipped{
		CompletedTurns: turns,
		Cell:           util.Cell{X: pos[0], Y: pos[1]},
	}
}

func calcAliveCells(p Params, world func(y, x int) uint8) []util.Cell {
	cells := make([]util.Cell, 0)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world(y, x) == 255 {
				cells = append(cells, util.Cell{
					X: x,
					Y: y,
				})
			}
		}
	}
	return cells
}

func calcAliveCellsCount(p Params, world func(y, x int) uint8, turns int) AliveCellsCount {
	c := 0
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world(y, x) == 255 {
				c++
			}
		}
	}

	return AliveCellsCount{
		CellsCount:     c,
		CompletedTurns: turns,
	}
}
