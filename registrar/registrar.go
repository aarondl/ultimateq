package registrar

import "github.com/aarondl/ultimateq/dispatch/cmd"

// Interface is the operations performable by a registrar.
type Interface interface {
	Register(network, channel, event string, handler interface{}) uint64
	RegisterCmd(network, channel string, command *cmd.Cmd) error

	Unregister(id uint64) bool
	UnregisterCmd(network, channel, ext, cmd string) bool
}
