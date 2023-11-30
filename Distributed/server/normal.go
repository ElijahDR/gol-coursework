package main

import (
	"math"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/util"
)

func masterNormal(s *ServerCommands, world [][]uint16, turns int) [][]uint8 {
	uint16World := world

	channels := make([]chan [][]uint16, len(NODES))
	for i := 0; i < len(NODES); i++ {
		channels[i] = make(chan [][]uint16, 2)
	}

	for j := 0; j < turns; j++ {
		if len(s.keyPresses) > 0 {
			key := <-s.keyPresses
			if key == 'p' {
				for {
					key := <-s.keyPresses
					if key == 'p' {
						break
					}
				}
			} else if key == 'q' {
				s.returnMain <- 1
				return util.ConvertToUint8(uint16World)
			} else if key == 'k' {
				defer func() {
					go func() {
						s.returnMain <- 2
					}()
				}()
				return util.ConvertToUint8(uint16World)
			}
		}
		slices := util.CalcSlices(uint16World, len(world), len(NODES)-len(blacklist))
		for i, slice := range slices {
			if i == s.id {
				s.slice = slice
				continue
			}
			go callIterateSlice(i, slice, channels[i])
		}

		newSlice := iterateSlice(s.slice)

		// fmt.Println("Combining world...")
		var newWorld [][]uint16
		for i, channel := range channels {
			// fmt.Println("Getting world from", i)
			if i == s.id {
				// fmt.Println("That's me!")
				newWorld = append(newWorld, newSlice...)
			} else {
				data := <-channel
				newWorld = append(newWorld, data...)
			}
		}

		uint16World = newWorld
		s.currentWorld = uint16World
		s.currentTurn = j + 1
	}

	s.returnMain <- 0
	return util.ConvertToUint8(uint16World)
}

func callIterateSlice(id int, slice [][]uint16, channel chan [][]uint16) {
	destIP := NODES[id] + ":8030"
	// fmt.Println("Asking", destIP, "to iterate slice")

	client, err := rpc.Dial("tcp", destIP)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	request := IterateSliceReq{
		Slice: slice,
	}
	response := new(HaloExchangeRes)
	client.Call("ServerCommands.IterateSlice", request, response)

	channel <- response.Slice
}

func (s *ServerCommands) IterateSlice(req IterateSliceReq, res *IterateSliceRes) (err error) {
	slice := req.Slice
	newSlice := iterateSlice(slice)
	res.Slice = newSlice
	return
}

func iterateSlice(slice [][]uint16) [][]uint16 {
	dataChannel := make(chan [][]uint16)
	stopChannel := make(chan int)
	nThreads := int(math.Min(float64(len(slice)), 8))
	go util.SimulateSlice(slice, dataChannel, stopChannel, 1, nThreads)
	data := <-dataChannel
	return data
}
