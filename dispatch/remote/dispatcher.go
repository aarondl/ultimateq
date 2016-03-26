package remote

import (
	"errors"
	"io"
	"sync"

	"github.com/aarondl/ultimateq/registrar"
	"gopkg.in/inconshreveable/log15.v2"
)

var (
	// ErrAlreadyExist is returned from New() if
	// an ExtHandler already exists.
	ErrAlreadyExist = errors.New("handler already exists")
)

// Dispatcher holds dispatchers for each extension, or creates them
// when they're requested.
type Dispatcher struct {
	reg    registrar.Interface
	logger log15.Logger

	mut  sync.Mutex
	exts map[string]*ExtHandler
}

// NewDispatcher returns a new dispatcher for extensions.
func NewDispatcher(reg registrar.Interface, logger log15.Logger) *Dispatcher {
	r := &Dispatcher{
		logger: logger,
		reg:    reg,
		exts:   make(map[string]*ExtHandler),
	}

	return r
}

// New creates a new handler, killing any old ones laying around by the same
// name.
func (d *Dispatcher) New(ext string, rwc io.ReadWriteCloser, dc OnDisconnect) *ExtHandler {
	d.mut.Lock()
	defer d.mut.Unlock()

	if e, ok := d.exts[ext]; ok {
		e.Close()
	}

	handler := NewExtHandler(ext, rwc, d.logger, dc)
	d.exts[ext] = handler

	return handler
}

// Get an extension handler by it's name
func (d *Dispatcher) Get(ext string) *ExtHandler {
	d.mut.Lock()
	defer d.mut.Unlock()

	return d.exts[ext]
}
