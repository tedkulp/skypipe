package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"code.google.com/p/go.net/websocket"
	"github.com/nu7hatch/gouuid"
	"github.com/oleiade/lane"
)

const VERSION = "0.2"
const PROTOCOL_VERSION = "SKYPIPE/0.2"

var daemon *bool = flag.Bool("d", false, "run the server daemon")
var server *string = flag.String("s", "localhost:3000", "use a different server to start session")

type session struct {
	Name         string
	Transmitters *clients
	Receivers    *clients
	Buffers      *lane.Queue
	EOF          chan int
}

func (s session) Close() {
	// Put an item in the channel for each receiver
	for _ = range s.Receivers.c {
		s.EOF <- 0
	}
}

func (s session) HasBufferedData() (bool) {
	return s.Buffers.Size() > 0
}

func (s session) BufferData(buff *bytes.Buffer) {
	s.Buffers.Enqueue(buff)
}

func (s session) GetNextData() (*bytes.Buffer) {
	return s.Buffers.Dequeue().(*bytes.Buffer)
}

type sessions struct {
	s map[string]*session
}

func (s sessions) Get(name string) (sess *session, err error) {
	sess, found := s.s[name]
	if !found {
		err = errors.New("session not found")
		return
	}
	return
}

func (s sessions) Create(name string) (*session, error) {
	if sess, _ := s.Get(name); sess != nil {
		return nil, errors.New("session already exists")
	}

	sess := &session{
		Name:         name,
		Transmitters: &clients{c: make(map[string]io.Writer)},
		Receivers:    &clients{c: make(map[string]io.Writer)},
		Buffers:      lane.NewQueue(),
		EOF:          make(chan int),
	}

	s.s[name] = sess

	return sess, nil
}

func (s sessions) Delete(name string) {
	delete(s.s, name)
}

type clients struct {
	c map[string]io.Writer
}

func (c clients) Add(client io.Writer) (string) {
	id, _ := uuid.NewV4()

	c.c[id.String()] = client

	return id.String()
}

func (c clients) Remove(id string) {
	delete(c.c, id)
}

func (c clients) Write(data []byte) (n int, err error) {
	for key := range c.c {
		log.Println(key, c.c[key])
		n, err = c.c[key].Write(data)
		// if err != nil || n != len(data) {
		//   delete(c.c, w)
		// }
	}
	return len(data), nil
}

func handleInputMode(id string, pipeName string) {
	// log.Println("input", id, pipeName)
	conn, err := websocket.Dial("ws://" + *server + "/" + strings.Join([]string{id, pipeName}, "-") + "/in", "", "http://" + *server)

	if err != nil {
		panic(err)
	}

	io.Copy(conn, os.Stdin)
}

func handleOutputMode(id string, pipeName string) {
	conn, err := websocket.Dial("ws://" + *server + "/" + strings.Join([]string{id, pipeName}, "-") + "/out", "", "http://" + *server)

	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdin, conn)
}

func startDaemon() {
	sessions := sessions{s: make(map[string]*session)}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
			case r.RequestURI == "/":
				http.Redirect(w, r, "https://github.com/tedkulp/skypipe", 301)
			case r.RequestURI == "/version":
				w.Write([]byte("v" + VERSION))
			case r.RequestURI == "/protocol":
				w.Write([]byte(PROTOCOL_VERSION))
			default:
				parts := strings.Split(r.RequestURI, "/")
				sessionName := parts[1]
				direction   := parts[2]

				isWebsocket := r.Header.Get("Upgrade") == "websocket"
				session, err := sessions.Get(sessionName)
				if err != nil {
					session, err = sessions.Create(sessionName)
					if err != nil {
						log.Panic(err)
					}
				}

				if isWebsocket {
					websocket.Handler(func(conn *websocket.Conn) {
						switch direction {
							case "in":
								// This will probably go -- don't think we're going
								// to support multiple transmitters, but don't want to
								// count it out yet
								transId := session.Transmitters.Add(conn)

								if len(session.Receivers.c) > 0 {
									// We have receivers -- just echo everything
									// directly to them and EOF out
									io.Copy(io.MultiWriter(session.Receivers), conn)
									session.Close()
								} else {
									// Put the data into a buffer and we'll run it
									// out the next connecting receiver
									buff := new(bytes.Buffer)
									io.Copy(buff, conn)
									session.BufferData(buff)
								}

								// This will probably go
								session.Transmitters.Remove(transId)

							case "out":
								transId := session.Receivers.Add(conn)

								log.Println(sessionName + ": viewer connected [websocket]")

								if session.HasBufferedData() {
									io.Copy(conn, session.GetNextData())
								} else {
									// Wait on an EOF
									<-session.EOF
								}

								// We've EOF'd, remove the listener from the session
								session.Receivers.Remove(transId)
							default:
								log.Println("not a valid command")
						}
					}).ServeHTTP(w, r)
				}
		}
	})

	port := "3000"

	val := os.Getenv("PORT")
	if val != "" {
		port = val
	} else {
		parts := strings.Split(*server, ":")
		if len(parts) > 1 {
			port = parts[1]
		}
	}

	log.Println("Skypipe server started on " + port + "...")
	log.Fatal(http.ListenAndServe(":" + port, nil))
}

func main() {
	flag.Parse()

	if *daemon {
		startDaemon()
	} else {
		stdin, _ := os.Stdin.Stat()
		pipeName := ""

		if len(flag.Args()) > 0 {
			pipeName = flag.Args()[0];
		}

		// id, err := uuid.NewV4()
		// if err != nil {
		//   log.Panic(err)
		// }

		if stdin.Mode() & os.ModeNamedPipe > 0 {
			handleInputMode("12345"/*id.String()*/, pipeName)
		} else {
			handleOutputMode("12345"/*id.String()*/, pipeName)
		}
	}
}
