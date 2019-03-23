package bot

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"testing"
	"time"

	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"gopkg.in/inconshreveable/log15.v2"
)

type testHandler struct {
	callback func(irc.Writer, *irc.Event)
}

func (h testHandler) Handle(w irc.Writer, ev *irc.Event) {
	if h.callback != nil {
		h.callback(w, ev)
	}
}

type testCommand struct {
	callback func(string, irc.Writer, *cmd.Event) error
}

func (h testCommand) Cmd(cmd string,
	w irc.Writer, cdata *cmd.Event) error {

	if h.callback != nil {
		return h.callback(cmd, w, cdata)
	}
	return nil
}

var devNull = func() log15.Handler { return log15.DiscardHandler() }

type reconnErr struct{}

func (r reconnErr) Error() string   { return "reconnErr" }
func (r reconnErr) Timeout() bool   { return true }
func (r reconnErr) Temporary() bool { return true }

func init() {
	log.SetOutput(ioutil.Discard)
	log15.Root().SetHandler(log15.DiscardHandler())

	data.StoredUserPwdCost = 4 // See bcrypt.MinCost
}

var netID = "test"

var fakeConfig = config.New().FromString(`
nick = "nobody"
altnick = "nobody1"
username = "nobody"
realname = "ultimateq"
noreconnect = true
nostore = true
noverifycert = true
sslcert = "fakecert"
ssl = true
[networks.test]
	servers = ["irc.test.net"]
`)

//==================================
// Tests begin
//==================================
func TestBot_Create(t *testing.T) {
	t.Parallel()
	bot, err := New(fakeConfig)
	if bot == nil {
		t.Error("Bot should be created.")
	}
	if err != nil {
		t.Error(err)
	}

	log15.Root().SetHandler(log15.DiscardHandler())
	_, err = New(config.New())
	if err != errInvalidConfig {
		t.Error("Expected error:", errInvalidConfig, "got", err)
	}
}

func TestBot_CreateLogger(t *testing.T) {
	t.Parallel()

	loglvlCfg := fakeConfig.Clone().SetLogLevel("crit")
	bot, err := New(loglvlCfg)
	if bot == nil || err != nil {
		t.Error("Bot should be created.")
	}
}

func TestBot_Start(t *testing.T) {
	t.Parallel()
	connProvider := func(srv string) (net.Conn, error) {
		return nil, io.EOF
	}
	var err error
	conf := fakeConfig.Clone()
	conf.NewNetwork("otherserver").SetServers([]string{"o.com"})
	b, _ := createBot(conf, connProvider, nil, devNull, false, false)
	dead := 0
	for err = range b.Start() {
		if err != io.EOF {
			t.Error("Was expecting the error from connect.")
		}
		dead++
	}
	if dead != len(conf.Networks()) {
		t.Error("It should die once for each server.")
	}
}

func TestBot_StartStopNetwork(t *testing.T) {
	t.Parallel()
	conn1 := mocks.NewConn()
	conn2 := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		if srv == "other:6667" {
			return conn1, nil
		}
		conn2.ResetDeath()
		return conn2, nil
	}
	conf := fakeConfig.Clone()
	conf.NewNetwork("othersrv").SetServers([]string{"other:6667"})
	b, _ := createBot(conf, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	done := make(chan int)
	start := make(chan Status)
	stop := make(chan Status)
	srv.addStatusListener(start, STATUS_STARTED)
	srv.addStatusListener(stop, STATUS_STOPPED)

	go func() {
		for i := 0; i < 2; i++ {
			<-start
			if !b.StopNetwork(netID) {
				t.Error("There was a problem stopping the server.")
			}

			<-stop
			if !b.StartNetwork(netID) {
				t.Fatal("There was an error starting the server.")
			}
		}

		<-start
		go b.Stop()
		<-stop
		done <- 0
	}()

	for _ = range b.Start() {
	}

	<-done
}

func TestBot_Dispatching(t *testing.T) {
	t.Parallel()
	conn := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)

	result := make(chan *irc.Event)
	thandler := &testHandler{
		func(_ irc.Writer, ev *irc.Event) {
			result <- ev
		},
	}
	cresult := make(chan string)
	tcommand := &testCommand{
		func(command string, _ irc.Writer, _ *cmd.Event) error {
			cresult <- command
			return nil
		},
	}
	b.RegisterGlobal(irc.PRIVMSG, thandler)
	id, err := b.RegisterGlobalCmd(cmd.New(
		"a", "cmd", "b", tcommand, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error("Should have registered a command successfully.")
	}

	end := b.Start()

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

	for range end {
	}

	if !b.UnregisterCmd(id) {
		t.Error("Should have unregistered a command.")
	}
}

func TestBot_Dispatch_ConnectDisconnect(t *testing.T) {
	t.Parallel()
	conn := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)

	result := make(chan *irc.Event)
	thandler := &testHandler{
		func(w irc.Writer, ev *irc.Event) {
			result <- ev
		},
	}
	b.RegisterGlobal(irc.CONNECT, thandler)
	b.RegisterGlobal(irc.DISCONNECT, thandler)

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

func TestBot_Reconnect(t *testing.T) {
	t.Parallel()
	conn := mocks.NewConn()
	wantedConn := make(chan int)
	connProvider := func(srv string) (net.Conn, error) {
		<-wantedConn
		conn.ResetDeath()
		return conn, nil
	}

	conf := fakeConfig.Clone()
	conf.Network("").SetNoReconnect(false).SetReconnectTimeout(1)
	b, _ := createBot(conf, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]
	srv.reconnScale = time.Millisecond

	go func() {
		wantedConn <- 0

		conn.Send(nil, 0, io.EOF)
		wantedConn <- 0

		conn.Send(nil, 0, io.EOF)
		wantedConn <- 0

		b.Stop()
	}()

	for err := range b.Start() {
		if err != errServerKilled {
			t.Error("Expected it to die during running state.")
		}
	}
}

func TestBot_ReconnectConnection(t *testing.T) {
	t.Parallel()
	wantedConn := make(chan int)
	connProvider := func(srv string) (net.Conn, error) {
		<-wantedConn
		return nil, reconnErr{}
	}

	conf := fakeConfig.Clone()
	conf.Network("").SetNoReconnect(false).SetReconnectTimeout(1)
	b, _ := createBot(conf, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]
	srv.reconnScale = time.Millisecond

	listen := make(chan Status)
	srv.addStatusListener(listen, STATUS_CONNECTING)

	end := b.Start()
	<-listen
	wantedConn <- 0
	<-listen
	wantedConn <- 0
	<-listen
	wantedConn <- 0
	<-listen

	b.Stop()
	for err := range end {
		if err != errServerKilledConn {
			t.Error("Expected it to die during connecting:", err)
		}
	}

	close(wantedConn)
}

func TestBot_ReconnectKill(t *testing.T) {
	t.Parallel()
	connProvider := func(srv string) (net.Conn, error) {
		return nil, reconnErr{}
	}

	conf := fakeConfig.Clone()
	conf.Network("").SetNoReconnect(false).SetReconnectTimeout(3)
	b, _ := createBot(conf, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	listen := make(chan Status)
	srv.addStatusListener(listen, STATUS_RECONNECTING)

	end := b.Start()

	<-listen
	b.Stop()
	for err := range end {
		if err != errServerKilledReconn {
			t.Error("Expected it to die during connection:", err)
		}
	}
}

func TestBot_RegisterGlobal(t *testing.T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, devNull, false, false)
	gid := b.RegisterGlobal(irc.PRIVMSG, &coreHandler{})
	id := b.Register(netID, "", irc.PRIVMSG, &coreHandler{})

	if !b.Unregister(id) {
		t.Error("Should unregister the global registration.")
	}
	if !b.Unregister(gid) {
		t.Error("Should unregister the global registration.")
	}
}

func TestBot_RegisterGlobalCmd(t *testing.T) {
	var id uint64
	var err error
	var success bool
	b, _ := createBot(fakeConfig, nil, nil, devNull, false, false)
	command := "cmd"
	id, err = b.RegisterGlobalCmd(cmd.New("ext", command, "desc", &testCommand{},
		cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	_, err = b.RegisterGlobalCmd(cmd.New("ext", command, "desc", &testCommand{},
		cmd.AnyKind, cmd.AnyScope))
	if err == nil {
		t.Error("Expecting error about duplicates.")
	}
	if success = b.UnregisterCmd(id); !success {
		t.Error("It should unregister correctly.")
	}

	id, err = b.RegisterCmd(netID, channel, cmd.New("e", command, "d",
		&testCommand{}, cmd.AnyKind, cmd.AnyScope))
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	success = b.UnregisterCmd(id)
	if !success {
		t.Error("It should unregister correctly.")
	}
}

func TestBot_Providers(t *testing.T) {
	t.Parallel()
	storeConf1 := fakeConfig.Clone()
	storeConf2 := storeConf1.Clone()
	storeConf3 := storeConf1.Clone()
	storeConf1.Network("").SetNoStore(false)
	storeConf2.Network(netID).SetNoStore(false)
	storeConf3.Network(netID).SetNoStore(true)

	badConnProv := func(s string) (net.Conn, error) {
		return nil, net.ErrWriteToConnected
	}
	badStoreProv := func(s string) (*data.Store, error) {
		return nil, io.EOF
	}

	b, err := createBot(fakeConfig, badConnProv, nil, devNull, false, false)
	if err = <-b.Start(); err != net.ErrWriteToConnected {
		t.Error("Expected:", net.ErrWriteToConnected, "got:", err)
	}

	b, err = createBot(fakeConfig, nil, badStoreProv, devNull, false, false)
	if err != nil {
		t.Error("Expected no errors.")
	}
	b, err = createBot(storeConf1, nil, badStoreProv, devNull, false, false)
	if err != io.EOF {
		t.Error("Expected an error creating the store.")
	}
	b, err = createBot(storeConf2, nil, badStoreProv, devNull, false, false)
	if err != io.EOF {
		t.Error("Expected an error creating the store.")
	}
	b, err = createBot(storeConf3, nil, badStoreProv, devNull, false, false)
	if err != nil {
		t.Error("Expected no errors.")
	}
}

func TestBot_Store(t *testing.T) {
	t.Parallel()
	conf := fakeConfig.Clone()
	conf.Network("").SetNoStore(false)
	goodStoreProv := func(s string) (*data.Store, error) {
		return data.NewStore(data.MemStoreProvider)
	}
	b, err := createBot(conf, nil, goodStoreProv, devNull, false, false)
	if err != nil {
		t.Error("Expected no errors.")
	}
	if b.store == nil {
		t.Error("Store should not be nil.")
	}
	b.Close()
	b.Close() // Nothing bad should happen
}

func TestBot_Stop(t *testing.T) {
	t.Parallel()
	conn := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	listen := make(chan Status)
	srv.addStatusListener(listen, STATUS_STARTED)

	end := b.Start()

	<-listen

	b.Stop()
	for _ = range end {
	}
}

func TestBot_Locker(t *testing.T) {
	t.Parallel()

	goodStoreProv := func(s string) (*data.Store, error) {
		return data.NewStore(data.MemStoreProvider)
	}

	conf := fakeConfig.Clone()
	conf.Network("").SetNoStore(false)
	b, err := createBot(conf, nil, goodStoreProv, devNull, false, false)

	if err != nil {
		t.Error("Unexpected err:", err)
	}
	var provider data.Provider = b // Check conformity

	state := provider.State(netID)
	if state == nil {
		t.Error("State should not be nil.")
	}
	store := provider.Store()
	if store == nil {
		t.Error("Store should not be nil.")
	}
}

func TestBot_GetEndpoint(t *testing.T) {
	t.Parallel()
	conn := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	listen := make(chan Status)
	srv.addStatusListener(listen, STATUS_STARTED)

	end := b.Start()

	ep := b.NetworkWriter(netID)

	test := "test\r\n"
	result := make(chan string)
	go func() {
		result <- string(conn.Receive(len(test), io.EOF))
	}()

	<-listen

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
