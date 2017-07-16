package cmd

import (
	"strings"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

// Event represents the data about the event that occurred. The commander
// fills the Event structure with information about the user and channel
// involved. It also embeds the State and Store for easy access.
//
// Some parts of Event will be nil under certain circumstances so elements
// within must be checked for nil, see each element's documentation
// for further information.
type Event struct {
	*irc.Event

	// User can be nil if the bot's State is disabled.
	User *data.User
	// StoredUser will be nil when there is no required access.
	StoredUser *data.StoredUser
	// UserChannelModes will be nil when the message was not sent to a channel.
	UserChannelModes *data.UserModes
	// Channel will be nil when the message was not sent to a channel.
	Channel *data.Channel
	// TargetChannel will not be nil when the command has the #channel
	// parameter. The parameter can still be nil when the channel is not known
	// to the bot.
	TargetChannel *data.Channel
	// TargetUsers is populated when the arguments contain a ~nick argument, and
	// as a byproduct of looking up authentication, when the arguments contain
	// a *user argument, and a nickname is passed instead of a *username.
	TargetUsers map[string]*data.User
	// TargetStoredUser is populated when the arguments contain a *user
	// argument.
	TargetStoredUser map[string]*data.StoredUser
	// TargetVarUsers is populated when the arguments contain a ~nick...
	// argument. When a *user... parameter is used, it will be sparsely filled
	// whenever a user is requested by nickname not *username.
	TargetVarUsers []*data.User
	// TargetVarUsers is populated when the arguments contain a *user...
	// argument.
	TargetVarStoredUser []*data.StoredUser

	args map[string]string
}

// Arg gets an argument that was passed in to the command by the user. The
// name of the argument passed into Register() is required to get the argument.
func (ev *Event) Arg(arg string) string {
	return ev.args[arg]
}

// SplitArg behaves exactly like GetArg but calls strings.Fields on the
// argument. Useful for varargs...
func (ev *Event) SplitArg(arg string) (args []string) {
	if str, ok := ev.args[arg]; ok && len(str) > 0 {
		args = strings.Fields(str)
	}
	return
}
