package mocks

import (
	"net"
	"sync"
	"time"
)

const (
	panicMsg = "This function is not properly mocked."
)

type IOReturn struct {
	n   int
	err error
}

// Mock of Conn interface
type Conn struct {
	writechan   chan []byte
	writereturn chan IOReturn
	readchan    chan []byte
	readreturn  chan IOReturn
	deathWaiter sync.WaitGroup
}

func CreateConn() (conn *Conn) {
	conn = &Conn{
		writechan:   make(chan []byte),
		writereturn: make(chan IOReturn),
		readchan:    make(chan []byte),
		readreturn:  make(chan IOReturn),
	}

	conn.deathWaiter.Add(1)
	return
}

func (m *Conn) Receive(n int, err error) []byte {
	read := <-m.writechan
	m.writereturn <- IOReturn{n, err}
	return read
}

func (m *Conn) Write(written []byte) (int, error) {
	m.writechan <- written
	ret := <-m.writereturn
	return ret.n, ret.err
}

func (m *Conn) Send(buffer []byte, n int, err error) {
	m.readchan <- buffer
	m.readreturn <- IOReturn{n, err}
}

func (m *Conn) Read(buffer []byte) (int, error) {
	read := <-m.readchan
	copy(buffer, read)
	ret := <-m.readreturn
	return ret.n, ret.err
}

func (m *Conn) ResetDeath() {
	m.deathWaiter.Add(1)
}

func (m *Conn) WaitForDeath() {
	m.deathWaiter.Wait()
}

func (m *Conn) Close() error {
	m.deathWaiter.Done()
	return nil
}

func (m *Conn) LocalAddr() net.Addr {
	panic(panicMsg)
	return nil
}

func (m *Conn) RemoteAddr() net.Addr {
	panic(panicMsg)
	return nil
}

func (m *Conn) SetDeadline(_param0 time.Time) error {
	panic(panicMsg)
	return nil
}

func (m *Conn) SetReadDeadline(_param0 time.Time) error {
	panic(panicMsg)
	return nil
}

func (m *Conn) SetWriteDeadline(_param0 time.Time) error {
	panic(panicMsg)
	return nil
}
