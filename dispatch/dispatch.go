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
	"sync"
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
	finder *irc.ChannelFinder
	chans  []string
	waiter sync.WaitGroup

	// Protects all state variables.
	protect sync.RWMutex
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

	d := &Dispatcher{
		events: make(eventTableStore),
	}

	err := d.protocaps(caps)
	if err != nil {
		return nil, err
	}
	d.channels(activeChannels)

	return d, nil
}

// Protocaps sets the protocaps for this dispatcher.
func (d *Dispatcher) Protocaps(caps *irc.ProtoCaps) (err error) {
	d.protect.Lock()
	defer d.protect.Unlock()
	err = d.protocaps(caps)
	return
}

// protocaps sets the protocaps for this dispatcher. Not thread safe.
func (d *Dispatcher) protocaps(caps *irc.ProtoCaps) (err error) {
	d.finder, err = irc.CreateChannelFinder(caps.Chantypes())
	return
}

// Channels sets the active channels for this dispatcher.
func (d *Dispatcher) Channels(chans []string) {
	d.protect.Lock()
	d.channels(chans)
	d.protect.Unlock()
}

// GetChannels gets the active channels for this dispatcher.
func (d *Dispatcher) GetChannels() (chans []string) {
	d.protect.Lock()
	defer d.protect.Unlock()

	if d.chans == nil {
		return
	}
	chans = make([]string, len(d.chans))
	copy(chans, d.chans)
	return
}

// AddChannels adds channels to the active channels for this dispatcher.
func (d *Dispatcher) AddChannels(chans ...string) {
	if 0 == len(chans) {
		return
	}
	d.protect.Lock()
	defer d.protect.Unlock()

	if d.chans == nil {
		d.chans = make([]string, 0, len(chans))
	}

	for i := 0; i < len(chans); i++ {
		addchan := strings.ToLower(chans[i])
		found := false
		for j, length := 0, len(d.chans); j < length; j++ {
			if d.chans[j] == addchan {
				found = true
				break
			}
		}
		if !found {
			d.chans = append(d.chans, addchan)
		}
	}
}

// RemoveChannels removes channels to the active channels for this dispatcher.
func (d *Dispatcher) RemoveChannels(chans ...string) {
	if 0 == len(chans) {
		return
	}
	d.protect.Lock()
	defer d.protect.Unlock()

	if d.chans == nil || 0 == len(d.chans) {
		return
	}

	for i := 0; i < len(chans); i++ {
		removechan := strings.ToLower(chans[i])
		for j, length := 0, len(d.chans); j < length; j++ {
			if d.chans[j] == removechan {
				if length == 1 {
					d.chans = nil
					return
				}
				if j < length-1 {
					d.chans[j], d.chans[length-1] =
						d.chans[length-1], d.chans[j]
				}
				d.chans = d.chans[:length-1]
				length--
			}
		}
	}
}

// channels sets the active channels for this dispatcher. Not thread
// safe.
func (d *Dispatcher) channels(chans []string) {
	length := len(chans)
	if length == 0 {
		d.chans = nil
	} else {
		d.chans = make([]string, length)
		for i := 0; i < length; i++ {
			d.chans[i] = strings.ToLower(chans[i])
		}
	}
}

// Register registers an event handler to a particular event. In return a
// unique identifer is given to later pass into Unregister in case of a need
// to unregister the event handler.
func (d *Dispatcher) Register(event string, handler interface{}) int {
	event = strings.ToUpper(event)
	id := rand.Int()

	d.protect.Lock()
	defer d.protect.Unlock()

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

	d.protect.Lock()
	defer d.protect.Unlock()

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

	d.protect.RLock()
	defer d.protect.RUnlock()

	handled := d.dispatchHelper(event, msg, sender)
	d.dispatchHelper(irc.RAW, msg, sender)

	return handled
}

// WaitForCompletion waits on all active event handlers to return. Bad event
// handlers may never return.
func (d *Dispatcher) WaitForCompletion() {
	d.waiter.Wait()
}

// dispatchHelper locates a handler and attempts to resolve it with
// resolveHandler. It returns true if it was able to find an event table.
func (d *Dispatcher) dispatchHelper(event string,
	msg *irc.IrcMessage, sender irc.Sender) bool {

	if evtable, ok := d.events[event]; ok {
		for _, handler := range evtable {
			d.waiter.Add(1)
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
	case PrivmsgHandler, PrivmsgUserHandler, PrivmsgChannelHandler:

		if channelHandler, ok := t.(PrivmsgChannelHandler); ok &&
			d.shouldDispatch(true, msg) {

			channelHandler.PrivmsgChannel(&irc.Message{msg}, sender)
		} else if userHandler, ok := t.(PrivmsgUserHandler); ok &&
			d.shouldDispatch(false, msg) {

			userHandler.PrivmsgUser(&irc.Message{msg}, sender)
		} else if privmsgHandler, ok := t.(PrivmsgHandler); ok {
			privmsgHandler.Privmsg(&irc.Message{msg}, sender)
		}

	case NoticeHandler, NoticeUserHandler, NoticeChannelHandler:

		if channelHandler, ok := t.(NoticeChannelHandler); ok &&
			d.shouldDispatch(true, msg) {

			channelHandler.NoticeChannel(&irc.Message{msg}, sender)
		} else if userHandler, ok := t.(NoticeUserHandler); ok &&
			d.shouldDispatch(false, msg) {

			userHandler.NoticeUser(&irc.Message{msg}, sender)
		} else if noticeHandler, ok := t.(NoticeHandler); ok {
			noticeHandler.Notice(&irc.Message{msg}, sender)
		}
	case EventHandler:
		t.HandleRaw(msg, sender)
	}

	d.waiter.Done()
}

// shouldDispatch checks if we should dispatch this event. Works for user and
// channel based messages.
func (d *Dispatcher) shouldDispatch(channel bool, msg *irc.IrcMessage) bool {
	d.protect.RLock()
	defer d.protect.RUnlock()
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
