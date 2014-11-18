package data

import (
	"strings"

	"github.com/aarondl/ultimateq/irc"
)

const (
	// banMode is the universal irc mode for bans
	banMode = 'b'
)

// Channel encapsulates all the data associated with a channel.
type Channel struct {
	name  string
	topic string
	ChannelModes
}

// NewChannel instantiates a channel object.
func NewChannel(name string, m *modeKinds) *Channel {
	if len(name) == 0 {
		return nil
	}

	return &Channel{
		name:         name,
		ChannelModes: NewChannelModes(m),
	}
}

// Clone copies the channel into new memory.
func (c *Channel) Clone() *Channel {
	return &Channel{c.name, c.topic, c.ChannelModes.Clone()}
}

// Name gets the name of the channel.
func (c *Channel) Name() string {
	return c.name
}

// SetTopic sets the topic of the channel.
func (c *Channel) SetTopic(topic string) {
	c.topic = topic
}

// Topic gets the topic of the channel.
func (c *Channel) Topic() string {
	return c.topic
}

// IsBanned checks a host to see if it's banned.
func (c *Channel) IsBanned(host irc.Host) bool {
	if !strings.ContainsAny(string(host), "!@") {
		host += "!@"
	}
	bans := c.Addresses(banMode)
	for i := 0; i < len(bans); i++ {
		if irc.Mask(bans[i]).Match(host) {
			return true
		}
	}

	return false
}

// SetBans sets the bans of the channel.
func (c *Channel) SetBans(bans []string) {
	delete(c.modes, banMode)
	for i := 0; i < len(bans); i++ {
		c.setAddress(banMode, bans[i])
	}
}

// AddBan adds to the channel's bans.
func (c *Channel) AddBan(ban string) {
	c.setAddress(banMode, ban)
}

// Bans gets the bans of the channel.
func (c *Channel) Bans() []string {
	getBans := c.Addresses(banMode)
	if getBans == nil {
		return nil
	}
	bans := make([]string, len(getBans))
	copy(bans, getBans)
	return bans
}

// HasBan checks to see if a specific mask is present in the banlist.
func (c *Channel) HasBan(ban string) bool {
	return c.isAddressSet(banMode, ban)
}

// DeleteBan deletes a ban from the list.
func (c *Channel) DeleteBan(ban string) {
	c.unsetAddress(banMode, ban)
}

// String returns the name of the channel.
func (c *Channel) String() string {
	return c.name
}

// DeleteBans deletes all bans that match a mask.
func (c *Channel) DeleteBans(mask irc.Host) {
	bans := c.Addresses(banMode)
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
		c.unsetAddress(banMode, toRemove[i])
	}
}
