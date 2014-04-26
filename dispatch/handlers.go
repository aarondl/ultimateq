package dispatch

import "github.com/aarondl/ultimateq/irc"

// PrivmsgHandler is for handling privmsgs going to channel or user targets.
type PrivmsgHandler interface {
	Privmsg(*irc.Event, irc.Writer)
}

// PrivmsgUserHandler is for handling privmsgs going to user targets.
type PrivmsgUserHandler interface {
	PrivmsgUser(*irc.Event, irc.Writer)
}

// PrivmsgChannelHandler is for handling privmsgs going to channel targets.
type PrivmsgChannelHandler interface {
	PrivmsgChannel(*irc.Event, irc.Writer)
}

// NoticeHandler is for handling privmsgs going to channel or user targets.
type NoticeHandler interface {
	Notice(*irc.Event, irc.Writer)
}

// NoticeUserHandler is for handling privmsgs going to user targets.
type NoticeUserHandler interface {
	NoticeUser(*irc.Event, irc.Writer)
}

// NoticeChannelHandler is for handling privmsgs going to channel targets.
type NoticeChannelHandler interface {
	NoticeChannel(*irc.Event, irc.Writer)
}

// CTCPHandler is for handling ctcp messages that are directly to the bot.
// Automatically parses the tag & data portions out.
type CTCPHandler interface {
	CTCP(*irc.Event, string, string, irc.Writer)
}

// CTCPChannelHandler is for handling ctcp messages that go to a channel.
// Automatically parses the tag & data portions out.
type CTCPChannelHandler interface {
	CTCPChannel(*irc.Event, string, string, irc.Writer)
}

// CTCPReplyHandler is for handling ctcp replies from clients.
// Automatically parses the tag & data portions out.
type CTCPReplyHandler interface {
	CTCPReply(*irc.Event, string, string, irc.Writer)
}
