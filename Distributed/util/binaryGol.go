package util

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
func CalcSlices(world [][]uint16, p Params, n int) [][][]uint16 {
	rows := CalcSharing(p.ImageHeight, n)
	start := 0
	// fmt.Println("Rows:", rows)
	var slices [][][]uint16
	for i := 0; i < n; i++ {
		var slice [][]uint16
		// If this is the only slice, just include the whole board with wraparound rows
		if i == 0 && i == n-1 {
			slice = append(slice, world[p.ImageHeight-1])
			slice = append(slice, world[start:start+rows[i]]...)
			slice = append(slice, world[0])
		} else if i == 0 {
			// If its the first, give it the last row at the beginning
			slice = append(slice, world[p.ImageHeight-1])
			slice = append(slice, world[start:start+rows[i]+1]...)
		} else if i == n-1 {
			// If its the last row, give it the first row at the end
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
