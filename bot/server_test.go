package bot

import (
	"bytes"
	"crypto/x509"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
	"io"
	"net"
	"runtime"
	. "testing"
	"time"
)

func TestServer_createIrcClient(t *T) {
	t.Parallel()
	errch := make(chan error)
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, false, false)
	srv := b.servers[serverID]

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

func TestServer_createIrcClient_failConn(t *T) {
	t.Parallel()
	errch := make(chan error)
	connProvider := func(srv string) (net.Conn, error) {
		return nil, io.EOF
	}
	b, _ := createBot(fakeConfig, connProvider, nil, false, false)
	srv := b.servers[serverID]

	go func() {
		errch <- srv.createIrcClient()
	}()

	if <-errch != io.EOF {
		t.Error("Expected a failed connection.")
	}
}

func TestServer_createIrcClient_killConn(t *T) {
	t.Parallel()
	errch := make(chan error)
	connProvider := func(srv string) (net.Conn, error) {
		time.Sleep(time.Second * 5)
		return nil, io.EOF
	}
	b, _ := createBot(fakeConfig, connProvider, nil, false, false)
	srv := b.servers[serverID]

	go func() {
		errch <- srv.createIrcClient()
	}()

	srv.kill <- 0
	if <-errch != errKilledConn {
		t.Error("Expected a killed connection.")
	}
}

func TestServer_createTlsConfig(t *T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, false, false)
	srv := b.servers[serverID]

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

func TestServerSender(t *T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, false, false)
	ep := b.GetEndpoint(serverID)
	if ep.GetKey() != serverID {
		t.Error("Expected the key to represent the server.")
	}
	if b.servers[serverID].endpoint.server != b.servers[serverID] {
		t.Error("Endpoints are being constructed with wrong servers.")
	}
}

func TestServer_Status(t *T) {
	t.Parallel()
	srv := &Server{}

	var tests = []struct {
		Value, Lock, Expect bool
	}{
		{true, false, true},
		{false, false, false},
		{true, true, true},
		{false, true, false},
	}

	var funcs = []func(int){
		func(i int) {
			test := tests[i]
			srv.setConnecting(test.Value, test.Lock)
			if was := srv.IsConnecting(); was != test.Expect {
				t.Error("In Test:", test)
				t.Error("Expected:", test.Expect, "was:", was)
			}
		},
		func(i int) {
			test := tests[i]
			srv.setConnected(test.Value, test.Lock)
			if was := srv.IsConnected(); was != test.Expect {
				t.Error("In Test:", test)
				t.Error("Expected:", test.Expect, "was:", was)
			}
		},
		func(i int) {
			test := tests[i]
			srv.setStarted(test.Value, test.Lock)
			if was := srv.IsStarted(); was != test.Expect {
				t.Error("In Test:", test)
				t.Error("Expected:", test.Expect, "was:", was)
			}
		},
		func(i int) {
			test := tests[i]
			srv.setReconnecting(test.Value, test.Lock)
			if was := srv.IsReconnecting(); was != test.Expect {
				t.Error("In Test:", test)
				t.Error("Expected:", test.Expect, "was:", was)
			}
		},
	}

	for _, fn := range funcs {
		for j := range tests {
			fn(j)
		}
	}
}

func TestServer_StatusMultiple(t *T) {
	srv := &Server{}
	srv.status = 0
	if !srv.IsStopped() {
		t.Error("If there are no flags set, isstopped should be true.")
	}
	srv.status = ^byte(0)
	if srv.IsStopped() {
		t.Error("If there is any flags set, isstopped should be false.")
	}
	if !srv.IsConnecting() {
		t.Error("All flags should all be able to be set together.")
	}
	if !srv.IsConnected() {
		t.Error("All flags should all be able to be set together.")
	}
	if !srv.IsStarted() {
		t.Error("All flags should all be able to be set together.")
	}
	if !srv.IsReconnecting() {
		t.Error("All flags should all be able to be set together.")
	}
}

func TestServer_rehashProtocaps(t *T) {
	b, _ := createBot(fakeConfig, nil, nil, false, false)
	srv := b.servers[serverID]

	srv.caps.ParseISupport(&irc.Message{Args: []string{
		"NICK", "CHANTYPES=@",
	}})
	err := srv.rehashProtocaps()
	if err != nil {
		t.Error("Unexpected:", err)
	}

	if srv.caps.Chantypes() != "@" {
		t.Error("Protocaps were not set by rehash.")
	}
	if b.caps.Chantypes() != "#&~@" {
		t.Error("Protocaps were not merged correctly.")
	}
}

func TestServer_Write(t *T) {
	conn := mocks.CreateConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, _ := createBot(fakeConfig, connProvider, nil, false, false)
	srv := b.servers[serverID]

	var err error
	_, err = srv.Write(nil)
	if err != nil {
		t.Error("Expected:", err)
	}
	_, err = srv.Write([]byte{1})
	if err != errNotConnected {
		t.Error("Expected:", errNotConnected, "got:", err)
	}

	end := b.Start()

	for !srv.IsConnected() {
		runtime.Gosched()
	}

	message := []byte("PONG :msg\r\n")
	if _, err = srv.Write(message); err != nil {
		t.Error("Unexpected write error:", err)
	}
	got := conn.Receive(len(message), nil)
	if bytes.Compare(got, message) != 0 {
		t.Errorf("Socket received wrong message: (%s) != (%s)", got, message)
	}

	b.Stop()
	for _ = range end {
	}
}
