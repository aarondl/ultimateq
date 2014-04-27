// The ultimateq bot framework.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/query"
	"github.com/aarondl/quotes"
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
)

var (
	sanitizeNewline = strings.NewReplacer("\r\n", " ", "\n", " ")
	rgxSpace        = regexp.MustCompile(`\s{2,}`)
	queryConf       query.Config
)

const (
	dateFormat = "January 02, 2006 at 3:04pm MST"
)

/* =====================
 Helper methods.
===================== */
func sanitize(str string) string {
	return rgxSpace.ReplaceAllString(sanitizeNewline.Replace(str), " ")
}

func isNick(potential string) (is bool) {
	if len(potential) > 0 {
		first := potential[0]
		is = (first >= 'a' && first <= 'z') ||
			(first >= 'A' && first <= 'Z')
	}
	return
}

func respond(e *data.DataEndpoint, msgtarget, nickname, message string) {
	if isNick(msgtarget) {
		e.Notice(nickname, message)
	} else {
		e.Privmsg(msgtarget, message)
	}
}

type Quoter struct {
	db *quotes.QuoteDB
}

type Queryer struct {
}

type Handler struct {
}

// Let reflection hook up the commands, instead of doing it here.
func (_ *Quoter) Cmd(_ string, _ *data.DataEndpoint, _ *cmd.Event) error {
	return nil
}

func (_ *Queryer) Cmd(_ string, _ *data.DataEndpoint, _ *cmd.Event) error {
	return nil
}

func (_ *Handler) Cmd(_ string, _ *data.DataEndpoint, _ *cmd.Event) error {
	return nil
}

/* =====================
 Quoter methods.
===================== */

func (q *Quoter) Addquote(e *data.DataEndpoint, ev cmd.Event) error {
	nick := ev.Nick()
	quote := ev.GetArg("quote")
	if len(quote) == 0 {
		return nil
	}

	ev.Close()

	err := q.db.AddQuote(nick, quote)
	if err != nil {
		e.Noticef(nick, "\x02Quote:\x02 %v", err)
	}
	e.Notice(nick, "\x02Quote:\x02 Added.")
	return nil
}

func (q *Quoter) Delquote(e *data.DataEndpoint, ev cmd.Event) error {
	nick := ev.Nick()
	id, err := strconv.Atoi(ev.GetArg("id"))
	ev.Close()

	if err != nil {
		e.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}
	if did, err := q.db.DelQuote(int(id)); err != nil {
		e.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else if !did {
		e.Notice(nick, "\x02Quote:\x02 Could not find quote %d.", id)
	} else {
		e.Noticef(nick, "\x02Quote:\x02 Quote %d deleted.", id)
	}
	return nil
}

func (q *Quoter) Editquote(e *data.DataEndpoint, ev cmd.Event) error {
	nick := ev.Nick()
	quote := ev.GetArg("quote")
	id, err := strconv.Atoi(ev.GetArg("id"))
	ev.Close()

	if len(quote) == 0 {
		return nil
	}

	if err != nil {
		e.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}
	if did, err := q.db.EditQuote(int(id), quote); err != nil {
		e.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else if !did {
		e.Notice(nick, "\x02Quote:\x02 Could not find quote %d.", id)
	} else {
		e.Noticef(nick, "\x02Quote:\x02 Quote %d updated.", id)
	}
	return nil
}

func (q *Quoter) Quote(e *data.DataEndpoint, ev cmd.Event) error {
	strid := ev.GetArg("id")
	nick := ev.Nick()
	targ := ev.Target()
	ev.Close()

	var quote string
	var id int
	var err error
	if len(strid) > 0 {
		getid, err := strconv.Atoi(strid)
		id = int(getid)
		if err != nil {
			e.Notice(nick, "\x02Quote:\x02 Not a valid id.")
			return nil
		}
		quote, err = q.db.GetQuote(id)
	} else {
		id, quote, err = q.db.RandomQuote()
	}
	if err != nil {
		e.Noticef(nick, "\x02Quote:\x02 %v", err)
		return nil
	}

	if len(quote) == 0 {
		respond(e, targ, nick, "\x02Quote:\x02 Does not exist.")
	} else {
		respond(e, targ, nick,
			fmt.Sprintf("\x02Quote (\x02#%d\x02):\x02 %s", id, quote))
	}
	return nil
}

func (q *Quoter) Quotes(e *data.DataEndpoint, ev cmd.Event) error {
	targ, nick := ev.Target(), ev.Nick()
	ev.Close()

	respond(e, targ, nick,
		fmt.Sprintf("\x02Quote:\x02 %d quote(s) in database.", q.db.NQuotes()))
	return nil
}

func (q *Quoter) Details(e *data.DataEndpoint, ev cmd.Event) error {
	nick := ev.Nick()
	id, err := strconv.Atoi(ev.GetArg("id"))
	targ := ev.Target()
	ev.Close()

	if err != nil {
		e.Notice(nick, "\x02Quote:\x02 Not a valid id.")
		return nil
	}

	if date, author, err := q.db.GetDetails(int(id)); err != nil {
		e.Noticef(nick, "\x02Quote:\x02 %v", err)
	} else {
		respond(e, targ, nick,
			fmt.Sprintf("\x02Quote (\x02#%d\x02):\x02 Created on %s by %s",
				id, time.Unix(date, 0).UTC().Format(dateFormat), author))
	}

	return nil
}

/* =====================
 Queryer methods.
===================== */

func (_ *Queryer) PrivmsgChannel(m *irc.Message, endpoint irc.Endpoint) {
	if out, err := query.YouTube(m.Message()); len(out) != 0 {
		endpoint.Privmsg(m.Target(), out)
	} else if err != nil {
		nick := m.Nick()
		endpoint.Notice(nick, err.Error())
	}
}

func (_ *Queryer) Calc(e *data.DataEndpoint, ev cmd.Event) error {
	q := ev.GetArg("query")
	nick, targ := ev.Nick(), ev.Target()
	ev.Close()

	if out, err := query.Wolfram(q, &queryConf); len(out) != 0 {
		out = sanitize(out)
		if targ := targ; isNick(targ) {
			e.Notice(nick, out)
		} else {
			e.Privmsg(targ, out)
		}
	} else if err != nil {
		e.Notice(nick, err.Error())
	}

	return nil
}

func (_ *Queryer) Google(e *data.DataEndpoint, ev cmd.Event) error {
	q := ev.GetArg("query")
	nick, targ := ev.Nick(), ev.Target()
	ev.Close()

	if out, err := query.Google(q); len(out) != 0 {
		out = sanitize(out)
		if targ := targ; isNick(targ) {
			e.Notice(nick, out)
		} else {
			e.Privmsg(targ, out)
		}
	} else if err != nil {
		e.Notice(nick, err.Error())
	}

	return nil
}

/* =====================
 Handler methods.
===================== */

func (h *Handler) Up(e *data.DataEndpoint, ev cmd.Event) error {
	user := ev.UserAccess
	ch := ev.TargetChannel
	if ch == nil {
		return fmt.Errorf("Must be a channel that the bot is on.")
	}
	chname := ch.Name()

	if !putPeopleUp(ev.Message, chname, user, e) {
		return cmd.MakeFlagsError("ov")
	}
	return nil
}

func (h *Handler) HandleRaw(m *irc.Message, endpoint irc.Endpoint) {
	if m.Name == irc.JOIN {
		end := endpoint.(*bot.ServerEndpoint)
		end.UsingStore(func(s *data.Store) {
			a := s.GetAuthedUser(endpoint.GetKey(), m.Sender)
			ch := m.Target()
			putPeopleUp(m, ch, a, endpoint)
		})
	}
}

func putPeopleUp(m *irc.Message, ch string,
	a *data.UserAccess, e irc.Endpoint) (did bool) {
	if a != nil {
		nick := m.Nick()
		if a.HasFlag(e.GetKey(), ch, 'o') {
			e.Sendf("MODE %s +o :%s", ch, nick)
			did = true
		} else if a.HasFlag(e.GetKey(), ch, 'v') {
			e.Sendf("MODE %s +v :%s", ch, nick)
			did = true
		}
	}
	return
}

func (h *Handler) PrivmsgUser(m *irc.Message, endpoint irc.Endpoint) {
	flds := strings.Fields(m.Message())
	if m.Nick() == "Aaron" && flds[0] == "do" {
		endpoint.Send(strings.Join(flds[1:], " "))
	}
}

func main() {
	log.SetOutput(os.Stdout)

	b, err := bot.NewBot(bot.ConfigureFile("config.yaml"))
	if err != nil {
		log.Fatalln("Error creating bot:", err)
	}
	defer b.Close()

	var handler Handler
	var queryer Queryer
	if conf := query.NewConfig("wolfid.toml"); conf != nil {
		queryConf = *conf
	}
	qdb, err := quotes.OpenDB("quotes.sqlite3")
	if err != nil {
		log.Fatalln("Error opening quotes db:", err)
	}
	defer qdb.Close()
	var quoter = Quoter{qdb}

	// Quote commands
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Retrieves a quote. Randomly selects a quote if no id is provided.",
		"quote",
		&quoter,
		cmd.PRIVMSG, cmd.ALL, "[id]",
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Shows the number of quotes in the database.",
		"quotes",
		&quoter,
		cmd.PRIVMSG, cmd.ALL,
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Gets the details for a specific quote.",
		"details",
		&quoter,
		cmd.PRIVMSG, cmd.ALL, "id",
	))
	b.RegisterCmd(cmd.MkCmd(
		"quote",
		"Adds a quote to the database.",
		"addquote",
		&quoter,
		cmd.PRIVMSG, cmd.ALL, "quote...",
	))
	b.RegisterCmd(cmd.MkAuthCmd(
		"quote",
		"Removes a quote from the database.",
		"delquote",
		&quoter,
		cmd.PRIVMSG, cmd.ALL, 0, "Q", "id",
	))
	b.RegisterCmd(cmd.MkAuthCmd(
		"quote",
		"Edits an existing quote.",
		"editquote",
		&quoter,
		cmd.PRIVMSG, cmd.ALL, 0, "Q", "id", "quote...",
	))

	// Queryer commands
	b.Register(irc.PRIVMSG, &queryer)
	b.RegisterCmd(cmd.MkCmd(
		"query",
		"Submits a query to Google.",
		"google",
		&queryer,
		cmd.PRIVMSG, cmd.ALL, "query...",
	))
	b.RegisterCmd(cmd.MkCmd(
		"query",
		"Submits a query to Wolfram Alpha.",
		"calc",
		&queryer,
		cmd.PRIVMSG, cmd.ALL, "query...",
	))

	// Handler commands
	b.Register(irc.PRIVMSG, &handler)
	b.Register(irc.JOIN, &handler)
	b.RegisterCmd(cmd.MkAuthCmd(
		"simple",
		"Gives the user ops or voice if they have o or v flags respectively.",
		"up",
		&handler,
		cmd.PRIVMSG, cmd.ALL, 0, "", "#chan",
	))

	end := b.Start()

	input, quit := make(chan int), make(chan os.Signal, 2)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input <- 0
	}()

	signal.Notify(quit, os.Interrupt, os.Kill)

	stop := false
	for !stop {
		select {
		case <-input:
			b.Stop()
			stop = true
		case <-quit:
			b.Stop()
			stop = true
		case err, ok := <-end:
			log.Println("Server death:", err)
			stop = !ok
		}
	}

	log.Println("Shutting down...")
	<-time.After(1 * time.Second)
}
