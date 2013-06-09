package bot

import (
	"bytes"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"io"
	. "launchpad.net/gocheck"
	"net"
)

var zeroConnProvider = func(srv string) (net.Conn, error) {
	return nil, nil
}

func (s *s) TestBot_ReadConfig(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Check(err, IsNil)

	b.ReadConfig(func(conf *config.Config) {
		c.Check(
			conf.Servers[serverId].GetNick(),
			Equals,
			fakeConfig.Servers[serverId].GetNick(),
		)
	})
}

func (s *s) TestBot_WriteConfig(c *C) {
	b, err := createBot(fakeConfig, nil, nil, false)
	c.Check(err, IsNil)

	b.WriteConfig(func(conf *config.Config) {
		c.Check(
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

	c3 := &config.Config{}

	b, err := createBot(c1, nil, connProvider, false)
	c.Check(err, IsNil)
	c.Check(len(b.servers), Equals, 2)

	oldsrv1, oldsrv2 := b.servers[serverId], b.servers["newserver"]

	errs := b.Connect()
	c.Check(len(errs), Equals, 0)
	b.start(true, false)

	c.Check(elementsEquals(b.conf.Global.Channels, chans1), Equals, true)
	c.Check(elementsEquals(oldsrv1.conf.GetChannels(), chans1), Equals, true)
	c.Check(elementsEquals(oldsrv2.conf.GetChannels(), chans1), Equals, true)
	c.Check(elementsEquals(b.dispatcher.GetChannels(), chans1), Equals, true)
	c.Check(elementsEquals(oldsrv1.dispatcher.GetChannels(), chans1),
		Equals, true)
	c.Check(elementsEquals(oldsrv2.dispatcher.GetChannels(), chans1),
		Equals, true)

	servers := b.ReplaceConfig(c3)
	c.Check(len(servers), Equals, 0)
	servers = b.ReplaceConfig(c2)
	c.Check(len(servers), Equals, 1)
	c.Check(len(b.servers), Equals, 2)

	c.Check(elementsEquals(b.conf.Global.Channels, chans2), Equals, true)
	c.Check(elementsEquals(oldsrv1.conf.GetChannels(), chans3), Equals, true)
	c.Check(elementsEquals(servers[0].server.conf.GetChannels(), chans2),
		Equals, true)
	c.Check(elementsEquals(b.dispatcher.GetChannels(), chans2), Equals, true)
	c.Check(elementsEquals(oldsrv1.dispatcher.GetChannels(), chans3),
		Equals, true)
	c.Check(elementsEquals(servers[0].server.dispatcher.GetChannels(), chans2),
		Equals, true)

	c.Check(servers[0].Err, IsNil)
	c.Check(servers[0].ServerName, Equals, "anothernewserver")

	c.Check(
		bytes.Compare(conns[serverId].Receive(len(nick), nil), nick),
		Equals,
		0,
	)

	server := servers[0].server
	c.Check(server, NotNil)

	errs = b.Connect()
	c.Check(len(errs), Equals, 2)
	c.Check(errs[0].Error(), Matches, ".*already connected.\n")

	c.Check(oldsrv1.IsConnected(), Equals, true)
	c.Check(server.IsConnected(), Equals, true)

	conns["anothernewserver"].Send([]byte{}, 0, io.EOF)

	b.Stop()
	b.Disconnect()
	b.WaitForHalt()
}

func (s *s) TestBot_testElementEquals(c *C) {
	a := []string{"a", "b"}
	b := []string{"b", "a"}
	c.Check(elementsEquals(a, b), Equals, true)

	a = []string{"a", "b", "c"}
	c.Check(elementsEquals(a, b), Equals, false)

	a = []string{"x", "y"}
	c.Check(elementsEquals(a, b), Equals, false)

	a = []string{}
	b = []string{}
	c.Check(elementsEquals(a, b), Equals, true)

	b = []string{"a"}
	c.Check(elementsEquals(a, b), Equals, false)

	a = []string{"a"}
	b = []string{}
	c.Check(elementsEquals(a, b), Equals, false)
}
