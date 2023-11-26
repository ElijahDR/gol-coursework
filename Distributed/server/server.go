package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
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
	if height < 512 && width < 512 {
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

func (s *ServerCommands) ReceiveHaloRegions(req HaloRegionReq, res *HaloRegionRes) (err error) {
	region := req.Region
	turn := req.CurrentTurn
	fmt.Println("Receiving halo regions for turn", turn)
	go updateHaloRegions(s, region, turn, req.Type)

	return
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
