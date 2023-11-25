package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
)

var NODES = []string{
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
	newWorld := broker(convertToUint16(world), params, 1)

	res.World = convertToNormal(newWorld)

	return
}

func broker(world [][]uint16, p Params, n int) [][]uint16 {
	channels := make([]chan [][]uint16, n)
	for i := 0; i < len(channels); i++ {
		channels[i] = make(chan [][]uint16)
	}

	slices := calcSlices(world, p, n)

	for i, channel := range channels {
		go callWorker(i, slices[i], p, channel)
	}

	var newWorld [][]uint16
	for _, channel := range channels {
		data := <-channel
		newWorld = append(newWorld, data...)
	}

	return newWorld
}

func calcSlices(world [][]uint16, p Params, n int) [][][]uint16 {
	rows := calcRows(p, n)
	start := 0
	var slices [][][]uint16
	for i := 0; i < n; i++ {
		var slice [][]uint16
		if i == 0 {
			slice = append(slice, world[p.ImageHeight-1])
			slice = append(slice, world[start:start+rows[i]+1]...)
		} else if i == n {
			slice = append(slice, world[start:p.ImageHeight-1]...)
			slice = append(slice, world[0])
		} else {
			slice = append(slice, world[start-1:start+rows[i]+1]...)
		}
		slices = append(slices, slice)
	}

	return slices
}

func callWorker(id int, slice [][]uint16, p Params, channel chan [][]uint16) {
	server := NODES[id]
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

	channel <- response.Slice
	return
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

func convertToUint16(world [][]uint8) [][]uint16 {
	var byteWorld [][]uint16
	for _, line := range world {
		var byteLine []uint16
		for i := 0; i < len(line); i += 16 {
			b := uint16(0)
			for j := 16; j >= 0; j-- {
				b = (b) | uint16(line[i+j]<<uint8(j))
			}
			byteLine = append(byteLine, b)
		}
		// fmt.Printf("%016b\n", byte(n))
		byteWorld = append(byteWorld, byteLine)
	}

	return byteWorld
}

func convertToNormal(world [][]uint16) [][]uint8 {
	var newWorld [][]uint8
	maxX := len(world[0])
	for y := 0; y < len(world); y++ {
		var newLine []uint8
		for x := 0; x < maxX; y++ {
			n := world[y][x]
			for i := 0; i < 16; i++ {
				newLine = append(newLine, uint8(n>>uint8(15-i))&1)
			}
		}
		newWorld = append(newWorld, newLine)
	}

	return newWorld
}

func calcRows(p Params, n int) []int {
	rowsEach := p.ImageHeight / n
	nBigger := p.ImageHeight - (rowsEach * n)
	var rows []int
	for i := 0; i < n-nBigger; i++ {
		rows = append(rows, rowsEach)
	}
	for i := 0; i < nBigger; i++ {
		rows = append(rows, rowsEach+1)
	}

	return rows
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolCommands{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
