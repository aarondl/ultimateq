package apiserver

import (
	"sync"
	"time"

	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"gopkg.in/inconshreveable/log15.v2"
)

const (
	msgPipeEventTimeout = 30 * time.Second
)

type ircEvent struct {
	streamID uint32
	ev       *irc.Event
}

type cmdEvent struct {
	streamID uint32
	command  string
	ev       *cmd.Event
}

type msgPipe struct {
	log log15.Logger

	Events    chan ircEvent
	CmdEvents chan cmdEvent

	mut      sync.Mutex
	streamID uint32
	writers  map[uint32]chan<- []byte
}

func newMsgPipe(log log15.Logger) *msgPipe {
	return &msgPipe{
		log:       log,
		Events:    make(chan ircEvent),
		CmdEvents: make(chan cmdEvent),
		writers:   make(map[uint32]chan<- []byte),
	}
}

func (m *msgPipe) openWriter() (uint32, <-chan []byte) {
	writer := make(chan []byte)
	var streamID uint32

	msgPipe.mut.Lock()
	m.streamID++
	streamID = m.streamID

	m.writers[m.streamID] = writer
	msgPipe.mut.Unlock()

	return streamID, writer
}

func (m *msgPipe) getWriter(streamID uint32) chan<- []byte {
	msgPipe.mut.Lock()
	w := msgPipe.writers[streamID]
	m.log.Debug("get writer", "streamid", streamID)
	msgPipe.mut.Unlock()

	return w
}

func (m *msgPipe) closeWriter(streamID uint32) {
	msgPipe.mut.Lock()
	w, ok := msgPipe.writers[streamID]
	delete(msgPipe.writers, streamID)
	m.log.Debug("delete writer", "streamid", streamID)
	msgPipe.mut.Unlock()

	if !ok {
		return
	}

	close(w)
}

func (m *msgPipe) closeAll() {
	m.log.Debug("close all")

	msgPipe.mut.Lock()
	for k, w := range m.writers {
		close(w)
		delete(msgPipe.writers, streamID)
	}
	msgPipe.mut.Unlock()
}

func (m *msgPipe) HandleRaw(w irc.Writer, ev *irc.Event) {
	streamID, writer := m.openWriter()

	select {
	case m.Events <- ircEvent{streamID: streamID, ev: ev}:
		m.log.Debug("raw event fired", "evname", ev.Name, "streamid", streamID)
	case <-time.After(msgPipeEventTimeout):
		// No one picked up the phone
		m.log.Debug("raw event expired", "evname", ev.Name, "streamid", streamID)
		m.closeWriter(streamID)
		return
	}

	for {
		msg, ok := <-writer
		if !ok {
			return
		}

		w.Write(msg)
	}
}

func (m *msgPipe) Cmd(command string, w irc.Writer, ev *cmd.Event) error {
	writer := m.openWriter()

	select {
	case m.Events <- ircEvent{streamID: streamID, ev: ev}:
		m.log.Debug("raw cmd fired", "cmdname", command, "streamid", streamID)
	case <-time.After(msgPipeEventTimeout):
		// No one picked up the phone
		m.log.Debug("raw cmd expired", "cmdname", command, "streamid", streamID)
		m.closeWriter(streamID)
		return
	}

	for {
		msg, ok := <-writer
		if !ok {
			return
		}

		w.Write(msg)
	}
}
