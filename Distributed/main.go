package main

import (
	"flag"
	"fmt"
	"runtime"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/sdl"
)

func getGOMAXPROCS() int {
	return runtime.GOMAXPROCS(0)
}

// main is the function called when starting Game of Life with 'go run .'
func main() {
	runtime.LockOSThread()
	var params gol.Params

	flag.IntVar(
		&params.Threads,
		"t",
		8,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.ImageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.ImageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.IntVar(
		&params.Turns,
		"turns",
		10000000000,
		"Specify the number of turns to process. Defaults to 10000000000.")

	// flag.StringVar(
	// 	&params.Type,
	// 	"type",
	// 	"p",
	// 	"Specify the type of GOL to run: p for parallel, d for distributed")

	noVis := flag.Bool(
		"noVis",
		false,
		"Disables the SDL window, so there is no visualisation during the tests.")

	flag.Parse()

	fmt.Println("Threads:", params.Threads)
	fmt.Println("Width:", params.ImageWidth)
	fmt.Println("Height:", params.ImageHeight)
	// fmt.Printf("GOMAXPROCS is %d\n", getGOMAXPROCS())

	// if params.Type == "p" {
	// 	fmt.Println("Running Parallel")
	// } else if params.Type == "d" {
	// 	fmt.Println("Running Distributed")
	// } else {
	// 	panic("Unkown type! Please use p or d")
	// }

	keyPresses := make(chan rune, 10)
	events := make(chan gol.Event, 1000)

	go gol.Run(params, events, keyPresses)
	if !(*noVis) {
		sdl.Run(params, events, keyPresses)
	} else {
		complete := false
		for !complete {
			event := <-events
			switch event.(type) {
			case gol.FinalTurnComplete:
				complete = true
			}
		}
	}
}
