package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"net/rpc"
)

func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

func updateWorld(g *GolCommands, newWorld [][]uint8) {
	g.mu.Lock()
	g.world = newWorld
	g.alive = calcAliveCells(newWorld)
	g.turn = g.turn + 1
	g.mu.Unlock()
}

func getWorld(g *GolCommands) [][]uint8 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.world
}

func calcAliveCells(world [][]uint8) int {
	c := 0
	for _, line := range world {
		for _, cell := range line {
			if cell == 255 {
				c++
			}
		}
	}
	return c
}

func worldUpdater(g *GolCommands, worldChan chan [][]uint8, done chan bool) {
	for {
		select {
		case <-done:
			break
		case world := <-worldChan:
			updateWorld(g, world)
		default:
		}
	}
}

func (g *GolCommands) SingleThreadGOL(req SingleThreadGolRequest, res *SingleThreadGolResponse) (err error) {
	fmt.Println("Started SingleThreadGOL", req.Params)
	g.params = req.Params
	// g.keyPresses = make(chan rune)
	updateWorld(g, req.World)
	g.turn = 0
	worldChan := make(chan [][]uint8)
	// defer close(worldChan)
	stopDistributer := make(chan int)
	stopUpdater := make(chan bool)

	initial := getWorld(g)
	go distributor(worldChan, g.params, initial, g.keyPresses, stopDistributer)
	go worldUpdater(g, worldChan, stopUpdater)

	distributerCode := <-stopDistributer
	// fmt.Println("FLag")
	if distributerCode == -1 {
		defer g.finish()
	}

	stopUpdater <- true

	// fmt.Println("FLag")

	res.World = getWorld(g)
	res.Turns = g.turn

	defer close(stopDistributer)
	defer close(stopUpdater)
	defer close(worldChan)
	// worldChan = nil

	fmt.Println("Finished SingleThreadGOL", req.Params)
	return
}

func (g *GolCommands) AliveCellsCount(req AliveCellsCountRequest, res *AliveCellsCountResponse) (err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	res.Count = g.alive
	res.Turn = g.turn
	return
}

func (g *GolCommands) finish() {
	g.finished <- true
}

func (g *GolCommands) KeyPress(req KeyPressRequest, res *KeyPressResponse) (err error) {
	g.keyPresses <- req.Key
	// for {
	// 	if len(g.keyPresses) == 0 {
	// 		break
	// 	}
	// }
	res.Turn = g.turn
	res.World = getWorld(g)
	return
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

func worker(id int, channel chan [][]byte, p Params, world func(y, x int) uint8, startY int, endY int, startX int, endX int) {
	var newWorld [][]byte
	for y := startY; y < endY; y++ {
		var newLine []byte
		for x := startX; x < endX; x++ {
			n := neighbours(p, world, []int{x, y})
			cell := world(y, x)
			newCell := golLogic(cell, n)
			newLine = append(newLine, newCell)
		}
		newWorld = append(newWorld, newLine)
	}

	channel <- newWorld
	// fmt.Println("Worker", id, "done")
}

func worker1D(id int, channel chan []byte, p Params, world func(y, x int) uint8, start int, end int) {
	var newWorld []byte
	for i := start; i < end; i++ {
		coord := convert1Dto2D(i, p)
		n := neighbours(p, world, []int{coord[0], coord[1]})
		cell := world(coord[1], coord[0])
		newCell := golLogic(cell, n)
		// if len(newLine) < p.ImageWidth {
		// 	newLine = append(newLine, newCell)
		// } else {
		// 	newWorld = append(newWorld, newLine)
		// 	newLine = []byte{}
		// }
		newWorld = append(newWorld, newCell)
	}

	channel <- newWorld
}

func worker1Dv2(id int, channel chan []byte, p Params, world func(y, x int) uint8, start int, end int) {
	var newWorld []byte
	startCoord := convert1Dto2D(start, p)
	endCoord := convert1Dto2D(end, p)
	for y := startCoord[1]; y < endCoord[1]; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if y == startCoord[1] && x < startCoord[0]-1 {
				x = startCoord[0] - 1
				continue
			} else if y == endCoord[1] {
				if x >= endCoord[0] {
					break
				}
			}

			n := neighbours(p, world, []int{x, y})
			cell := world(y, x)
			newCell := golLogic(cell, n)
			newWorld = append(newWorld, newCell)
		}
	}

	// fmt.Println("Flag")
	channel <- newWorld

	// fmt.Println("worker", id, "done")
}

func handleKeyPresses(keyPresses chan rune) int {
	if len(keyPresses) > 0 {
		key := <-keyPresses
		if key == 'q' {
			return 0
		} else if key == 'p' {
			for {
				key := <-keyPresses
				if key == 'p' {
					break
				}
			}
		} else if key == 'k' {
			return -1
		}
	}
	return 1
}

func distributor(worldChan chan [][]uint8, p Params, world [][]uint8, keyPresses chan rune, done chan int) {
	distribution := "row"

	if distribution == "row" {
		channels := make([]chan [][]byte, p.Threads)
		for i := 0; i < len(channels); i++ {
			channels[i] = make(chan [][]byte)
		}
		startY := calcStartY(p)
		for i := 0; i < p.Turns; i++ {
			handleKey := handleKeyPresses(keyPresses)
			if handleKey == 0 {
				break
			} else if handleKey == -1 {
				done <- -1
				break
			}
			newWorld := rowDistribution(channels, p, world, startY)
			worldChan <- newWorld
			world = newWorld
			// fmt.Println("Completed", i, "turn")
		}

		for _, c := range channels {
			close(c)
		}
	} else if distribution == "cell" {
		channels := make([]chan []byte, p.Threads)
		for i := 0; i < len(channels); i++ {
			channels[i] = make(chan []byte)
		}
		coords := calcCoords(p)
		for i := 0; i < p.Turns; i++ {
			handleKey := handleKeyPresses(keyPresses)
			if handleKey == 0 {
				break
			} else if handleKey == -1 {
				done <- -1
				break
			}
			newWorld := cellDistribution(channels, p, world, coords)
			// fmt.Println("Completed", i, "turn")
			worldChan <- newWorld
			world = newWorld
		}

		for _, c := range channels {
			close(c)
		}
	}

	// fmt.Println("Distributor Function Finished")
	done <- 1
}

func convert1Dto2D(index int, p Params) []int {
	return []int{index % p.ImageWidth, index / p.ImageWidth}
}

func calcCoords(p Params) [][]int {
	totalCells := int(float64(p.ImageHeight) * float64(p.ImageWidth))
	cellsPerWorker := int(math.Floor(float64(totalCells) / float64(p.Threads)))
	plusOnes := totalCells - cellsPerWorker*p.Threads
	var nCells []int
	for i := 0; i < p.Threads; i++ {
		if i+1 > p.Threads-plusOnes {
			nCells = append(nCells, cellsPerWorker+1)
		} else {
			nCells = append(nCells, cellsPerWorker)
		}
	}

	// fmt.Println("Cells:", totalCells, "Threads:", p.Threads, "Extra:", plusOnes, "Cells Distribution:", nCells)

	var coords [][]int
	current := 0
	for i := 0; i < p.Threads; i++ {
		start := current
		end := start + nCells[i]
		// coords = append(coords, append(convert1Dto2D(start, p), convert1Dto2D(end, p)...))
		coords = append(coords, []int{start, end})
		current = end
	}

	// fmt.Println(coords)
	return coords
}

func rowDistribution(channels []chan [][]byte, p Params, world [][]uint8, startY []int) [][]uint8 {
	immutableWorld := makeImmutableMatrix(world)
	for i := 0; i < len(channels); i++ {

		go worker(i, channels[i], p, immutableWorld, startY[i], startY[i+1], 0, p.ImageWidth)
	}

	var newWorld [][]byte
	for _, channel := range channels {
		data := <-channel
		for _, d := range data {
			newWorld = append(newWorld, d)
		}
	}

	// fmt.Println("RowDistribution Function finished")

	return newWorld
}

func cellDistribution(channels []chan []byte, p Params, world [][]uint8, coords [][]int) [][]uint8 {
	immutableWorld := makeImmutableMatrix(world)
	// channels := make([]chan []byte, p.Threads)
	for i := 0; i < len(channels); i++ {
		// channels[i] = make(chan []byte)
		go worker1Dv2(i, channels[i], p, immutableWorld, coords[i][0], coords[i][1])
	}

	var allData []byte
	for _, channel := range channels {
		data := <-channel
		for _, d := range data {
			allData = append(allData, d)
		}
	}

	var newWorld [][]byte
	for i := 0; i < p.ImageHeight; i++ {
		newWorld = append(newWorld, allData[i*p.ImageWidth:(i+1)*p.ImageWidth])
	}

	// fmt.Println("RowDistribution Function finished")

	return newWorld
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	finishedChan := make(chan bool)
	keyPresses := make(chan rune, 1)
	rpc.Register(&GolCommands{finished: finishedChan, keyPresses: keyPresses})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	go rpc.Accept(listener)
	<-finishedChan
}
