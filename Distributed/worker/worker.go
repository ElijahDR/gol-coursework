package main

import (
	"flag"
	"fmt"
	"math/bits"
	"net"
	"net/rpc"
)

func (g *GolCommands) GOLWorker(req GolWorkerRequest, res *GolWorkerResponse) (err error) {
	fmt.Println("Worker", req.ID, "received slice")

	slice := req.Slice
	workerChan := make(chan [][]uint16)
	go worker(slice, workerChan)
	data := <-workerChan
	res.Slice = data

	return
}

// Doing it this was is my biggest regret since taking physics a level
func worker(slice [][]uint16, c chan [][]uint16) {
	nuint16 := len(slice[0])
	for y := 1; y < len(slice)-1; y++ {
		var newLine []uint16
		for x := 0; x < nuint16; x++ {
			var newuint16 uint16

			if x == 0 {
				var area []byte
				for j := -1; j <= 1; j++ {
					// Get the last bit of the furthest right uint16 and the first 2 of the first uint16
					area[j] = (byte(slice[y+j][nuint16-1]&1) << 2) | byte(slice[y+j][0]>>14)
				}
				newuint16 = uint16(golLogic(area))
			} else {
				var area []byte
				for j := -1; j <= 1; j++ {
					area[j] = byte(slice[y+j][x-1]&1)<<2 | byte(slice[y+j][x])>>uint8(14)
				}
				newuint16 = uint16(golLogic(area))
			}

			for i := 1; i < 15; i++ {
				var area []byte
				for j := -1; j <= 1; j++ {
					area[j] = byte(slice[y+j][x]>>uint8(14-i)) & uint8(111)
				}
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			}

			if x == nuint16-1 {
				var area []byte
				for j := -1; j <= 1; j++ {
					// Get the first bit of the leftmost uint16 and the last two of the rightmost uint16
					area[j] = byte(slice[y+j][nuint16-1]&11)<<1 | byte(slice[y+j][0]>>15)
				}
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			} else {
				var area []byte
				for j := -1; j <= 1; j++ {
					area[j] = byte(slice[y+j][x])&11 | byte(slice[y+j][x+1])>>15
				}
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			}

			newLine = append(newLine, newuint16)
		}
	}
}

func golLogic(area []byte) byte {
	cell := (area[1] >> uint8(1)) & 1
	count := bits.OnesCount8(area[0]) + bits.OnesCount8(area[1]) + bits.OnesCount8(area[1]) - int(cell)
	if cell == 1 && (count == 2 || count == 3) {
		return 1
	} else if cell == 0 && count == 2 {
		return 1
	} else {
		return 0
	}
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&GolCommands{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
