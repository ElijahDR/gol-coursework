package main

import "uk.ac.bris.cs/gameoflife/util"

func masterLocal(s *ServerCommands, world [][]uint8, turns int) [][]uint8 {
	uint16World := util.ConvertToUint16(world)

	dataChannel := make(chan [][]uint16)

	slices := util.CalcSlices(uint16World, len(world), 1)
	stopChannelSim := make(chan int)
	go util.SimulateSlice(slices[0], dataChannel, stopChannelSim, turns)
	stopChannelUpdate := make(chan int)
	go updateSliceLocal(s, dataChannel, stopChannelUpdate)
	<-stopChannelSim

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
