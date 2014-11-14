package data

import (
	"strings"
	"testing"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/aarondl/ultimateq/irc"
)

func init() {
	// Speed up bcrypt for tests.
	StoredUserPwdCost = bcrypt.MinCost
	// Invalidate the Store cache enough to be testable.
	nMaxCache = 1
}

var (
	uname    = "user"
	password = "pass"
	host     = `nick!user@host`
	network  = "irc.network.net"
	users    = []string{"nick1!user1@host1", "nick2!user2@host2"}
	nicks    = []string{"nick1", "nick2"}
	channels = []string{"#CHAN1", "#CHAN2"}
	channel  = "#CHAN1"

	self = Self{
		User: NewUser("me!my@host.com"),
	}

	netInfo = irc.NewNetworkInfo()
)

func TestState(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if st == nil {
		t.Error("Unexpected nil.")
	}
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if st.Self.ChannelModes == nil {
		t.Error("Unexpected nil.")
	}

	st, err = NewState(nil)
	if exp, got := err, errProtoCapsMissing; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	// Should die on creating kinds
	fakeCaps := &irc.NetworkInfo{}
	fakeCaps.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=#&", "PREFIX=(ov)@+",
	}})
	st, err = NewState(fakeCaps)
	if got := st; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}

	// Should die on creating user modes
	fakeCaps = &irc.NetworkInfo{}
	fakeCaps.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=#&", "CHANMODES=a,b,c,d",
	}})
	st, err = NewState(fakeCaps)
	if got := st; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}
}

func TestState_UpdateProtoCaps(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	fakeNetInfo := &irc.NetworkInfo{}
	fakeNetInfo.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=!", "PREFIX=(q)@", "CHANMODES=,,,q",
	}})
	fakeNetInfo.ParseMyInfo(&irc.Event{Args: []string{
		"nick", "irc.test.net", "test-12", "q", "abc",
	}})

	if exp, got := st.kinds.kinds['q'], 0; exp != got {
		t.Fatalf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.umodes.GetModeBit('q'), byte(0); exp != got {
		t.Fatalf("Expected: %v, got: %v", exp, got)
	}
	st.SetNetworkInfo(fakeNetInfo)
	if exp, got := st.kinds.kinds['q'], 0; exp == got {
		t.Fatalf("Did not want: %v, got: %v", exp, got)
	}
	if exp, got := st.umodes.GetModeBit('q'), byte(0); exp == got {
		t.Fatalf("Did not want: %v, got: %v", exp, got)
	}
}

func TestState_GetUser(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetUser(users[1]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addUser(users[0])
	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if got := st.GetUser(users[1]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addUser(users[1])
	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if st.GetUser(users[1]) == nil {
		t.Error("Unexpected nil.")
	}

	st, err = NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	oldHost := "nick!user@host.com"
	newHost := "nick!user@host.net"
	st.addUser(oldHost)
	if exp, got := st.GetUser(oldHost).Host(), oldHost; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetUser(newHost).Host(), newHost; exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}
	st.addUser(newHost)
	if exp, got := st.GetUser(oldHost).Host(), oldHost; exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}
	if exp, got := st.GetUser(newHost).Host(), newHost; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_GetChannel(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if got := st.GetChannel(channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetChannel(channels[1]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addChannel(channels[0])
	if st.GetChannel(channels[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if got := st.GetChannel(channels[1]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addChannel(channels[1])
	if st.GetChannel(channels[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if st.GetChannel(channels[1]) == nil {
		t.Error("Unexpected nil.")
	}
}

func TestState_GetUsersChannelModes(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addUser(users[0])
	if got := st.GetUsersChannelModes(users[0], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addChannel(channels[0])
	if got := st.GetUsersChannelModes(users[0], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}

	st.addToChannel(users[0], channels[0])
	if st.GetUsersChannelModes(users[0], channels[0]) == nil {
		t.Error("Unexpected nil.")
	}
}

func TestState_GetNUsers(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := st.GetNUsers(), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addUser(users[0])
	st.addUser(users[0]) // Test that adding a user twice does nothing.
	if exp, got := st.GetNUsers(), 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addUser(users[1])
	if exp, got := st.GetNUsers(), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_GetNChannels(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := st.GetNChannels(), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[0]) // Test that adding a channel twice does nothing.
	if exp, got := st.GetNChannels(), 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addChannel(channels[1])
	if exp, got := st.GetNChannels(), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_GetNUserChans(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := st.GetNUserChans(users[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetNUserChans(users[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	if exp, got := st.GetNUserChans(users[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetNUserChans(users[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[0]) // Test no duplicate adds.
	st.addToChannel(users[0], channels[1])
	st.addToChannel(users[1], channels[0])
	if exp, got := st.GetNUserChans(users[0]), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetNUserChans(users[1]), 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_GetNChanUsers(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if exp, got := st.GetNChanUsers(channels[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetNChanUsers(channels[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	if exp, got := st.GetNChanUsers(channels[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetNChanUsers(channels[0]), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	st.addToChannel(users[1], channels[0])
	if exp, got := st.GetNChanUsers(channels[0]), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetNChanUsers(channels[1]), 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_EachUser(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	i := 0
	st.EachUser(func(u *User) {
		has := false
		for _, user := range users {
			if user == u.Host() {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		i++
	})
	if exp, got := i, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_EachChannel(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	i := 0
	st.EachChannel(func(ch *Channel) {
		has := false
		for _, channel := range channels {
			if channel == ch.String() {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		i++
	})
	if exp, got := i, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_EachUserChan(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	i := 0
	st.EachUserChan(users[0], func(uc *UserChannel) {
		has := false
		for _, channel := range channels {
			if channel == uc.Channel.String() {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		i++
	})
	if exp, got := i, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_EachChanUser(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	i := 0
	st.EachChanUser(channels[0], func(cu *ChannelUser) {
		has := false
		for _, user := range users {
			if user == cu.User.Host() {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		i++
	})
	if exp, got := i, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_GetUsers(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	if exp, got := len(st.GetUsers()), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, u := range st.GetUsers() {
		has := false
		for _, user := range users {
			if user == u {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_GetChannels(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	if exp, got := len(st.GetChannels()), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, ch := range st.GetChannels() {
		has := false
		for _, channel := range channels {
			if channel == ch {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_GetUserChans(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if got := st.GetUserChans(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	if exp, got := len(st.GetUserChans(users[0])), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, uc := range st.GetUserChans(users[0]) {
		has := false
		for _, channel := range channels {
			if channel == uc {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_GetChanUsers(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if got := st.GetChanUsers(channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	if exp, got := len(st.GetChanUsers(channels[0])), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, cu := range st.GetChanUsers(channels[0]) {
		has := false
		for _, user := range users {
			if user == cu {
				has = true
				break
			}
		}
		if exp, got := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_IsOn(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	st.addChannel(channels[0])
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	st.addUser(users[0])
	st.addToChannel(users[0], channels[0])
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
}

func TestState_UpdateNick(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}

	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if got := st.GetUser(users[1]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[1], channels[0])
	}
	for nick := range st.channelUsers[strings.ToLower(channels[0])] {
		if exp, got := nick, nicks[0]; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}

	u := st.Update(ev)
	if u.Nick[0] != users[0] {
		t.Errorf("Expected: %v, got: %v", users[0], u.Nick[0])
	}
	exp := strings.Replace(users[0], nicks[0], nicks[1], -1)
	if exp != u.Nick[1] {
		t.Errorf("Expected: %v, got: %v", exp, u.Nick[1])
	}

	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if st.GetUser(users[1]) == nil {
		t.Error("Unexpected nil.")
	}
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[1], channels[0])
	}
	for nick := range st.channelUsers[strings.ToLower(channels[0])] {
		if exp, got := nick, nicks[1]; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}

	ev.Sender = users[0]
	ev.Args = []string{"newnick"}
	st.Update(ev)
	if st.GetUser("newnick") == nil {
		t.Error("Unexpected nil.")
	}
	if got := st.GetUser(nicks[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
}

func TestState_UpdateNickSelfNilMaps(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}
	st.addUser(users[0])
	st.Update(ev)

	_, ok := st.userChannels[nicks[0]]
	if exp, got := ok, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	_, ok = st.userChannels[nicks[1]]
	if exp, got := ok, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateJoin(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.JOIN,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	st.addChannel(channels[0])
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	u := st.Update(ev)
	if u.Seen != users[0] {
		t.Error("Expected %v to be seen.", users[0])
	}
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}

	st, _ = NewState(netInfo)
	st.Self = self
	st.addChannel(channels[0])

	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	st.Update(ev)
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
}

func TestState_UpdateJoinSelf(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.JOIN,
		Sender: self.Host(),
		Args:   []string{channels[0]},
	}

	if got := st.GetChannel(channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if st.IsOn(st.Self.Nick(), channels[0]) {
		t.Errorf("Expected %v to not be on %v", st.Self.Nick(), channels[0])
	}
	u := st.Update(ev)
	if len(u.Seen) > 0 {
		t.Error("Expected self not to be seen.")
	}
	if st.GetChannel(channels[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if !st.IsOn(st.Self.Nick(), channels[0]) {
		t.Errorf("Expected %v to be on %v", st.Self.Nick(), channels[0])
	}
}

func TestState_UpdatePart(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.PART,
		Sender: users[0],
		Args:   []string{channels[0]},
	}

	st.addUser(users[0])
	st.addUser(users[1])

	// Test coverage, make sure adding to a channel that doesn't exist does
	// nothing.
	st.addToChannel(users[0], channels[0])
	if st.IsOn(users[0], channels[0]) {
		t.Error("Expected the user to not be on the channel.")
	}

	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	st.addToChannel(users[0], channels[1])

	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[0])
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v.", users[1], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if st.IsOn(users[1], channels[1]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[1])
	}

	u := st.Update(ev)
	if len(u.Unseen) > 0 {
		t.Errorf("Did not expect anyone to be unseen.")
	}
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[0], channels[0])
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v.", users[1], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if st.IsOn(users[1], channels[1]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[1])
	}

	ev.Sender = users[1]
	st.Update(ev)
	if u.Unseen != users[1] {
		t.Errorf("Expected %v to be unseen.", users[1])
	}
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[0], channels[0])
	}
	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if st.IsOn(users[1], channels[1]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[1])
	}

	ev.Sender = users[0]
	ev.Args[0] = channels[1]
	st.Update(ev)
	if u.Unseen != users[0] {
		t.Errorf("Expected %v to be unseen.", users[0])
	}

	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[0], channels[0])
	}
	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[0])
	}
	if st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to not be on %v.", users[0], channels[1])
	}
	if st.IsOn(users[1], channels[1]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[1])
	}
}

func TestState_UpdatePartSelf(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.PART,
		Sender: self.Host(),
		Args:   []string{channels[0]},
	}

	st.addUser(users[0])
	st.addUser(self.Host())
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	st.addToChannel(self.Nick(), channels[0])

	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if !st.IsOn(self.Nick(), channels[0]) {
		t.Errorf("Expected %v to be on %v.", self.Nick(), channels[0])
	}
	st.Update(ev)
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[0], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if st.IsOn(self.Nick(), channels[0]) {
		t.Errorf("Expected %v to not be on %v.", self.Nick(), channels[0])
	}
}

func TestState_UpdateQuit(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.QUIT,
		Sender: users[0],
		Args:   []string{"quit message"},
	}

	// Test Quitting when we don't know the user
	st.Update(ev)
	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}

	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[1], channels[0])
	}
	if st.GetUser(users[1]) == nil {
		t.Error("Unexpected nil.")
	}

	st.Update(ev)

	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[1], channels[0])
	}
	if st.GetUser(users[1]) == nil {
		t.Error("Unexpected nil.")
	}

	ev.Sender = users[1]
	st.Update(ev)

	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[1], channels[0])
	}
	if got := st.GetUser(users[1]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
}

func TestState_UpdateKick(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], users[0]},
	}

	st.addUser(users[0])
	st.addUser(users[1])

	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
	st.Update(ev)
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
}

func TestState_UpdateKickSelf(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], st.Self.Nick()},
	}

	st.addUser(st.Self.Host())
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	if st.GetChannel(channels[0]) == nil {
		t.Error("Unexpected nil.")
	}
	st.Update(ev)
	if got := st.GetChannel(channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
}

func TestState_UpdateMode(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.MODE,
		Sender: users[0],
		Args: []string{channels[0],
			"+ovmb-vn", nicks[0], nicks[0], "*!*mask", nicks[1],
		},
		NetworkInfo: netInfo,
	}

	fail := st.GetUsersChannelModes(users[0], channels[0])
	if got := fail; got != nil {
		t.Error("Expected: %v to be nil.", got)
	}

	st.addChannel(channels[0])
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	u1modes := st.GetUsersChannelModes(users[0], channels[0])
	u2modes := st.GetUsersChannelModes(users[1], channels[0])
	u2modes.SetMode('v')
	st.GetChannel(channels[0]).Set("n")

	if exp, got := st.GetChannel(channels[0]).IsSet("n"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetChannel(channels[0]).IsSet("mb"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u1modes.HasMode('o'), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u1modes.HasMode('v'), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u2modes.HasMode('v'), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.GetChannel(channels[0]).IsSet("n"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetChannel(channels[0]).IsSet("mb *!*mask"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u1modes.HasMode('o'), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u1modes.HasMode('v'), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := u2modes.HasMode('v'), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateModeSelf(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self.User = self.User
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:        irc.MODE,
		Sender:      self.Host(),
		Args:        []string{self.Nick(), "+i-o"},
		NetworkInfo: netInfo,
	}

	st.Self.Set("o")

	if exp, got := st.Self.IsSet("i"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.Self.IsSet("o"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.Self.IsSet("i"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.Self.IsSet("o"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateTopic(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.TOPIC,
		Sender: users[1],
		Args:   []string{channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	if exp, got := st.GetChannel(channels[0]).Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.GetChannel(channels[0]).Topic(), "topic topic"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateRplTopic(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.RPL_TOPIC,
		Sender: network,
		Args:   []string{self.Nick(), channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	if exp, got := st.GetChannel(channels[0]).Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.GetChannel(channels[0]).Topic(), "topic topic"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateEmptyTopic(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.TOPIC,
		Sender: users[1],
		Args:   []string{channels[0], ""},
	}

	ch := st.addChannel(channels[0])
	ch.SetTopic("topic topic")

	if exp, got := st.GetChannel(channels[0]).Topic(), "topic topic"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.GetChannel(channels[0]).Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdatePrivmsg(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:        irc.PRIVMSG,
		Sender:      users[0],
		Args:        []string{channels[0]},
		NetworkInfo: netInfo,
	}

	st.addChannel(channels[0])

	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetUsersChannelModes(users[0], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if st.GetUsersChannelModes(users[0], channels[0]) == nil {
		t.Error("Unexpected nil.")
	}

	ev.Sender = network
	size := len(st.users)
	st.Update(ev)
	if exp, got := len(st.users), size; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateNotice(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:        irc.NOTICE,
		Sender:      users[0],
		Args:        []string{channels[0]},
		NetworkInfo: netInfo,
	}

	st.addChannel(channels[0])

	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetUsersChannelModes(users[0], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if st.GetUsersChannelModes(users[0], channels[0]) == nil {
		t.Error("Unexpected nil.")
	}

	ev.Sender = network
	size := len(st.users)
	st.Update(ev)
	if exp, got := len(st.users), size; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateWelcome(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}
	ev := &irc.Event{
		Name:   irc.RPL_WELCOME,
		Sender: network,
		Args:   []string{nicks[1], "Welcome to"},
	}

	st.Update(ev)
	if exp, got := st.Self.Host(), nicks[1]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.users[nicks[1]].Host(), st.Self.Host(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	ev = &irc.Event{
		Name:   irc.RPL_WELCOME,
		Sender: network,
		Args:   []string{nicks[1], "Welcome to " + users[1]},
	}

	st.Update(ev)
	if exp, got := st.Self.Host(), users[1]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.users[nicks[1]].Host(), st.Self.Host(); exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateRplNamereply(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.RPL_NAMREPLY,
		Sender: network,
		Args: []string{
			self.Nick(), "=", channels[0],
			"@" + nicks[0] + " +" + nicks[1] + " " + self.Nick(),
		},
	}

	st.addChannel(channels[0])

	if got := st.GetUsersChannelModes(users[0], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetUsersChannelModes(users[1], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetUsersChannelModes(self.Nick(), channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	got := st.GetUsersChannelModes(users[0], channels[0]).String()
	if got != "o" {
		t.Errorf(`Expected: "o", got: %q`, got)
	}
	got = st.GetUsersChannelModes(users[1], channels[0]).String()
	if got != "v" {
		t.Errorf(`Expected: "v", got: %q`, got)
	}
	got = st.GetUsersChannelModes(self.Nick(), channels[0]).String()
	if len(got) > 0 {
		t.Errorf(`Expected: empty string, got: %q`, got)
	}
}

func TestState_RplWhoReply(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.RPL_WHOREPLY,
		Sender: network,
		Args: []string{
			self.Nick(), channels[0], irc.Username(users[0]),
			irc.Hostname(users[0]), "*.network.net", nicks[0], "Hx@d",
			"3 real name",
		},
	}

	st.addChannel(channels[0])

	if got := st.GetUser(users[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	if got := st.GetUsersChannelModes(users[0], channels[0]); got != nil {
		t.Error("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	if st.GetUser(users[0]) == nil {
		t.Error("Unexpected nil.")
	}
	if exp, got := st.GetUser(users[0]).Host(), users[0]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := st.GetUser(users[0]).Realname(), "real name"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	got := st.GetUsersChannelModes(users[0], channels[0]).String()
	if got != "o" {
		t.Errorf(`Expected: "o", got: %q`, got)
	}

}

func TestState_UpdateRplMode(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.RPL_CHANNELMODEIS,
		Sender: network,
		Args:   []string{self.Nick(), channels[0], "+ntzl", "10"},
	}

	st.addChannel(channels[0])
	if exp, got := st.GetChannel(channels[0]).IsSet("ntzl 10"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.GetChannel(channels[0]).IsSet("ntzl 10"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateRplBanlist(t *testing.T) {
	t.Parallel()

	st, err := NewState(netInfo)
	st.Self = self
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	ev := &irc.Event{
		Name:   irc.RPL_BANLIST,
		Sender: network,
		Args: []string{self.Nick(), channels[0], nicks[0] + "!*@*", nicks[1],
			"1367197165"},
	}

	st.addChannel(channels[0])
	if exp, got := st.GetChannel(channels[0]).HasBan(nicks[0]+"!*@*"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if exp, got := st.GetChannel(channels[0]).HasBan(nicks[0]+"!*@*"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}
