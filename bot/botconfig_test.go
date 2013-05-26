package bot

import (
	"github.com/aarondl/ultimateq/config"
	. "launchpad.net/gocheck"
	"net"
)

func (s *s) TestBot_ReadConfig(c *C) {
	c.SucceedNow()
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)
	_ = b

	b.ReadConfig(func(conf *config.Config) {
		c.Assert(
			conf.Servers[serverId].GetNick(),
			Equals,
			fakeConfig.Servers[serverId].GetNick(),
		)
	})
}

func (s *s) TestBot_WriteConfig(c *C) {
	c.SucceedNow()
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider)
	c.Assert(err, IsNil)
	_ = b

	b.WriteConfig(func(conf *config.Config) {
		c.Assert(
			conf.Servers[serverId].GetNick(),
			Equals,
			fakeConfig.Servers[serverId].GetNick(),
		)
	})
}

func (s *s) TestBot_ReplaceConfig(c *C) {
	c.Skip("Not implemented")
}
