package data

import (
	"strings"
)

// Channel encapsulates all the data associated with a channel.
type Channel struct {
	name  string
	topic string
	bans  []WildMask
	Modes *Modeset
}

// CreateChannel instantiates a channel object.
func CreateChannel(name string) *Channel {
	return &Channel{
		name:  strings.ToLower(name),
		Modes: CreateModeset(),
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
func (c *Channel) IsBanned(mask Mask) bool {
	if !strings.ContainsAny(string(mask), "!@") {
		mask += "!@"
	}

	for _, ban := range c.bans {
		if ban.Match(mask) {
			return true
		}
	}

	return false
}

// Bans sets the bans of the channel.
func (c *Channel) Bans(bans []WildMask) {
	c.bans = make([]WildMask, len(bans))
	copy(c.bans, bans)
}

// AddBans adds to the channel's bans.
func (c *Channel) AddBan(ban WildMask) {
	c.bans = append(c.bans, ban)
}

// GetBans gets the bans of the channel.
func (c *Channel) GetBans() []WildMask {
	bans := make([]WildMask, len(c.bans))
	copy(bans, c.bans)
	return bans
}

// HasBan checks to see if a specific mask is present in the banlist.
func (c *Channel) HasBan(ban WildMask) bool {
	for i := 0; i < len(c.bans); i++ {
		if c.bans[i] == ban {
			return true
		}
	}
	return false
}

// DeletBan deletes a ban from the list.
func (c *Channel) DeleteBan(ban WildMask) bool {
	ln := len(c.bans)
	for i := 0; i < ln; i++ {
		if c.bans[i] == ban {
			c.bans[i], c.bans[ln-1] = c.bans[ln-1], c.bans[i]
			c.bans = c.bans[:ln-1]
			return true
		}
	}

	return false
}
