package data

import (
	"github.com/aarondl/ultimateq/irc"
	"strings"
)

const (
	// banMode is the universal irc mode for bans
	banMode = 'b'
)

// Channel encapsulates all the data associated with a channel.
type Channel struct {
	name  string
	topic string
	*ChannelModes
}

// CreateChannel instantiates a channel object.
func CreateChannel(name string,
	kinds *ChannelModeKinds, userKinds *UserModeKinds) *Channel {

	if len(name) == 0 {
		return nil
	}

	return &Channel{
		name:         name,
		ChannelModes: CreateChannelModes(kinds, userKinds),
	}
}

// GetName gets the name of the channel.
func (c *Channel) GetName() string {
	return c.name
}

// Topic sets the topic of the channel.
func (c *Channel) Topic(topic string) {
	c.topic = topic
}

// GetTopic gets the topic of the channel.
func (c *Channel) GetTopic() string {
	return c.topic
}

// IsBanned checks a mask to see if it's banned.
func (c *Channel) IsBanned(mask irc.Mask) bool {
	if !strings.ContainsAny(string(mask), "!@") {
		mask += "!@"
	}
	bans := c.GetAddresses(banMode)
	for i := 0; i < len(bans); i++ {
		if irc.WildMask(bans[i]).Match(mask) {
			return true
		}
	}

	return false
}

// Bans sets the bans of the channel.
func (c *Channel) Bans(bans []string) {
	delete(c.modes, banMode)
	for i := 0; i < len(bans); i++ {
		c.setAddress(banMode, bans[i])
	}
}

// AddBans adds to the channel's bans.
func (c *Channel) AddBan(ban string) {
	c.setAddress(banMode, ban)
}

// GetBans gets the bans of the channel.
func (c *Channel) GetBans() []string {
	getBans := c.GetAddresses(banMode)
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

// DeletBan deletes a ban from the list.
func (c *Channel) DeleteBan(ban string) {
	c.unsetAddress(banMode, ban)
}

// String returns the name of the channel.
func (c *Channel) String() string {
	return c.name
}

// DeleteBans deletes all bans that match a mask.
func (c *Channel) DeleteBans(mask irc.Mask) {
	bans := c.GetAddresses(banMode)
	if 0 == len(bans) {
		return
	}

	if !strings.ContainsAny(string(mask), "!@") {
		mask += "!@"
	}

	toRemove := make([]string, 0, 1) // Assume only one ban will match.
	for i := 0; i < len(bans); i++ {
		if irc.WildMask(bans[i]).Match(mask) {
			toRemove = append(toRemove, bans[i])
		}
	}

	for i := 0; i < len(toRemove); i++ {
		c.unsetAddress(banMode, toRemove[i])
	}
}
