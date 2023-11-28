package util

import (
	"fmt"
	"math"
	"math/bits"
)

// Converts the standard 2D array of uint8 values of either 0 or 255 into a 2d array of uint16
// Each cell is instead represeted by a binary digit in a uint16 meaning that a 16x16 grid has 16 rows of one uint16.
// Small caveat that this means that every GOL grid given to it is technically a multiple of 16 but seems reasonable.
func ConvertToUint16(world [][]uint8) [][]uint16 {
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

// Calculates how to split a certain number of rows between N "workers"
func CalcSharing(x int, n int) []int {
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

// Converts a uint16 "world" to a uint8 "world"
func ConvertToUint8(world [][]uint16) [][]uint8 {
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

// Calculated the section of world that is needed to calculate a given area. This means that it gives the
// 		row above and below the given area as well.
func CalcSlices(world [][]uint16, height int, n int) [][][]uint16 {
	rows := CalcSharing(height, n)
	start := 0
	// fmt.Println("Rows:", rows)
	var slices [][][]uint16
	for i := 0; i < n; i++ {
		var slice [][]uint16
		// If this is the only slice, just include the whole board with wraparound rows
		if i == 0 && i == n-1 {
			slice = append(slice, world[height-1])
			slice = append(slice, world[start:start+rows[i]]...)
			slice = append(slice, world[0])
		} else if i == 0 {
			// If its the first, give it the last row at the beginning
			slice = append(slice, world[height-1])
			slice = append(slice, world[start:start+rows[i]+1]...)
		} else if i == n-1 {
			// If its the last row, give it the first row at the end
			slice = append(slice, world[start-1:height]...)
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

// Not sure if this is needed anymore, was when I was originally containing the GOL structure exclusively in bytes.
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

func CalcAliveCellsUint16(world [][]uint16) int {
	c := 0
	for _, line := range world {
		for _, u := range line {
			c += bits.OnesCount16(u)
		}
	}
	return c
}

// func CalcOverlapRows(height int, n int, id int) map[int]int {
// 	shares := CalcSharing(height, n)
// 	if id == 0 {
// 		return
// 	}
// }

func GolLogic(area []byte) byte {
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

func SimulateSliceHalo(slice [][]uint16, dataChannel chan [][]uint16, stopChannel chan int, turns int, receiveHaloChannel chan [][]uint16) {
	var data [][]uint16
	sliceSize := len(slice)

	nThreads := int(math.Min(float64(sliceSize), 8))
	startingY := CalcSharing(sliceSize-2, nThreads)

	workerChannels := make([]chan [][]uint16, nThreads)
	for i := 0; i < nThreads; i++ {
		workerChannels[i] = make(chan [][]uint16)
	}

	workingSlice := slice
	PrintUint16World(workingSlice)
	for i := 0; i < turns; i++ {
		if len(stopChannel) > 1 {
			break
		}
		if i > 0 {
			fmt.Println("Waiting for halo channels for turn", i, "...")
			newRegions := <-receiveHaloChannel
			fmt.Println("Received for halo channels for turn", i, "!")
			workingSlice = append([][]uint16{newRegions[0]}, workingSlice...)
			workingSlice = append(workingSlice, newRegions[1])
		}

		currentY := 1
		for j := 0; j < nThreads; j++ {
			go SliceWorker(currentY, currentY+startingY[j], workingSlice, workerChannels[j])
			currentY += startingY[j]
		}

		for j := 0; j < nThreads; j++ {
			d := <-workerChannels[j]
			data = append(data, d...)
		}

		fmt.Println("Finished turn", i, "in SimulateSliceHalo")
		dataChannel <- data
		workingSlice = data
		PrintUint16World(workingSlice)
	}

	stopChannel <- 1
}

func SimulateSlice(slice [][]uint16, dataChannel chan [][]uint16, stopChannel chan int, turns int) [][]uint16 {
	var data [][]uint16
	sliceSize := len(slice)

	nThreads := int(math.Min(float64(sliceSize), 8))
	startingY := CalcSharing(sliceSize-2, nThreads)

	workerChannels := make([]chan [][]uint16, nThreads)
	for i := 0; i < nThreads; i++ {
		workerChannels[i] = make(chan [][]uint16, 2)
	}

	workingSlice := slice

	for i := 0; i < turns; i++ {
		select {
		case <-stopChannel:
			return workingSlice
		default:
			if i > 0 {
				workingSlice = append([][]uint16{workingSlice[len(workingSlice)-1]}, workingSlice...)
				workingSlice = append(workingSlice, workingSlice[0])
			}
			currentY := 1
			for j := 0; j < nThreads; j++ {
				go SliceWorker(currentY, currentY+startingY[j], workingSlice, workerChannels[j])
				currentY += startingY[j]
			}

			for j := 0; j < nThreads; j++ {
				d := <-workerChannels[j]
				// PrintUint16World(d)
				data = append(data, d...)
			}

			// PrintUint16World(data)
			workingSlice = data
			dataChannel <- workingSlice
		}
	}

	stopChannel <- 1
	return slice
}

func SliceWorker(startY int, endY int, slice [][]uint16, c chan [][]uint16) {
	nuint16 := len(slice[0])
	// printRows(slice, y)

	// PrintUint16World(slice[startY:endY])
	// fmt.Println(startY, endY)
	var newSlice [][]uint16
	for y := startY; y < endY; y++ {
		var newLine []uint16
		for x := 0; x < nuint16; x++ {
			var newuint16 uint16

			area := make([]byte, 3)
			if x == 0 {
				for j := -1; j <= 1; j++ {
					// Get the last bit of the furthest right uint16 and the first 2 of the first uint16
					area[j+1] = (byte(slice[y+j][nuint16-1]&1) << 2) | byte(slice[y+j][0]>>14)
				}
			} else {
				for j := -1; j <= 1; j++ {
					area[j+1] = byte(slice[y+j][x-1]&1)<<2 | byte(slice[y+j][x]>>uint8(14))
				}
			}
			newuint16 = uint16(GolLogic(area))

			for i := 1; i < 15; i++ {
				area := make([]byte, 3)
				for j := -1; j <= 1; j++ {
					area[j+1] = byte(slice[y+j][x]>>uint8(14-i)) & uint8(7)
				}

				newuint16 = newuint16<<uint8(1) | uint16(GolLogic(area))
			}

			area = make([]byte, 3)
			if x == nuint16-1 {
				for j := -1; j <= 1; j++ {
					// Get the first bit of the leftmost uint16 and the last two of the rightmost uint16
					area[j+1] = byte(slice[y+j][nuint16-1]&3)<<1 | byte(slice[y+j][0]>>15)
				}
			} else {
				for j := -1; j <= 1; j++ {
					area[j+1] = (byte(slice[y+j][x])&3)<<1 | byte(slice[y+j][x+1]>>15)
				}
				// printArea(area, (x+1)*16, y)]
			}
			newuint16 = newuint16<<uint8(1) | uint16(GolLogic(area))

			newLine = append(newLine, newuint16)
		}
		newSlice = append(newSlice, newLine)
	}

	// PrintUint16World(newSlice)
	c <- newSlice
}
