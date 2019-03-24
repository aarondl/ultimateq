package remote

import (
	"context"
	"sync"
	"time"

	"github.com/aarondl/ultimateq/dispatch"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"

	"github.com/aarondl/ultimateq/api"
)

// grpcClient.RegisterCmd(bla bla bla)
// remote.NewClient(grpcClient) - Begins calling it
//
// The client runs two methods constantly (blocking):
// HandleEvents
//  This dispatches to the things registered with Register
// HandleCommands
//  This dispatches to the things registered with RegisterCmd
//
// On the server:
// The server's responses to HandleEvents and HandleCommands
// is a bidirectional stream. All messages for a given event handler
// are sent to a specific opened stream.
//
// Essentially when a user registers an event/cmd they are given an id
// that links not only their irc.Writer back to the server, but their
// bot-registered-event back to the correct grpc server stream

// Client helps handle event and command dispatching remotely.
type Client struct {
	client    api.ExtClient
	extension string

	mut      sync.RWMutex
	events   map[uint64]dispatch.Handler
	commands map[uint64]cmd.Handler
}

// NewClient returns a new dispatcher for extensions.
func NewClient(extension string, client api.ExtClient) *Client {
	r := &Client{
		extension: extension,
		client:    client,
		events:    make(map[uint64]dispatch.Handler),
		commands:  make(map[uint64]cmd.Handler),
	}

	return r
}

type remoteIRCWriter struct {
	client api.ExtClient
	extID  string
	netID  string
}

func (r remoteIRCWriter) Write(b []byte) (n int, err error) {
	writeReq := &api.WriteRequest{
		Ext: r.extID,
		Net: r.netID,
		Msg: &api.RawIRC{Msg: b},
	}

	_, err = r.client.Write(context.Background(), writeReq)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func newWriter(client api.ExtClient, extID, netID string) irc.Writer {
	return irc.Helper{
		Writer: remoteIRCWriter{client: client, extID: extID, netID: netID},
	}
}

// Listen for events and commands and dispatch them to handlers. It blocks
// forever on its two listening goroutines.
func (c *Client) Listen() error {
	var eventIDs, cmdIDs []uint64
	c.mut.RLock()
	for id := range c.events {
		eventIDs = append(eventIDs, id)
	}
	for id := range c.commands {
		cmdIDs = append(cmdIDs, id)
	}
	c.mut.RUnlock()

	evSub := &api.SubscriptionRequest{Ext: c.extension, Ids: eventIDs}
	cmdSub := &api.SubscriptionRequest{Ext: c.extension, Ids: cmdIDs}

	evStream, err := c.client.Events(context.Background(), evSub)
	if err != nil {
		return err
	}

	cmdStream, err := c.client.Commands(context.Background(), cmdSub)
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	var evErr, cmdErr error

	go func() {
		for {
			ircEventResp, err := evStream.Recv()
			if err != nil {
				evErr = err
				break
			}

			writer := newWriter(c.client, c.extension, ircEventResp.Event.NetworkId)

			c.mut.RLock()
			handler := c.events[ircEventResp.Id]
			c.mut.RUnlock()

			if handler == nil {
				// How did this happen?
				continue
			}

			ev := &irc.Event{
				Name:      ircEventResp.Event.Name,
				Sender:    ircEventResp.Event.Sender,
				Args:      ircEventResp.Event.Args,
				Time:      time.Unix(ircEventResp.Event.Time, 0),
				NetworkID: ircEventResp.Event.NetworkId,
			}

			go handler.Handle(writer, ev)
		}

		wg.Done()
	}()

	go func() {
		for {
			cmdEventResp, err := cmdStream.Recv()
			if err != nil {
				cmdErr = err
				break
			}

			writer := newWriter(c.client, c.extension, cmdEventResp.Event.IrcEvent.NetworkId)

			c.mut.RLock()
			handler := c.commands[cmdEventResp.Id]
			c.mut.RUnlock()

			if handler == nil {
				// How did this happen?
				continue
			}

			ircEvent := cmdEventResp.Event.IrcEvent
			iev := &irc.Event{
				Name:      ircEvent.Name,
				Sender:    ircEvent.Sender,
				Args:      ircEvent.Args,
				Time:      time.Unix(ircEvent.Time, 0),
				NetworkID: ircEvent.NetworkId,
			}

			ev := &cmd.Event{
				Event: iev,
				Args:  cmdEventResp.Event.Args,
			}

			go handler.Cmd(cmdEventResp.Name, writer, ev)
		}

		wg.Done()
	}()

	wg.Wait()

	if evErr != nil {
		return evErr
	}

	if cmdErr != nil {
		return cmdErr
	}

	return nil
}

// Register an event handler with the bot
func (c *Client) Register(network string, channel string, event string, handler dispatch.Handler) (uint64, error) {
	req := &api.RegisterRequest{
		Ext:     c.extension,
		Network: network,
		Channel: channel,
		Event:   event,
	}

	resp, err := c.client.Register(context.Background(), req)
	if err != nil {
		return 0, err
	}

	c.mut.Lock()
	c.events[resp.Id] = handler
	c.mut.Unlock()

	return resp.Id, nil
}

// RegisterCmd with the bot
func (c *Client) RegisterCmd(network string, channel string, command *cmd.Command) (uint64, error) {
	req := &api.RegisterCmdRequest{
		Ext:     c.extension,
		Network: network,
		Channel: channel,
		Cmd: &api.Cmd{
			Name:        command.Name,
			Ext:         command.Extension,
			Desc:        command.Description,
			Kind:        int32(command.Kind),
			Scope:       int32(command.Scope),
			Args:        command.Args,
			RequireAuth: command.RequireAuth,
			ReqLevel:    int32(command.ReqLevel),
			ReqFlags:    command.ReqFlags,
		},
	}

	resp, err := c.client.RegisterCmd(context.Background(), req)
	if err != nil {
		return 0, err
	}

	c.mut.Lock()
	c.commands[resp.Id] = command.Handler
	c.mut.Unlock()

	return resp.Id, nil
}

// Unregister an event handler
func (c *Client) Unregister(id uint64) (bool, error) {
	resp, err := c.client.Unregister(context.Background(), &api.UnregisterRequest{Id: id})
	if err != nil {
		return false, err
	}

	c.mut.Lock()
	delete(c.events, id)
	c.mut.Unlock()

	return resp.Ok, nil
}

// UnregisterCmd from the bot
func (c *Client) UnregisterCmd(id uint64) (bool, error) {
	resp, err := c.client.UnregisterCmd(context.Background(), &api.UnregisterRequest{Id: id})
	if err != nil {
		return false, err
	}

	c.mut.Lock()
	delete(c.commands, id)
	c.mut.Unlock()

	return resp.Ok, nil
}
