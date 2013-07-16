package dispatch

import (
	"github.com/aarondl/ultimateq/irc"
	"math/rand"
	"strings"
	"sync"
)

// EventHandler is the basic interface that will deal with handling any message
// as a raw IrcMessage event. However there are other message types are specific
// to very common irc events that are more helpful than this interface.
type EventHandler interface {
	HandleRaw(event *irc.Message, endpoint irc.Endpoint)
}

type (
	// eventTable is the storage used to keep id -> interface{} mappings in the
	// eventTableState map.
	eventTable map[int]interface{}
	// eventTableState is the map used to hold the event handlers for an event
	eventTableState map[string]eventTable
)

// Dispatcher is made for handling dispatching of raw-ish irc events.
type Dispatcher struct {
	*DispatchCore
	events        eventTableState
	protectEvents sync.RWMutex
}

// CreateDispatcher initializes an empty dispatcher ready to register events.
func CreateDispatcher(core *DispatchCore) *Dispatcher {
	return &Dispatcher{
		DispatchCore: core,
		events:       make(eventTableState),
	}
}

// Register registers an event handler to a particular event. In return a
// unique identifer is given to later pass into Unregister in case of a need
// to unregister the event handler.
func (d *Dispatcher) Register(event string, handler interface{}) int {
	event = strings.ToUpper(event)
	id := rand.Int()

	d.protectEvents.Lock()
	defer d.protectEvents.Unlock()

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

	d.protectEvents.Lock()
	defer d.protectEvents.Unlock()

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
func (d *Dispatcher) Dispatch(msg *irc.Message, ep irc.Endpoint) bool {
	event := strings.ToUpper(msg.Name)

	d.protectEvents.RLock()
	defer d.protectEvents.RUnlock()

	handled := d.dispatchHelper(event, msg, ep)
	d.dispatchHelper(irc.RAW, msg, ep)

	return handled
}

// dispatchHelper locates a handler and attempts to resolve it with
// resolveHandler. It returns true if it was able to find an event table.
func (d *Dispatcher) dispatchHelper(event string,
	msg *irc.Message, ep irc.Endpoint) bool {

	if evtable, ok := d.events[event]; ok {
		for _, handler := range evtable {
			d.HandlerStarted()
			go d.resolveHandler(handler, event, msg, ep)
		}
		return true
	}
	return false
}

// resolveHandler checks the type of the handler passed in, resolves it to a
// real type, coerces the IrcMessage in whatever way necessary and then
// calls that handlers primary dispatch method with the coerced message.
func (d *Dispatcher) resolveHandler(
	handler interface{}, event string, msg *irc.Message, ep irc.Endpoint) {

	var handled bool
	switch msg.Name {
	case irc.PRIVMSG:
		handled = d.dispatchPrivmsg(handler, msg, ep)
	case irc.NOTICE:
		handled = d.dispatchNotice(handler, msg, ep)
	}

	if !handled {
		if evHandler, ok := handler.(EventHandler); ok {
			evHandler.HandleRaw(msg, ep)
		}
	}
	d.HandlerFinished()
}

// dispatchPrivmsg dispatches only a private message. Returns true if the
// event was handled.
func (d *Dispatcher) dispatchPrivmsg(
	handler interface{}, msg *irc.Message, ep irc.Endpoint) (handled bool) {

	if channelHandler, ok := handler.(PrivmsgChannelHandler); ok &&
		d.shouldDispatch(true, msg.Args[0]) {
		channelHandler.PrivmsgChannel(msg, ep)
		handled = true
	} else if userHandler, ok := handler.(PrivmsgUserHandler); ok &&
		d.shouldDispatch(false, msg.Args[0]) {
		userHandler.PrivmsgUser(msg, ep)
		handled = true
	} else if privmsgHandler, ok := handler.(PrivmsgHandler); ok {
		privmsgHandler.Privmsg(msg, ep)
		handled = true
	}
	return
}

// dispatchNotice dispatches only a notice message. Returns true if the
// event was handled.
func (d *Dispatcher) dispatchNotice(
	handler interface{}, msg *irc.Message, ep irc.Endpoint) (handled bool) {

	if channelHandler, ok := handler.(NoticeChannelHandler); ok &&
		d.shouldDispatch(true, msg.Args[0]) {
		channelHandler.NoticeChannel(msg, ep)
		handled = true
	} else if userHandler, ok := handler.(NoticeUserHandler); ok &&
		d.shouldDispatch(false, msg.Args[0]) {
		userHandler.NoticeUser(msg, ep)
		handled = true
	} else if noticeHandler, ok := handler.(NoticeHandler); ok {
		noticeHandler.Notice(msg, ep)
		handled = true
	}
	return
}

// shouldDispatch checks if we should dispatch this event. Works for user and
// channel based messages.
func (d *DispatchCore) shouldDispatch(channelMsg bool, channel string) bool {
	isChan, hasChan := d.CheckTarget(channel)
	return channelMsg == isChan && (!channelMsg || hasChan)
}
