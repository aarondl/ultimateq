package data

import (
	"strings"

	"github.com/aarondl/ultimateq/api"
	"github.com/aarondl/ultimateq/irc"
)

const (
	// banMode is the universal irc mode for bans
	banMode = 'b'
)

// Channel encapsulates all the data associated with a channel.
type Channel struct {
	Name  string       `json:"name"`
	Topic string       `json:"topic"`
	Modes ChannelModes `json:"channel_modes"`
}

// NewChannel instantiates a channel object.
func NewChannel(name string, m *modeKinds) *Channel {
	if len(name) == 0 {
		return nil
	}

	return &Channel{
		Name:  name,
		Modes: NewChannelModes(m),
	}
}

// Clone deep copies this Channel.
func (c *Channel) Clone() *Channel {
	return &Channel{c.Name, c.Topic, c.Modes.Clone()}
}

// IsBanned checks a host to see if it's banned.
func (c *Channel) IsBanned(host irc.Host) bool {
	if !strings.ContainsAny(string(host), "!@") {
		host += "!@"
	}
	bans := c.Modes.Addresses(banMode)
	for i := 0; i < len(bans); i++ {
		if irc.Mask(bans[i]).Match(host) {
			return true
		}
	}

	return false
}

// SetBans sets the bans of the channel.
func (c *Channel) SetBans(bans []string) {
	delete(c.Modes.modes, banMode)
	for i := 0; i < len(bans); i++ {
		c.Modes.setAddress(banMode, bans[i])
	}
}

// AddBan adds to the channel's bans.
func (c *Channel) AddBan(ban string) {
	c.Modes.setAddress(banMode, ban)
}

// Bans gets the bans of the channel.
func (c *Channel) Bans() []string {
	getBans := c.Modes.Addresses(banMode)
	if getBans == nil {
		return nil
	}
	bans := make([]string, len(getBans))
	copy(bans, getBans)
	return bans
}

// HasBan checks to see if a specific mask is present in the banlist.
func (c *Channel) HasBan(ban string) bool {
	return c.Modes.isAddressSet(banMode, ban)
}

// DeleteBan deletes a ban from the list.
func (c *Channel) DeleteBan(ban string) {
	c.Modes.unsetAddress(banMode, ban)
}

// String returns the name of the channel.
func (c *Channel) String() string {
	return c.Name
}

// DeleteBans deletes all bans that match a mask.
func (c *Channel) DeleteBans(mask irc.Host) {
	bans := c.Modes.Addresses(banMode)
	if 0 == len(bans) {
		return
	}

	if !strings.ContainsAny(string(mask), "!@") {
		mask += "!@"
	}

	toRemove := make([]string, 0, 1) // Assume only one ban will match.
	for i := 0; i < len(bans); i++ {
		if irc.Mask(bans[i]).Match(mask) {
			toRemove = append(toRemove, bans[i])
		}
	}

	for i := 0; i < len(toRemove); i++ {
		c.Modes.unsetAddress(banMode, toRemove[i])
	}
}

// ToProto converts to a protocol buffer
func (c *Channel) ToProto() *api.StateChannel {
	ch := new(api.StateChannel)

	ch.Name = c.Name
	ch.Topic = c.Topic
	ch.Modes = c.Modes.ToProto()

	return ch
}
