package dispatch

import (
	"fmt"
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

// eventTable is used to store an event key -> registration id -> handler
type eventTable map[string]map[uint64]interface{}

// Dispatcher is made for handling dispatching of raw-ish irc events.
type Dispatcher struct {
	*DispatchCore
	events        eventTable
	protectEvents sync.RWMutex
	eventID       uint64
}

// NewDispatcher initializes an empty dispatcher ready to register events.
func NewDispatcher(core *DispatchCore) *Dispatcher {
	return &Dispatcher{
		DispatchCore: core,
		events:       make(eventTable),
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

	d.protectEvents.Lock()
	defer d.protectEvents.Unlock()

	network = strings.ToLower(network)
	channel = strings.ToLower(channel)
	event = strings.ToLower(event)

	key := mkKey(network, channel, event)
	idTable, ok := d.events[key]
	if !ok {
		idTable = make(map[uint64]interface{})
		d.events[key] = idTable
	}

	d.eventID++
	idTable[d.eventID] = handler
	return d.eventID
}

// Unregister uses the identifier returned by Register to unregister a
// callback from the Dispatcher. If the callback was removed it returns
// true, false if it could not be found.
func (d *Dispatcher) Unregister(id uint64) bool {
	d.protectEvents.Lock()
	defer d.protectEvents.Unlock()

	for _, idTable := range d.events {
		if _, ok := idTable[id]; ok {
			delete(idTable, id)
			return true
		}
	}
	return false
}

// Dispatch an IrcMessage to event handlers handling event also ensures all raw
// handlers receive all messages.
func (d *Dispatcher) Dispatch(w irc.Writer, ev *irc.Event) {
	d.protectEvents.RLock()
	defer d.protectEvents.RUnlock()

	var isChan bool
	network := strings.ToLower(ev.NetworkID)
	channel := ""
	event := strings.ToLower(ev.Name)
	if isChan = len(ev.Args) > 1 && ev.IsTargetChan(); isChan {
		channel = strings.ToLower(ev.Target())
	}

	// Try most specific key to most generic keys, ending in the global key.
	d.tryKey(network, channel, event, w, ev)
	d.tryKey(network, "", "", w, ev)
	d.tryKey(network, "", event, w, ev)
	if isChan {
		d.tryKey("", channel, "", w, ev)
		d.tryKey(network, channel, "", w, ev)
		d.tryKey("", channel, event, w, ev)
	}
	d.tryKey("", "", event, w, ev)
	d.tryKey("", "", "", w, ev)

	// Raw handlers
	d.tryKey(network, channel, irc.RAW, w, ev)
	d.tryKey(network, "", irc.RAW, w, ev)
	d.tryKey("", channel, irc.RAW, w, ev)
}

// tryKey attempts to use a key to fire off an event.
func (d *Dispatcher) tryKey(net, ch, e string, w irc.Writer, ev *irc.Event) {
	key := mkKey(net, ch, e)
	idTable, ok := d.events[key]
	if ok {
		for _, handler := range idTable {
			d.HandlerStarted()
			go d.resolveHandler(handler, w, ev)
		}
	}
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

// mkKey creates a key for event lookups.
func mkKey(network, channel, event string) string {
	return fmt.Sprintf("%s:%s:%s", network, channel, event)
}
