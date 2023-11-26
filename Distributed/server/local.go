package main

import "uk.ac.bris.cs/gameoflife/util"

func masterLocal(s *ServerCommands, world [][]uint8, turns int) [][]uint8 {
	uint16World := util.ConvertToUint16(world)

	dataChannel := make(chan [][]uint16)
	stopChannel := make(chan int)

	slices := util.CalcSlices(uint16World, len(world), 1)
	go util.SimulateSlice(slices[0], dataChannel, stopChannel, turns)
	go updateSliceLocal(s, dataChannel, stopChannel)
	<-stopChannel

	return util.ConvertToUint8(s.slice)
}

func updateSliceLocal(s *ServerCommands, dataChannel chan [][]uint16, stopChannel chan int) {
	for {
		select {
		case <-stopChannel:
			break
		case newSlice := <-dataChannel:
			s.mu.Lock()
			s.slice = newSlice
			s.currentTurn++
			s.mu.Unlock()
		default:
		}
	}
}
