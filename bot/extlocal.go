package bot

// Extension provides methods to initialize and de-initialize itself.
// The bot type is passed in as the interface with which to register event
// handlers, commands, and access the databases. The de-init should destroy the
// extension until it is called upon again.
//
// Event handlers can execute in parallel and so there must be provisions to
// protect mutable state within the implementation.
type Extension interface {
	// Init attaches event handlers and commands
	Init(*Bot) error
	// Deinit detaches event handlers and commands
	Deinit(*Bot) error
}

var extensions = map[string]Extension{}

// RegisterExtension with the bot. This should be called on init(), and
// the extension should rely upon import side effects in order to have that
// init() called. Panics if a name is registered twice.
func RegisterExtension(name string, ext Extension) {
	if _, ok := extensions[name]; ok {
		panic("Extension [" + name + "] is already registered.")
	}

	extensions[name] = ext
}
