package gol

import (
	"flag"
	"fmt"
	"net/rpc"
)

func testHalo() {
	server := "23.22.135.15:8030"
	// server := "127.0.0.1:8031"
	client, err := rpc.Dial("tcp", server)
	fmt.Println(err)
	defer client.Close()

	request := HaloExchangeReq{}
	response := new(HaloExchangeRes)
	client.Call("ServerCommands.HaloExchange", request, response)
	fmt.Println(response)
}

func serverHandleKeyPresses(client *rpc.Client, c distributorChannels, keyPresses <-chan rune, stopChannel chan int) {
	for {
		select {
		case <-stopChannel:
			break
		case key := <-keyPresses:
			fmt.Println("KEY PRESSED", string(key))
			req := KeyPressRequest{
				Key: key,
			}
			res := new(KeyPressResponse)
			client.Call("ServerCommands.KeyPress", req, res)
			if key == 'p' {
				fmt.Println("Paused! Current Turn:", res.Turn)
				for {
					key = <-keyPresses
					if key == 'p' {
						req := KeyPressRequest{
							Key: key,
						}
						res := new(KeyPressResponse)
						client.Call("ServerCommands.KeyPress", req, res)
						fmt.Println("Continuing...")
					}
				}
			} else if key == 's' {
				go writePGMServer(c, res.Turn, res.World)
			} else if key == 'q' {
				return
			} else if key == 'k' {
				go writePGMServer(c, res.Turn, res.World)
				return
			}
		default:
		}
	}
}

func writePGMServer(c distributorChannels, turn int, world [][]uint8) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprint(len(world[0])*16, "x", len(world), "x", turn)
	for y := 0; y < len(world); y++ {
		for x := 0; x < len(world[y]); x++ {
			c.ioOutput <- world[y][x]
		}
	}
}

func server_distribution(p Params, c distributorChannels, keyPresses <-chan rune) {

	// TODO: Create a 2D slice to store the world.

	turn := 0
	world := readWorld(p, c)

	// testHalo()

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

	stopChannel := make(chan int)
	go serverHandleKeyPresses(client, c, keyPresses, stopChannel)

	client.Call("ServerCommands.RunGOL", request, response)

	world = response.World

	immutableWorld := makeImmutableMatrix(world)

	// TODO: Report the final state using FinalTurnCompleteEvent.

	c.events <- FinalTurnComplete{
		CompletedTurns: p.Turns,
		Alive:          calcAliveCells(p, immutableWorld),
	}

	stopChannel <- 1

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
