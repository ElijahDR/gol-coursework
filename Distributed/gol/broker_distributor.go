package gol

import (
	"flag"
	"fmt"
	"net/rpc"
)

func callBroker(client *rpc.Client, params Params, world [][]uint8) [][]uint8 {
	fmt.Println(params)
	request := GolBrokerRequest{
		Params: params,
		World:  world,
	}
	response := new(GolBrokerResponse)
	client.Call("GolCommands.GOLBroker", request, response)

	return response.World
}

func readWorld(p Params, c distributorChannels) [][]uint8 {
	var world [][]uint8

	c.ioCommand <- ioInput
	filename := fmt.Sprint(p.ImageWidth, "x", p.ImageHeight)

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

	return world
}

func broker_distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

	// TODO: Create a 2D slice to store the world.

	turn := 0
	world := readWorld(p, c)

	server := "23.22.135.15:8030"
	// server := "127.0.0.1:8031"
	flag.Parse()
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	world = callBroker(client, p, world)

	immutableWorld := makeImmutableMatrix(world)

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calcAliveCells(p, immutableWorld),
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
