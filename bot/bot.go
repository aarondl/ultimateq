package bot

import (
	"bufio"
	"bytes"
	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/inet"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/parse"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
)

const (
	// defaultChanTypes is used to create a barebones ProtoCaps until
	// the real values can be filled in by the handler.
	defaultChanTypes = "#&~"
)

// Bot is used as an interface into the packages under ultimateq
type Bot struct {
	dispatcher *dispatch.Dispatcher
	client     *inet.IrcClient
	caps       *irc.ProtoCaps
	config     *fakeBotConfig
}

type Handler struct {
}

func (h Handler) HandleRaw(msg *irc.IrcMessage) {
	if msg.Name == "PING" {
	}
}

// ConnProvider transforms a server:port string into a // net.Conn
type ConnProvider func(string) (net.Conn, error)

// TODO: Remove this in favor of config.go's configuration.
type fakeBotConfig struct {
	server, nick, username, host, fullname string
	port                                   uint
}

// CreateBot initializes all the package helper types for use within the bot.
func CreateBot(config fakeBotConfig, prov ConnProvider) (*Bot, error) {
	b := Bot{
		caps:   &irc.ProtoCaps{Chantypes: defaultChanTypes},
		config: &config,
	}

	var err error
	if err = b.createDispatcher(); err != nil {
		return nil, err
	}
	if err = b.createIrcClient(prov); err != nil {
		return nil, err
	}

	return &b, nil
}

// createDispatcher uses the bot's current ProtoCaps to create a dispatcher.
func (b *Bot) createDispatcher() error {
	var err error
	b.dispatcher, err = dispatch.CreateRichDispatcher(b.caps)
	if err != nil {
		return err
	}
	return nil
}

// createIrcClient connects to the configured server, and creates an IrcClient
// for use with that connection.
func (b *Bot) createIrcClient(provider ConnProvider) error {
	var conn net.Conn
	var err error

	port := strconv.Itoa(int(b.config.port))
	server := b.config.server + ":" + port

	if provider == nil {
		if conn, err = net.Dial("tcp", server); err != nil {
			return err
		}
	} else {
		if conn, err = provider(server); err != nil {
			return err
		}
	}

	b.client = inet.CreateIrcClient(conn)
	return nil
}

func (b *Bot) dostuff() {
	var waiter sync.WaitGroup
	waiter.Add(1)
	go func() {
		for {
			msg, ok := b.client.ReadMessage()
			if !ok {
				log.Println("Socket closed.")
				break
			}
			ircMsg, err := parse.Parse(string(msg))
			if err != err {
				log.Println("Error parsing message:", err)
			} else {
				b.dispatcher.Dispatch(ircMsg.Name, ircMsg)
			}
		}
		waiter.Done()
	}()

	// Main goroutine will be reading from stdin and writing our commands
	// to the server
	reader := bufio.NewReader(os.Stdin)
	for {
		str, err := reader.ReadBytes('\n')
		if err != nil {
			log.Println("Error while getting input:", err)
			break
		}

		str = str[:len(str)-1]

		if 0 == bytes.Compare(str, []byte("QUIT")) {
			b.client.Write([]byte("QUIT :Quitting"))
			break
		} else {
			b.client.Write(str)
		}
	}

	// Exit and wait for all goroutines to return
	log.Println("Exiting.")
	b.client.Close()
	b.client.Wait()
	waiter.Wait()
}
