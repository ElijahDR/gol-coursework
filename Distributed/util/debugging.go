package util

import "fmt"

func PrintArea(area []byte, x int, y int) {
	fmt.Println("printing area at", x, y)
	for _, line := range area {
		fmt.Printf("%03b\n", line)
	}
}

func PrintRows(arr [][]uint16, y int) {
	fmt.Printf("%0v %016b\n", y-1, arr[y-1])
	fmt.Printf("%0v %016b\n", y, arr[y])
	fmt.Printf("%0v %016b\n", y+1, arr[y+1])
}

func PrintUint16World(arr [][]uint16) {
	for _, line := range arr {
		for _, u := range line {
			fmt.Printf("%016b", u)
		}
		fmt.Print("\n")
	}
	fmt.Print("\n")
}

func PrintLine(line []uint8) {
	for _, cell := range line {
		if cell == 255 {
			fmt.Print(1)
		} else {
			fmt.Print(0)
		}
	}
	fmt.Println()
}

func PrintUint8World(world [][]uint8) {
	for _, line := range world {
		for _, cell := range line {
			if cell == 255 {
				fmt.Print(1)
			} else {
				fmt.Print(0)
			}
		}
		fmt.Println()
	}
}

func CompareWorlds(w1 [][]uint8, w2 [][]uint16) bool {
	var str1 string
	var str2 string
	for y := 0; y < len(w1); y++ {
		for _, cell := range w1[y] {
			if cell == 255 {
				str1 += "1"
			} else {
				str1 += "0"
			}
		}

		for _, u := range w2[y] {
			str2 += fmt.Sprintf("%016b", u)
		}
	}

	fmt.Println(str1 == str2)
	return str1 == str2
}
