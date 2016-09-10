package apiserver

import (
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

type msgPipe struct {
	Events    chan *irc.Event
	CmdEvents chan *cmd.Event
}

func (m msgPipe) HandleRaw(w irc.Writer, ev *irc.Event) {
}

func (m msgPipe) Cmd(command string, w irc.Writer, ev *cmd.Event) {
}
