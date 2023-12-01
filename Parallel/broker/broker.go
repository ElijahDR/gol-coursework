package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/util"
)

var NODES = []string{
	// "127.0.0.1",
	"35.174.225.191",
	"44.208.149.39",
	"3.214.156.90",
	"44.208.47.178",
}

var N_NODES = 4

func (g *GolCommands) GOLBroker(req GolBrokerRequest, res *GolBrokerResponse) (err error) {
	params := req.Params
	fmt.Println("Broker Received Request:", params.ImageWidth, "x", params.ImageHeight, "for", params.Turns, "turns")

	world := req.World
	uint16World := util.ConvertToUint16(world)
	// newWorld := broker(uint16World, params, 1)
	newWorld := broker(uint16World, params, 4)
	res.World = util.ConvertToUint8(newWorld)

	return
}

func broker(world [][]uint16, p util.Params, n int) [][]uint16 {
	channels := make([]chan [][]uint16, n)
	for i := 0; i < len(channels); i++ {
		channels[i] = make(chan [][]uint16)
	}

	for i := 0; i < p.Turns; i++ {
		slices := util.CalcSlices(world, p.ImageHeight, n)

		for id, channel := range channels {
			go callWorker(id, slices[id], p, channel)
		}

		var newWorld [][]uint16
		for _, channel := range channels {
			data := <-channel
			newWorld = append(newWorld, data...)
		}

		world = newWorld
	}

	return world
}

func callWorker(id int, slice [][]uint16, p util.Params, channel chan [][]uint16) {
	server := NODES[id] + ":8030"
	// fmt.Println("Sending request to", server, "id", id)
	flag.Parse()
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	request := GolWorkerRequest{
		Slice: slice,
		ID:    id,
	}
	response := new(GolWorkerResponse)
	client.Call("GolCommands.GOLWorker", request, response)
	// fmt.Println(response.World)

	// fmt.Println(response.Slice)
	channel <- response.Slice
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	// pAddr := flag.String("port", "8031", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolCommands{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
