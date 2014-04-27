package dispatch

import "github.com/aarondl/ultimateq/irc"

// PrivmsgHandler is for handling privmsgs going to channel or user targets.
type PrivmsgHandler interface {
	Privmsg(irc.Writer, *irc.Event)
}

// PrivmsgUserHandler is for handling privmsgs going to user targets.
type PrivmsgUserHandler interface {
	PrivmsgUser(irc.Writer, *irc.Event)
}

// PrivmsgChannelHandler is for handling privmsgs going to channel targets.
type PrivmsgChannelHandler interface {
	PrivmsgChannel(irc.Writer, *irc.Event)
}

// NoticeHandler is for handling privmsgs going to channel or user targets.
type NoticeHandler interface {
	Notice(irc.Writer, *irc.Event)
}

// NoticeUserHandler is for handling privmsgs going to user targets.
type NoticeUserHandler interface {
	NoticeUser(irc.Writer, *irc.Event)
}

// NoticeChannelHandler is for handling privmsgs going to channel targets.
type NoticeChannelHandler interface {
	NoticeChannel(irc.Writer, *irc.Event)
}

// CTCPHandler is for handling ctcp messages that are directly to the bot.
// Automatically parses the tag & data portions out.
type CTCPHandler interface {
	CTCP(irc.Writer, *irc.Event, string, string)
}

// CTCPChannelHandler is for handling ctcp messages that go to a channel.
// Automatically parses the tag & data portions out.
type CTCPChannelHandler interface {
	CTCPChannel(irc.Writer, *irc.Event, string, string)
}

// CTCPReplyHandler is for handling ctcp replies from clients.
// Automatically parses the tag & data portions out.
type CTCPReplyHandler interface {
	CTCPReply(irc.Writer, *irc.Event, string, string)
}
