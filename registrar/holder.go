package registrar

import (
	"strings"

	"github.com/aarondl/ultimateq/dispatch/cmd"
)

// holder stores registrations to an underlying Registrar and has the ability
// to unregister everything.
type holder struct {
	registrar Interface

	ids      map[uint64]struct{}
	commands map[string]struct{}
}

func newHolder(registrar Interface) *holder {
	h := &holder{
		registrar: registrar,
	}
	h.initMaps()

	return h
}

func (h *holder) initMaps() {
	h.ids = make(map[uint64]struct{})
	h.commands = make(map[string]struct{})
}

// Register and save the token we get back.
func (h *holder) Register(network, channel, event string, handler interface{}) uint64 {
	id := h.registrar.Register(network, channel, event, handler)
	h.ids[id] = struct{}{}

	return id
}

// RegisterCmd and save the names we use for later
func (h *holder) RegisterCmd(network, channel string, command *cmd.Cmd) error {
	if err := h.registrar.RegisterCmd(network, channel, command); err != nil {
		return err
	}

	key := strings.Join([]string{network, channel, command.Extension, command.Cmd}, ":")
	h.commands[key] = struct{}{}

	return nil
}

// Unregister and discard our token that matches.
func (h *holder) Unregister(id uint64) bool {
	if did := h.registrar.Unregister(id); !did {
		return false
	}

	delete(h.ids, id)
	return true
}

// UnregisterCmd and discard our record of its registration.
func (h *holder) UnregisterCmd(network, channel, ext, cmd string) bool {
	ok := h.registrar.UnregisterCmd(network, channel, ext, cmd)

	key := strings.Join([]string{network, channel, ext, cmd}, ":")
	delete(h.commands, key)
	return ok
}

// unregisterAll applies unregister to all known registered things as well
// as empties the maps.
func (h *holder) unregisterAll() {
	for k := range h.ids {
		h.registrar.Unregister(k)
	}

	for k := range h.commands {
		spl := strings.Split(k, ":")

		network, channel, ext, cmd := spl[0], spl[1], spl[2], spl[3]
		h.registrar.UnregisterCmd(network, channel, ext, cmd)
	}

	h.initMaps()
}
