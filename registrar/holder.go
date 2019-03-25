package registrar

import (
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
)

// holder stores registrations to an underlying Registrar and has the ability
// to unregister everything.
type holder struct {
	registrar Interface

	events   map[uint64]struct{}
	commands map[uint64]struct{}
}

func newHolder(registrar Interface) *holder {
	h := &holder{
		registrar: registrar,
	}
	h.initMaps()

	return h
}

func (h *holder) initMaps() {
	h.events = make(map[uint64]struct{})
	h.commands = make(map[uint64]struct{})
}

// Register and save the token we get back.
func (h *holder) Register(network, channel, event string, handler dispatch.Handler) uint64 {
	id := h.registrar.Register(network, channel, event, handler)
	h.events[id] = struct{}{}

	return id
}

// RegisterCmd and save the names we use for later
func (h *holder) RegisterCmd(network, channel string, command *cmd.Command) (uint64, error) {
	id, err := h.registrar.RegisterCmd(network, channel, command)
	if err != nil {
		return 0, err
	}

	h.commands[id] = struct{}{}

	return id, nil
}

// Unregister and discard our token that matches.
func (h *holder) Unregister(id uint64) bool {
	if did := h.registrar.Unregister(id); !did {
		return false
	}

	delete(h.events, id)
	return true
}

// UnregisterCmd and discard our record of its registration.
func (h *holder) UnregisterCmd(id uint64) bool {
	ok := h.registrar.UnregisterCmd(id)

	delete(h.commands, id)
	return ok
}

// unregisterAll applies unregister to all known registered things as well
// as empties the maps.
func (h *holder) unregisterAll() {
	for k := range h.events {
		h.registrar.Unregister(k)
	}

	for k := range h.commands {
		h.registrar.UnregisterCmd(k)
	}

	h.initMaps()
}
