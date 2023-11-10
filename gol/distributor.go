package gol

import (
	"fmt"

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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

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
		}
		world = append(world, line)
	}

	// TODO: Execute all turns of the Game of Life.
	for i := 0; i < p.Turns; i++ {
		world = calculateStep(p, world)
	}

	// fmt.Println("All done")

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calcAliveCells(world),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calcAliveCells(world [][]uint8) []util.Cell {
	cells := make([]util.Cell, 0)
	for y, line := range world {
		for x, cell := range line {
			if cell == 255 {
				cells = append(cells, util.Cell{
					X: x,
					Y: y,
				})
			}
		}
	}
	return cells
}

func neighbours(world [][]byte, pos []int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			//fmt.Println(x+dx, y+dy)
			//fmt.Println(world)
			newPos := [2]int{(pos[0] + dx + len(world[0])) % len(world[0]), (pos[1] + dy + len(world)) % len(world)}
			if world[newPos[0]][newPos[1]] == 255 {
				count += 1
			}
		}
	}

	return count
}

func calculateStep(p Params, world [][]byte) [][]byte {
	var newWorld [][]byte
	for y, line := range world {
		newLine := make([]byte, 0)
		for x, cell := range line {
			count := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					//if x+dx < 0 || x+dx >= len(line) {
					//	continue
					//}
					if dx == 0 && dy == 0 {
						continue
					}
					//fmt.Println(x+dx, y+dy)
					//fmt.Println(world)
					newX := (x + dx + p.ImageWidth) % p.ImageWidth
					newY := (y + dy + p.ImageHeight) % p.ImageWidth
					if world[newY][newX] == 255 {
						count += 1
					}
				}
			}
			if cell == 255 {
				if count < 2 {
					//newWorld[y][x] = 0
					newLine = append(newLine, 0)
				} else if count > 3 {
					//newWorld[y][x] = 0
					newLine = append(newLine, 0)
				} else {
					//newWorld[y][x] = 255
					newLine = append(newLine, 255)
				}
			} else {
				if count == 3 {
					//newWorld[y][x] = 255
					newLine = append(newLine, 255)
				} else {
					//newWorld[y][x] = 0
					newLine = append(newLine, 0)
				}
			}
		}

		newWorld = append(newWorld, newLine)
	}

	return newWorld
}
