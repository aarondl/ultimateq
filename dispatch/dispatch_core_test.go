package dispatch

import (
	"github.com/aarondl/ultimateq/irc"
	. "launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { TestingT(t) } //Hook into testing package
type s struct{}

var _ = Suite(&s{})

var caps = irc.CreateProtoCaps()

func (s *s) TestDispatchCore(c *C) {
	d, err := CreateDispatchCore(caps)
	c.Check(d, NotNil)
	c.Check(err, IsNil)

	d, err = CreateDispatchCore(nil)
	c.Check(err, Equals, errProtoCapsMissing)

	p := irc.CreateProtoCaps()
	p.ParseISupport(&irc.Message{Args: []string{"nick", "CHANTYPES=H"}})
	d, err = CreateDispatchCore(p)
	c.Check(err, NotNil)
}

func (s *s) TestDispatchCore_Synchronization(c *C) {
	d, err := CreateDispatchCore(caps)
	c.Check(err, IsNil)
	d.HandlerStarted()
	d.HandlerStarted()
	d.HandlerStarted()
	d.HandlerFinished()
	d.HandlerFinished()
	d.HandlerFinished()
	d.WaitForHandlers()
	c.Succeed()
}

func (s *s) TestDispatchCore_AddRemoveChannels(c *C) {
	chans := []string{"#chan1", "#chan2", "#chan3"}
	d, err := CreateDispatchCore(caps, chans...)
	c.Check(err, IsNil)

	c.Check(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Check(d.chans[i], Equals, v)
	}

	d.RemoveChannels(chans...)
	c.Check(d.chans, IsNil)
	d.RemoveChannels(chans...)
	c.Check(d.chans, IsNil)
	d.RemoveChannels()
	c.Check(d.chans, IsNil)

	d.Channels(chans)
	d.RemoveChannels(chans[1:]...)
	c.Check(len(d.chans), Equals, len(chans)-2)
	for i, v := range chans[:1] {
		c.Check(d.chans[i], Equals, v)
	}
	d.AddChannels(chans[1:]...)
	c.Check(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Check(d.chans[i], Equals, v)
	}
	d.AddChannels(chans[0])
	d.AddChannels()
	c.Check(len(d.chans), Equals, len(chans))
	d.RemoveChannels(chans...)
	d.AddChannels(chans...)
	c.Check(len(d.chans), Equals, len(chans))
}

func (s *s) TestDispatchCore_GetChannels(c *C) {
	d, err := CreateDispatchCore(caps)
	c.Check(err, IsNil)

	c.Check(d.GetChannels(), IsNil)
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)

	for i, ch := range d.GetChannels() {
		c.Check(d.chans[i], Equals, ch)
	}

	first := d.GetChannels()
	first[0] = "#chan3"
	for i, ch := range d.GetChannels() {
		c.Check(d.chans[i], Equals, ch)
	}
}

func (s *s) TestDispatchCore_UpdateChannels(c *C) {
	d, err := CreateDispatchCore(caps)
	c.Check(err, IsNil)
	chans := []string{"#chan1", "#chan2"}
	d.Channels(chans)
	c.Check(len(d.chans), Equals, len(chans))
	for i, v := range chans {
		c.Check(d.chans[i], Equals, v)
	}
	d.Channels([]string{})
	c.Check(len(d.chans), Equals, 0)
	d.Channels(chans)
	c.Check(len(d.chans), Equals, len(chans))
	d.Channels(nil)
	c.Check(len(d.chans), Equals, 0)
}

func (s *s) TestDispatchCore_UpdateProtoCaps(c *C) {
	p := irc.CreateProtoCaps()
	p.ParseISupport(&irc.Message{Args: []string{"nick", "CHANTYPES=#"}})
	d, err := CreateDispatchCore(p)
	c.Check(err, IsNil)
	var isChan bool
	isChan, _ = d.CheckTarget("#chan")
	c.Check(isChan, Equals, true)
	isChan, _ = d.CheckTarget("&chan")
	c.Check(isChan, Equals, false)

	p = irc.CreateProtoCaps()
	p.ParseISupport(&irc.Message{Args: []string{"nick", "CHANTYPES=&"}})
	err = d.Protocaps(p)
	c.Check(err, IsNil)
	isChan, _ = d.CheckTarget("#chan")
	c.Check(isChan, Equals, false)
	isChan, _ = d.CheckTarget("&chan")
	c.Check(isChan, Equals, true)
}

func (s *s) TestDispatchCore_CheckTarget(c *C) {
	d, err := CreateDispatchCore(caps, "#chan")
	c.Check(err, IsNil)

	var isChan, hasChan bool
	isChan, hasChan = d.CheckTarget("#chan")
	c.Check(isChan, Equals, true)
	c.Check(hasChan, Equals, true)
	isChan, hasChan = d.CheckTarget("#chan2")
	c.Check(isChan, Equals, true)
	c.Check(hasChan, Equals, false)
	isChan, hasChan = d.CheckTarget("!chan")
	c.Check(isChan, Equals, false)
	c.Check(hasChan, Equals, false)
	isChan, hasChan = d.CheckTarget("user")
	c.Check(isChan, Equals, false)
	c.Check(hasChan, Equals, false)
}

func (s *s) TestDispatchCore_filterChannelDispatch(c *C) {
	d, err := CreateDispatchCore(caps, []string{"#CHAN"}...)
	c.Check(err, IsNil)
	c.Check(d.chans, NotNil)

	var should bool
	should = d.hasChannel("#chan")
	c.Check(should, Equals, true)
	should = d.hasChannel("#chan2")
	c.Check(should, Equals, false)
}
