package main

import (
	"os"
	"fmt"
	"io/ioutil"
	zmq "github.com/pebbe/zmq4"
)

func PrintError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(-1)
}

func GetConnection() (*zmq.Socket) {
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

func HandleInputMode(pipeName string) {
	conn := GetConnection()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		PrintError(err)
	}

	SendData(conn, pipeName, data)

	conn.Close()
}

func SendCommand(conn *zmq.Socket, cmd string, args... string) ([]string, error) {
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

func SendData(conn *zmq.Socket, pipeName string, data []byte) {
	_, err := conn.SendMessage("SKYPIPE/0.1", "DATA", pipeName, data)
	if err != nil {
		PrintError(err)
	}

	_, err = conn.RecvMessage(0)
	if err != nil {
		PrintError(err)
	}
}

func HandleOutputMode(pipeName string) {
	conn := GetConnection()

	SendCommand(conn, "HELLO")
	msg, err := SendCommand(conn, "LISTEN", pipeName)

	for !HandleResponse(msg, err) {
		msg, err = conn.RecvMessage(0)
		if err != nil {
			PrintError(err)
		}
	}

	msg, err = SendCommand(conn, "UNLISTEN", pipeName)
	conn.Close()
}

func HandleResponse(msg []string, _ error) (shouldExit bool) {
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
		HandleInputMode(pipeName)
	} else {
		HandleOutputMode(pipeName)
	}
}
