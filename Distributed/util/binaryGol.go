package util

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

func CalculateNRows(p Params, n int) []int {
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
