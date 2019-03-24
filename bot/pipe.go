package bot

import (
	"sync"
	"time"

	"github.com/aarondl/ultimateq/api"

	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
)

const (
	msgPipeEventTimeout = 5 * time.Second
)

var _ dispatch.Handler = &pipeHandler{}
var _ cmd.Handler = &pipeHandler{}

type pipeHandler struct {
	logger log15.Logger
	ext    string

	cleanupFn func(ext string, id uint64)

	// A pipeHandler briefly exists during a time where there could be no
	// eventID set (after register) but events are firing (before eventID can
	// be set) so this protects eventID while setting initially
	eventIDMut sync.RWMutex
	eventID    uint64

	eventChan   chan<- *api.IRCEventResponse
	commandChan chan<- *api.CmdEventResponse
}

func (p *pipeHandler) setEventID(evID uint64) {
	p.eventIDMut.Lock()
	p.eventID = evID
	p.eventIDMut.Unlock()
}

func (p *pipeHandler) Handle(w irc.Writer, ev *irc.Event) {
	p.eventIDMut.RLock()
	evID := p.eventID
	p.eventIDMut.RUnlock()
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

	select {
	case p.eventChan <- event:
	case <-time.After(msgPipeEventTimeout):
		p.logger.Info("remote event send timeout", "ext", p.ext, "handler_id", evID)
		p.cleanupFn(p.ext, evID)
	}
}

func (p *pipeHandler) Cmd(name string, w irc.Writer, ev *cmd.Event) error {
	p.eventIDMut.RLock()
	evID := p.eventID
	p.eventIDMut.RUnlock()
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

	select {
	case p.commandChan <- command:
	case <-time.After(msgPipeEventTimeout):
		p.logger.Info("remote cmd send timeout", "ext", p.ext, "handler_id", evID)
		p.cleanupFn(p.ext, evID)
	}

	return nil
}
