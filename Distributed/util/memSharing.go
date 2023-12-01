package util

func SimulateSliceMemSharing(slice *[][]uint16, stopChannel chan int, turns int, nThreads int, currentTurn *int) {
	sliceSize := len(*slice)

	// nThreads := int(math.Min(float64(sliceSize), 8))
	// nThreads = 1

	startingY := CalcSharing(sliceSize, nThreads)

	workerChannels := make([]chan bool, nThreads)
	for i := 0; i < nThreads; i++ {
		workerChannels[i] = make(chan bool)
	}

	var workerRows [][]int
	current := 1
	for j := 0; j < nThreads; j++ {
		workerRows = append(workerRows, []int{current, current + startingY[j]})
		current += startingY[j]
	}

	for i := 0; i < turns; i++ {
		// start := time.Now()
		workingSlice := *slice
		select {
		case <-stopChannel:
			return
		default:
			workingSlice = append([][]uint16{workingSlice[len(workingSlice)-1]}, workingSlice...)
			workingSlice = append(workingSlice, workingSlice[1])

			// fmt.Println("Current Turn Working Slice:", i)
			// PrintUint16World(workingSlice)
			var newSlice [][]uint16
			newSlice = make([][]uint16, sliceSize)
			// fmt.Println(newSlice)
			for j := 0; j < nThreads; j++ {
				go SliceWorkerMemSharing(workerRows[j][0], workerRows[j][1], workingSlice, &newSlice, workerChannels[j])
			}

			for j := 0; j < nThreads; j++ {
				<-workerChannels[j]
			}
			*slice = newSlice
			(*currentTurn)++

		}
	}
	// PrintUint16World(*slice)
	stopChannel <- 1
}

func SliceWorkerMemSharing(startY int, endY int, slice [][]uint16, newSlice *[][]uint16, done chan bool) {
	nuint16 := len(slice[0])
	// printRows(slice, y)

	// fmt.Println("Working Incoming")
	// fmt.Println(slice[startY:endY])
	// fmt.Println(startY, endY)
	// fmt.Println("Length of slice given to worker:", len(slice))
	for y := startY; y < endY; y++ {
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

			(*newSlice)[y-1] = append((*newSlice)[y-1], newuint16)
		}
	}

	done <- true
}
