package dispatch

import (
	"fmt"
	"github.com/aarondl/ultimateq/irc"
	. "testing"
)

func checkArrays(expected []string, actual []string) error {
	if len(expected) != len(actual) {
		return fmt.Errorf("Length Expected: %v got: %v",
			len(expected), len(actual))
	}
	for i, v := range expected {
		if v != actual[i] {
			return fmt.Errorf("Expected: %v got: %v", v, actual[i])
		}
	}
	return nil
}

var caps = irc.CreateProtoCaps()

func TestDispatchCore(t *T) {
	t.Parallel()
	d := CreateDispatchCore(caps)
	if d == nil {
		t.Error("Create should create things.")
	}
}

func TestDispatchCore_Synchronization(t *T) {
	t.Parallel()
	d := CreateDispatchCore(caps)
	d.HandlerStarted()
	d.HandlerStarted()
	d.HandlerFinished()
	d.HandlerFinished()
	d.WaitForHandlers()
}

func TestDispatchCore_AddRemoveChannels(t *T) {
	t.Parallel()
	chans := []string{"#chan1", "#chan2", "#chan3"}
	d := CreateDispatchCore(caps, chans...)

	if err := checkArrays(chans, d.chans); err != nil {
		t.Error(err)
	}

	d.RemoveChannels(chans...)
	if d.chans != nil {
		t.Error("Removing everything should remove everything.")
	}
	d.RemoveChannels(chans...)
	if d.chans != nil {
		t.Error("Removing everything should remove everything.")
	}
	d.RemoveChannels()
	if d.chans != nil {
		t.Error("Removing nothing should add test coverage.")
	}

	d.Channels(chans)
	d.RemoveChannels(chans[1:]...)
	if err := checkArrays(chans[:1], d.chans); err != nil {
		t.Error(err)
	}
	d.AddChannels(chans[1:]...)
	if err := checkArrays(chans, d.chans); err != nil {
		t.Error(err)
	}
	d.AddChannels(chans[0])
	d.AddChannels()
	if err := checkArrays(chans, d.chans); err != nil {
		t.Error(err)
	}
	d.RemoveChannels(chans...)
	d.AddChannels(chans...)
	if err := checkArrays(chans, d.chans); err != nil {
		t.Error(err)
	}
}

func TestDispatchCore_GetChannels(t *T) {
	t.Parallel()
	d := CreateDispatchCore(caps)

	if d.GetChannels() != nil {
		t.Error("Should start uninitialized.")
	}
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)
	if err := checkArrays(d.chans, d.GetChannels()); err != nil {
		t.Error(err)
	}

	first := d.GetChannels()
	first[0] = "#chan3"
	if err := checkArrays(d.chans, d.GetChannels()); err != nil {
		t.Error(err)
		t.Error("The array should be copied so data changes do not affect it.")
	}
}

func TestDispatchCore_UpdateChannels(t *T) {
	t.Parallel()
	d := CreateDispatchCore(caps)
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)
	if err := checkArrays(chans, d.chans); err != nil {
		t.Error("Channels were not set correctly.")
	}
	d.Channels([]string{})
	if d.chans != nil {
		t.Error("It should be nil after an empty set.")
	}
	d.Channels(chans)
	if err := checkArrays(chans, d.chans); err != nil {
		t.Error("Channels were not set correctly.")
	}
	d.Channels(nil)
	if d.chans != nil {
		t.Error("It should be nil after an empty set.")
	}
}

func TestDispatchCore_UpdateProtoCaps(t *T) {
	t.Parallel()
	p := irc.CreateProtoCaps()
	p.ParseISupport(&irc.Message{Args: []string{"nick", "CHANTYPES=#"}})
	d := CreateDispatchCore(p)
	if isChan, _ := d.CheckTarget("#chan"); !isChan {
		t.Error("Expected it to be a channel.")
	}
	if isChan, _ := d.CheckTarget("&chan"); isChan {
		t.Error("Expected it to not be a channel.")
	}

	p = irc.CreateProtoCaps()
	p.ParseISupport(&irc.Message{Args: []string{"nick", "CHANTYPES=&"}})
	d.Protocaps(p)
	if isChan, _ := d.CheckTarget("#chan"); isChan {
		t.Error("Expected it to not be a channel.")
	}
	if isChan, _ := d.CheckTarget("&chan"); !isChan {
		t.Error("Expected it to be a channel.")
	}
}

func TestDispatchCore_CheckTarget(t *T) {
	t.Parallel()
	d := CreateDispatchCore(caps, "#chan")

	var tests = []struct {
		IsChan  bool
		HasChan bool
		Chan    string
	}{
		{true, true, "#chan"},
		{true, false, "#chan2"},
		{false, false, "!chan"},
		{false, false, "user"},
	}

	for _, test := range tests {
		isChan, hasChan := d.CheckTarget(test.Chan)
		if isChan != test.IsChan || hasChan != test.HasChan {
			t.Error("Fail:", test)
			t.Errorf("Expected: IsChan(%v) HasChan(%v)",
				test.IsChan, test.HasChan)
			t.Errorf("Actual: IsChan(%v) HasChan(%v)",
				isChan, hasChan)
		}
	}
}

func TestDispatchCore_filterChannelDispatch(t *T) {
	t.Parallel()
	d := CreateDispatchCore(caps, []string{"#CHAN"}...)
	if d.chans == nil {
		t.Error("Initialization failed.")
	}

	if has := d.hasChannel("#chan"); !has {
		t.Error("It should have this channel.")
	}
	if has := d.hasChannel("#chan2"); has {
		t.Error("It should not have this channel.")
	}
}
