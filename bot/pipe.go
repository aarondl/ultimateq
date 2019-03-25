package bot

import (
	"sync"

	"github.com/aarondl/ultimateq/api"

	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
)

const (
	// The initial error in the subscriber on Send() counts for a misfire of
	// sorts
	misfireThreshold = 4
)

var _ dispatch.Handler = &pipeHandler{}
var _ cmd.Handler = &pipeHandler{}

type pipeHelper interface {
	broadcastEvent(ext string, r *api.IRCEventResponse) bool
	broadcastCmd(ext string, r *api.CmdEventResponse) bool
	unregEvent(ext string, id uint64)
	unregCmd(ext string, id uint64)
}

type pipeHandler struct {
	logger log15.Logger
	ext    string

	helper pipeHelper

	// A pipeHandler briefly exists during a time where there could be no
	// eventID set (after register) but events are firing (before eventID can
	// be set) so this protects eventID while setting initially.
	//
	// misfires must also be protected since event handlers are fired from
	// multiple goroutines and they're technically editing the data
	mut     sync.RWMutex
	eventID uint64
	// Misfires is incremented every time this handler fails to deliver
	// to at least one remote subscriber. It marks obsolesence and will be
	// garbage collected upon reaching a threshold.
	misfires int
}

func (p *pipeHandler) setEventID(evID uint64) {
	p.mut.Lock()
	p.eventID = evID
	p.mut.Unlock()
}

func (p *pipeHandler) Handle(w irc.Writer, ev *irc.Event) {
	p.mut.RLock()
	evID := p.eventID
	p.mut.RUnlock()
	if evID == 0 {
		return
	}

	p.logger.Debug("remote event dispatch", "id", evID)

	event := &api.IRCEventResponse{
		Id: evID,
		Event: &api.IRCEvent{
			Name:      ev.Name,
			Sender:    ev.Sender,
			Args:      ev.Args,
			Time:      ev.Time.Unix(),
			NetworkId: ev.NetworkID,
		},
	}

	sent := p.helper.broadcastEvent(p.ext, event)
	if sent {
		return
	}

	p.logger.Debug("remote misfire", "ext", p.ext, "id", evID)

	var misfires int
	p.mut.Lock()
	p.misfires++
	misfires = p.misfires
	p.mut.Unlock()

	if misfires > misfireThreshold {
		p.logger.Debug("unreg event misfire threshold", "ext", p.ext, "id", evID)
		p.helper.unregEvent(p.ext, evID)
	}
}

func (p *pipeHandler) Cmd(name string, w irc.Writer, ev *cmd.Event) error {
	p.mut.RLock()
	evID := p.eventID
	p.mut.RUnlock()
	if evID == 0 {
		return nil
	}

	p.logger.Debug("remote cmd dispatch", "id", evID)

	iev := &api.IRCEvent{
		Name:      ev.Event.Name,
		Sender:    ev.Event.Sender,
		Args:      ev.Event.Args,
		Time:      ev.Event.Time.Unix(),
		NetworkId: ev.Event.NetworkID,
	}

	command := &api.CmdEventResponse{
		Id: evID,
		Event: &api.CmdEvent{
			IrcEvent: iev,
			Args:     ev.Args,
		},
	}

	sent := p.helper.broadcastCmd(p.ext, command)
	if sent {
		return nil
	}

	p.logger.Debug("remote misfire", "ext", p.ext, "id", evID)

	var misfires int
	p.mut.Lock()
	p.misfires++
	misfires = p.misfires
	p.mut.Unlock()

	if misfires > misfireThreshold {
		p.logger.Debug("unreg cmd misfire threshold", "ext", p.ext, "id", evID)
		p.helper.unregEvent(p.ext, evID)
	}
	return nil
}
