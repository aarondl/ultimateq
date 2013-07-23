// The ultimateq bot framework.
package main

import (
	"bufio"
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/config"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/commander"
	"github.com/aarondl/ultimateq/irc"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
)

type Handler struct {
}

func (h *Handler) Command(cmd string, msg *irc.Message,
	endpoint *data.DataEndpoint, cdata *commander.CommandData) error {

	endpoint.Notice(cdata.User.GetNick(), "hello")
	return nil
}

func (h *Handler) PrivmsgUser(m *irc.Message, endpoint irc.Endpoint) {
	flds := strings.Fields(m.Message())
	if irc.Mask(m.Sender).GetNick() == "Aaron" && flds[0] == "do" {
		endpoint.Send(strings.Join(flds[1:], " "))
	}
}

func (h *Handler) PrivmsgChannel(m *irc.Message, endpoint irc.Endpoint) {
	if m.Message() == "hello" {
		endpoint.Privmsg(m.Target(), "Hello to you too!")
	}
	if irc.Mask(m.Sender).GetNick() == "Aaron" {
		end := endpoint.(*bot.ServerEndpoint)
		split := strings.Fields(m.Message())
		switch split[0] {
		case "whoami":
			end.UsingState(func(s *data.State) {
				end.Privmsgf(m.Target(), "I am (%v) on channels: %v",
					s.Self.GetFullhost(),
					strings.Join(s.GetUserChans(s.Self.GetFullhost()), ", "))
			})
		case "who":
			end.UsingState(func(s *data.State) {
				user := s.GetUser("Aaron")
				cu := s.GetUsersChannelModes("Aaron", m.Target())
				end.Privmsgf(m.Target(), "Aaron is: %v, modes on %v: %v",
					user.GetFullhost(), m.Target(), cu)
			})
		case "channels":
			end.UsingState(func(s *data.State) {
				channels := make([]string, 0, s.GetNChannels())
				s.EachChannel(func(c *data.Channel) {
					channels = append(channels, c.String())
				})
				end.Privmsgf(m.Target(), "Channels (%v)",
					strings.Join(channels, ", "))
			})
		case "users":
			end.UsingState(func(s *data.State) {
				users := make([]string, 0, s.GetNUsers())
				s.EachUser(func(u *data.User) {
					if u == nil {
						log.Println("There was a nil user.")
					} else {
						log.Println(u)
					}
					users = append(users, u.String())
				})
				end.Privmsgf(m.Target(), "Users (%v)",
					strings.Join(users, ", "))
			})
		case "userchans":
			user := m.Sender
			if len(split) > 1 {
				user = split[1]
			}
			end.UsingState(func(s *data.State) {
				if s.GetUser(user) == nil {
					end.Privmsgf(m.Target(), "No user: %v", user)
					return
				}
				channels := make([]string, 0, s.GetNChanUsers(user))
				s.EachUserChan(user, func(uc *data.UserChannel) {
					channels = append(channels, uc.Channel.String())
				})
				end.Privmsgf(m.Target(), "%v is on (%v)",
					user, strings.Join(channels, ", "))
			})
		case "chanusers":
			ch := m.Target()
			if len(split) > 1 {
				ch = split[1]
			}
			end.UsingState(func(s *data.State) {
				if s.GetChannel(ch) == nil {
					end.Privmsgf(m.Target(), "No channel: %v", ch)
					return
				}
				users := make([]string, 0, s.GetNChanUsers(ch))
				s.EachChanUser(ch, func(cu *data.ChannelUser) {
					users = append(users, cu.User.String())
				})
				end.Privmsgf(m.Target(), "Users on %v (%v)",
					ch, strings.Join(users, ", "))
			})
		case "modes":
			end.UsingState(func(s *data.State) {
				ch := s.GetChannel(m.Target())
				end.Privmsgf(m.Target(), "%v has modes: %v",
					ch, ch.ChannelModes)
			})
		case "topic":
			end.UsingState(func(s *data.State) {
				ch := s.GetChannel(m.Target())
				end.Privmsgf(m.Target(), "Topic is %v", ch.GetTopic())
			})
		}
	}
}

func conf(c *config.Config) *config.Config {
	c. // Defaults
		Nick("nobody__").
		Altnick("nobody_").
		Realname("there").
		Username("guy").
		Userhost("friend").
		NoReconnect(true)

	/*
		c. // First server
			Server("irc.gamesurge.net1").
			Host("irc.gamesurge.net").
			Nick("Aaron").
			Altnick("nobody1").
			ReconnectTimeout(5)
	*/

	c. // Second Server
		Server("irc.gamesurge.net2").
		Host("localhost").
		Nick("nobody2")

	return c
}

func main() {
	log.SetOutput(os.Stdout)

	b, err := bot.CreateBot(bot.ConfigureFile("config.yaml"))
	if err != nil {
		log.Fatalln(err)
	}
	defer b.Close()

	b.Register(irc.PRIVMSG, &Handler{})

	ers := b.Connect()
	if len(ers) != 0 {
		log.Println(ers)
		return
	}
	b.Start()

	input, dead, quit := make(chan int), make(chan int), make(chan os.Signal, 2)

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input <- 0
	}()
	go func() {
		b.WaitForHalt()
		dead <- 0
	}()

	signal.Notify(quit, os.Interrupt, os.Kill)

	select {
	case <-input:
	case <-dead:
	case <-quit:
	}

	b.Stop()
	b.Disconnect()
	log.Println("Shutting down...")
	<-time.After(1 * time.Second)
}
