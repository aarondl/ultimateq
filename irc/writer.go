package irc

import (
	"fmt"
	"io"
	"strings"
)

const (
	// IRC_MAX_LENGTH is the maximum length for an irc message. Normally it is
	// 510 bytes + crlf but the server has to truncate extra to allow for our
	// fullhost on rebroadcast to clients, so we should send less than
	// this by the maximum allowed fullhost length.
	IRC_MAX_LENGTH = 510 - 62
	// SPLIT_BACKWARD is the maximum number of characters split will search
	// backwards from IRC_MAX_LENGTH for a space when spliting message to long
	// to fit on one line
	SPLIT_BACKWARD = 20
	// fmtPrivmsgHeader creates the beginning of a privmsg.
	fmtPrivmsgHeader = PRIVMSG + " %s :"
	// fmtNoticeHeader creates the beginning of a notice.
	fmtNoticeHeader = NOTICE + " %s :"
	// fmtCTCP creates a CTCP message.
	fmtCTCP = PRIVMSG + " %s :%s"
	// fmtCTCPReply creates a CTCPReply message.
	fmtCTCPReply = NOTICE + " %s :%s"
	// fmtNotifyHeader creates a notice or privmsg message.
	fmtNotifyHeader = "%s %s :"
	// fmtJoin creates a join message.
	fmtJoin = JOIN + " :%s"
	// fmtPart creates a part message.
	fmtPart = PART + " :%s"
	// fmtQuit creates a quit message.
	fmtQuit = QUIT + " :%s"
)

// Writer provides common write operations in IRC protocol fashion to an
// underlying io.Writer.
type Writer interface {
	io.Writer
	// Send sends a string with spaces between non-strings.
	Send(...interface{}) error
	// Sendln sends a string with spaces between everything.
	// Does not send newline.
	Sendln(...interface{}) error
	// Sendf sends a formatted string.
	Sendf(string, ...interface{}) error

	// Privmsg sends a privmsg with spaces between non-strings.
	Privmsg(string, ...interface{}) error
	// Privmsgln sends a privmsg with spaces between everything.
	// Does not send newline.
	Privmsgln(string, ...interface{}) error
	// Privmsgf sends a formatted privmsg.
	Privmsgf(string, string, ...interface{}) error

	// Notice sends a notice with spaces between non-strings.
	Notice(string, ...interface{}) error
	// Noticeln sends a notice with spaces between everything.
	// Does not send newline.
	Noticeln(string, ...interface{}) error
	// Noticef sends a formatted notice.
	Noticef(string, string, ...interface{}) error

	// CTCP sends a CTCP with spaces between non-strings.
	CTCP(string, string, ...interface{}) error
	// CTCPln sends a CTCP with spaces between everything.
	// Does not send newline.
	CTCPln(string, string, ...interface{}) error
	// CTCPf sends a formatted CTCP.
	CTCPf(string, string, string, ...interface{}) error
	// CTCPReply sends a string with spaces between non-strings.

	// CTCPReply sends a CTCPReply with spaces between non-strings.
	CTCPReply(string, string, ...interface{}) error
	// CTCPReplyln sends a CTCPReply with spaces between everything.
	// Does not send newline.
	CTCPReplyln(string, string, ...interface{}) error
	// CTCPReplyf sends a formatted CTCPReply.
	CTCPReplyf(string, string, string, ...interface{}) error

	// Notify sends a notification with spaces between non-strings.
	// If the Event is designated towards a #channel, then the notification
	// will be sent to that channel. If the Event is designated towards a user
	// (normally the bot itself) then the notification will be sent to the
	// given target.
	Notify(*Event, string, ...interface{}) error
	// Notifyln sends a notification with spaces between everything.
	// Does not send newline. See Notify for details of use.
	Notifyln(*Event, string, ...interface{}) error
	// Notifyf sends a formatted notification. See Notify for details of use.
	Notifyf(*Event, string, string, ...interface{}) error

	// Sends a join message to the writer.
	Join(...string) error
	// Sends a part message to the writer.
	Part(...string) error
	// Sends a quit message to the writer.
	Quit(string) error
}

// Helper fullfills the Writer's many interface requirements.
type Helper struct {
	io.Writer
}

// Send sends a string with spaces between non-strings.
func (h Helper) Send(args ...interface{}) error {
	_, err := fmt.Fprint(h, args...)
	return err
}

// Sendln sends a string with spaces between everything. Does not send newline.
func (h Helper) Sendln(args ...interface{}) error {
	str := fmt.Sprintln(args...)
	_, err := h.Write([]byte(str[:len(str)-1]))
	return err
}

// Sendf sends a formatted string.
func (h Helper) Sendf(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(h, format, args...)
	return err
}

// Privmsg sends a string with spaces between non-strings.
func (h Helper) Privmsg(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtPrivmsgHeader, target))
	msg := []byte(fmt.Sprint(args...))
	return h.splitSend(header, msg)
}

// Privmsgln sends a privmsg with spaces between everything.
// Does not send newline.
func (h Helper) Privmsgln(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtPrivmsgHeader, target))
	str := fmt.Sprintln(args...)
	str = str[:len(str)-1]
	return h.splitSend(header, []byte(str))
}

// Privmsgf sends a formatted privmsg.
func (h Helper) Privmsgf(target, format string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtPrivmsgHeader, target))
	msg := []byte(fmt.Sprintf(format, args...))
	return h.splitSend(header, msg)
}

// Notice sends a string with spaces between non-strings.
func (h Helper) Notice(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtNoticeHeader, target))
	msg := []byte(fmt.Sprint(args...))
	return h.splitSend(header, msg)
}

// Noticeln sends a notice with spaces between everything.
// Does not send newline.
func (h Helper) Noticeln(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtNoticeHeader, target))
	str := fmt.Sprintln(args...)
	str = str[:len(str)-1]
	return h.splitSend(header, []byte(str))
}

// Noticef sends a formatted notice.
func (h Helper) Noticef(target, format string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtNoticeHeader, target))
	msg := []byte(fmt.Sprintf(format, args...))
	return h.splitSend(header, msg)
}

// CTCP sends a string with spaces between non-strings.
func (h Helper) CTCP(target, tag string, data ...interface{}) error {
	msg := CTCPpack([]byte(tag), []byte(fmt.Sprint(data...)))
	_, err := fmt.Fprintf(h, fmtCTCP, target, msg)
	return err
}

// CTCPln sends a CTCP with spaces between everything.
// Does not send newline.
func (h Helper) CTCPln(target, tag string, data ...interface{}) error {
	str := fmt.Sprintln(data...)
	str = str[:len(str)-1]
	msg := CTCPpack([]byte(tag), []byte(str))
	_, err := fmt.Fprintf(h, fmtCTCP, target, msg)
	return err
}

// CTCPf sends a formatted CTCP.
func (h Helper) CTCPf(target, tag, format string, data ...interface{}) error {
	msg := CTCPpack([]byte(tag), []byte(fmt.Sprintf(format, data...)))
	_, err := fmt.Fprintf(h, fmtCTCP, target, msg)
	return err
}

// CTCPReply sends a string with spaces between non-strings.
func (h Helper) CTCPReply(target, tag string, data ...interface{}) error {
	msg := CTCPpack([]byte(tag), []byte(fmt.Sprint(data...)))
	_, err := fmt.Fprintf(h, fmtCTCPReply, target, msg)
	return err
}

// CTCPReplyln sends a CTCPReply with spaces between everything.
// Does not send newline.
func (h Helper) CTCPReplyln(target, tag string, data ...interface{}) error {
	str := fmt.Sprintln(data...)
	str = str[:len(str)-1]
	msg := CTCPpack([]byte(tag), []byte(str))
	_, err := fmt.Fprintf(h, fmtCTCPReply, target, msg)
	return err
}

// CTCPReplyf sends a formatted CTCPReply.
func (h Helper) CTCPReplyf(target, tag, format string,
	data ...interface{}) error {

	msg := CTCPpack([]byte(tag), []byte(fmt.Sprintf(format, data...)))
	_, err := fmt.Fprintf(h, fmtCTCPReply, target, msg)
	return err
}

// Notify sends a string with spaces between non-strings.
// See irc.Writer.Notify for details of use.
func (h Helper) Notify(ev *Event, target string, args ...interface{}) error {
	msgType := NOTICE
	if ev.IsTargetChan() {
		msgType = PRIVMSG
		target = ev.Target()
	}
	header := []byte(fmt.Sprintf(fmtNotifyHeader, msgType, target))
	msg := []byte(fmt.Sprint(args...))
	return h.splitSend(header, msg)
}

// Notifyln sends a notify with spaces between everything.
// Does not send newline. See irc.Writer.Notify for details of use.
func (h Helper) Notifyln(ev *Event, target string, args ...interface{}) error {
	msgType := NOTICE
	if ev.IsTargetChan() {
		msgType = PRIVMSG
		target = ev.Target()
	}
	header := []byte(fmt.Sprintf(fmtNotifyHeader, msgType, target))
	str := fmt.Sprintln(args...)
	str = str[:len(str)-1]
	return h.splitSend(header, []byte(str))
}

// Notifyf sends a formatted notification.
// See irc.Writer.Notify for details of use.
func (h Helper) Notifyf(ev *Event, target, format string,
	args ...interface{}) error {

	msgType := NOTICE
	if ev.IsTargetChan() {
		msgType = PRIVMSG
		target = ev.Target()
	}
	header := []byte(fmt.Sprintf(fmtNotifyHeader, msgType, target))
	msg := []byte(fmt.Sprintf(format, args...))
	return h.splitSend(header, msg)
}

// Join sends a join message to the writer.
func (h Helper) Join(targets ...string) error {
	if len(targets) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(h, fmtJoin, strings.Join(targets, ","))
	return err
}

// Part sends a part message to the writer.
func (h Helper) Part(targets ...string) error {
	if len(targets) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(h, fmtPart, strings.Join(targets, ","))
	return err
}

// Quit sends a quit message to the writer.
func (h Helper) Quit(msg string) error {
	_, err := fmt.Fprintf(h, fmtQuit, msg)
	return err
}

// splitSend breaks a message down into irc-digestable chunks based on
// IRC_MAX_LENGTH, and appends the header to each message. Will also use
// SPLIT_BACKWARD character look-back to see if it can split on a space instead
// of in the middle of a word. If it can, it will eliminate the space from
// the following message.
func (h Helper) splitSend(header, msg []byte) error {
	var err error
	ln, lnh := len(msg), len(header)
	msgMax := IRC_MAX_LENGTH - lnh
	if ln <= msgMax {
		_, err = h.Write(append(header, msg...))
		return err
	}

	var size int
	buf := make([]byte, IRC_MAX_LENGTH)
	for ln > 0 {
		nextWriteOffset := 0
		size = msgMax
		if ln <= msgMax {
			size = ln
		} else {
			for i := msgMax; i != 0 && i > msgMax-SPLIT_BACKWARD; i-- {
				if msg[i] == ' ' {
					size = i
					nextWriteOffset = 1
					break
				}
			}
		}
		copy(buf, header)
		copy(buf[lnh:], msg[:size])
		_, err = h.Write(buf[:lnh+size])
		if err != nil {
			return err
		}
		msg = msg[size+nextWriteOffset:]
		ln, lnh = len(msg), len(header)
	}

	return nil
}
