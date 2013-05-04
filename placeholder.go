// The ultimateq bot framework.
package main

import (
)

var dispatcher dispatch.Dispatcher
var client *inet.IrcClient

func main() {
	log.SetOutput(os.Stdout)

	// Create a dispatcher

	// Connect to the server

	// Create an IRC Client on top of the connection
	client = inet.CreateIrcClient(conn)
	// Create goroutines to read/write
	client.SpawnWorkers()
	// Write our initial nicks
	client.Write([]byte("NICK :nobody_"))
	client.Write([]byte("USER nobody bitforge.ca 0 :nobody"))

	// Set up the parsing/dispatching goroutine
}
