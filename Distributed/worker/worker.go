package main

import (
	"flag"
	"math/bits"
	"net"
	"net/rpc"
)

func (g *GolCommands) GOLWorker(req GolWorkerRequest, res *GolWorkerResponse) (err error) {
	// fmt.Println("Worker", req.ID, "received slice")

	// area := []byte{
	// 	byte(7),
	// 	byte(0),
	// 	byte(0),
	// }
	// fmt.Println(golLogic(area))
	// area = []byte{
	// 	byte(6),
	// 	byte(2),
	// 	byte(0),
	// }
	// fmt.Println(golLogic(area))
	// area = []byte{
	// 	byte(1),
	// 	byte(2),
	// 	byte(0),
	// }
	// fmt.Println(golLogic(area))

	slice := req.Slice
	// fmt.Println("Slice:", slice)

	// nThreads := 8
	// startingY := calcThreads(len(slice)-2, nThreads)

	// channels := make([]chan [][]uint16, nThreads)
	// for i := 0; i < len(channels); i++ {
	// 	channels[i] = make(chan [][]uint16, 2)
	// }

	// current := 1
	// for i := 0; i < len(channels); i++ {
	// 	go parallelWorker(current, current+startingY[i], slice, channels[i])
	// 	current += startingY[i]
	// }

	// var data [][]uint16
	// for i := 0; i < len(channels); i++ {
	// 	d := <-channels[i]
	// 	data = append(data, d...)
	// }

	workerChan := make(chan [][]uint16)
	go worker(slice, workerChan)
	data := <-workerChan

	// fmt.Println("New Slice:", data)
	res.Slice = data

	return
}

func calcThreads(x int, n int) []int {
	rowsEach := x / n
	nBigger := x - (rowsEach * n)
	var rows []int
	for i := 0; i < n-nBigger; i++ {
		rows = append(rows, rowsEach)
	}
	for i := 0; i < nBigger; i++ {
		rows = append(rows, rowsEach+1)
	}

	return rows
}

func parallelWorker(startY int, endY int, slice [][]uint16, c chan [][]uint16) {
	nuint16 := len(slice[0])
	// printRows(slice, y)
	var newSlice [][]uint16
	for y := startY; y < endY; y++ {
		var newLine []uint16
		for x := 0; x < nuint16; x++ {
			var newuint16 uint16

			if x == 0 {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					// Get the last bit of the furthest right uint16 and the first 2 of the first uint16
					area[j+1] = (byte(slice[y+j][nuint16-1]&1) << 2) | byte(slice[y+j][0]>>14)
				}
				newuint16 = uint16(golLogic(area))
			} else {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = byte(slice[y+j][x-1]&1)<<2 | byte(slice[y+j][x]>>uint8(14))
				}
				newuint16 = uint16(golLogic(area))
			}

			for i := 1; i < 15; i++ {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = byte(slice[y+j][x]>>uint8(14-i)) & uint8(7)
				}

				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			}

			if x == nuint16-1 {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					// Get the first bit of the leftmost uint16 and the last two of the rightmost uint16
					area[j+1] = byte(slice[y+j][nuint16-1]&3)<<1 | byte(slice[y+j][0]>>15)
				}
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			} else {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = (byte(slice[y+j][x])&3)<<1 | byte(slice[y+j][x+1]>>15)
				}
				// printArea(area, (x+1)*16, y)
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			}

			newLine = append(newLine, newuint16)
		}
		newSlice = append(newSlice, newLine)
	}

	c <- newSlice
}

// Doing it this was is my biggest regret since taking physics a level
func worker(slice [][]uint16, c chan [][]uint16) {
	nuint16 := len(slice[0])
	var newSlice [][]uint16
	for y := 1; y < len(slice)-1; y++ {
		// printRows(slice, y)
		var newLine []uint16
		for x := 0; x < nuint16; x++ {
			var newuint16 uint16

			if x == 0 {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					// Get the last bit of the furthest right uint16 and the first 2 of the first uint16
					area[j+1] = (byte(slice[y+j][nuint16-1]&1) << 2) | byte(slice[y+j][0]>>14)
				}
				newuint16 = uint16(golLogic(area))
			} else {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = byte(slice[y+j][x-1]&1)<<2 | byte(slice[y+j][x]>>uint8(14))
				}
				newuint16 = uint16(golLogic(area))
			}

			for i := 1; i < 15; i++ {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = byte(slice[y+j][x]>>uint8(14-i)) & uint8(7)
				}

				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			}

			if x == nuint16-1 {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					// Get the first bit of the leftmost uint16 and the last two of the rightmost uint16
					area[j+1] = byte(slice[y+j][nuint16-1]&3)<<1 | byte(slice[y+j][0]>>15)
				}
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			} else {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = (byte(slice[y+j][x])&3)<<1 | byte(slice[y+j][x+1]>>15)
				}
				// printArea(area, (x+1)*16, y)
				newuint16 = newuint16<<uint8(1) | uint16(golLogic(area))
			}

			newLine = append(newLine, newuint16)
		}
		newSlice = append(newSlice, newLine)
	}

	c <- newSlice
}

func golLogic(area []byte) byte {
	// for _, c := range area {
	// 	fmt.Printf("%03b\n", c)
	// }
	cell := (area[1] >> uint8(1)) & 1
	// fmt.Println("Cell:", cell)
	count := bits.OnesCount8(area[0]) + bits.OnesCount8(area[1]) + bits.OnesCount8(area[2]) - int(cell)
	// fmt.Println("Count:", count)
	if cell == 1 && (count == 2 || count == 3) {
		return 1
	} else if cell == 0 && count == 3 {
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
