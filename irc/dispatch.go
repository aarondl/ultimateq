package irc

import (
	"math/rand"
	"strings"
)

// EventHandler is the basic interface that will deal with handling any message
// as a raw IrcMessage event. However there are other message types are specific
// to very common irc events that are more helpful than this interface.
type EventHandler interface {
	HandleRaw(event *IrcMessage)
}

// eventTable is the storage used to keep id -> EventHandler mappings in the
// eventTableStore map.
type eventTable map[int]EventHandler

// eventTableStore is the map used to hold the event handlers for an event
type eventTableStore map[string]eventTable

// Dispatcher is made for handling bot-local dispatching of irc
// events.
type Dispatcher struct {
	events eventTableStore
}

// CreateDispatcher initializes an empty dispatcher ready to register events.
func CreateDispatcher() *Dispatcher {
	return &Dispatcher{
		events: make(eventTableStore),
	}
}

// Register registers an event handler to a particular event. In return a
// unique identifer is given to later pass into Unregister in case of a need
// to unregister the event handler.
func (d *Dispatcher) Register(event string, handler EventHandler) int {
	event = strings.ToUpper(event)
	id := rand.Int()
	if ev, ok := d.events[event]; !ok {
		d.events[event] = make(eventTable)
	} else {
		for _, has := ev[id]; has; id = rand.Int() {
		}
	}

	d.events[event][id] = handler
	return id
}

// Unregister uses the event name, and the identifier returned by Register to
// unregister a callback from the Dispatcher.
func (d *Dispatcher) Unregister(event string, id int) bool {
	event = strings.ToUpper(event)
	if ev, ok := d.events[event]; ok {
		if _, ok := ev[id]; ok {
			delete(ev, id)
			return true
		}
	}
	return false
}
