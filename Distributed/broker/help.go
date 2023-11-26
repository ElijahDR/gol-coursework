package broker

func calculateStep(p Params, world [][]byte) [][]uint8 {
	var newWorld [][]uint8
	for y, line := range world {
		newLine := make([]uint8, 0)
		for x, cell := range line {
			count := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					//if x+dx < 0 || x+dx >= len(line) {
					//	continue
					//}
					if dx == 0 && dy == 0 {
						continue
					}
					//fmt.Println(x+dx, y+dy)
					//fmt.Println(world)
					newX := (x + dx + p.ImageWidth) % p.ImageWidth
					newY := (y + dy + p.ImageHeight) % p.ImageWidth
					if world[newY][newX] == 255 {
						count += 1
					}
				}
			}
			if cell == 255 {
				if count < 2 {
					//newWorld[y][x] = 0
					newLine = append(newLine, 0)
				} else if count > 3 {
					//newWorld[y][x] = 0
					newLine = append(newLine, 0)
				} else {
					//newWorld[y][x] = 255
					newLine = append(newLine, 255)
				}
			} else {
				if count == 3 {
					//newWorld[y][x] = 255
					newLine = append(newLine, 255)
				} else {
					//newWorld[y][x] = 0
					newLine = append(newLine, 0)
				}
			}
		}

		newWorld = append(newWorld, newLine)
	}

	return newWorld
}
