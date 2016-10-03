package dispatch

import (
	"sync"

	"github.com/aarondl/ultimateq/irc"
)

// Handler is the interface for use with normal dispatching
type Handler interface {
	Handle(w irc.Writer, ev *irc.Event)
}

// Dispatcher is made for handling dispatching of raw-ish irc events.
type Dispatcher struct {
	*DispatchCore

	trieMut sync.RWMutex
	trie    *trie
}

// NewDispatcher initializes an empty dispatcher ready to register events.
func NewDispatcher(core *DispatchCore) *Dispatcher {
	return &Dispatcher{
		DispatchCore: core,
		trie:         newTrie(false),
	}
}

// Register registers an event handler to a particular event. In return a
// unique identifer is given to later pass into Unregister in case of a need
// to unregister the event handler. Pass in an empty string to any of network,
// channel or event to prevent filtering on that parameter. Panics if it's
// given a type that doesn't implement any of the correct interfaces.
func (d *Dispatcher) Register(network, channel, event string, handler Handler) uint64 {
	d.trieMut.Lock()
	id := d.trie.register(network, channel, event, handler)
	d.trieMut.Unlock()

	return id
}

// Unregister uses the identifier returned by Register to unregister a
// callback from the Dispatcher. If the callback was removed it returns
// true, false if it could not be found.
func (d *Dispatcher) Unregister(id uint64) bool {
	d.trieMut.Lock()
	did := d.trie.unregister(id)
	d.trieMut.Unlock()

	return did
}

// Dispatch an IrcMessage to event handlers handling event also ensures all raw
// handlers receive all messages.
func (d *Dispatcher) Dispatch(w irc.Writer, ev *irc.Event) {
	network := ev.NetworkID
	event := ev.Name
	var channel string
	if len(ev.Args) > 0 && ev.IsTargetChan() {
		channel = ev.Target()
	}

	d.trieMut.RLock()
	handlers := d.trie.handlers(network, channel, event)
	d.trieMut.RUnlock()

	for _, handler := range handlers {
		h := handler.(Handler)
		d.HandlerStarted()
		go func() {
			defer d.HandlerFinished()
			defer d.PanicHandler()
			h.Handle(w, ev)
		}()
	}
}
