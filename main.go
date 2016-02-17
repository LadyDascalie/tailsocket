package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/dleung/gotail"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var fname string

func main() {
	// Limit to 1 proc to avoid running intensively for no reason
	runtime.GOMAXPROCS(1)

	flag.StringVar(&fname, "file", "", "File to tail. Ex: ./tailsocket -file=\"/var/log/my_awesome_log\"")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		index, err := Asset("index.html")
		check(err)

		// Prints the bundled index.html directly to the connection
		fmt.Fprintf(w, "%s", index)
	})

	// The handler for the websocket
	http.HandleFunc("/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)

		// Listens to messages from the client and closes the connection when necessary
		go func(conn *websocket.Conn) {
			for {
				// Prevent using the CPU massively just to listen to close request
				time.Sleep(1 * time.Second)
				_, _, err := conn.ReadMessage()
				if err != nil {
					conn.Close()
				}
			}
		}(conn)

		// Sends data back to the client
		go func(conn *websocket.Conn) {
			tail, err := gotail.NewTail(fname, gotail.Config{Timeout: 2})
			check(err)
			for line := range tail.Lines {
				conn.WriteJSON(Log{
					LogLine: html.EscapeString(line),
				})
				// Delay sending the message back, again to provide CPU relief
				time.Sleep(500 * time.Millisecond)
			}
		}(conn)
	})

	http.ListenAndServe(":3000", nil)
}

// Log is the struct for the logfile you want to tail
// it doesnt need to be complicated, a single property per line is fine
type Log struct {
	LogLine string `json:"logline"`
}

// Just a simple error checking wrapper that logs errors to console
func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
