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

// eventTable is the storage used to keep id -> interface{} mappings in the
// eventTableStore map.
type eventTable map[int]interface{}

// eventTableStore is the map used to hold the event handlers for an event
type eventTableStore map[string]eventTable

// Dispatcher is made for handling bot-local dispatching of irc
// events.
type Dispatcher struct {
	events eventTableStore
	finder *ChannelFinder
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
func (d *Dispatcher) Register(event string, handler interface{}) int {
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
// unregister a callback from the Dispatcher. If the callback was removed it
// returns true, false if it could not be found.
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

// Dispatch an IrcMessage to event handlers handling event also ensures all raw
// handlers receive all messages. Returns false if no eventtable was found for
// the primary sent event.
func (d *Dispatcher) Dispatch(event string, msg *IrcMessage) bool {
	event = strings.ToUpper(event)
	handled := d.dispatchHelper(event, msg)
	d.dispatchHelper(RAW, msg)

	return handled
}

// dispatchHelper locates a handler and attempts to resolve it with
// resolveHandler. It returns true if it was able to find an event table.
func (d *Dispatcher) dispatchHelper(event string, msg *IrcMessage) bool {
	var (
		pmsg  []*Privmsg
		pumsg []*PrivmsgTarget
		pcmsg []*PrivmsgTarget
		nmsg  []*Notice
		numsg []*NoticeTarget
		ncmsg []*NoticeTarget
	)
	initp := func() {
		if pmsg == nil {
			pmsg, pumsg, pcmsg = d.PrivmsgParse(msg)
		}
	}
	initn := func() {
		if nmsg == nil {
			nmsg, numsg, ncmsg = d.NoticeParse(msg)
		}
	}

	if evtable, ok := d.events[event]; ok {
		for _, handler := range evtable {
			switch t := handler.(type) {
			case EventHandler:
				t.HandleRaw(msg)
			case PrivmsgHandler:
				initp()
				for _, v := range pmsg {
					t.Privmsg(v)
				}
			case PrivmsgUserHandler:
				initp()
				for _, v := range pumsg {
					t.PrivmsgUser(v)
				}
			case PrivmsgChannelHandler:
				initp()
				for _, v := range pcmsg {
					t.PrivmsgChannel(v)
				}
			case NoticeHandler:
				initn()
				for _, v := range nmsg {
					t.Notice(v)
				}
			case NoticeUserHandler:
				initn()
				for _, v := range numsg {
					t.NoticeUser(v)
				}
			case NoticeChannelHandler:
				initn()
				for _, v := range ncmsg {
					t.NoticeChannel(v)
				}
			}
		}
		return true
	}
	return false
}
