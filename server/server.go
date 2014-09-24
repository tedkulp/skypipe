package main

import (
	"log"
	"github.com/tedkulp/skypipe/common"
	zmq "github.com/pebbe/zmq4"
)

func HandleHello(conn *zmq.Socket, clientId []byte) {
	log.Println("HELLO")
	conn.SendMessage(clientId, "HELLO")
}

func SendData(conn *zmq.Socket, clientId []byte, pipeName string, data []byte) (count int) {
	count, err := conn.SendMessage(clientId, "SKYPIPE/0.1", "DATA", pipeName, data)
	if err != nil {
		common.PrintError(err)
	}

	return count
}

func SendAck(conn *zmq.Socket, clientId []byte) (count int) {
	count, err := conn.SendMessage(clientId, "SKYPIPE/0.1", "ACK")
	if err != nil {
		common.PrintError(err)
	}

	return count
}

func HandleListen(conn *zmq.Socket, clients map[string][][]byte, buffers map[string][][]byte, clientId []byte, pipeName string) {
	log.Println("LISTEN", pipeName)

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
	log.Println("UNLISTEN", pipeName)

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

func main() {
	server, _ := zmq.NewSocket(zmq.ROUTER)
	err := server.Bind("tcp://*:5556")
	if err != nil {
		log.Println(err)
	}

	buffers := make(map[string][][]byte)
	clients := make(map[string][][]byte)

	for {
		msg, err := server.RecvMessageBytes(0)
		clientId, cmd := msg[0], msg[2]

		switch string(cmd) {
			case "HELLO":
				HandleHello(server, clientId)
			case "LISTEN":
				HandleListen(server, clients, buffers, clientId, string(msg[3]))
			case "UNLISTEN":
				HandleUnlisten(server, clients, msg[0], string(msg[3]))
			case "DATA":
				pipeName := string(msg[3])
				log.Println("DATA", pipeName)

				_, pipeHasClients := clients[pipeName]
				if pipeHasClients && len(clients[pipeName]) > 0 {
					for _, outputClientId := range clients[pipeName] {
						SendData(server, outputClientId, pipeName, msg[4])
					}
				} else {
					buffers[pipeName] = append(buffers[pipeName], msg[4])
				}
				SendAck(server, clientId)
		}

		log.Println(clients)

		if err != nil {
			break //  Interrupted
		}
	}
}
