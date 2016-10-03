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

// DispatchCore is a core for any dispatching mechanisms that includes a sync'd
// list of channels, channel identification services, and a waiter to
// synchronize the exit of all the event handlers sharing this core.
type DispatchCore struct {
	log     log15.Logger
	waiter  sync.WaitGroup
	protect sync.RWMutex
}

// NewDispatchCore initializes a dispatch core
func NewDispatchCore(logger log15.Logger) *DispatchCore {
	d := &DispatchCore{log: logger}

	return d
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

// PanicHandler catches any panics and logs a stack trace
func (d *DispatchCore) PanicHandler() {
	recovered := recover()
	if recovered == nil {
		return
	}
	buf := make([]byte, 1024)
	runtime.Stack(buf, false)
	d.log.Error("Handler failed", "panic", recovered)
	d.log.Error(string(buf))
}
