package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/rpc"
	"time"

	"uk.ac.bris.cs/gameoflife/util"
)

var NODES = []string{
	"23.22.135.15",
	"35.174.225.191",
	"44.208.149.39",
	"3.214.156.90",
	"44.208.47.178",
	"3.93.57.107",
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
	if s.currentTurn != 0 && len(req.World) == len(s.currentWorld) && s.totalTurns == req.Turns {
		world := s.currentWorld
		turns := req.Turns - s.currentTurn
		height := len(world)
		width := len(world[0]) * 16

		fmt.Println("Server Received Continuation of:", width, "x", height, "for", req.Turns, "turns from turn", s.currentTurn)
		// if height < 64 && width < 64 {
		// 	res.World = masterLocal(s, world, turns)
		// if height < 512 && width < 512 {
		// 	res.World = masterNormal(s, world, turns)
		// } else {
		// 	res.World = masterHaloExchange(s, world, turns)
		// }
		go masterNormal(s, world, turns, req.Threads)
		// res.World = masterNormal(s, world, turns)
		code := <-s.returnMain
		res.World = util.ConvertToUint8(s.currentWorld)
		// util.PrintUint8World(res.World)
		if code == 0 {
			s.currentTurn = 0
		}
		// s.broker = false
		if code == 2 {
			defer func() {
				go func() {
					if s.broker {
						s.quit <- 2
					} else {
						s.quit <- 1
					}
				}()
			}()
		}
		return

	}
	world := req.World
	turns := req.Turns
	height := len(world)
	width := len(world[0])
	s.totalTurns = turns
	s.broker = true

	s.keyPresses = make(chan rune, 5)
	s.returnMain = make(chan int)

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
	go masterNormal(s, util.ConvertToUint16(world), turns, req.Threads)
	// res.World = masterNormal(s, world, turns)
	code := <-s.returnMain
	res.World = util.ConvertToUint8(s.currentWorld)
	// util.PrintUint8World(res.World)
	if code == 0 {
		s.currentTurn = 0
	}
	// s.broker = false
	fmt.Println("Returned main RUNGOL Request")
	if code == 2 {
		defer func() {
			go func() {
				if s.broker {
					s.quit <- 2
				} else {
					s.quit <- 1
				}
			}()
		}()
	}
	return

}

func (s *ServerCommands) KeyPress(req KeyPressRequest, res *KeyPressResponse) (err error) {
	key := req.Key
	fmt.Println("Key Press Request:", string(key))
	res.World = util.ConvertToUint8(s.currentWorld)
	res.Turn = s.currentTurn
	s.keyPresses <- key
	// fmt.Println("returned keyPress")
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

func (s *ServerCommands) Quit(req QuitReq, res *QuitRes) (err error) {
	fmt.Println("Shutting Down...")
	s.quit <- 1
	return nil
}

func (s *ServerCommands) Ping(req PingReq, res *PingRes) (err error) {
	res.Ping = true
	return
}

func (s *ServerCommands) NominateBroker(req NomBrokerReq, res *NomBrokerRes) (err error) {
	if s.broker {
		fmt.Println("I am already broker!")
		res.ID = s.id
		return
	}
	totalPing := 0
	for i, ip := range NODES {
		if i == s.id {
			continue
		}
		server := ip + ":8030"
		client, _ := rpc.Dial("tcp", server)

		request := PingReq{}
		response := new(PingRes)
		t := time.Now()
		client.Call("ServerCommands.Ping", request, response)
		ping := int(time.Since(t) / time.Millisecond)
		totalPing += ping
		client.Close()
	}

	min := totalPing
	id := s.id
	for i, ip := range NODES {
		if i == s.id {
			continue
		}
		server := ip + ":8030"
		client, _ := rpc.Dial("tcp", server)

		request := TotalPingReq{}
		response := new(TotalPingRes)
		client.Call("ServerCommands.TotalPing", request, response)

		if response.Broker {
			fmt.Println(ip, "already broker!")
			res.ID = i
			return
		}

		// fmt.Println("Still running")
		if response.TotalPing < min {
			id = i
			min = response.TotalPing
		}

		client.Close()
	}

	res.ID = id
	return
}

func (s *ServerCommands) TotalPing(req TotalPingReq, res *TotalPingRes) (err error) {
	totalPing := 0
	if s.broker {
		res.Broker = true
		return
	}
	for i, ip := range NODES {
		if i == s.id {
			continue
		}
		server := ip + ":8030"
		client, _ := rpc.Dial("tcp", server)

		request := PingReq{}
		response := new(PingRes)
		t := time.Now()
		client.Call("ServerCommands.Ping", request, response)
		ping := int(time.Since(t) / time.Millisecond)
		totalPing += ping
		client.Close()
	}
	res.TotalPing = totalPing
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

	quit := make(chan int)
	broker := false
	rpc.Register(&ServerCommands{id: id, quit: quit, broker: broker, currentTurn: 0})
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
	code := <-quit
	time.Sleep(2 * time.Second)
	if code == 2 {
		// for i, conn := range CONNECTIONS {
		// 	fmt.Println("closing", i, "...")
		// 	if i == id {
		// 		continue
		// 	}
		// 	req := new(QuitReq)
		// 	res := new(QuitRes)
		// 	conn.Call("ServerCommands.Quit", req, res)
		// 	defer conn.Close()
		// 	fmt.Println("closed", i, "!")
		// }
		for i, ip := range NODES {
			if i == id {
				continue
			}
			client, _ := rpc.Dial("tcp", ip+":8030")
			req := QuitReq{}
			res := new(QuitRes)
			client.Call("ServerCommands.Quit", req, res)
		}
	}
}
