/*
Package dispatch supplies a low level core that can be shared between
multiple dispatcher implementations. It also provides one basic
dispatcher implementation.
*/
package dispatch

import (
	"errors"
	"github.com/aarondl/ultimateq/irc"
	"strings"
	"sync"
)

var (
	// errProtoCapsMissing is returned when a core is instantiated with a nil
	// protocaps.
	errProtoCapsMissing = errors.New(
		"dispatching: Cannot create a dispatcher without ProtoCaps.")
)

// DispatchCore is a core for any dispatching mechanisms that includes a sync'd
// list of channels, a finder, and a waiter to synchronize the exit of all the
// event handlers sharing this core.
type DispatchCore struct {
	waiter  sync.WaitGroup
	finder  *irc.ChannelFinder
	chans   []string
	protect sync.RWMutex
}

// CreateDispatchCore initializes a dispatch core, it takes a protocaps in order
// to filter between channel and other messages.
func CreateDispatchCore(
	caps *irc.ProtoCaps, activeChannels ...string) (*DispatchCore, error) {

	if caps == nil {
		return nil, errProtoCapsMissing
	}

	d := &DispatchCore{}

	err := d.protocaps(caps)
	if err != nil {
		return nil, err
	}
	d.channels(activeChannels)

	return d, nil
}

// Protocaps sets the protocaps for this dispatcher.
func (d *DispatchCore) Protocaps(caps *irc.ProtoCaps) (err error) {
	d.protect.Lock()
	defer d.protect.Unlock()
	err = d.protocaps(caps)
	return
}

// protocaps sets the protocaps for this dispatcher. Not thread safe.
func (d *DispatchCore) protocaps(caps *irc.ProtoCaps) (err error) {
	d.finder, err = irc.CreateChannelFinder(caps.Chantypes())
	return
}

// Channels sets the active channels for this dispatcher.
func (d *DispatchCore) Channels(chans []string) {
	d.protect.Lock()
	d.channels(chans)
	d.protect.Unlock()
}

// GetChannels gets the active channels for this dispatcher.
func (d *DispatchCore) GetChannels() (chans []string) {
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
func (d *DispatchCore) AddChannels(chans ...string) {
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
func (d *DispatchCore) RemoveChannels(chans ...string) {
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
func (d *DispatchCore) channels(chans []string) {
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

// HandlerStarted tells the core that a handler has started and it should be
// waited on.
func (d *DispatchCore) HandlerStarted() {
	d.waiter.Add(1)
}

// HandlerFinished tells the core that a handler has ended.
func (d *DispatchCore) HandlerFinished() {
	d.waiter.Done()
}

// WaitForHandlers waits for the unfinished handlers to finish.
func (d *DispatchCore) WaitForHandlers() {
	d.waiter.Wait()
}

// CheckTarget describes a dispatching target. It checks both if it is a
// channel, and if it is a channel, if that channel is an active one for
// this dispatchcore.
func (d *DispatchCore) CheckTarget(target string) (isChan, hasChan bool) {
	d.protect.RLock()
	defer d.protect.RUnlock()
	target = strings.ToLower(target)
	isChan = d.finder.IsChannel(target)
	hasChan = isChan && d.hasChannel(target)
	return
}

// hasChannel checks to see if the dispatch core's channel list includes a
// channel.
func (d *DispatchCore) hasChannel(channel string) bool {
	if d.chans == nil {
		return true
	}

	targ := strings.ToLower(channel)
	for i := 0; i < len(d.chans); i++ {
		if targ == d.chans[i] {
			return true
		}
	}
	return false
}
