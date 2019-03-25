package dispatch

import (
	"testing"

	"github.com/aarondl/ultimateq/irc"
)

var netInfo = irc.NewNetworkInfo()

func TestCore(t *testing.T) {
	t.Parallel()
	d := NewCore(nil)
	if d == nil {
		t.Error("Create should create things.")
	}
}

func TestCore_Synchronization(t *testing.T) {
	t.Parallel()
	d := NewCore(nil)
	d.HandlerStarted()
	d.HandlerStarted()
	d.HandlerFinished()
	d.HandlerFinished()
	d.WaitForHandlers()
}
