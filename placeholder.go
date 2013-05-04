// The ultimateq bot framework.
package main

import (
	"bufio"
	"bytes"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/parse"
	"log"
	"net"
	"os"
	"sync"
)

var dispatcher dispatch.Dispatcher
var client *inet.IrcClient

type Handler struct {
}

func (h Handler) HandleRaw(msg *irc.IrcMessage) {
	if msg.Name == "PING" {
		client.Write([]byte("PONG :" + msg.Args[0] + "\r\n"))
	}
}

func main() {
	log.SetOutput(os.Stdout)

	// Create a dispatcher
	dispatcher, err := dispatch.CreateRichDispatcher(
		&irc.ProtoCaps{Chantypes: "#&~"},
	)
	if err != nil {
		log.Println("Failed to create dispatcher:", err)
		return
	}
	dispatcher.Register(irc.RAW, Handler{})

	// Connect to the server
	conn, err := net.Dial("tcp", "irc.gamesurge.net:6667")
	if err != nil {
		log.Println("Could not connect:", err)
		return
	}

	// Create an IRC Client on top of the connection
	client = inet.CreateIrcClient(conn)
	// Create goroutines to read/write
	client.SpawnWorkers()
	// Write our initial nicks
	client.Write([]byte("NICK :nobody_"))
	client.Write([]byte("USER nobody bitforge.ca 0 :nobody"))

	// Set up the parsing/dispatching goroutine
	var waiter sync.WaitGroup
	waiter.Add(1)
	go func() {
		for {
			msg, ok := client.ReadMessage()
			if !ok {
				log.Println("Socket closed.")
				break
			}
			ircMsg, err := parse.Parse(string(msg))
			if err != err {
				log.Println("Error parsing message:", err)
			} else {
				dispatcher.Dispatch(ircMsg.Name, ircMsg)
			}
		}
		waiter.Done()
	}()

	// Main goroutine will be reading from stdin and writing our commands
	// to the server
	reader := bufio.NewReader(os.Stdin)
	for {
		str, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("Error while getting input:", err)
			break
		}

		str = str[:len(str)-1]

		if 0 == bytes.Compare(str, []byte("QUIT")) {
			client.Write([]byte("QUIT :Quitting"))
			break
		} else {
			client.Write(str)
		}
	}

	// Exit and wait for all goroutines to return
	log.Println("Exiting.")
	client.Close()
	client.Wait()
	waiter.Wait()
}
