package common

import (
	"os"
	"fmt"
	// "io/ioutil"
	// zmq "github.com/pebbe/zmq4"
)

const SkypipeHeader string = "SKYPIPE/0.2"

func PrintError(err error) {
	fmt.Fprintln(os.Stderr, err)
}

func PrintErrorAndQuit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(-1)
}
