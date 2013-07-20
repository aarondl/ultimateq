package bot

import (
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"io"
	"launchpad.net/gocheck"
	"log"
	"net"
	"os"
	"regexp/syntax"
	"runtime"
	. "testing"
	"time"
)

func Test(t *T) { gocheck.TestingT(t) } //Hook into testing package
type s struct{}

var _ = gocheck.Suite(&s{})

type testHandler struct {
	callback func(*irc.Message, irc.Endpoint)
}

func (h testHandler) HandleRaw(m *irc.Message, send irc.Endpoint) {
	if h.callback != nil {
		h.callback(m, send)
	}
}

type testCommand struct {
	callback func(string, *irc.Message,
		*data.DataEndpoint, *commander.CommandData) error
}

func (h testCommand) Command(cmd string, msg *irc.Message,
	ep *data.DataEndpoint, cdata *commander.CommandData) error {

	if h.callback != nil {
		return h.callback(cmd, msg, ep, cdata)
	}
	return nil
}

func init() {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Println("Could not set logger:", err)
	} else {
		log.SetOutput(f)
	}

	data.UserAccessPwdCost = 4 // See bcrypt.MinCost
}

var serverID = "irc.test.net"

var fakeConfig = Configure().
	Nick("nobody").
	Altnick("nobody1").
	Username("nobody").
	Userhost("bitforge.ca").
	Realname("ultimateq").
	NoReconnect(true).
	NoStore(true).
	NoVerifyCert(true).
	SslCert("fakecert").
	Ssl(true).
	Server(serverID)

//==================================
// Tests begin
//==================================
func TestBot_Create(t *T) {
	t.Parallel()
	bot, err := CreateBot(fakeConfig)
	if bot == nil {
		t.Error("Bot should be created.")
	}
	if err != nil {
		t.Error(err)
	}
	_, err = CreateBot(Configure())
	if err != errInvalidConfig {
		t.Error("Expected error:", errInvalidConfig, "got", err)
	}

	_, err = CreateBot(ConfigureFunction(
		func(conf *config.Config) *config.Config {
			return fakeConfig
		}),
	)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
}

func TestBot_Start(t *T) {
	t.Parallel()
	connProvider := func(srv string) (net.Conn, error) {
		return nil, io.EOF
	}
	var err error
	conf := fakeConfig.Clone()
	conf.Server("otherserver")
	b, _ := createBot(conf, nil, connProvider, nil, false, false)
	dead := 0
	for err = range b.Start() {
		if err != io.EOF {
			t.Error("Was expecting the error from connect.")
		}
		dead++
	}
	if dead != len(conf.Servers) {
		t.Error("It should die once for each server.")
	}
}

func TestBot_Dispatching(t *T) {
	t.Parallel()
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, nil, connProvider, nil, false, false)
	srv := b.servers[serverID]

	result := make(chan *irc.Message)
	thandler := &testHandler{
		func(m *irc.Message, ep irc.Endpoint) {
			result <- m
		},
	}
	cresult := make(chan string)
	tcommand := &testCommand{
		func(cmd string, _ *irc.Message, _ *data.DataEndpoint,
			_ *commander.CommandData) error {

			cresult <- cmd
			return nil
		},
	}
	b.Register(irc.PRIVMSG, thandler)
	if err := b.RegisterCommand(commander.MkCmd(
		"a", "b", "cmd", tcommand, commander.ALL, commander.ALL)); err != nil {
		t.Error("Should have registered a command successfully.")
	}

	end := b.Start()
	for !srv.IsConnected() {
		runtime.Gosched()
	}
	for !srv.IsStarted() {
		runtime.Gosched()
	}

	testMsg := "cmd"
	msg := []byte("PRIVMSG bot :" + testMsg + "\r\n")
	go func() {
		// First send should simply log.
		conn.Send([]byte(testMsg+"\r\n"), len(testMsg)+2, nil)
		conn.Send(msg, len(msg), io.EOF)
	}()

	if d := <-result; d == nil || d.Message() != testMsg {
		t.Error("Expected:", string(msg), "got:", d)
	}
	if c := <-cresult; c != testMsg {
		t.Error("Expected:", testMsg, "got:", c)
	}

	for _ = range end {
	}

	if !b.UnregisterCommand("cmd") {
		t.Error("Should have unregistered a command.")
	}
}

func TestBot_Dispatch_ConnectDisconnect(t *T) {
	t.Parallel()
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, nil, connProvider, nil, false, false)

	result := make(chan *irc.Message)
	thandler := &testHandler{
		func(m *irc.Message, ep irc.Endpoint) {
			result <- m
		},
	}
	b.Register(irc.CONNECT, thandler)
	b.Register(irc.DISCONNECT, thandler)

	end := b.Start()

	go func() {
		conn.Send(nil, 0, io.EOF)
	}()

	if d := <-result; d == nil || d.Name != irc.CONNECT {
		t.Error("Expected a dispatch of connect:", d)
	}
	if d := <-result; d == nil || d.Name != irc.DISCONNECT {
		t.Error("Expected a dispatch of connect:", d)
	}

	for _ = range end {
	}
}

func TestBot_Reconnect(t *T) {
	t.Parallel()
	conn := mocks.CreateConn()
	wantedConn := make(chan int)
	connProvider := func(srv string) (net.Conn, error) {
		<-wantedConn
		return conn, nil
	}

	conf := fakeConfig.Clone().GlobalContext().NoReconnect(false).
		ReconnectTimeout(1)
	b, _ := createBot(conf, nil, connProvider, nil, false, false)
	srv := b.servers[serverID]
	srv.reconnScale = time.Millisecond

	end := b.Start()
	wantedConn <- 0

	conn.Send(nil, 0, io.EOF)
	conn.ResetDeath()
	wantedConn <- 0

	conn.Send(nil, 0, io.EOF)
	conn.ResetDeath()
	wantedConn <- 0

	for !srv.IsConnecting() {
		runtime.Gosched()
	}
	srv.kill <- 0
	for err := range end {
		if err != errServerKilled {
			t.Error("Expected it to die during connection.")
		}
	}
}

func TestBot_ReconnectConnection(t *T) {
	t.Parallel()
	wantedConn := make(chan int)
	connProvider := func(srv string) (net.Conn, error) {
		<-wantedConn
		return nil, io.EOF
	}

	conf := fakeConfig.Clone().GlobalContext().NoReconnect(false).
		ReconnectTimeout(1)
	b, _ := createBot(conf, nil, connProvider, nil, false, false)
	srv := b.servers[serverID]
	srv.reconnScale = time.Millisecond

	end := b.Start()
	wantedConn <- 0
	wantedConn <- 0
	wantedConn <- 0

	for !srv.IsConnecting() {
		runtime.Gosched()
	}
	srv.kill <- 0
	for err := range end {
		if err != errServerKilledReconn {
			t.Error("Expected it to die during reconnection.")
		}
	}
}

func TestBot_ReconnectKill(t *T) {
	t.Parallel()
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	conf := fakeConfig.Clone().GlobalContext().NoReconnect(false).
		ReconnectTimeout(1)
	b, _ := createBot(conf, nil, connProvider, nil, false, false)
	srv := b.servers[serverID]

	result := make(chan *irc.Message)
	thandler := &testHandler{
		func(m *irc.Message, ep irc.Endpoint) {
			result <- m
		},
	}
	b.Register(irc.CONNECT, thandler)

	end := b.Start()

	if d := <-result; d == nil || d.Name != irc.CONNECT {
		t.Error("Expected a dispatch of connect:", d)
	}
	conn.Send(nil, 0, io.EOF)
	for !srv.IsReconnecting() {
		runtime.Gosched()
	}
	srv.kill <- 0
	for err := range end {
		if err != errServerKilledReconn {
			t.Error("Expected it to die during reconnection:", err)
		}
	}
}

func TestBot_Register(t *T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, nil, false, false)
	gid := b.Register(irc.PRIVMSG, &coreHandler{})
	id, err := b.RegisterServer(serverID, irc.PRIVMSG, &coreHandler{})

	if b.Unregister(irc.PRIVMSG, id) {
		t.Error("Unregister should not know about server events.")
	}
	if !b.Unregister(irc.PRIVMSG, gid) {
		t.Error("Should unregister the global registration.")
	}

	if ok, _ := b.UnregisterServer(serverID, irc.PRIVMSG, gid); ok {
		t.Error("Unregister server should not know about global events.")
	}
	if ok, _ := b.UnregisterServer(serverID, irc.PRIVMSG, id); !ok {
		t.Error("Unregister should unregister events.")
	}

	_, err = b.RegisterServer("", "", &coreHandler{})
	if err != errUnknownServerID {
		t.Error("Expecting:", errUnknownServerID, "got:", err)
	}
	_, err = b.UnregisterServer("", "", 0)
	if err != errUnknownServerID {
		t.Error("Expecting:", errUnknownServerID, "got:", err)
	}
}

func TestBot_RegisterCommand(t *T) {
	// t.Parallel() Cannot be parallel due to the nature of command registration
	var err error
	var success bool
	b, _ := createBot(fakeConfig, nil, nil, nil, false, false)
	cmd := "cmd"
	err = b.RegisterCommand(commander.MkCmd("ext", "desc", cmd, &testCommand{},
		commander.ALL, commander.ALL))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	err = b.RegisterCommand(commander.MkCmd("ext", "desc", cmd, &testCommand{},
		commander.ALL, commander.ALL))
	if err == nil {
		t.Error("Expecting error about duplicates.")
	}
	if success = b.UnregisterCommand(cmd); !success {
		t.Error("It should unregister correctly.")
	}

	err = b.RegisterServerCommand(serverID, commander.MkCmd("e", "d", cmd,
		&testCommand{}, commander.ALL, commander.ALL))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	if success = b.UnregisterServerCommand(serverID, cmd); !success {
		t.Error("It should unregister correctly.")
	}

	err = b.RegisterServerCommand("badServer", commander.MkCmd("e", "d", cmd,
		&testCommand{}, commander.ALL, commander.ALL))
	if err != errUnknownServerID {
		t.Error("Expecting:", errUnknownServerID, "got:", err)
	}

	if success = b.UnregisterServerCommand("badServer", cmd); success {
		t.Error("It should not unregister from non existent servers.")
	}
}

func TestBot_Providers(t *T) {
	t.Parallel()
	storeConf1 := fakeConfig.Clone().GlobalContext().NoStore(false)
	storeConf2 := storeConf1.Clone().ServerContext(serverID).NoStore(false)
	storeConf3 := storeConf1.Clone().ServerContext(serverID).NoStore(true)

	capsProv := func() *irc.ProtoCaps {
		p := irc.CreateProtoCaps()
		p.ParseISupport(&irc.Message{Args: []string{"nick", "CHANTYPES=H"}})
		return p
	}
	badConnProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}
	goodConnProv := func(s string) (net.Conn, error) {
		return mocks.CreateConn(), nil
	}
	badStoreProv := func(s string) (*data.Store, error) {
		return nil, io.EOF
	}

	b, err := createBot(
		fakeConfig, capsProv, goodConnProv, badStoreProv, false, false)
	if _, ok := err.(*syntax.Error); !ok {
		t.Error("The error was not a syntax error:", err)
	}

	b, err = createBot(fakeConfig, nil, badConnProv, badStoreProv, false, false)
	if err = <-b.Start(); err != net.ErrWriteToConnected {
		t.Error("Expected:", net.ErrWriteToConnected, "got:", err)
	}

	b, err = createBot(fakeConfig, nil, nil, badStoreProv, false, false)
	if err != nil {
		t.Error("Expected no errors.")
	}
	b, err = createBot(storeConf1, nil, nil, badStoreProv, false, false)
	if err != io.EOF {
		t.Error("Expected an error creating the store.")
	}
	b, err = createBot(storeConf2, nil, nil, badStoreProv, false, false)
	if err != io.EOF {
		t.Error("Expected an error creating the store.")
	}
	b, err = createBot(storeConf3, nil, nil, badStoreProv, false, false)
	if err != nil {
		t.Error("Expected no errors.")
	}
}

func TestBot_Store(t *T) {
	t.Parallel()
	conf := fakeConfig.Clone().GlobalContext().NoStore(false)
	goodStoreProv := func(s string) (*data.Store, error) {
		return data.CreateStore(data.MemStoreProvider)
	}
	b, err := createBot(conf, nil, nil, goodStoreProv, false, false)
	if err != nil {
		t.Error("Expected no errors.")
	}
	if b.store == nil {
		t.Error("Store should not be nil.")
	}
	b.Close()
	b.Close() // Nothing bad should happen
}

func TestBot_Stop(t *T) {
	t.Parallel()
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, nil, connProvider, nil, false, false)
	srv := b.servers[serverID]

	end := b.Start()

	for !srv.IsStarted() {
		runtime.Gosched()
	}

	b.Stop()
	for _ = range end {
	}
}

func TestBot_GetEndpoint(t *T) {
	t.Parallel()
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, nil, connProvider, nil, false, false)
	srv := b.servers[serverID]

	end := b.Start()

	ep := b.GetEndpoint(serverID)

	test := "test\r\n"
	result := make(chan string)
	go func() {
		result <- string(conn.Receive(len(test), io.EOF))
	}()

	for !srv.IsConnected() {
		runtime.Gosched()
	}

	if err := ep.Send(test); err != nil {
		t.Fatal("Unexpected error:", err)
	}

	if res := <-result; res != test {
		t.Error("Expected:", test, "got:", res)
	}

	b.Stop()
	for _ = range end {
	}
}

/*
func (s *s) TestBot_createBot(c *C) {
	capsProvider := func() *irc.ProtoCaps {
		return irc.CreateProtoCaps()
	}
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, err := createBot(fakeConfig, capsProvider, connProvider,
		nil, false, false)
	c.Check(b, NotNil)
	c.Check(err, IsNil)
	c.Check(len(b.servers), Equals, 1)
	c.Check(b.caps, NotNil)
	c.Check(b.capsProvider, NotNil)
	c.Check(b.connProvider, NotNil)
}

func (s *s) TestBot_createServer(c *C) {
	b, err := createBot(fakeConfig, nil, nil, nil, true, false)
	c.Check(err, IsNil)
	srv := b.servers[serverID]
	c.Check(srv.dispatcher, NotNil)
	c.Check(srv.commander, NotNil)
	c.Check(srv.dispatchCore, NotNil)
	c.Check(srv.state, NotNil)
	c.Check(srv.handler, NotNil)

	cnf := fakeConfig.Clone()
	cnf.GlobalContext().NoState(true)
	b, err = createBot(cnf, nil, nil, nil, false, false)
	c.Check(err, IsNil)
	srv = b.servers[serverID]
	c.Check(srv.dispatcher, NotNil)
	c.Check(srv.state, IsNil)
	c.Check(srv.handler, IsNil)
}

func (s *s) TestBot_createIrcClient(c *C) {
	connProv := func(server string) (net.Conn, error) {
		return nil, errFailedToLoadCertificate
	}
	b, err := createBot(fakeConfig, nil, connProv, nil, false, false)
	c.Check(err, IsNil)
	ers := b.Connect()
	c.Check(ers[0], Equals, errFailedToLoadCertificate)
}

func (s *s) TestBot_createDispatcher(c *C) {
	_, err := createBot(fakeConfig, func() *irc.ProtoCaps {
		return nil
	}, nil, nil, false, false)
	c.Check(err, NotNil)
}*/
