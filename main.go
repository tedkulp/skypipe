package main

import (
	"os"
	"fmt"
	"io/ioutil"
	"github.com/tedkulp/skypipe/common"
	zmq "github.com/pebbe/zmq4"
)

func GetConnection() (*zmq.Socket) {
	conn, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	err = conn.Connect("tcp://127.0.0.1:5556")
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	return conn
}

func HandleInputMode(pipeName string) {
	conn := GetConnection()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	SendData(conn, pipeName, data)

	conn.Close()
}

func SendCommand(conn *zmq.Socket, cmd string, args... string) ([]string, error) {
	_, err := conn.SendMessage(common.SkypipeHeader, cmd, args)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	msg, err := conn.RecvMessage(0)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	return msg, err
}

func SendData(conn *zmq.Socket, pipeName string, data []byte) {
	_, err := conn.SendMessage(common.SkypipeHeader, "DATA", pipeName, data)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	_, err = conn.RecvMessage(0)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}
}

func HandleOutputMode(pipeName string) {
	conn := GetConnection()

	SendCommand(conn, "HELLO")
	msg, err := SendCommand(conn, "LISTEN", pipeName)

	for !HandleResponse(msg, err) {
		msg, err = conn.RecvMessage(0)
		if err != nil {
			common.PrintErrorAndQuit(err)
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
