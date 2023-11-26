package gol

import (
	"flag"
	"fmt"
	"net/rpc"
)

func testHalo() {
	server := "23.22.135.15:8030"
	// server := "127.0.0.1:8031"
	flag.Parse()
	client, err := rpc.Dial("tcp", server)
	fmt.Println(err)
	defer client.Close()

	request := HaloExchangeReq{}
	response := new(HaloExchangeRes)
	client.Call("ServerCommands.HaloExchange", request, response)
	fmt.Println(response)
}

func server_distribution(p Params, c distributorChannels, keyPresses <-chan rune) {

	// TODO: Create a 2D slice to store the world.

	turn := 0
	world := readWorld(p, c)

	testHalo()

	server := "23.22.135.15:8030"
	// server := "127.0.0.1:8031"
	flag.Parse()
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	request := GolRequest{
		World: world,
		Turns: p.Turns,
	}
	response := new(GolResponse)
	client.Call("ServerCommands.RunGOL", request, response)

	world = response.World

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
