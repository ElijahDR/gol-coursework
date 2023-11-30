package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
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

var CONNECTIONS []*rpc.Client

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
	s.totalTurns = turns

	s.keyPresses = make(chan rune, 5)
	s.returnMain = make(chan bool)

	if turns == 0 {
		res.World = world
		return
	}

	fmt.Println("Server Received Request:", width, "x", height, "for", req.Turns, "turns")
	// if height < 64 && width < 64 {
	// 	res.World = masterLocal(s, world, turns)
	// if height < 512 && width < 512 {
	// 	res.World = masterNormal(s, world, turns)
	// } else {
	// 	res.World = masterHaloExchange(s, world, turns)
	// }
	go masterNormal(s, world, turns)
	// res.World = masterNormal(s, world, turns)
	<-s.returnMain
	res.World = util.ConvertToUint8(s.currentWorld)
	// util.PrintUint8World(res.World)

	s.currentTurn = 0
	return
}

func (s *ServerCommands) KeyPress(req KeyPressRequest, res *KeyPressResponse) (err error) {
	key := req.Key
	fmt.Println("KEY PRESSED", string(key))
	s.keyPresses <- key
	res.World = util.ConvertToUint8(s.currentWorld)
	res.Turn = s.currentTurn
	return
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
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	flag.StringVar(&args.port, "port", "8030", "Port to listen on")
	flag.StringVar(&args.ip, "ip", "127.0.0.1", "IP of this machine")
	// pAddr := flag.String("port", "8031", "Port to listen on")
	flag.Parse()

	if args.ip == "127.0.0.1" {
		fmt.Println("Running as 127.0.0.1, did you set this correct?")
	}
	// reader := bufio.NewReader(os.Stdin)
	// blank, _ := reader.ReadString('\n')
	// fmt.Println(blank)

	CONNECTIONS = make([]*rpc.Client, len(NODES))
	id := -1
	for i, ip := range NODES {
		if ip == args.ip {
			id = i
		}
	}
	if id == -1 {
		panic("ID not in list of nodes, please update")
	}

	quit := make(chan bool)
	rpc.Register(&ServerCommands{id: id, quit: quit})
	listener, _ := net.Listen("tcp", ":"+args.port)
	fmt.Println("I am", args.ip+":"+args.port)
	defer listener.Close()

	for i, ip := range NODES {
		if i == id {
			continue
		}
		for {
			client, _ := rpc.Dial("tcp", ip+":8030")
			CONNECTIONS[i] = client
			if client != nil {
				break
			}
		}
	}
	fmt.Println(CONNECTIONS)

	go rpc.Accept(listener)
	<-quit

	for i, conn := range CONNECTIONS {
		if i == id {
			continue
		}
		conn.Close()
	}
}
