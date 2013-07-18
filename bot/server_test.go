package bot

import (
	//"bytes"
	//"github.com/aarondl/ultimateq/data"
	//"github.com/aarondl/ultimateq/irc"
	//"github.com/aarondl/ultimateq/mocks"
	//. "launchpad.net/gocheck"
	"io"
	"crypto/x509"
	. "testing"
	"time"
	"net"
)

func TestServer_createIrcClient(t *T) {
	t.Parallel()
	doSleep := false
	errch := make(chan error)
	var retErr error

	connProvider := func(srv string) (net.Conn, error) {
		if doSleep {
			time.Sleep(time.Second * 5)
		}
		return nil, retErr
	}
	b, _ := createBot(fakeConfig, nil, connProvider, nil, false, false)
	srv := b.servers[serverId]

	doSleep = true
	go func() {
		errch <- srv.createIrcClient()
	}()

	srv.kill <- 0
	if <-errch != errKilledConn {
		t.Error("Expected a killed connection.")
	}

	doSleep = false
	retErr = io.EOF
	go func() {
		errch <- srv.createIrcClient()
	}()

	if <-errch != io.EOF {
		t.Error("Expected a failed connection.")
	}

	doSleep = false
	retErr = nil
	go func() {
		errch <- srv.createIrcClient()
	}()

	if <-errch != nil {
		t.Error("Expected a clean connect.")
	}
	if srv.client == nil {
		t.Error("Client should have been instantiated.")
	}
}

func TestServer_createTlsConfig(t *T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, nil, false, false)
	srv := b.servers[serverId]

	pool := x509.NewCertPool()
	tlsConfig, _ := srv.createTlsConfig(func(_ string) (*x509.CertPool, error) {
		return pool, nil
	})

	if !tlsConfig.InsecureSkipVerify {
		t.Error("This should have been set to fakeconfig's value.")
	}
	if tlsConfig.RootCAs != pool {
		t.Error("The provided root ca pool should be used.")
	}
}

/*func (s *s) TestServerSender(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	c.Check(srv.endpoint.GetKey(), Equals, serverId)
}

func (s *s) TestServerSender_UsingState(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]

	c.Check(srv.endpoint.GetKey(), Equals, serverId)
	called := false
	reportCalled := false
	reportCalled = srv.endpoint.UsingState(func(*data.State) {
		called = true
	})
	c.Check(called, Equals, true)
	c.Check(reportCalled, Equals, true)

	srv.state = nil
	srv.createServerEndpoint(nil, nil)
	called = false
	reportCalled = false
	reportCalled = srv.endpoint.UsingState(func(*data.State) {
		called = true
	})
	c.Check(called, Equals, false)
	c.Check(reportCalled, Equals, false)
}

func (s *s) TestServerSender_UsingStore(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	store, err := data.CreateStore(data.MemStoreProvider)
	srv.createServerEndpoint(store, &b.protectStore)
	c.Check(err, IsNil)

	called := false
	reportCalled := false
	reportCalled = srv.endpoint.UsingStore(func(*data.Store) {
		called = true
	})
	c.Check(called, Equals, true)
	c.Check(reportCalled, Equals, true)

	srv.createServerEndpoint(nil, &b.protectStore)
	called = false
	reportCalled = false
	reportCalled = srv.endpoint.UsingStore(func(*data.Store) {
		called = true
	})
	c.Check(called, Equals, false)
	c.Check(reportCalled, Equals, false)
}

func (s *s) TestServerSender_OpenState(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	srv.createServerEndpoint(nil, nil)

	c.Check(srv.endpoint.OpenState(), Equals, srv.state)
	srv.endpoint.CloseState()

	srv.protectState.Lock()
	srv.protectState.Unlock()
	c.Succeed()
}

func (s *s) TestServerSender_OpenStore(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]
	store, err := data.CreateStore(data.MemStoreProvider)
	c.Check(err, IsNil)
	srv.createServerEndpoint(store, &b.protectStore)

	c.Check(srv.endpoint.OpenStore(), Equals, b.store)
	srv.endpoint.CloseStore()

	b.protectStore.Lock()
	b.protectStore.Unlock()
	c.Succeed()
}

func (s *s) TestServer_Write(c *C) {
	str := "PONG :msg\r\n"

	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, nil, connProvider, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]

	_, err = srv.Write([]byte{})
	c.Check(err, Equals, errNotConnected)

	ers := b.Connect()
	c.Check(len(ers), Equals, 0)
	b.start(true, false)

	err = srv.Writeln(str)
	c.Check(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Check(err, IsNil)
	_, err = srv.Write([]byte(str))
	c.Check(bytes.Compare(conn.Receive(len(str), nil), []byte(str)), Equals, 0)
	c.Check(err, IsNil)
	err = b.Writeln("notrealserver", str)
	c.Check(err, NotNil)
	b.WaitForHalt()
	b.Disconnect()
}

func (s *s) TestServer_rehashProtocaps(c *C) {
	originalCaps := irc.CreateProtoCaps()
	originalCaps.ParseISupport(&irc.Message{Args: []string{
		"NICK", "CHANTYPES=!",
	}})
	capsProv := func() *irc.ProtoCaps {
		return originalCaps
	}

	b, err := createBot(fakeConfig, capsProv, nil, nil, false, false)
	c.Check(err, IsNil)
	srv := b.servers[serverId]

	c.Check(srv.caps.Chantypes(), Equals, "!")

	srv.caps.ParseISupport(&irc.Message{Args: []string{
		"NICK", "CHANTYPES=#",
	}})
	err = srv.rehashProtocaps()
	c.Check(err, IsNil)

	c.Check(b.caps.Chantypes(), Equals, "!#")
}

func (s *s) TestServer_State(c *C) {
	srv := &Server{}

	srv.setStarted(true, false)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setStarted(false, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setStarted(true, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setStarted(false, true)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setReading(true, false)
	c.Check(srv.IsReading(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setReading(false, false)
	c.Check(srv.IsReading(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setReading(true, true)
	c.Check(srv.IsReading(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setReading(false, true)
	c.Check(srv.IsReading(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setWriting(true, false)
	c.Check(srv.IsWriting(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setWriting(false, false)
	c.Check(srv.IsWriting(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setWriting(true, true)
	c.Check(srv.IsWriting(), Equals, true)
	c.Check(srv.IsStarted(), Equals, true)
	srv.setWriting(false, true)
	c.Check(srv.IsWriting(), Equals, false)
	c.Check(srv.IsStarted(), Equals, false)

	srv.setConnected(true, false)
	c.Check(srv.IsConnected(), Equals, true)
	srv.setConnected(false, false)
	c.Check(srv.IsConnected(), Equals, false)

	srv.setConnected(true, true)
	c.Check(srv.IsConnected(), Equals, true)
	srv.setConnected(false, true)
	c.Check(srv.IsConnected(), Equals, false)

	srv.setReconnecting(true, false)
	c.Check(srv.IsReconnecting(), Equals, true)
	srv.setReconnecting(false, false)
	c.Check(srv.IsReconnecting(), Equals, false)

	srv.setReconnecting(true, true)
	c.Check(srv.IsReconnecting(), Equals, true)
	srv.setReconnecting(false, true)
	c.Check(srv.IsReconnecting(), Equals, false)
}*/
