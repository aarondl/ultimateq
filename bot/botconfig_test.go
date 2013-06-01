package bot

import (
	"bytes"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	. "launchpad.net/gocheck"
	"net"
)

var zeroConnProvider = func(srv string) (net.Conn, error) {
	return nil, nil
}

func (s *s) TestBot_ReadConfig(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Assert(err, IsNil)

	b.ReadConfig(func(conf *config.Config) {
		c.Assert(
			conf.Servers[serverId].GetNick(),
			Equals,
			fakeConfig.Servers[serverId].GetNick(),
		)
	})
}

func (s *s) TestBot_WriteConfig(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Assert(err, IsNil)

	b.WriteConfig(func(conf *config.Config) {
		c.Assert(
			conf.Servers[serverId].GetNick(),
			Equals,
			fakeConfig.Servers[serverId].GetNick(),
		)
	})
}

func (s *s) TestBot_ReplaceConfig(c *C) {
	nick := []byte(irc.NICK + " :newnick\r\n")

	conns := make(map[string]*mocks.Conn)
	connProvider := func(srv string) (net.Conn, error) {
		conn := mocks.CreateConn()
		conns[srv[:len(srv)-5]] = conn //Remove port
		return conn, nil
	}

	chans1 := []string{"#chan1", "#chan2", "#chan3"}
	chans2 := []string{"#chan1", "#chan3"}
	chans3 := []string{"#chan1"}

	c1 := fakeConfig.Clone().
		GlobalContext().
		Channels(chans1...).
		Server("newserver")

	c2 := fakeConfig.Clone().
		GlobalContext().
		Channels(chans2...).
		ServerContext(serverId).
		Nick("newnick").
		Channels(chans3...).
		Server("anothernewserver")

	b, err := createBot(c1, nil, connProvider, false)
	c.Assert(err, IsNil)
	c.Assert(len(b.servers), Equals, 2)

	oldsrv1, oldsrv2 := b.servers[serverId], b.servers["newserver"]

	errs := b.Connect()
	c.Assert(len(errs), Equals, 0)
	b.start(true, false)

	c.Assert(elementsEquals(b.conf.Global.Channels, chans1), Equals, true)
	c.Assert(elementsEquals(oldsrv1.conf.GetChannels(), chans1), Equals, true)
	c.Assert(elementsEquals(oldsrv2.conf.GetChannels(), chans1), Equals, true)
	c.Assert(elementsEquals(b.dispatcher.GetChannels(), chans1), Equals, true)
	c.Assert(elementsEquals(oldsrv1.dispatcher.GetChannels(), chans1),
		Equals, true)
	c.Assert(elementsEquals(oldsrv2.dispatcher.GetChannels(), chans1),
		Equals, true)

	servers := b.ReplaceConfig(c2)
	c.Assert(len(servers), Equals, 1)
	c.Assert(len(b.servers), Equals, 2)

	c.Assert(elementsEquals(b.conf.Global.Channels, chans2), Equals, true)
	c.Assert(elementsEquals(oldsrv1.conf.GetChannels(), chans3), Equals, true)
	c.Assert(elementsEquals(servers[0].server.conf.GetChannels(), chans2),
		Equals, true)
	c.Assert(elementsEquals(b.dispatcher.GetChannels(), chans2), Equals, true)
	c.Assert(elementsEquals(oldsrv1.dispatcher.GetChannels(), chans3),
		Equals, true)
	c.Assert(elementsEquals(servers[0].server.dispatcher.GetChannels(), chans2),
		Equals, true)

	c.Assert(servers[0].Err, IsNil)
	c.Assert(servers[0].ServerName, Equals, "anothernewserver")

	c.Assert(
		bytes.Compare(conns[serverId].Receive(len(nick), nil), nick),
		Equals,
		0,
	)

	server := servers[0].server
	c.Assert(server, NotNil)

	errs = b.Connect()
	c.Assert(len(errs), Equals, 1)
	c.Assert(errs[0].Error(), Matches, ".*already connected.\n")

	b.start(true, false)

	c.Assert(oldsrv1.IsConnected(), Equals, true)
	c.Assert(server.IsConnected(), Equals, true)

	b.Stop()
	b.Disconnect()
	b.WaitForHalt()
}

func (s *s) TestBot_testElementEquals(c *C) {
	a := []string{"a", "b"}
	b := []string{"b", "a"}
	c.Assert(elementsEquals(a, b), Equals, true)

	a = []string{"a", "b", "c"}
	c.Assert(elementsEquals(a, b), Equals, false)

	a = []string{}
	b = []string{}
	c.Assert(elementsEquals(a, b), Equals, true)

	b = []string{"a"}
	c.Assert(elementsEquals(a, b), Equals, false)

	a = []string{"a"}
	b = []string{}
	c.Assert(elementsEquals(a, b), Equals, false)
}
