package irc

import "strings"

const (
	// nTargetsAssumed is the typical number of targets for notices and privmsgs
	nTargetsAssumed = 1
)

// Privmsg type holds information about a privmsg event.
type Privmsg struct {
	Message string
	User    string
	Channel string
	Sender  string
}

// PrivmsgHandler handles private messages of any form.
type PrivmsgHandler interface {
	Privmsg(event *Privmsg)
}

// Parses private messages into three separate arrays for caching. This function
// is declared on Dispatcher because of it's need for a ChannelFinder object.
func (d *Dispatcher) PrivmsgParse(msg *IrcMessage) (
	[]*Privmsg, []*PrivmsgTarget, []*PrivmsgTarget) {

	p := make([]*Privmsg, 0, nTargetsAssumed)
	c := make([]*PrivmsgTarget, 0, nTargetsAssumed)
	u := make([]*PrivmsgTarget, 0, nTargetsAssumed)

	for _, v := range strings.Split(msg.Args[0], ",") {
		if d.finder != nil && d.finder.IsChannel(v) {
			pmsg := &Privmsg{Channel: v, Sender: msg.Sender, Message: msg.Args[1]}
			trg := &PrivmsgTarget{Target: v, Sender: msg.Sender, Message: msg.Args[1]}
			p = append(p, pmsg)
			c = append(c, trg)
		} else {
			pmsg := &Privmsg{User: v, Sender: msg.Sender, Message: msg.Args[1]}
			trg := &PrivmsgTarget{Target: v, Sender: msg.Sender, Message: msg.Args[1]}
			p = append(p, pmsg)
			u = append(u, trg)
		}
	}

	return p, u, c
}

// PrivmsgTarget holds information about a privmsg to a specific target.
type PrivmsgTarget struct {
	Message string
	Target  string
	Sender  string
}

// PrivmsgChannelHandler handles channel messages only.
type PrivmsgChannelHandler interface {
	PrivmsgChannel(msg *PrivmsgTarget)
}

// PrivmsgUserHandler handles channel messages only.
type PrivmsgUserHandler interface {
	PrivmsgUser(msg *PrivmsgTarget)
}

// Notice type holds information about a privmsg event.
type Notice struct {
	Message string
	User    string
	Channel string
	Sender  string
}

// NoticeHandler handles private messages of any form.
type NoticeHandler interface {
	Notice(event *Notice)
}

// NoticeTarget holds information about a privmsg to a specific target.
type NoticeTarget struct {
	Message string
	Target  string
	Sender  string
}

// NoticeChannelHandler handles channel messages only.
type NoticeChannelHandler interface {
	NoticeChannel(msg *NoticeTarget)
}

// NoticeUserHandler handles channel messages only.
type NoticeUserHandler interface {
	NoticeUser(msg *NoticeTarget)
}

// Parses notices into three separate arrays for caching.
func (d *Dispatcher) NoticeParse(msg *IrcMessage) (
	[]*Notice, []*NoticeTarget, []*NoticeTarget) {

	n := make([]*Notice, 0, nTargetsAssumed)
	c := make([]*NoticeTarget, 0, nTargetsAssumed)
	u := make([]*NoticeTarget, 0, nTargetsAssumed)

	for _, v := range strings.Split(msg.Args[0], ",") {
		if d.finder != nil && d.finder.IsChannel(v) {
			nmsg := &Notice{Channel: v, Sender: msg.Sender, Message: msg.Args[1]}
			trg := &NoticeTarget{Target: v, Sender: msg.Sender, Message: msg.Args[1]}
			n = append(n, nmsg)
			c = append(c, trg)
		} else {
			nmsg := &Notice{User: v, Sender: msg.Sender, Message: msg.Args[1]}
			trg := &NoticeTarget{Target: v, Sender: msg.Sender, Message: msg.Args[1]}
			n = append(n, nmsg)
			u = append(u, trg)
		}
	}

	return n, u, c
}
