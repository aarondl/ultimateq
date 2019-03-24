package bot

import (
	"bytes"
	"crypto/x509"
	"io"
	"net"
	"testing"

	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/mocks"
)

func TestServer_createIrcClient(t *testing.T) {
	t.Parallel()
	errch := make(chan error)
	connProvider := func(srv string) (net.Conn, error) {
		return nil, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	go func() {
		err, _ := srv.createIrcClient()
		errch <- err
	}()

	if <-errch != nil {
		t.Error("Expected a clean connect.")
	}
	if srv.client == nil {
		t.Error("Client should have been instantiated.")
	}
}

func TestServer_createIrcClient_failConn(t *testing.T) {
	t.Parallel()
	errch := make(chan error)
	connProvider := func(srv string) (net.Conn, error) {
		return nil, io.EOF
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	go func() {
		err, _ := srv.createIrcClient()
		errch <- err
	}()

	if <-errch != io.EOF {
		t.Error("Expected a failed connection.")
	}
}

func TestServer_createIrcClient_killConn(t *testing.T) {
	t.Parallel()
	errch := make(chan error)
	connCh := make(chan int)
	connProvider := func(srv string) (net.Conn, error) {
		<-connCh
		return nil, io.EOF
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]
	srv.killable = make(chan int)

	go func() {
		err, _ := srv.createIrcClient()
		errch <- err
	}()

	close(srv.killable)
	if <-errch != errServerKilledConn {
		t.Error("Expected a killed connection.")
	}

	close(connCh)
}

func TestServer_createTlsConfig(t *testing.T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, devNull, false, false)
	srv := b.servers[netID]

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

func TestServer_Close(t *testing.T) {
	t.Parallel()
	errch := make(chan error)
	conn := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}
	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	go func() {
		err, _ := srv.createIrcClient()
		errch <- err
	}()

	if err := <-errch; err != nil {
		t.Error("Unexpected:", err)
	}

	if err := srv.Close(); err != nil {
		t.Error("Unexpected:", err)
	}

	if srv.client != nil {
		t.Error("Expected client to be nil.")
	}
}

func TestServer_Status(t *testing.T) {
	t.Parallel()
	srv := &Server{}

	status := make(chan Status)
	connAndStop := make(chan Status)
	srv.addStatusListener(connAndStop, STATUS_CONNECTING, STATUS_STOPPED)
	srv.addStatusListener(status)

	done := make(chan int)

	go func() {
		srv.setStatus(STATUS_CONNECTING)
		srv.setStatus(STATUS_STARTED)
		srv.setStatus(STATUS_STOPPED)
	}()

	go func() {
		ers := 0
		if st := <-status; st != STATUS_CONNECTING {
			t.Error("Received the wrong state:", st)
			ers++
		}
		if st := <-status; st != STATUS_STARTED {
			t.Error("Received the wrong state:", st)
			ers++
		}
		if st := <-status; st != STATUS_STOPPED {
			t.Error("Received the wrong state:", st)
			ers++
		}
		done <- ers
	}()

	go func() {
		ers := 0
		if st := <-connAndStop; st != STATUS_CONNECTING {
			t.Error("Received the wrong state:", st)
			ers++
		}
		if st := <-connAndStop; st != STATUS_STOPPED {
			t.Error("Received the wrong state:", st)
			ers++
		}
		done <- ers
	}()

	if ers := <-done; ers > 0 {
		t.Error(ers, " errors encountered during run.")
	}
	if ers := <-done; ers > 0 {
		t.Error(ers, " errors encountered during run.")
	}
}

func TestServer_rehashNetworkInfo(t *testing.T) {
	t.Parallel()
	b, _ := createBot(fakeConfig, nil, nil, devNull, false, false)
	srv := b.servers[netID]

	srv.netInfo.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=@",
	}})
	err := srv.rehashNetworkInfo()
	if err != nil {
		t.Error("Unexpected:", err)
	}

	if srv.netInfo.Chantypes() != "@" {
		t.Error("Protocaps were not set by rehash.")
	}
}

func TestServer_Write(t *testing.T) {
	t.Parallel()
	conn := mocks.NewConn()
	connProvider := func(srv string) (net.Conn, error) {
		return conn, nil
	}

	b, _ := createBot(fakeConfig, connProvider, nil, devNull, false, false)
	srv := b.servers[netID]

	var err error
	_, err = srv.Write(nil)
	if err != nil {
		t.Error("Expected:", err)
	}
	_, err = srv.Write([]byte{1})
	if err != errNotConnected {
		t.Error("Expected:", errNotConnected, "got:", err)
	}

	listen := make(chan Status)
	srv.addStatusListener(listen, STATUS_STARTED)

	end := b.Start()

	for <-listen != STATUS_STARTED {
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
	for range end {
	}
}
