package gol

import (
	"flag"
	"fmt"
	"net/rpc"
)

var NODES = []string{
	"23.22.135.15",
	"35.174.225.191",
	"44.208.149.39",
	"3.214.156.90",
	"44.208.47.178",
	"3.93.57.107",
	"18.208.253.79",
}

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

// func testPings() int {
// 	min := 999999999999
// 	id := -1
// 	for i, ip := range NODES {
// 		server := ip + ":8030"
// 		client, _ := rpc.Dial("tcp", server)

// 		request := PingReq{}
// 		response := new(PingRes)
// 		t := time.Now()
// 		client.Call("ServerCommands.Ping", request, response)
// 		ping := int(time.Since(t) / time.Millisecond)
// 		fmt.Println(i, ping)
// 		if ping < min {
// 			id = i
// 			min = ping
// 		}
// 		client.Close()
// 	}
// 	return id
// }

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
			fmt.Println("KeyPress returned...")
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
						break
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

func halo_distribution(p Params, c distributorChannels, keyPresses <-chan rune) {

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
	client.Call("ServerCommands.ClientRunHalo", request, response)

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

func server_distribution(p Params, c distributorChannels, keyPresses <-chan rune) {

	// TODO: Create a 2D slice to store the world.

	turn := 0
	world := readWorld(p, c)

	// testHalo()

	server := NODES[0] + ":8030"
	// server := "127.0.0.1:8031"
	flag.Parse()
	client, _ := rpc.Dial("tcp", server)

	req := NomBrokerReq{}
	res := new(NomBrokerRes)
	client.Call("ServerCommands.NominateBroker", req, res)

	brokerID := res.ID
	fmt.Println(brokerID)
	client.Close()

	server = NODES[brokerID] + ":8030"
	client, _ = rpc.Dial("tcp", server)
	defer client.Close()

	request := GolRequest{
		World:   world,
		Turns:   p.Turns,
		Threads: p.Threads,
	}
	response := new(GolResponse)

	stopChannel := make(chan int)
	go serverHandleKeyPresses(client, c, keyPresses, stopChannel)

	client.Call("ServerCommands.RunGOL", request, response)
	fmt.Println("RunGOL returned")

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
