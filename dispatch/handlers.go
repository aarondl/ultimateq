package dispatch

import "github.com/aarondl/ultimateq/irc"

// PrivmsgHandler is for handling privmsgs going to channel or user targets.
type PrivmsgHandler interface {
	Privmsg(*irc.Message, irc.Sender)
}

// PrivmsgUserHandler is for handling privmsgs going to user targets.
type PrivmsgUserHandler interface {
	PrivmsgUser(*irc.Message, irc.Sender)
}

// PrivmsgChannelHandler is for handling privmsgs going to channel targets.
type PrivmsgChannelHandler interface {
	PrivmsgChannel(*irc.Message, irc.Sender)
}

// NoticeHandler is for handling privmsgs going to channel or user targets.
type NoticeHandler interface {
	Notice(*irc.Message, irc.Sender)
}

// NoticeUserHandler is for handling privmsgs going to user targets.
type NoticeUserHandler interface {
	NoticeUser(*irc.Message, irc.Sender)
}

// NoticeChannelHandler is for handling privmsgs going to channel targets.
type NoticeChannelHandler interface {
	NoticeChannel(*irc.Message, irc.Sender)
}
