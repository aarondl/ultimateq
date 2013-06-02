package data

import (
	"strings"
)

// Channel encapsulates all the data associated with a channel.
type Channel struct {
	name     string
	topic    string
	banmasks []string
	Modes    *Modeset
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

// Banmasks sets the banmasks of the channel.
func (c *Channel) Banmasks(banmasks []string) {
	c.banmasks = make([]string, len(banmasks))
	copy(c.banmasks, banmasks)
}

// GetBanmasks gets the banmasks of the channel.
func (c *Channel) GetBanmasks() []string {
	banmasks := make([]string, len(c.banmasks))
	copy(banmasks, c.banmasks)
	return banmasks
}

// IsBanned checks a mask to see if it's banned.
func (c *Channel) IsBanned(banmask string) bool {
	return false
}

// HasBanmask checks to see if a specific mask is present in the banlist.
func (c *Channel) HasBanmask(banmask string) bool {
	for i := 0; i < len(c.banmasks); i++ {
		if c.banmasks[i] == banmask {
			return true
		}
	}
	return false
}

// DeletBanmask deletes a banmask from the list.
func (c *Channel) DeleteBanmask(banmask string) bool {
	ln := len(c.banmasks)
	for i := 0; i < ln; i++ {
		if c.banmasks[i] == banmask {
			c.banmasks[i], c.banmasks[ln-1] = c.banmasks[ln-1], c.banmasks[i]
			c.banmasks = c.banmasks[:ln-1]
			return true
		}
	}

	return false
}
