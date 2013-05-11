/*
dispatch package is used to dispatch irc messages to event handlers in an
asynchronous fashion. It supports various event handling types to easily
extract information from events, as well as define more succint handlers.
*/
package dispatch

import (
	"errors"
	"github.com/aarondl/ultimateq/irc"
	"math/rand"
	"strings"
)

var (
	// errProtoCapsMissing is returned by CreateRichDispatch if nil is provided
	// instead of a irc.ProtoCaps pointer.
	errProtoCapsMissing = errors.New(
		"dispatch: Protocaps missing in CreateRichDispatch")
)

// EventHandler is the basic interface that will deal with handling any message
// as a raw IrcMessage event. However there are other message types are specific
// to very common irc events that are more helpful than this interface.
type EventHandler interface {
	HandleRaw(event *irc.IrcMessage, sender irc.Sender)
}

type (
	// eventTable is the storage used to keep id -> interface{} mappings in the
	// eventTableStore map.
	eventTable map[int]interface{}
	// eventTableStore is the map used to hold the event handlers for an event
	eventTableStore map[string]eventTable
)

// Dispatcher is made for handling bot-local dispatching of irc
// events.
type Dispatcher struct {
	events eventTableStore
	caps   *irc.ProtoCaps
	finder *irc.ChannelFinder
	chans  []string
}

// CreateDispatcher initializes an empty dispatcher ready to register events.
func CreateDispatcher() *Dispatcher {
	return &Dispatcher{
		events: make(eventTableStore),
	}
}

// CreateRichDispatcher initializes empty dispatcher ready to register events
// and additionally creates a channelfinder from a set of irc.ProtoCaps in order
// to properly send Privmsg(User|Channel)/Notice(User|Channel) events. If
// activeChannels is not nil, (Privmsg|Notice)Channel events are filtered on
// the list of channels.
func CreateRichDispatcher(caps *irc.ProtoCaps,
	activeChannels []string) (*Dispatcher, error) {

	if caps == nil {
		return nil, errProtoCapsMissing
	}
	f, err := irc.CreateChannelFinder(caps.Chantypes)
	if err != nil {
		return nil, err
	}

	var chans []string = nil
	length := len(activeChannels)
	if length > 0 {
		chans = make([]string, length)
		for i := 0; i < length; i++ {
			chans[i] = strings.ToLower(activeChannels[i])
		}
	}

	return &Dispatcher{
		events: make(eventTableStore),
		caps:   caps,
		finder: f,
		chans:  chans,
	}, nil
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
func (d *Dispatcher) Dispatch(msg *irc.IrcMessage, sender irc.Sender) bool {
	event := strings.ToUpper(msg.Name)
	handled := d.dispatchHelper(event, msg, sender)
	d.dispatchHelper(irc.RAW, msg, sender)

	return handled
}

// dispatchHelper locates a handler and attempts to resolve it with
// resolveHandler. It returns true if it was able to find an event table.
func (d *Dispatcher) dispatchHelper(event string,
	msg *irc.IrcMessage, sender irc.Sender) bool {

	if evtable, ok := d.events[event]; ok {
		for _, handler := range evtable {
			go d.resolveHandler(handler, event, msg, sender)
		}
		return true
	}
	return false
}

// resolveHandler checks the type of the handler passed in, resolves it to a
// real type, coerces the IrcMessage in whatever way necessary and then
// calls that handlers primary dispatch method with the coerced message.
func (d *Dispatcher) resolveHandler(
	handler interface{}, event string, msg *irc.IrcMessage, sender irc.Sender) {

	switch t := handler.(type) {
	case PrivmsgUserHandler:
		if d.shouldDispatch(false, msg) {
			t.PrivmsgUser(&Message{msg}, sender)
		}
	case PrivmsgChannelHandler:
		if d.shouldDispatch(true, msg) {
			t.PrivmsgChannel(&Message{msg}, sender)
		}
	case PrivmsgHandler:
		t.Privmsg(&Message{msg}, sender)
	case NoticeUserHandler:
		if d.shouldDispatch(false, msg) {
			t.NoticeUser(&Message{msg}, sender)
		}
	case NoticeChannelHandler:
		if d.shouldDispatch(true, msg) {
			t.NoticeChannel(&Message{msg}, sender)
		}
	case NoticeHandler:
		t.Notice(&Message{msg}, sender)
	case EventHandler:
		t.HandleRaw(msg, sender)
	}
}

// shouldDispatch checks if we should dispatch this event. Works for user and
// channel based messages.
func (d *Dispatcher) shouldDispatch(channel bool, msg *irc.IrcMessage) bool {
	return d.finder != nil && channel == d.finder.IsChannel(msg.Args[0]) &&
		(!channel || d.checkChannels(msg))
}

// filterChannelDispatch is used for any channel-specific message handlers
// that exist. It scans the list of targets given to CreateRichDispatch to
// check if this event should be dispatched.
func (d *Dispatcher) checkChannels(msg *irc.IrcMessage) bool {
	if d.chans == nil {
		return true
	}

	targ := strings.ToLower(msg.Args[0])
	for i := 0; i < len(d.chans); i++ {
		if targ == d.chans[i] {
			return true
		}
	}
	return false
}
