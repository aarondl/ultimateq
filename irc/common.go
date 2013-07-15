/*
Package irc defines types to be used by most other packages in
the ultimateq system. It is small and comprised mostly of helper like types
and constants.
*/
package irc

import (
	"fmt"
	"io"
	"strings"
)

const (
	// IRC_MAX_LENGTH is the maximum length for an irc message.
	IRC_MAX_LENGTH = 510
	// fmtPrivmsgHeader creates the beginning of a privmsg.
	fmtPrivmsgHeader = PRIVMSG + " %v :"
	// fmtNoticeHeader creates the beginning of a notice.
	fmtNoticeHeader = NOTICE + " %v :"
	// fmtJoin creates a join message.
	fmtJoin = JOIN + " :%v"
	// fmtPart creates a part message.
	fmtPart = PART + " :%v"
	// fmtQuit creates a quit message.
	fmtQuit = QUIT + " :%v"
)

// IRC Messages, these messages are 1-1 constant to string lookups for ease of
// use when registering handlers etc.
const (
	JOIN    = "JOIN"
	KICK    = "KICK"
	MODE    = "MODE"
	NICK    = "NICK"
	NOTICE  = "NOTICE"
	PART    = "PART"
	PING    = "PING"
	PONG    = "PONG"
	PRIVMSG = "PRIVMSG"
	QUIT    = "QUIT"
	TOPIC   = "TOPIC"
)

// IRC Reply and Error Messages. These are sent in reply to a previous message.
const (
	RPL_WELCOME         = "001"
	RPL_YOURHOST        = "002"
	RPL_CREATED         = "003"
	RPL_MYINFO          = "004"
	RPL_ISUPPORT        = "005"
	RPL_BOUNCE          = "005"
	RPL_USERHOST        = "302"
	RPL_ISON            = "303"
	RPL_AWAY            = "301"
	RPL_UNAWAY          = "305"
	RPL_NOWAWAY         = "306"
	RPL_WHOISUSER       = "311"
	RPL_WHOISSERVER     = "312"
	RPL_WHOISOPERATOR   = "313"
	RPL_WHOISIDLE       = "317"
	RPL_ENDOFWHOIS      = "318"
	RPL_WHOISCHANNELS   = "319"
	RPL_WHOWASUSER      = "314"
	RPL_ENDOFWHOWAS     = "369"
	RPL_LISTSTART       = "321"
	RPL_LIST            = "322"
	RPL_LISTEND         = "323"
	RPL_UNIQOPIS        = "325"
	RPL_CHANNELMODEIS   = "324"
	RPL_NOTOPIC         = "331"
	RPL_TOPIC           = "332"
	RPL_INVITING        = "341"
	RPL_SUMMONING       = "342"
	RPL_INVITELIST      = "346"
	RPL_ENDOFINVITELIST = "347"
	RPL_EXCEPTLIST      = "348"
	RPL_ENDOFEXCEPTLIST = "349"
	RPL_VERSION         = "351"
	RPL_WHOREPLY        = "352"
	RPL_ENDOFWHO        = "315"
	RPL_NAMREPLY        = "353"
	RPL_ENDOFNAMES      = "366"
	RPL_LINKS           = "364"
	RPL_ENDOFLINKS      = "365"
	RPL_BANLIST         = "367"
	RPL_ENDOFBANLIST    = "368"
	RPL_INFO            = "371"
	RPL_ENDOFINFO       = "374"
	RPL_MOTDSTART       = "375"
	RPL_MOTD            = "372"
	RPL_ENDOFMOTD       = "376"
	RPL_YOUREOPER       = "381"
	RPL_REHASHING       = "382"
	RPL_YOURESERVICE    = "383"
	RPL_TIME            = "391"
	RPL_USERSSTART      = "392"
	RPL_USERS           = "393"
	RPL_ENDOFUSERS      = "394"
	RPL_NOUSERS         = "395"
	RPL_TRACELINK       = "200"
	RPL_TRACECONNECTING = "201"
	RPL_TRACEHANDSHAKE  = "202"
	RPL_TRACEUNKNOWN    = "203"
	RPL_TRACEOPERATOR   = "204"
	RPL_TRACEUSER       = "205"
	RPL_TRACESERVER     = "206"
	RPL_TRACESERVICE    = "207"
	RPL_TRACENEWTYPE    = "208"
	RPL_TRACECLASS      = "209"
	RPL_TRACERECONNECT  = "210"
	RPL_TRACELOG        = "261"
	RPL_TRACEEND        = "262"
	RPL_STATSLINKINFO   = "211"
	RPL_STATSCOMMANDS   = "212"
	RPL_ENDOFSTATS      = "219"
	RPL_STATSUPTIME     = "242"
	RPL_STATSOLINE      = "243"
	RPL_UMODEIS         = "221"
	RPL_SERVLIST        = "234"
	RPL_SERVLISTEND     = "235"
	RPL_LUSERCLIENT     = "251"
	RPL_LUSEROP         = "252"
	RPL_LUSERUNKNOWN    = "253"
	RPL_LUSERCHANNELS   = "254"
	RPL_LUSERME         = "255"
	RPL_ADMINME         = "256"
	RPL_ADMINLOC1       = "257"
	RPL_ADMINLOC2       = "258"
	RPL_ADMINEMAIL      = "259"
	RPL_TRYAGAIN        = "263"

	ERR_NOSUCHNICK        = "401"
	ERR_NOSUCHSERVER      = "402"
	ERR_NOSUCHCHANNEL     = "403"
	ERR_CANNOTSENDTOCHAN  = "404"
	ERR_TOOMANYCHANNELS   = "405"
	ERR_WASNOSUCHNICK     = "406"
	ERR_TOOMANYTARGETS    = "407"
	ERR_NOSUCHSERVICE     = "408"
	ERR_NOORIGIN          = "409"
	ERR_NORECIPIENT       = "411"
	ERR_NOTEXTTOSEND      = "412"
	ERR_NOTOPLEVEL        = "413"
	ERR_WILDTOPLEVEL      = "414"
	ERR_BADMASK           = "415"
	ERR_UNKNOWNCOMMAND    = "421"
	ERR_NOMOTD            = "422"
	ERR_NOADMININFO       = "423"
	ERR_FILEERROR         = "424"
	ERR_NONICKNAMEGIVEN   = "431"
	ERR_ERRONEUSNICKNAME  = "432"
	ERR_NICKNAMEINUSE     = "433"
	ERR_NICKCOLLISION     = "436"
	ERR_UNAVAILRESOURCE   = "437"
	ERR_USERNOTINCHANNEL  = "441"
	ERR_NOTONCHANNEL      = "442"
	ERR_USERONCHANNEL     = "443"
	ERR_NOLOGIN           = "444"
	ERR_SUMMONDISABLED    = "445"
	ERR_USERSDISABLED     = "446"
	ERR_NOTREGISTERED     = "451"
	ERR_NEEDMOREPARAMS    = "461"
	ERR_ALREADYREGISTRED  = "462"
	ERR_NOPERMFORHOST     = "463"
	ERR_PASSWDMISMATCH    = "464"
	ERR_YOUREBANNEDCREEP  = "465"
	ERR_YOUWILLBEBANNED   = "466"
	ERR_KEYSET            = "467"
	ERR_CHANNELISFULL     = "471"
	ERR_UNKNOWNMODE       = "472"
	ERR_INVITEONLYCHAN    = "473"
	ERR_BANNEDFROMCHAN    = "474"
	ERR_BADCHANNELKEY     = "475"
	ERR_BADCHANMASK       = "476"
	ERR_NOCHANMODES       = "477"
	ERR_BANLISTFULL       = "478"
	ERR_NOPRIVILEGES      = "481"
	ERR_CHANOPRIVSNEEDED  = "482"
	ERR_CANTKILLSERVER    = "483"
	ERR_RESTRICTED        = "484"
	ERR_UNIQOPPRIVSNEEDED = "485"
	ERR_NOOPERHOST        = "491"
	ERR_UMODEUNKNOWNFLAG  = "501"
	ERR_USERSDONTMATCH    = "502"
)

// Pseudo Messages, these messages are not real messages defined by the irc
// protocol but the bot provides them to allow for additional messages to be
// handled such as connect or disconnects which the irc protocol has no protocol
// defined for.
const (
	RAW        = "RAW"
	CONNECT    = "CONNECT"
	DISCONNECT = "DISCONNECT"
)

// Endpoint represents the source of an event, and should allow replies on a
// writing interface as well as a way to identify itself.
type Endpoint interface {
	// An embedded io.Writer that can write back to the source of the event.
	io.Writer
	// Retrieves a key to identify the source of the event.
	GetKey() string
	// Send sends a string with spaces between non-strings.
	Send(...interface{}) error
	// Sendln sends a string with spaces between everything.
	// Does not send newline.
	Sendln(...interface{}) error
	// Sendf sends a formatted string.
	Sendf(string, ...interface{}) error

	// Privmsg sends a string with spaces between non-strings.
	Privmsg(string, ...interface{}) error
	// Privmsgln sends a privmsg with spaces between everything.
	// Does not send newline.
	Privmsgln(string, ...interface{}) error
	// Privmsgf sends a formatted privmsg.
	Privmsgf(string, string, ...interface{}) error
	// Notice sends a string with spaces between non-strings.
	Notice(string, ...interface{}) error
	// Noticeln sends a notice with spaces between everything.
	// Does not send newline.
	Noticeln(string, ...interface{}) error
	// Noticef sends a formatted notice.
	Noticef(string, string, ...interface{}) error
	// Sends a join message to the endpoint.
	Join(...string) error
	// Sends a part message to the endpoint.
	Part(...string) error
	// Sends a quit message to the endpoint.
	Quit(string) error
}

// IrcMessage contains all the information broken out of an irc message.
type IrcMessage struct {
	// Name of the message. Uppercase constant name or numeric.
	Name string
	// The server or user that sent the message, a fullhost if one was supplied.
	Sender string
	// The args split by space delimiting.
	Args []string
}

// Split splits string arguments. A convenience method to avoid having to call
// splits and import strings.
func (m *IrcMessage) Split(index int) []string {
	return strings.Split(m.Args[index], ",")
}

// Message type provides a view around an IrcMessage to access it's parts in a
// more convenient way.
type Message struct {
	// The underlying irc message.
	*IrcMessage
}

// Target retrieves the channel or user this message was sent to.
func (p *Message) Target() string {
	return p.Args[0]
}

// Message retrieves the message sent to the user or channel.
func (p *Message) Message() string {
	return p.Args[1]
}

// Helper fullfills the Endpoint's many interface requirements.
type Helper struct {
	io.Writer
}

// Send sends a string with spaces between non-strings.
func (h *Helper) Send(args ...interface{}) error {
	_, err := fmt.Fprint(h, args...)
	return err
}

// Sendln sends a string with spaces between everything. Does not send newline.
func (h *Helper) Sendln(args ...interface{}) error {
	str := fmt.Sprintln(args...)
	_, err := h.Write([]byte(str[:len(str)-1]))
	return err
}

// Sendf sends a formatted string.
func (h *Helper) Sendf(format string, args ...interface{}) error {
	_, err := fmt.Fprintf(h, format, args...)
	return err
}

// Privmsg sends a string with spaces between non-strings.
func (h *Helper) Privmsg(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtPrivmsgHeader, target))
	msg := []byte(fmt.Sprint(args...))
	return h.splitSend(header, msg)
}

// Privmsgln sends a privmsg with spaces between everything.
// Does not send newline.
func (h *Helper) Privmsgln(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtPrivmsgHeader, target))
	str := fmt.Sprintln(args...)
	str = str[:len(str)-1]
	return h.splitSend(header, []byte(str))
}

// Privmsgf sends a formatted privmsg.
func (h *Helper) Privmsgf(target, format string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtPrivmsgHeader, target))
	msg := []byte(fmt.Sprintf(format, args...))
	return h.splitSend(header, msg)
}

// Notice sends a string with spaces between non-strings.
func (h *Helper) Notice(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtNoticeHeader, target))
	msg := []byte(fmt.Sprint(args...))
	return h.splitSend(header, msg)
}

// Noticeln sends a notice with spaces between everything.
// Does not send newline.
func (h *Helper) Noticeln(target string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtNoticeHeader, target))
	str := fmt.Sprintln(args...)
	str = str[:len(str)-1]
	return h.splitSend(header, []byte(str))
}

// Noticef sends a formatted notice.
func (h *Helper) Noticef(target, format string, args ...interface{}) error {
	header := []byte(fmt.Sprintf(fmtNoticeHeader, target))
	msg := []byte(fmt.Sprintf(format, args...))
	return h.splitSend(header, msg)
}

// Join sends a join message to the endpoint.
func (h *Helper) Join(targets ...string) error {
	if len(targets) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(h, fmtJoin, strings.Join(targets, ","))
	return err
}

// Part sends a part message to the endpoint.
func (h *Helper) Part(targets ...string) error {
	if len(targets) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(h, fmtPart, strings.Join(targets, ","))
	return err
}

// Quit sends a quit message to the endpoint.
func (h *Helper) Quit(msg string) error {
	_, err := fmt.Fprintf(h, fmtQuit, msg)
	return err
}

// splitSend breaks a message down into irc-digestable chunks based on
// IRC_MAX_LENGTH, and appends the header to each message.
func (h *Helper) splitSend(header, msg []byte) error {
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
		size = msgMax
		if ln < msgMax {
			size = ln
		}
		copy(buf, header)
		copy(buf[lnh:], msg[:size])
		_, err = h.Write(buf[:lnh+size])
		if err != nil {
			return err
		}
		msg = msg[size:]
		ln, lnh = len(msg), len(header)
	}

	return nil
}
