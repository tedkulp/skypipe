package main

import (
	"os"
	"fmt"
	"log"
	zmq "github.com/pebbe/zmq4"
)

func PrintError(err error) {
	fmt.Fprintln(os.Stderr, err)
}

func HandleHello(conn *zmq.Socket, clientId []byte) {
	fmt.Println("HELLO")
	conn.SendMessage(clientId, "HELLO")
}

func SendData(conn *zmq.Socket, clientId []byte, pipeName string, data []byte) (count int) {
	count, err := conn.SendMessage(clientId, "SKYPIPE/0.1", "DATA", pipeName, data)
	if err != nil {
		PrintError(err)
	}

	return count
}

func SendAck(conn *zmq.Socket, clientId []byte) (count int) {
	count, err := conn.SendMessage(clientId, "SKYPIPE/0.1", "ACK")
	if err != nil {
		PrintError(err)
	}

	return count
}

func HandleListen(conn *zmq.Socket, clients map[string][][]byte, buffers map[string][][]byte, clientId []byte, pipeName string) {
	fmt.Println("LISTEN", pipeName)

	idx := SliceIndex(len(clients[pipeName]), func(i int) bool { return ByteArrayEquals(clients[pipeName][i], clientId) })
	if idx == -1 {
		clients[pipeName] = append(clients[pipeName], clientId)
	}

	// if only client after adding, then previously there were
	// no clients and it was buffering, so spit out buffered data
	if len(clients[pipeName]) == 1 {
		bufferQueue, ok := buffers[pipeName]
		if ok {
			if len(bufferQueue) > 0 {
				buffer, bufferQueue := bufferQueue[0], bufferQueue[1:]
				buffers[pipeName] = bufferQueue

				SendData(conn, clientId, pipeName, buffer)
			} else {
				SendAck(conn, clientId)
			}
		} else {
			SendAck(conn, clientId)
		}
	}
}

func HandleUnlisten(conn *zmq.Socket, clients map[string][][]byte, clientId []byte, pipeName string) {
	fmt.Println("UNLISTEN", pipeName)

	idx := SliceIndex(len(clients[pipeName]), func(i int) bool { return ByteArrayEquals(clients[pipeName][i], clientId) })
	if idx > -1 {
		clients[pipeName] = append(clients[pipeName][:idx], clients[pipeName][idx+1:]...) // https://code.google.com/p/go-wiki/wiki/SliceTricks
	}

	SendAck(conn, clientId)
}

// http://stackoverflow.com/a/18561233/402347
func ByteArrayEquals(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// http://stackoverflow.com/a/18203895/402347
func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

func rep_socket_monitor(addr string) {
	s, err := zmq.NewSocket(zmq.PAIR)
	if err != nil {
		log.Fatalln(err)
	}
	err = s.Connect(addr)
	if err != nil {
		log.Fatalln(err)
	}
	for {
		a, b, c, err := s.RecvEvent(0)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(a, b, c)
	}
	s.Close()
}

func main() {
	server, _ := zmq.NewSocket(zmq.ROUTER)
	err := server.Bind("tcp://*:5556")
	if err != nil {
		fmt.Println(err)
	}

	// REP socket monitor, all events
	err = server.Monitor("inproc://monitor.rep", zmq.EVENT_ALL)
	if err != nil {
		log.Fatalln(err)
	}
	go rep_socket_monitor("inproc://monitor.rep")

	buffers := make(map[string][][]byte)
	clients := make(map[string][][]byte)

	for {
		msg, err := server.RecvMessageBytes(0)

		switch string(msg[2]) {
			case "HELLO":
				HandleHello(server, msg[0])
			case "LISTEN":
				HandleListen(server, clients, buffers, msg[0], string(msg[3]))
			case "UNLISTEN":
				HandleUnlisten(server, clients, msg[0], string(msg[3]))
			case "DATA":
				_, pipeHasClients := clients[string(msg[3])]
				if pipeHasClients && len(clients[string(msg[3])]) > 0 {
					for _, clientId := range clients[string(msg[3])] {
						SendData(server, clientId, string(msg[3]), msg[4])
					}
				} else {
					buffers[string(msg[3])] = append(buffers[string(msg[3])], msg[4])
				}
				server.SendMessage(msg[0], "SKYPIPE/0.1", "ACK")
		}

		fmt.Println(clients)

		if err != nil {
			break //  Interrupted
		}
	}
}
