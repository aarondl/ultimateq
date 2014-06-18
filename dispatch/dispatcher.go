package dispatch

import (
	"strings"
	"sync"

	"github.com/aarondl/ultimateq/irc"
)

// EventHandler is the basic interface that will deal with handling any message
// as a raw IrcMessage event. However there are other message types are specific
// to very common irc events that are more helpful than this interface.
type EventHandler interface {
	HandleRaw(w irc.Writer, ev *irc.Event)
}

type (
	// eventTable is the storage used to keep id -> interface{} mappings in the
	// eventIDs map.
	eventTable map[uint64]interface{}
	// eventIDs is the map used to hold the event handlers for an event
	eventIDs map[string]eventTable
	// channelEvents is the map used to filter based on channels.
	channelEvents map[string]eventIDs
	// networkEvents is the map used to filter based on networks.
	networkEvents map[string]channelEvents
)

// Dispatcher is made for handling dispatching of raw-ish irc events.
type Dispatcher struct {
	*DispatchCore
	events        networkEvents
	protectEvents sync.RWMutex
	eventID       uint64
}

// NewDispatcher initializes an empty dispatcher ready to register events.
func NewDispatcher(core *DispatchCore) *Dispatcher {
	return &Dispatcher{
		DispatchCore: core,
		events:       make(networkEvents),
	}
}

// Register registers an event handler to a particular event. In return a
// unique identifer is given to later pass into Unregister in case of a need
// to unregister the event handler. Pass in an empty string to any of network,
// channel or event to prevent filtering on that parameter. Panics if it's
// given a type that doesn't implement any of the correct interfaces.
func (d *Dispatcher) Register(
	network, channel, event string, handler interface{}) uint64 {

	switch handler.(type) {
	case EventHandler:
	case PrivmsgHandler, PrivmsgChannelHandler, PrivmsgUserHandler:
	case NoticeHandler, NoticeChannelHandler, NoticeUserHandler:
	case CTCPHandler, CTCPChannelHandler, CTCPReplyHandler:
	default:
		panic("dispatch: Handler must implement dispatch handler interfaces.")
	}

	event = strings.ToUpper(event)
	network = strings.ToLower(network)
	channel = strings.ToLower(channel)

	var ets eventIDs
	var et eventTable
	var ce channelEvents
	var ok bool

	d.protectEvents.Lock()
	defer d.protectEvents.Unlock()

	if ce, ok = d.events[network]; !ok {
		ce = make(channelEvents)
		d.events[network] = ce
	}
	if ets, ok = ce[channel]; !ok {
		ets = make(eventIDs)
		ce[channel] = ets
	}
	if et, ok = ets[event]; !ok {
		et = make(eventTable)
		ets[event] = et
	}

	d.eventID++
	et[d.eventID] = handler
	return d.eventID
}

// Unregister uses the identifier returned by Register to unregister a
// callback from the Dispatcher. If the callback was removed it returns
// true, false if it could not be found.
func (d *Dispatcher) Unregister(id uint64) bool {
	d.protectEvents.Lock()
	defer d.protectEvents.Unlock()

	for _, ce := range d.events {
		for _, ets := range ce {
			for _, events := range ets {
				for eID := range events {
					if eID == id {
						delete(events, eID)
						return true
					}
				}
			}
		}
	}
	return false
}

// Dispatch an IrcMessage to event handlers handling event also ensures all raw
// handlers receive all messages. Returns false if no eventtable was found for
// the primary sent event.
func (d *Dispatcher) Dispatch(w irc.Writer, ev *irc.Event) bool {

	d.protectEvents.RLock()
	defer d.protectEvents.RUnlock()

	handled := d.dispatchHelper(w, ev)

	return handled
}

// dispatchHelper locates a handler and attempts to resolve it with
// resolveHandler. It returns true if it was able to find an event table.
func (d *Dispatcher) dispatchHelper(w irc.Writer, ev *irc.Event) bool {
	called := false

	networkID := strings.ToLower(ev.NetworkID)

	if ce, ok := d.events[networkID]; ok {
		called = d.filterChannel(ce, w, ev)
	}

	if ce, ok := d.events[""]; ok {
		called = called || d.filterChannel(ce, w, ev)
	}

	return called
}

func (d *Dispatcher) filterChannel(
	ce channelEvents, w irc.Writer, ev *irc.Event) bool {
	called := false

	if len(ev.Args) > 0 && ev.IsTargetChan() {
		target := strings.ToLower(ev.Target())
		if ets, ok := ce[target]; ok {
			called = d.filterEvent(ets, w, ev)
		}
	}

	if ets, ok := ce[""]; ok {
		called = called || d.filterEvent(ets, w, ev)
	}

	return called
}

func (d *Dispatcher) filterEvent(
	ets eventIDs, w irc.Writer, ev *irc.Event) bool {

	called := false

	if et, ok := ets[ev.Name]; ok {
		for _, handler := range et {
			d.HandlerStarted()
			go d.resolveHandler(handler, w, ev)
			called = true
		}
	}

	if et, ok := ets[""]; ok {
		for _, handler := range et {
			d.HandlerStarted()
			go d.resolveHandler(handler, w, ev)
			called = true
		}
	}

	return called
}

// resolveHandler checks the type of the handler passed in, resolves it to a
// real type, coerces the IrcMessage in whatever way necessary and then
// calls that handlers primary dispatch method with the coerced message.
func (d *Dispatcher) resolveHandler(
	handler interface{}, w irc.Writer, ev *irc.Event) {

	defer d.PanicHandler()
	defer d.HandlerFinished()

	var handled bool
	switch ev.Name {
	case irc.PRIVMSG, irc.NOTICE:
		if len(ev.Args) >= 2 && irc.IsCTCPString(ev.Message()) {
			if ev.Name == irc.PRIVMSG {
				handled = d.dispatchCTCP(handler, w, ev)
			} else {
				handled = d.dispatchCTCPReply(handler, w, ev)
			}
		} else {
			if ev.Name == irc.PRIVMSG {
				handled = d.dispatchPrivmsg(handler, w, ev)
			} else {
				handled = d.dispatchNotice(handler, w, ev)
			}
		}
	}

	if !handled {
		if evHandler, ok := handler.(EventHandler); ok {
			evHandler.HandleRaw(w, ev)
		}
	}
}

// dispatchPrivmsg dispatches only a private message. Returns true if the
// event was handled.
func (d *Dispatcher) dispatchPrivmsg(
	handler interface{}, w irc.Writer, ev *irc.Event) (handled bool) {

	if channelHandler, ok := handler.(PrivmsgChannelHandler); ok &&
		d.shouldDispatch(true, ev) {
		channelHandler.PrivmsgChannel(w, ev)
		handled = true
	} else if userHandler, ok := handler.(PrivmsgUserHandler); ok &&
		d.shouldDispatch(false, ev) {
		userHandler.PrivmsgUser(w, ev)
		handled = true
	} else if privmsgHandler, ok := handler.(PrivmsgHandler); ok {
		privmsgHandler.Privmsg(w, ev)
		handled = true
	}
	return
}

// dispatchNotice dispatches only a notice message. Returns true if the
// event was handled.
func (d *Dispatcher) dispatchNotice(
	handler interface{}, w irc.Writer, ev *irc.Event) (handled bool) {

	if channelHandler, ok := handler.(NoticeChannelHandler); ok &&
		d.shouldDispatch(true, ev) {
		channelHandler.NoticeChannel(w, ev)
		handled = true
	} else if userHandler, ok := handler.(NoticeUserHandler); ok &&
		d.shouldDispatch(false, ev) {
		userHandler.NoticeUser(w, ev)
		handled = true
	} else if noticeHandler, ok := handler.(NoticeHandler); ok {
		noticeHandler.Notice(w, ev)
		handled = true
	}
	return
}

// dispatchCTCP dispatches only a ctcp message. Returns true if the
// event was handled.
func (d *Dispatcher) dispatchCTCP(
	handler interface{}, w irc.Writer, ev *irc.Event) (handled bool) {

	tag, data := irc.CTCPunpackString(ev.Message())

	if channelHandler, ok := handler.(CTCPChannelHandler); ok &&
		d.shouldDispatch(true, ev) {
		channelHandler.CTCPChannel(w, ev, tag, data)
		handled = true
	} else if directHandler, ok := handler.(CTCPHandler); ok {
		directHandler.CTCP(w, ev, tag, data)
		handled = true
	}
	return
}

// dispatchCTCPReply dispatches only a ctcpreply message. Returns true if the
// event was handled.
func (d *Dispatcher) dispatchCTCPReply(
	handler interface{}, w irc.Writer, ev *irc.Event) (handled bool) {

	tag, data := irc.CTCPunpackString(ev.Message())

	if directHandler, ok := handler.(CTCPReplyHandler); ok {
		directHandler.CTCPReply(w, ev, tag, data)
		handled = true
	}
	return
}

// shouldDispatch checks if we should dispatch this event. Works for user and
// channel based messages.
func (d *DispatchCore) shouldDispatch(channelMsg bool, ev *irc.Event) bool {
	isChan, hasChan := d.CheckTarget(ev)
	return channelMsg == isChan && (!channelMsg || hasChan)
}
