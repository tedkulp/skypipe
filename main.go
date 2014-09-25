package main

import (
	"os"
	"flag"
	"fmt"
	"io/ioutil"
	zmq "github.com/pebbe/zmq4"

	"github.com/tedkulp/skypipe/common"
)

var hostname = flag.String("hostname", "127.0.0.1", "IP Address to connect to. Defaults to: 127.0.0.1")
var port     = flag.Int("port", 9000, "Port to bind the server to. Defaults to: 9000")

func GetConnection() (*zmq.Socket) {
	conn, err := zmq.NewSocket(zmq.DEALER)
	if err != nil {
		common.PrintErrorAndQuit(err)
	}

	address  := fmt.Sprintf("tcp://%s:%d", *hostname, *port)
	if os.Getenv("SATELLITE") != "" {
		address = os.Getenv("SATELLITE")
	}

	err = conn.Connect(address)
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

func init() {
	flag.StringVar(hostname, "h", "127.0.0.1", "IP Address to connect to. Defaults to: 127.0.0.1")
	flag.IntVar(port, "p", 9000, "Port to bind the server to. Defaults to: 9000")
}

func main() {
	flag.Parse()

	stdin, _ := os.Stdin.Stat()
	pipeName := "__DEFAULT__"

	if len(flag.Args()) > 0 {
		pipeName = flag.Args()[0];
	}

	if stdin.Mode() & os.ModeNamedPipe > 0 {
		HandleInputMode(pipeName)
	} else {
		HandleOutputMode(pipeName)
	}
}
