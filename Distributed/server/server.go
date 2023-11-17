package main

import (
	"flag"
	"fmt"
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
	g.mu.Unlock()
}

func getWorld(g *GolCommands) [][]uint8 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.world
}

func (g *GolCommands) SingleThreadGOL(req SingleThreadGolRequest, res *SingleThreadGolResponse) (err error) {
	fmt.Println("Started SingleThreadGOL")
	g.params = req.Params
	updateWorld(g, req.World)
	newWorld := rowDistribution(g.params, g.world)
	updateWorld(g, newWorld)

	res.World = getWorld(g)
	res.Turns = g.turn
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

func worker(turns int, channel chan [][]byte, p Params, world func(y, x int) uint8, startY int, endY int, startX int, endX int) {
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
}

func rowDistribution(p Params, world [][]uint8) [][]uint8 {
	startX := 0
	endX := p.ImageWidth
	// go liveCellsReport(ticker, c, aliveCells, done)

	startY := calcStartY(p)

	for i := 1; i < p.Turns+1; i++ {
		immutableWorld := makeImmutableMatrix(world)
		channels := make([]chan [][]byte, p.Threads)
		for i := 0; i < len(channels); i++ {
			channels[i] = make(chan [][]byte)

			// step := int(math.Ceil(float64(p.ImageHeight / p.Threads)))
			// startY := (i * step)
			// endY := int(math.Min(float64(startY+step), float64(p.ImageHeight)))
			// fmt.Println(step, startY)

			go worker(i, channels[i], p, immutableWorld, startY[i], startY[i+1], startX, endX)
		}

		var newWorld [][]byte
		for _, channel := range channels {
			data := <-channel
			for _, d := range data {
				newWorld = append(newWorld, d)
			}
		}

		world = newWorld
		fmt.Println("Completed turns:", i)
	}

	return world
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolCommands{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
