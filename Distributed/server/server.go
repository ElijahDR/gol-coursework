package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/util"
)

var NODES = []string{
	"23.22.135.15",
	"35.174.225.191",
	"44.208.149.39",
	"3.214.156.90",
	"44.208.47.178",
}

type haloRegion struct {
	regions     [][]uint16
	currentTurn int
}

type sliceInfo struct {
	slice [][]uint16
	turn  int
}

var blacklist = []int{}

func (s *ServerCommands) RunGOL(req GolRequest, res *GolResponse) (err error) {
	world := req.World
	turns := req.Turns
	height := len(world)
	width := len(world[0])

	if turns == 0 {
		res.World = world
		return
	}

	fmt.Println("Server Received Request:", width, "x", height, "for", req.Turns, "turns")
	if height < 64 && width < 64 {
		res.World = masterNormal(s, world, turns)
	} else {
		res.World = masterHaloExchange(s, world, turns)
	}

	s.currentTurn = 0
	return
}

func (s *ServerCommands) IterateSlice(req IterateSliceReq, res *IterateSliceRes) (err error) {
	slice := req.Slice
	res.Slice = iterateSlice(slice)
	return
}

func iterateSlice(slice [][]uint16) [][]uint16 {
	dataChannel := make(chan [][]uint16)
	stopChannel := make(chan int)
	go util.SimulateSlice(slice, dataChannel, stopChannel, 1, nil)
	data := <-dataChannel
	return data
}

func masterNormal(s *ServerCommands, world [][]uint8, turns int) [][]uint8 {
	uint16World := util.ConvertToUint16(world)

	channels := make([]chan [][]uint16, len(NODES))
	for i := 0; i < len(NODES); i++ {
		channels[i] = make(chan [][]uint16, 2)
	}

	for j := 0; j < turns; j++ {
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
	}

	return util.ConvertToUint8(uint16World)
}

func callIterateSlice(id int, slice [][]uint16, channel chan [][]uint16) {
	destIP := NODES[id] + ":8030"
	fmt.Println("Asking", destIP, "to partake in Halo Exchange")

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

func (s *ServerCommands) HaloExchange(req HaloExchangeReq, res *HaloExchangeRes) (err error) {
	s.slice = req.Slice
	turns := req.Turns
	fmt.Println("Running Halo Exchange...")

	runHaloExchange(s, turns)

	res.Slice = s.slice
	res.CurrentTurn = turns
	s.haloRegions = make(map[int][][]uint16)
	s.currentTurn = 0
	return
}

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

	// fmt.Println("Combining world...")
	var finalWorld [][]uint16
	for i, channel := range channels {
		// fmt.Println("Getting world from", i)
		if i == s.id {
			// fmt.Println("That's me!")
			finalWorld = append(finalWorld, newSlice...)
		} else {
			data := <-channel
			finalWorld = append(finalWorld, data...)
		}
	}

	return util.ConvertToUint8(finalWorld)
}

func runHaloExchange(s *ServerCommands, turns int) [][]uint16 {
	dataChannel := make(chan [][]uint16)
	stopChannels := make(map[string]chan int)
	sendHaloChannel := make(chan haloRegion, 100)
	receiveHaloChannel := make(chan [][]uint16, 10)
	s.haloRegions = make(map[int][][]uint16)

	stopChannels["simulator"] = make(chan int)
	go util.SimulateSlice(s.slice, dataChannel, stopChannels["simulator"], turns, receiveHaloChannel)

	stopChannels["sliceUpdater"] = make(chan int)
	go sliceUpdater(s, dataChannel, stopChannels["sliceUpdater"], sendHaloChannel)

	stopChannels["sendHaloRegions"] = make(chan int)
	go sendHaloRegions(s, sendHaloChannel, stopChannels["sendHaloRegions"])

	stopChannels["receiveHaloRegions"] = make(chan int)
	go receiveHaloRegions(s, receiveHaloChannel, stopChannels["receiveHaloRegions"])

	fmt.Println("Waiting for finish...")
	<-stopChannels["simulator"]
	fmt.Println("Finished")

	return s.slice
}

func receiveHaloRegions(s *ServerCommands, receiveHaloChannel chan [][]uint16, stopChannel chan int) {
	haloTurn := 1
	for {
		select {
		case <-stopChannel:
			break
		default:
			if len(s.haloRegions[haloTurn]) == 2 {
				receiveHaloChannel <- s.haloRegions[haloTurn]
				s.haloLock.Lock()
				delete(s.haloRegions, haloTurn)
				s.haloLock.Unlock()
				haloTurn++
			}
		}
	}
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

func (s *ServerCommands) ReceiveHaloRegions(req HaloRegionReq, res *HaloRegionRes) (err error) {
	region := req.Region
	turn := req.CurrentTurn
	fmt.Println("Receiving halo regions for turn", turn)
	go updateHaloRegions(s, region, turn, req.Type)

	return
}

func makeSendHalo(id int, req HaloRegionReq) {
	client, _ := rpc.Dial("tcp", NODES[id]+":8030")
	defer client.Close()
	response := new(HaloRegionRes)
	client.Call("ServerCommands.ReceiveHaloRegions", req, response)
}

func makeHaloExchange(s *ServerCommands, region haloRegion) {
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

func sliceUpdater(s *ServerCommands, dataChannel chan [][]uint16, stopChannel chan int, sendHaloChannel chan haloRegion) {
	for {
		select {
		case <-stopChannel:
			break
		case newSlice := <-dataChannel:
			s.mu.Lock()
			s.slice = newSlice
			s.currentTurn++
			regions := append(append([][]uint16{}, s.slice[1]), s.slice[len(s.slice)-2])
			sendHaloChannel <- haloRegion{regions: regions, currentTurn: s.currentTurn}
			s.mu.Unlock()
		default:
		}
	}
}

// func currentHaloRegions(s *ServerCommands) haloRegion {
// 	var regions [][]uint16
// 	s.mu.Lock()
// 	regions = append(append(regions, s.slice[1]), s.slice[len(s.slice)-2])
// 	turn := s.currentTurn
// 	s.mu.Unlock()

// 	return haloRegion{
// 		regions:     regions,
// 		currentTurn: turn,
// 	}
// }

func (s *ServerCommands) CheckAlive(req CheckAliveReq, res *CheckAliveRes) (err error) {
	res.ResponseID = s.id
	return
}

func checkAlive(myID int, checkID int) {
	destIP := NODES[checkID]
	client, error := rpc.Dial("tcp", destIP+":8030")
	if error != nil {
		fmt.Println("Error connecting to", destIP)
		fmt.Println("Adding to blacklist...")
		blacklist = append(blacklist, checkID)
	}
	defer client.Close()
	request := CheckAliveReq{
		RequestID: myID,
	}
	response := new(CheckAliveRes)
	client.Call("ServerCommands.CheckAlive", request, response)
	if response.ResponseID == checkID {
		fmt.Println("ID:", checkID, ", IP:", destIP, "all good")
	}
}

func main() {
	var args ServerArgs
	flag.StringVar(&args.port, "port", "8030", "Port to listen on")
	flag.StringVar(&args.ip, "ip", "127.0.0.1", "IP of this machine")
	// pAddr := flag.String("port", "8031", "Port to listen on")
	flag.Parse()

	if args.ip == "127.0.0.1" {
		fmt.Println("Running as 127.0.0.1, did you set this correct?")
	}
	id := -1
	for i, ip := range NODES {
		if ip == args.ip {
			id = i
		}
	}
	if id == -1 {
		panic("ID not in list of nodes, please update")
	}
	rpc.Register(&ServerCommands{id: id})
	listener, _ := net.Listen("tcp", ":"+args.port)
	fmt.Println("I am", args.ip+":"+args.port)
	defer listener.Close()
	rpc.Accept(listener)
}
