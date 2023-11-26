package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/util"
)

var NODES = []string{
	"127.0.0.1",
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
	newWorld := broker(uint16World, params, 1)
	// newWorld := broker(uint16World, params, 4)
	res.World = util.ConvertToUint8(newWorld)

	return
}

func broker(world [][]uint16, p util.Params, n int) [][]uint16 {
	channels := make([]chan [][]uint16, n)
	for i := 0; i < len(channels); i++ {
		channels[i] = make(chan [][]uint16)
	}

	for i := 0; i < p.Turns; i++ {
		slices := calcSlices(world, p, n)

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

func calcSlices(world [][]uint16, p util.Params, n int) [][][]uint16 {
	rows := util.CalculateNRows(p, n)
	start := 0
	// fmt.Println("Rows:", rows)
	var slices [][][]uint16
	for i := 0; i < n; i++ {
		var slice [][]uint16
		if i == 0 && i == n-1 {
			slice = append(slice, world[p.ImageHeight-1])
			slice = append(slice, world[start:start+rows[i]]...)
			slice = append(slice, world[0])
		} else if i == 0 {
			slice = append(slice, world[p.ImageHeight-1])
			slice = append(slice, world[start:start+rows[i]+1]...)
		} else if i == n-1 {
			slice = append(slice, world[start-1:p.ImageHeight]...)
			slice = append(slice, world[0])
		} else {
			slice = append(slice, world[start-1:start+rows[i]+1]...)
		}
		slices = append(slices, slice)
		// fmt.Println(slices)
		start += rows[i]
	}

	return slices
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

func convertToBytes(world [][]uint8) [][]byte {
	var byteWorld [][]byte
	for _, line := range world {
		var byteLine []byte
		for i := 0; i < len(line); i += 8 {
			b := byte(0)
			for j := 7; j >= 0; j-- {
				b = (b) | (line[i+j] << uint8(j))
			}
			byteLine = append(byteLine, b)
		}
		// fmt.Printf("%016b\n", byte(n))
		byteWorld = append(byteWorld, byteLine)
	}

	return byteWorld
}

func main() {
	// pAddr := flag.String("port", "8030", "Port to listen on")
	pAddr := flag.String("port", "8031", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolCommands{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
