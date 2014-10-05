package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"code.google.com/p/go.net/websocket"
	"github.com/nu7hatch/gouuid"
)

const VERSION = "0.2"
const PROTOCOL_VERSION = "SKYPIPE/0.2"

var daemon *bool = flag.Bool("d", false, "run the server daemon")

type session struct {
	Name         string
	Transmitters *clients
	Receivers    *clients
	EOF          chan int
}

func (s session) Close() {
	// Put an item in the channel for each receiver
	for _ = range s.Receivers.c {
		s.EOF <- 0
	}
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
	conn, err := websocket.Dial("ws://localhost:3000/"+id+"/in", "", "http://localhost:3000")

	if err != nil {
		panic(err)
	}

	io.Copy(conn, os.Stdin)
}

func handleOutputMode(id string, pipeName string) {
	conn, err := websocket.Dial("ws://localhost:3000/"+id+"/out", "", "http://localhost:3000")

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
								}

								// This will probably go
								session.Transmitters.Remove(transId)

							case "out":
								transId := session.Receivers.Add(conn)

								log.Println(sessionName + ": viewer connected [websocket]")

								// Wait on an EOF
								<-session.EOF

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
	if (val != "") {
		port = val
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
		pipeName := "__DEFAULT__"

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