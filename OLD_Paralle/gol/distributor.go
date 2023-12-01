package gol

import (
	"fmt"
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

func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

// type insert[T any] func (array[]T, item T, i int) []T {

// }

func insert(array [][]byte, item []byte, i int) [][]byte {
	newArray := append(array, []byte{0})
	copy(newArray[i+1:], newArray[i:])
	newArray[i] = item
	return newArray
}

func calcSteps(p Params) []int {
	var steps []int
	for i := 0; i < p.Threads; i++ {
		steps = append(steps, 0)
	}
	c := 0
	for i := 0; i < p.ImageHeight; i++ {
		steps[c%p.Threads]++
		c++
	}

	return steps
}

func calcStartY(p Params) []int {
	steps := calcSteps(p)
	var startY []int
	startY = append(startY, 0)
	for i := 1; i <= p.Threads; i++ {
		startY = append(startY, startY[i-1]+steps[i-1])
	}

	return startY
}

func liveCellsReport(ticker *time.Ticker, c distributorChannels, cells chan AliveCellsCount, done chan bool) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			c.events <- (<-cells)
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

func rowDistribution(p Params, c distributorChannels, keyPresses <-chan rune, immutableWorld func(x, y int) uint8) func(x, y int) uint8 {
	startX := 0
	endX := p.ImageWidth

	aliveCells := make(chan AliveCellsCount, 1)
	done := make(chan bool)
	ticker := time.NewTicker(2000 * time.Millisecond)
	go liveCellsReport(ticker, c, aliveCells, done)
	aliveCells <- AliveCellsCount{CellsCount: 0, CompletedTurns: 0}

	startY := calcStartY(p)

	for i := 1; i < p.Turns+1; i++ {
		if !handleKeyPresses(keyPresses, c, p, immutableWorld, i) {
			break
		}
		channels := make([]chan [][]byte, p.Threads)
		for i := 0; i < len(channels); i++ {
			channels[i] = make(chan [][]byte)

			// step := int(math.Ceil(float64(p.ImageHeight / p.Threads)))
			// startY := (i * step)
			// endY := int(math.Min(float64(startY+step), float64(p.ImageHeight)))
			// fmt.Println(step, startY)

			go worker(c, i, channels[i], p, immutableWorld, startY[i], startY[i+1], startX, endX)
		}

		var newWorld [][]byte
		for _, channel := range channels {
			data := <-channel
			for _, d := range data {
				newWorld = append(newWorld, d)
			}
		}

		immutableWorld = makeImmutableMatrix(newWorld)

		// Reporting to alive cells report
		if len(aliveCells) == 0 {
			aliveCells <- calcAliveCellsCount(p, immutableWorld, i)
		} else {
			<-aliveCells
			aliveCells <- calcAliveCellsCount(p, immutableWorld, i)
		}

		c.events <- TurnComplete{
			CompletedTurns: i,
		}
	}

	ticker.Stop()
	done <- true

	return immutableWorld
}

// func calcCoords(p Params) [][]int {
// 	cells := p.ImageHeight * p.ImageWidth
// 	size := cells / p.Threads
// 	nBigger := cells - (size * p.Threads)
// 	var coords [][]int
// 	coords = append(coords, []int{
// 		0, 0, size % p.ImageWidth, size / p.ImageHeight,
// 	})
// 	for i := 1; i < p.Threads; i++ {
// 		if i >= p.Threads-nBigger {
// 			coords = append(coords, []int{
// 				coords[i-1][2], coords[i-1][3], ((i + 1) * size) % p.ImageWidth, ((i + 1) * size) / p.ImageHeight,
// 			})
// 		}
// 	}

// 	return coords
// }

// func cellDistribution(p Params, c distributorChannels, immutableWorld func(x, y int) uint8) func(x, y int) uint8 {

// }

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

	immutableWorld := rowDistribution(p, c, keyPresses, makeImmutableMatrix(world))

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calcAliveCells(p, immutableWorld),
	}

	writePGM(c, p, immutableWorld)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func reportCells(c distributorChannels, turns int, pos []int) {
	c.events <- CellFlipped{
		CompletedTurns: turns,
		Cell:           util.Cell{X: pos[0], Y: pos[1]},
	}
}

func worker(c distributorChannels, turns int, channel chan [][]byte, p Params, world func(y, x int) uint8, startY int, endY int, startX int, endX int) {
	var newWorld [][]byte
	for y := startY; y < endY; y++ {
		var newLine []byte
		for x := startX; x < endX; x++ {
			n := neighbours(p, world, []int{x, y})
			cell := world(y, x)
			newCell := golLogic(cell, n)
			if cell != newCell {
				reportCells(c, turns, []int{x, y})
			}
			newLine = append(newLine, newCell)
		}
		newWorld = append(newWorld, newLine)
	}

	channel <- newWorld
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

func neighbours(p Params, world func(y, x int) uint8, pos []int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			//fmt.Println(x+dx, y+dy)
			//fmt.Println(world)
			// Add the len of the line/world so that if it is negative, return a positive result.
			newPos := [2]int{(pos[0] + dx + p.ImageWidth) % p.ImageWidth, (pos[1] + dy + p.ImageHeight) % p.ImageHeight}
			if world(newPos[1], newPos[0]) == 255 {
				count += 1
			}
		}
	}

	return count
}

func golLogic(current byte, nCount int) byte {
	if current == 255 {
		if nCount < 2 {
			return 0
		} else if nCount > 3 {
			return 0
		} else {
			return 255
		}
	} else {
		if nCount == 3 {
			return 255
		} else {
			return 0
		}
	}
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
