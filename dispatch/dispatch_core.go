/*
Package dispatch is used to dispatch irc messages to event handlers in an
concurrent fashion. It supports various event handler types to easily
extract information from events, as well as define more succint handlers.
*/
package dispatch

import (
	"runtime"
	"sync"

	"gopkg.in/inconshreveable/log15.v2"
)

//Core is a core for any dispatching mechanisms that includes a sync'd
// a waiter to synchronize the exit of all the event handlers sharing this core.
type Core struct {
	log     log15.Logger
	waiter  sync.WaitGroup
	protect sync.RWMutex
}

// NewCore initializes a dispatch core
func NewCore(logger log15.Logger) *Core {
	d := &Core{log: logger}

	return d
}

// HandlerStarted tells the core that a handler has started and it should be
// waited on.
func (d *Core) HandlerStarted() {
	d.waiter.Add(1)
}

// HandlerFinished tells the core that a handler has ended.
func (d *Core) HandlerFinished() {
	d.waiter.Done()
}

// WaitForHandlers waits for the unfinished handlers to finish.
func (d *Core) WaitForHandlers() {
	d.waiter.Wait()
}

// PanicHandler catches any panics and logs a stack trace
func (d *Core) PanicHandler() {
	recovered := recover()
	if recovered == nil {
		return
	}
	buf := make([]byte, 1024)
	runtime.Stack(buf, false)
	d.log.Error("Handler failed", "panic", recovered)
	d.log.Error(string(buf))
}
