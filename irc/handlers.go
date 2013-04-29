package irc

// Message type provides a view around an IrcMessage to access it's parts in a
// more convenient way.
type Message struct {
	// Raw is the underlying irc message.
	Raw *IrcMessage
}

// Target retrieves the channel or user this message was sent to.
func (p *Message) Target() string {
	return p.Raw.Args[0]
}

// Message retrieves the message sent to the user or channel.
func (p *Message) Message() string {
	return p.Raw.Args[1]
}

// PrivmsgHandler is for handling privmsgs going to channel or user targets.
type PrivmsgHandler interface {
	Privmsg(*Message)
}

// PrivmsgUserHandler is for handling privmsgs going to user targets.
type PrivmsgUserHandler interface {
	PrivmsgUser(*Message)
}

// PrivmsgChannelHandler is for handling privmsgs going to channel targets.
type PrivmsgChannelHandler interface {
	PrivmsgChannel(*Message)
}

// NoticeHandler is for handling privmsgs going to channel or user targets.
type NoticeHandler interface {
	Notice(*Message)
}

// NoticeUserHandler is for handling privmsgs going to user targets.
type NoticeUserHandler interface {
	NoticeUser(*Message)
}

// NoticeChannelHandler is for handling privmsgs going to channel targets.
type NoticeChannelHandler interface {
	NoticeChannel(*Message)
}
