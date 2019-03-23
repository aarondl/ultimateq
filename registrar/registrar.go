package registrar

import (
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
)

// Interface is the operations performable by a registrar.
type Interface interface {
	Register(network, channel, event string, handler dispatch.Handler) uint64
	RegisterCmd(network, channel string, command *cmd.Command) (uint64, error)

	Unregister(id uint64) bool
	UnregisterCmd(id uint64) bool
}
