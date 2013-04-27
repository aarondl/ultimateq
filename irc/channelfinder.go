package irc

import "regexp"

const (
	// nStringsAssumed is the number of channels assumed to be in each irc message
	// if this number is too small, there could be memory thrashing due to append
	nChannelsAssumed = 1
)

// channelRegexp stores a cached regexp generated
type ChannelFinder struct {
	channelRegexp *regexp.Regexp
}

// BuildRegex creates a channel regex safely using the types that are passed in.
func (c *ChannelFinder) BuildRegex(types string) error {
	safetypes := ""
	for _, c := range types {
		safetypes += string(`\`) + string(c)
	}
	regex, err := regexp.Compile(`[` + safetypes + `][^\s,]+`)
	if err == nil {
		c.channelRegexp = regex
	}
	return err
}

// FindChannels retrieves all the channels in the string using a cached regex
// created using ProtoCaps.
func (c *ChannelFinder) FindChannels(msg string) []string {
	channels := make([]string, 0, nChannelsAssumed)

	for _, v := range c.channelRegexp.FindAllString(msg, -1) {
		channels = append(channels, v)
	}

	return channels
}
