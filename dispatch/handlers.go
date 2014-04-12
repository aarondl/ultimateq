package dispatch

import "github.com/aarondl/ultimateq/irc"

// PrivmsgHandler is for handling privmsgs going to channel or user targets.
type PrivmsgHandler interface {
	Privmsg(*irc.Message, irc.Endpoint)
}

// PrivmsgUserHandler is for handling privmsgs going to user targets.
type PrivmsgUserHandler interface {
	PrivmsgUser(*irc.Message, irc.Endpoint)
}

// PrivmsgChannelHandler is for handling privmsgs going to channel targets.
type PrivmsgChannelHandler interface {
	PrivmsgChannel(*irc.Message, irc.Endpoint)
}

// NoticeHandler is for handling privmsgs going to channel or user targets.
type NoticeHandler interface {
	Notice(*irc.Message, irc.Endpoint)
}

// NoticeUserHandler is for handling privmsgs going to user targets.
type NoticeUserHandler interface {
	NoticeUser(*irc.Message, irc.Endpoint)
}

// NoticeChannelHandler is for handling privmsgs going to channel targets.
type NoticeChannelHandler interface {
	NoticeChannel(*irc.Message, irc.Endpoint)
}

// CTCPHandler is for handling ctcp messages that are directly to the bot.
// Automatically parses the tag & data portions out.
type CTCPHandler interface {
	CTCP(*irc.Message, string, string, irc.Endpoint)
}

// CTCPChannelHandler is for handling ctcp messages that go to a channel.
// Automatically parses the tag & data portions out.
type CTCPChannelHandler interface {
	CTCPChannel(*irc.Message, string, string, irc.Endpoint)
}

// CTCPReplyHandler is for handling ctcp replies from clients.
// Automatically parses the tag & data portions out.
type CTCPReplyHandler interface {
	CTCPReply(*irc.Message, string, string, irc.Endpoint)
}
