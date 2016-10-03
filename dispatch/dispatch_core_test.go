package dispatch

import (
	"testing"

	"github.com/aarondl/ultimateq/irc"
)

var netInfo = irc.NewNetworkInfo()

func TestDispatchCore(t *testing.T) {
	t.Parallel()
	d := NewDispatchCore(nil)
	if d == nil {
		t.Error("Create should create things.")
	}
}

func TestDispatchCore_Synchronization(t *testing.T) {
	t.Parallel()
	d := NewDispatchCore(nil)
	d.HandlerStarted()
	d.HandlerStarted()
	d.HandlerFinished()
	d.HandlerFinished()
	d.WaitForHandlers()
}
