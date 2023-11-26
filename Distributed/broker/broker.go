package broker

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
	uint16World := convertToUint16(world)
	newWorld := broker(uint16World, params, 4)
	util.PrintUint16World(uint16World)
	res.World = convertToNormal(newWorld)

	return
}

func compareWorlds(w1 [][]uint16, w2 [][]uint8) {
	for y := 0; y < len(w1); y++ {
		for _, u := range w1[y] {
			fmt.Printf("%016b", u)
		}

		fmt.Print("\t\t")

		for _, cell := range w2[y] {
			if cell == 255 {
				fmt.Print(1)
				// s1 += "1"
			} else {
				fmt.Print(0)
				// s1 += "0"
			}
		}

		fmt.Print("\n")
	}

	fmt.Print("\n")
	fmt.Print("\n")
}

func broker(world [][]uint16, p Params, n int) [][]uint16 {
	channels := make([]chan [][]uint16, n)
	for i := 0; i < len(channels); i++ {
		channels[i] = make(chan [][]uint16)
	}

	// w2 := convertToNormal(world)

	// p.Turns = 1
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

		// fmt.Println("Turn", i)
		// printuint16(slices[0])
		// compareWorlds(world, w2)
		// w2 = calculateStep(p, w2)
		world = newWorld
	}

	return world
}

func calcSlices(world [][]uint16, p Params, n int) [][][]uint16 {
	rows := calcRows(p, n)
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

func callWorker(id int, slice [][]uint16, p Params, channel chan [][]uint16) {
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

func printByteLine(line []uint16) {
	for _, u := range line {
		fmt.Printf("%016b", u)
	}
	fmt.Print("\n")
}

func compareLines(l1 []uint8, l2 []uint16) bool {
	var s1 string
	var s2 string

	for _, cell := range l1 {
		if cell == 255 {
			// fmt.Print(1)
			s1 += "1"
		} else {
			// fmt.Print(0)
			s1 += "0"
		}
	}
	// fmt.Println()

	for _, u := range l2 {
		// fmt.Printf("%016b", u)
		s2 += fmt.Sprintf("%016b", u)
	}
	// fmt.Print("\n")

	fmt.Println(s1)
	fmt.Println(s2)
	fmt.Println(s1 == s2)
	return s1 == s2
}

func convertToUint16(world [][]uint8) [][]uint16 {
	var byteWorld [][]uint16
	for _, line := range world {
		var byteLine []uint16
		for i := 0; i < len(line); i += 16 {
			b := uint16(line[i] & 1)
			for j := 1; j < 16; j++ {
				b = (b << 1) | uint16((line[i+j] & 1))
			}
			byteLine = append(byteLine, b)
		}
		// compareLines(line, byteLine)
		byteWorld = append(byteWorld, byteLine)
	}

	return byteWorld
}

func convertToNormal(world [][]uint16) [][]uint8 {
	var newWorld [][]uint8
	maxX := len(world[0])
	for y := 0; y < len(world); y++ {
		var newLine []uint8
		for x := 0; x < maxX; x++ {
			n := world[y][x]
			for i := 0; i < 16; i++ {
				newLine = append(newLine, (uint8(n>>uint8(15-i))&1)*255)
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

	// fmt.Println(rows)
	return rows
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
