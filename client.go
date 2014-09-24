package main

import (
	"os"
	"fmt"
	// "io"
	"io/ioutil"
	// "bufio"
	zmq "github.com/pebbe/zmq4"
)

func PrintError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(-1)
}

func getConnection() (*zmq.Socket) {
	conn, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		PrintError(err)
	}

	err = conn.Connect("tcp://127.0.0.1:5556")
	if err != nil {
		PrintError(err)
	}

	return conn
}

func handleInputMode(pipeName string) {
	conn := getConnection()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		PrintError(err)
	}

	sendData(conn, pipeName, data)

	conn.Close()
}

func sendCommand(conn *zmq.Socket, cmd string, args... string) ([]string, error) {
	_, err := conn.SendMessage("SKYPIPE/0.1", cmd, args)
	if err != nil {
		PrintError(err)
	}

	msg, err := conn.RecvMessage(0)
	if err != nil {
		PrintError(err)
	}

	return msg, err
}

func sendData(conn *zmq.Socket, pipeName string, data []byte) {
	_, err := conn.SendMessage("SKYPIPE/0.1", "DATA", pipeName, data)
	if err != nil {
		PrintError(err)
	}

	_, err = conn.RecvMessage(0)
	if err != nil {
		PrintError(err)
	}
}

func handleOutputMode(pipeName string) {
	conn := getConnection()

	sendCommand(conn, "HELLO")
	msg, err := sendCommand(conn, "LISTEN", pipeName)

	for !handleResponse(msg, err) {
		msg, err = conn.RecvMessage(0)
		if err != nil {
			PrintError(err)
		}
	}

	msg, err = sendCommand(conn, "UNLISTEN", pipeName)
	conn.Close()
}

func handleResponse(msg []string, _ error) (shouldExit bool) {
	switch msg[1] {
		case "DATA":
			fmt.Println(msg[3])
			return true
		case "ACK":
			fallthrough
		default:
			fmt.Println("ACK or unknown")
	}

	return false
}

func main() {
	stdin, _  := os.Stdin.Stat()
	pipeName := "TEST"

	if stdin.Mode() & os.ModeNamedPipe > 0 {
		handleInputMode(pipeName)
	} else {
		handleOutputMode(pipeName)
	}
}
