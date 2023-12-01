package gol

import (
	"fmt"
	"time"

	_ "net/http/pprof"

	"uk.ac.bris.cs/gameoflife/util"
)

func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

var currentWorld [][]uint16
var currentTurn int

var WITHFLIPS bool

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
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

func liveCellsReport(ticker *time.Ticker, c distributorChannels, cells chan AliveCellsCount, done chan int) {
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

func bitSimulateGol(p Params, c distributorChannels, keyPresses <-chan rune, world [][]uint8) {
	stopChannels := make(map[string]chan int)
	// util.PrintUint8World(world)
	uint16World := util.ConvertToUint16(world)
	// util.PrintUint16World(uint16World)

	stopChannels["liveChells"] = make(chan int)
	ticker := time.NewTicker(2000 * time.Millisecond)
	aliveCells := make(chan AliveCellsCount, 1)
	go liveCellsReport(ticker, c, aliveCells, stopChannels["liveChells"])
	aliveCells <- AliveCellsCount{CellsCount: 0, CompletedTurns: 0}

	currentWorld = uint16World
	currentTurn = 0
	dataChannel := make(chan [][]uint16)
	stopChannels["updateWorld"] = make(chan int)
	go updateWorldBit(dataChannel, stopChannels["updateWorld"], c, aliveCells)

	// flipChannel := make(chan [][]int)

	stopChannels["simulator"] = make(chan int, 2)
	slice := util.CalcSlices(uint16World, len(uint16World), 1)[0]
	go util.SimulateSlice(slice, dataChannel, stopChannels["simulator"], p.Turns, p.Threads)

	// stopChannels["flipCells"] = make(chan int)
	// go flipCells(flipChannel, stopChannels["flipCells"], c, aliveCells)

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
						go util.SimulateSlice(util.CalcSlices(currentWorld, len(currentWorld), 1)[0], dataChannel, stopChannels["simulator"], p.Turns-currentTurn, p.Threads)
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
	writePGM(c, p, makeImmutableMatrix(util.ConvertToUint8(currentWorld)))
}

func updateWorldBit(dataChannel chan [][]uint16, stop chan int, c distributorChannels, aliveCells chan AliveCellsCount) {
	// var lock sync.Mutex
	for {
		select {
		case <-stop:
			return
		case data := <-dataChannel:
			// fmt.Println("Data Received: Updating World")
			flipped := calculateCellsFlipped(currentWorld, data)
			for _, cell := range flipped {
				c.events <- CellFlipped{
					Cell:           cell,
					CompletedTurns: currentTurn + 1,
				}
			}
			// fmt.Println(*currentTurn)
			// util.PrintUint16World(*world)
			// lock.Lock()
			currentWorld = data
			currentTurn++
			// lock.Unlock()
			c.events <- TurnComplete{
				CompletedTurns: currentTurn,
			}
			// if len(aliveCells) > 0 {
			// 	<-aliveCells
			// }
			// aliveCount := AliveCellsCount{
			// 	CompletedTurns: currentTurn,
			// 	CellsCount:     util.CalcAliveCellsCountUint16(data),
			// }
			// fmt.Println(currentTurn, aliveCount)
			// aliveCells <- aliveCount
		}
	}
}

func flipCells(flipChannel chan [][]int, stop chan int, c distributorChannels, aliveCells chan AliveCellsCount) {
	for {
		select {
		case <-stop:
			return
		case flips := <-flipChannel:
			if len(aliveCells) > 0 {
				<-aliveCells
			}
			aliveCount := AliveCellsCount{
				CompletedTurns: currentTurn,
				CellsCount:     len(flips),
			}
			aliveCells <- aliveCount
			for _, f := range flips {
				c.events <- CellFlipped{
					Cell:           util.Cell{X: f[0], Y: f[1]},
					CompletedTurns: currentTurn + 1,
				}
			}
		}
	}
}

func calculateCellsFlipped(oldWorld [][]uint16, newWorld [][]uint16) []util.Cell {
	var flipped []util.Cell
	for y := 0; y < len(oldWorld); y++ {
		for x := 0; x < len(oldWorld[y]); x++ {
			change := oldWorld[y][x] ^ newWorld[y][x]
			for i := 0; i < 16; i++ {
				if ((change >> uint8(15-i)) & 1) == 1 {
					flipped = append(flipped, util.Cell{
						X: (x * 16) + i,
						Y: y,
					})
				}
			}
		}
	}
	return flipped
}

func naiveSimulateGol(p Params, c distributorChannels, keyPresses <-chan rune, immutableWorld func(x, y int) uint8) func(x, y int) uint8 {
	startX := 0
	endX := p.ImageWidth

	aliveCells := make(chan AliveCellsCount, 1)
	done := make(chan int)
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
	done <- 1

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calcAliveCells(p, immutableWorld),
	}

	writePGM(c, p, immutableWorld)

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

func writePGMBit(c distributorChannels, turn int, world [][]uint16) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprint(len(world[0])*16, "x", len(world), "x", turn)
	for y := 0; y < len(world); y++ {
		for x := 0; x < len(world[y]); x++ {
			for i := 0; i < 16; i++ {
				c.ioOutput <- uint8((world[y][x] >> uint8(15-i)) & 1)
			}
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func parallel_distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	// go func() {
	// 	fmt.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// TODO: Create a 2D slice to store the world.

	// fmt.Println("Flag")
	// for i, b := range c.ioInput {
	// 	board[int32(math.Floor(float64(int(i)/(p.ImageWidth))))][int(int(i) % p.ImageWidth)] = b

	// }
	// var inp := c.ioInput

	turn := 0
	world := readWorld(p, c)

	if p.Turns == 0 {
		immutableWorld := makeImmutableMatrix(world)
		c.events <- FinalTurnComplete{
			CompletedTurns: p.Turns,
			Alive:          calcAliveCells(p, immutableWorld),
		}

		writePGM(c, p, immutableWorld)
	} else {
		bitSimulateGol(p, c, keyPresses, world)
		// naiveSimulateGol(p, c, keyPresses, makeImmutableMatrix(world))
		// bitSimulateGolMemShare(p, c, keyPresses, world)
	}

	// TODO: Execute all turns of the Game of Life.

	// TODO: Report the final state using FinalTurnCompleteEvent.

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
