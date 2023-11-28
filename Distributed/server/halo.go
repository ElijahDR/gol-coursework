package main

import (
	"fmt"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/util"
)

func (s *ServerCommands) HaloExchange(req HaloExchangeReq, res *HaloExchangeRes) (err error) {
	s.slice = req.Slice
	turns := req.Turns
	s.totalTurns = turns
	fmt.Println("Running Halo Exchange...")

	// finalChannel := make(chan [][]uint16, 1)

	res.Slice = runHaloExchange(s, turns)
	res.CurrentTurn = turns
	s.haloRegions = make(map[int][][]uint16)
	s.currentTurn = 0
	fmt.Println("Finished Halo Exchange")
	return
}

func (s *ServerCommands) ReceiveHaloRegions(req HaloRegionReq, res *HaloRegionRes) (err error) {
	region := req.Region
	turn := req.CurrentTurn
	fmt.Println("Receiving halo regions for turn", turn)
	go updateHaloRegions(s, region, turn, req.Type)

	return
}

// The function called by the node requested by the client
func masterHaloExchange(s *ServerCommands, world [][]uint8, turns int) [][]uint8 {
	uint16World := util.ConvertToUint16(world)

	slices := util.CalcSlices(uint16World, len(world), len(NODES)-len(blacklist))

	channels := make([]chan [][]uint16, len(NODES))
	for i := 0; i < len(NODES); i++ {
		channels[i] = make(chan [][]uint16, 2)
	}

	for i, slice := range slices {
		if i == s.id {
			s.slice = slice
			continue
		}
		go callHaloExchange(i, slice, turns, channels[i])
	}

	newSlice := runHaloExchange(s, turns)

	fmt.Println("Combining world...")
	var finalWorld [][]uint16
	for i, channel := range channels {
		fmt.Println("Getting world from", i)
		if i == s.id {
			fmt.Println("That's me!")
			finalWorld = append(finalWorld, newSlice...)
		} else {
			data := <-channel
			finalWorld = append(finalWorld, data...)
		}
	}

	s.haloRegions = make(map[int][][]uint16)
	s.currentTurn = 0
	return util.ConvertToUint8(finalWorld)
}

func runHaloExchange(s *ServerCommands, turns int) [][]uint16 {
	dataChannel := make(chan [][]uint16, 1)
	stopChannels := make(map[string]chan int)
	sendHaloChannel := make(chan haloRegion, 6)
	receiveHaloChannel := make(chan [][]uint16)
	s.haloRegions = make(map[int][][]uint16)
	s.currentTurn = 0

	stopChannels["simulator"] = make(chan int, 1)
	go util.SimulateSliceHalo(s.slice, dataChannel, stopChannels["simulator"], turns, receiveHaloChannel)

	stopChannels["sliceUpdater"] = make(chan int, 1)
	go updateSliceHalo(s, dataChannel, stopChannels["sliceUpdater"], sendHaloChannel)

	stopChannels["sendHaloRegions"] = make(chan int, 1)
	go sendHaloRegions(s, sendHaloChannel, stopChannels["sendHaloRegions"])

	stopChannels["receiveHaloRegions"] = make(chan int, 1)
	go receiveHaloRegions(s, receiveHaloChannel, stopChannels["receiveHaloRegions"])

	fmt.Println("Waiting for finish...")
	<-stopChannels["simulator"]
	fmt.Println("Finished")

	for _, stopChannel := range stopChannels {
		stopChannel <- 1
	}

	return s.slice
}

func makeSendHalo(id int, req HaloRegionReq) {
	client, _ := rpc.Dial("tcp", NODES[id]+":8030")
	defer client.Close()
	response := new(HaloRegionRes)
	client.Call("ServerCommands.ReceiveHaloRegions", req, response)
}

func updateSliceHalo(s *ServerCommands, dataChannel chan [][]uint16, stopChannel chan int, sendHaloChannel chan haloRegion) {
	for {
		select {
		case <-stopChannel:
			break
		case newSlice := <-dataChannel:
			fmt.Println("Updating slice and sending halo regions... for turn", s.currentTurn+1)
			s.mu.Lock()
			s.slice = newSlice
			s.currentTurn++
			regions := append(append([][]uint16{}, s.slice[1]), s.slice[len(s.slice)-2])
			newRegion := haloRegion{regions: regions, currentTurn: int(s.currentTurn)}
			sendHaloChannel <- newRegion
			s.mu.Unlock()
			fmt.Println("Finished updating slice and sending halo regions... for turn", s.currentTurn)
		default:
		}
	}
}

func sendHaloRegions(s *ServerCommands, sendHaloChannel chan haloRegion, stopChannel chan int) {
	for {
		select {
		case <-stopChannel:
			break
		case region := <-sendHaloChannel:
			go makeHaloExchange(s, region)
		default:
		}
	}
}

func receiveHaloRegions(s *ServerCommands, receiveHaloChannel chan [][]uint16, stopChannel chan int) {
	haloTurn := 1
	for {
		select {
		case <-stopChannel:
			break
		default:
			if len(s.haloRegions[haloTurn]) == 2 {
				fmt.Println("Sending halo regions down channel to worker...", haloTurn)
				var regions [][]uint16
				for i := 0; i < len(s.haloRegions[haloTurn]); i++ {
					regions = append(regions, s.haloRegions[haloTurn][i])
				}
				fmt.Println("Before delete:", regions)
				receiveHaloChannel <- regions
				s.haloLock.Lock()
				delete(s.haloRegions, haloTurn)
				s.haloLock.Unlock()
				haloTurn++
				fmt.Println("After delete:", regions)
				fmt.Println("Finished sending halo regions down channel to worker...", haloTurn)
			}
		}
	}
}

func makeHaloExchange(s *ServerCommands, region haloRegion) {
	if region.currentTurn >= s.totalTurns {
		fmt.Println("No need for more halo exchange...")
		return
	}
	bottomID := ((s.id - 1) + (len(NODES))) % (len(NODES))
	topID := (s.id + 1) % len(NODES)

	fmt.Println("Sending Halo Regions from", s.id, "to", bottomID, "for turn", region.currentTurn)
	request := HaloRegionReq{
		Region:      region.regions[0],
		CurrentTurn: region.currentTurn,
		Type:        1,
	}
	makeSendHalo(bottomID, request)

	fmt.Println("Sending Halo Regions from", s.id, "to", topID, "for turn", region.currentTurn)
	request = HaloRegionReq{
		Region:      region.regions[1],
		CurrentTurn: region.currentTurn,
		Type:        0,
	}
	makeSendHalo(topID, request)
}

func updateHaloRegions(s *ServerCommands, region []uint16, turn int, haloType int) {
	// fmt.Println("Waiting to unlock halo lock...")
	s.haloLock.Lock()
	// fmt.Println("Unlocked!")
	_, exists := s.haloRegions[turn]
	if exists {
		if haloType == 0 {
			s.haloRegions[turn] = append([][]uint16{region}, s.haloRegions[turn]...)
		} else {
			s.haloRegions[turn] = append(s.haloRegions[turn], region)
		}
	} else {
		s.haloRegions[turn] = [][]uint16{region}
	}
	s.haloLock.Unlock()
	fmt.Println("Current Halo Regions:", s.haloRegions)
}

func callHaloExchange(id int, slice [][]uint16, turns int, channel chan [][]uint16) {
	destIP := NODES[id] + ":8030"
	fmt.Println("Asking", destIP, "to partake in Halo Exchange")

	client, err := rpc.Dial("tcp", destIP)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	request := HaloExchangeReq{
		Slice: slice,
		Turns: turns,
	}
	response := new(HaloExchangeRes)
	client.Call("ServerCommands.HaloExchange", request, response)

	channel <- response.Slice
}
