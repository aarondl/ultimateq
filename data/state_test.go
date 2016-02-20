package data

import (
	"strings"
	"testing"

	"github.com/aarondl/ultimateq/irc"
	"golang.org/x/crypto/bcrypt"
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

	testNetInfo = irc.NewNetworkInfo()
)

func setupNewState() *State {
	st, err := NewState(testNetInfo)
	if err != nil {
		panic(err)
	}
	st.selfUser = NewUser("me!my@host.com")
	return st
}

func TestState(t *testing.T) {
	t.Parallel()

	st, err := NewState(testNetInfo)
	if st == nil {
		t.Error("Unexpected nil.")
	}
	if err != nil {
		t.Error("Unexpected Error:", err)
	}

	st, err = NewState(nil)
	if got, exp := err, errNetInfoMissing; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	// Should die on creating kinds
	fakeCaps := &irc.NetworkInfo{}
	fakeCaps.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=#&", "PREFIX=(ov)@+",
	}})
	st, err = NewState(fakeCaps)
	if got := st; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
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
		t.Errorf("Expected: %v to be nil.", got)
	}
	if err == nil {
		t.Error("Unexpected nil.")
	}
}

func TestState_UpdateProtoCaps(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	fakeNetInfo := &irc.NetworkInfo{}
	fakeNetInfo.ParseISupport(&irc.Event{Args: []string{
		"NICK", "CHANTYPES=!", "PREFIX=(q)@", "CHANMODES=,,,q",
	}})
	fakeNetInfo.ParseMyInfo(&irc.Event{Args: []string{
		"nick", "irc.test.net", "test-12", "q", "abc",
	}})

	if got, exp := st.kinds.channelModes['q'], 0; exp != got {
		t.Fatalf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := st.kinds.modeBit('q'), byte(0); exp != got {
		t.Fatalf("Expected: %v, got: %v", exp, got)
	}
	st.SetNetworkInfo(fakeNetInfo)
	if got, exp := st.kinds.channelModes['q'], 0; exp == got {
		t.Fatalf("Did not want: %v, got: %v", exp, got)
	}
	if got, exp := st.kinds.modeBit('q'), byte(0); exp == got {
		t.Fatalf("Did not want: %v, got: %v", exp, got)
	}
}

func TestState_User(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to not exist.", got)
	}
	if got, ok := st.User(users[1]); ok {
		t.Errorf("Expected: %v not exist.", got)
	}
	st.addUser(users[0])
	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if got, ok := st.User(users[1]); ok {
		t.Errorf("Expected: %v to not exist.", got)
	}
	st.addUser(users[1])
	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if _, ok := st.User(users[1]); !ok {
		t.Error("Unexpected nil.")
	}

	st = setupNewState()
	oldHost := "nick!user@host.com"
	newHost := "nick!user@host.net"
	st.addUser(oldHost)
	u, _ := st.User(oldHost)
	if got, exp := string(u.Host), oldHost; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	u, _ = st.User(oldHost)
	if got, exp := string(u.Host), newHost; exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}
	st.addUser(newHost)
	u, _ = st.User(oldHost)
	if got, exp := string(u.Host), oldHost; exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}
	u, _ = st.User(oldHost)
	if got, exp := string(u.Host), newHost; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_Channel(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	if got, ok := st.Channel(channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got, ok := st.Channel(channels[1]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.addChannel(channels[0])
	if _, ok := st.Channel(channels[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if got, ok := st.Channel(channels[1]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.addChannel(channels[1])
	if _, ok := st.Channel(channels[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if _, ok := st.Channel(channels[1]); !ok {
		t.Error("Unexpected nil.")
	}
}

func TestState_UserModes(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	st.addUser(users[0])
	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.addChannel(channels[0])
	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}

	st.addToChannel(users[0], channels[0])
	if _, ok := st.UserModes(users[0], channels[0]); !ok {
		t.Error("Unexpected nil.")
	}
}

func TestState_NUsers(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	if got, exp := st.NUsers(), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addUser(users[0])
	st.addUser(users[0]) // Test that adding a user twice does nothing.
	if got, exp := st.NUsers(), 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addUser(users[1])
	if got, exp := st.NUsers(), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_NChannels(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	if got, exp := st.NChannels(), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[0]) // Test that adding a channel twice does nothing.
	if got, exp := st.NChannels(), 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.addChannel(channels[1])
	if got, exp := st.NChannels(), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_NChannelsByUser(t *testing.T) {
	t.Parallel()

	var n int
	var ok bool

	st := setupNewState()
	if n, ok = st.NChannelsByUser(users[0]); ok {
		t.Error("Expected no user.")
	} else if n != 0 {
		t.Error("Expected no channels, got:", n)
	}
	if n, ok = st.NChannelsByUser(users[1]); ok {
		t.Error("Expected no user.")
	} else if n != 0 {
		t.Error("Expected no channels, got:", n)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	if n, ok = st.NChannelsByUser(users[0]); ok {
		t.Error("Expected no user.")
	} else if n != 0 {
		t.Error("Expected no channels, got:", n)
	}
	if n, ok = st.NChannelsByUser(users[1]); ok {
		t.Error("Expected no user.")
	} else if n != 0 {
		t.Error("Expected no channels, got:", n)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[0]) // Test no duplicate adds.
	st.addToChannel(users[0], channels[1])
	st.addToChannel(users[1], channels[0])
	if n, ok = st.NChannelsByUser(users[0]); !ok {
		t.Error("Expected a user.")
	} else if n != 2 {
		t.Error("Expected 2 channels, got:", n)
	}
	if n, ok = st.NChannelsByUser(users[1]); !ok {
		t.Error("Expected a user.")
	} else if n != 1 {
		t.Error("Expected 1 channels, got:", n)
	}
}

func TestState_NUsersByChannel(t *testing.T) {
	t.Parallel()

	var n int
	var ok bool

	st := setupNewState()
	if n, ok = st.NUsersByChannel(channels[0]); ok {
		t.Error("Expected no channel.")
	} else if n != 0 {
		t.Error("Expected no users, got:", n)
	}
	if n, ok = st.NUsersByChannel(channels[1]); ok {
		t.Error("Expected no channel.")
	} else if n != 0 {
		t.Error("Expected no users, got:", n)
	}
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	if n, ok = st.NUsersByChannel(channels[0]); ok {
		t.Error("Expected a channel.")
	} else if n != 0 {
		t.Error("Expected no users, got:", n)
	}
	if n, ok = st.NUsersByChannel(channels[1]); ok {
		t.Error("Expected a channel.")
	} else if n != 0 {
		t.Error("Expected no users, got:", n)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	st.addToChannel(users[1], channels[0])
	if n, ok = st.NUsersByChannel(channels[0]); !ok {
		t.Error("Expected a channel.")
	} else if n != 2 {
		t.Error("Expected 2 users, got:", n)
	}
	if n, ok = st.NUsersByChannel(channels[1]); !ok {
		t.Error("Expected a channel.")
	} else if n != 1 {
		t.Error("Expected 1 users, got:", n)
	}
}

func TestState_EachUser(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	st.addUser(users[0])
	st.addUser(users[1])
	i := 0
	st.EachUser(func(u User) bool {
		has := false
		for _, user := range users {
			if user == string(u.Host) {
				has = true
				break
			}
		}
		if got, exp := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		i++
		return false
	})
	if got, exp := i, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_EachChannel(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	i := 0
	st.EachChannel(func(ch Channel) bool {
		has := false
		for _, channel := range channels {
			if channel == ch.String() {
				has = true
				break
			}
		}
		if got, exp := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
		i++
		return false
	})
	if got, exp := i, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_Users(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	st.addUser(users[0])
	st.addUser(users[1])
	if got, exp := len(st.Users()), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, u := range st.Users() {
		has := false
		for _, user := range users {
			if user == u {
				has = true
				break
			}
		}
		if got, exp := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_Channels(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	if got, exp := len(st.Channels()), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, ch := range st.Channels() {
		has := false
		for _, channel := range channels {
			if channel == ch {
				has = true
				break
			}
		}
		if got, exp := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_ChannelsByUser(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	if got := st.ChannelsByUser(users[0]); got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[0], channels[1])
	if got, exp := len(st.ChannelsByUser(users[0])), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, uc := range st.ChannelsByUser(users[0]) {
		has := false
		for _, channel := range channels {
			if channel == uc {
				has = true
				break
			}
		}
		if got, exp := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_UsersByChannel(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	if got := st.UsersByChannel(channels[0]); got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	if got, exp := len(st.UsersByChannel(channels[0])), 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	for _, cu := range st.UsersByChannel(channels[0]) {
		has := false
		for _, user := range users {
			if user == cu {
				has = true
				break
			}
		}
		if got, exp := has, true; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}
}

func TestState_IsOn(t *testing.T) {
	t.Parallel()

	st := setupNewState()
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

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}

	st.addUser(users[0])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if got, ok := st.User(users[1]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[1], channels[0])
	}
	for nick := range st.channelUsers[strings.ToLower(channels[0])] {
		if got, exp := nick, nicks[0]; exp != got {
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

	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if _, ok := st.User(users[1]); !ok {
		t.Error("Unexpected nil.")
	}
	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[1], channels[0])
	}
	for nick := range st.channelUsers[strings.ToLower(channels[0])] {
		if got, exp := nick, nicks[1]; exp != got {
			t.Errorf("Expected: %v, got: %v", exp, got)
		}
	}

	ev.Sender = users[0]
	ev.Args = []string{"newnick"}
	st.Update(ev)
	if _, ok := st.User("newnick"); !ok {
		t.Error("Unexpected nil.")
	}
	if got, ok := st.User(nicks[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
}

func TestState_UpdateNickSelfNilMaps(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.NICK,
		Sender: users[0],
		Args:   []string{nicks[1]},
	}
	st.addUser(users[0])
	st.Update(ev)

	_, ok := st.userChannels[nicks[0]]
	if got, exp := ok, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	_, ok = st.userChannels[nicks[1]]
	if got, exp := ok, false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateJoin(t *testing.T) {
	t.Parallel()

	st := setupNewState()
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
	if len(u.Seen) != 1 || u.Seen[0] != users[0] {
		t.Errorf("Expected %v to be seen, got: %v", users[0], u.Seen)
	}
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}

	st = setupNewState()
	st.addChannel(channels[0])

	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	u = st.Update(ev)
	if len(u.Seen) != 1 || u.Seen[0] != users[0] {
		t.Errorf("Expected %v to be seen, got: %v", users[0], u.Seen)
	}
	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
}

func TestState_UpdateJoinSelf(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.JOIN,
		Sender: string(st.selfUser.Host),
		Args:   []string{channels[0]},
	}

	if got, ok := st.Channel(channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if st.IsOn(st.selfUser.Nick(), channels[0]) {
		t.Errorf("Expected %v to not be on %v", st.selfUser.Nick(), channels[0])
	}
	u := st.Update(ev)
	if len(u.Seen) > 0 {
		t.Error("Expected self not to be seen.")
	}
	if _, ok := st.Channel(channels[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if !st.IsOn(st.selfUser.Nick(), channels[0]) {
		t.Errorf("Expected %v to be on %v", st.selfUser.Nick(), channels[0])
	}
}

func TestState_UpdatePart(t *testing.T) {
	t.Parallel()

	st := setupNewState()

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
	u = st.Update(ev)
	if len(u.Unseen) != 1 || u.Unseen[0] != users[1] {
		t.Errorf("Expected %v to be unseen, got: %v", users[1], u.Unseen)
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
	u = st.Update(ev)
	if len(u.Unseen) != 1 || u.Unseen[0] != users[0] {
		t.Errorf("Expected %v to be unseen, got: %v", users[0], u.Unseen)
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

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.PART,
		Sender: string(st.selfUser.Host),
		Args:   []string{channels[0]},
	}

	st.addUser(users[0])
	st.addUser(users[1])
	st.addUser(string(st.selfUser.Host))
	st.addChannel(channels[0])
	st.addChannel(channels[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])
	st.addToChannel(users[0], channels[1])
	st.addToChannel(st.selfUser.Nick(), channels[0])
	st.addToChannel(st.selfUser.Nick(), channels[1])

	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v.", users[1], channels[0])
	}
	if !st.IsOn(st.selfUser.Nick(), channels[0]) {
		t.Errorf("Expected %v to be on %v.", st.selfUser.Nick(), channels[0])
	}
	if !st.IsOn(st.selfUser.Nick(), channels[1]) {
		t.Errorf("Expected %v to be on %v.", st.selfUser.Nick(), channels[1])
	}

	u := st.Update(ev)
	if len(u.Unseen) != 1 || users[1] != u.Unseen[0] {
		t.Errorf("Expected to unsee: %v, got: %v", users[1], u.Unseen)
	}

	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[0], channels[0])
	}
	if !st.IsOn(users[0], channels[1]) {
		t.Errorf("Expected %v to be on %v.", users[0], channels[1])
	}
	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v.", users[1], channels[0])
	}
	if st.IsOn(st.selfUser.Nick(), channels[0]) {
		t.Errorf("Expected %v to not be on %v.", st.selfUser.Nick(), channels[0])
	}
	if !st.IsOn(st.selfUser.Nick(), channels[1]) {
		t.Errorf("Expected %v to be on %v.", st.selfUser.Nick(), channels[1])
	}
}

func TestState_UpdateQuit(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.QUIT,
		Sender: users[0],
		Args:   []string{"quit message"},
	}

	// Test Quitting when we don't know the user
	st.Update(ev)
	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}

	st.addUser(users[0])
	st.addUser(users[1])
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	if !st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[0], channels[0])
	}
	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[1], channels[0])
	}
	if _, ok := st.User(users[1]); !ok {
		t.Error("Unexpected nil.")
	}

	st.Update(ev)

	if st.IsOn(users[0], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[0], channels[0])
	}
	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if !st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to be on %v", users[1], channels[0])
	}
	if _, ok := st.User(users[1]); !ok {
		t.Error("Unexpected nil.")
	}

	ev.Sender = users[1]
	st.Update(ev)

	if st.IsOn(users[1], channels[0]) {
		t.Errorf("Expected %v to not be on %v", users[1], channels[0])
	}
	if got, ok := st.User(users[1]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
}

func TestState_UpdateQuitSelf(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.QUIT,
		Sender: string(st.selfUser.Host),
		Args:   []string{"quit message"},
	}

	u := st.Update(ev)
	if len(u.Quit) > 0 {
		t.Error("Expected us not to quit, got:", u.Quit)
	}
}

func TestState_UpdateKick(t *testing.T) {
	t.Parallel()

	st := setupNewState()
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

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.KICK,
		Sender: users[1],
		Args:   []string{channels[0], st.selfUser.Nick()},
	}

	st.addUser(string(st.selfUser.Host))
	st.addChannel(channels[0])
	st.addToChannel(users[0], channels[0])

	if _, ok := st.Channel(channels[0]); !ok {
		t.Error("Unexpected nil.")
	}
	st.Update(ev)
	if got, ok := st.Channel(channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
}

func TestState_UpdateMode(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.MODE,
		Sender: users[0],
		Args: []string{channels[0],
			"+ovmb-vn", nicks[0], nicks[0], "*!*mask", nicks[1],
		},
		NetworkInfo: testNetInfo,
	}

	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}

	st.addChannel(channels[0])
	st.addUser(users[0])
	st.addUser(users[1])
	st.addToChannel(users[0], channels[0])
	st.addToChannel(users[1], channels[0])

	u1modes, _ := st.UserModes(users[0], channels[0])
	u2modes, _ := st.UserModes(users[1], channels[0])
	u2modes.SetMode('v')
	realCh := st.channel(channels[0])
	realCh.Set("n")

	ch, _ := st.Channel(channels[0])
	if got, exp := ch.IsSet("n"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := ch.IsSet("mb"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u1modes.HasMode('o'), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u1modes.HasMode('v'), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u2modes.HasMode('v'), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	u1modes, _ = st.UserModes(users[0], channels[0])
	u2modes, _ = st.UserModes(users[1], channels[0])
	ch, _ = st.Channel(channels[0])
	if got, exp := ch.IsSet("n"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := ch.IsSet("mb *!*mask"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u1modes.HasMode('o'), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u1modes.HasMode('v'), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := u2modes.HasMode('v'), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateModeSelf(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:        irc.MODE,
		Sender:      string(st.selfUser.Host),
		Args:        []string{st.selfUser.Nick(), "+i-o"},
		NetworkInfo: testNetInfo,
	}

	st.selfModes.Set("o")

	if got, exp := st.selfModes.IsSet("i"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := st.selfModes.IsSet("o"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	if got, exp := st.selfModes.IsSet("i"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := st.selfModes.IsSet("o"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateTopic(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.TOPIC,
		Sender: users[1],
		Args:   []string{channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	ch, _ := st.Channel(channels[0])
	if got, exp := ch.Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	ch, _ = st.Channel(channels[0])
	if got, exp := ch.Topic(), "topic topic"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	ev.Args = []string{channels[0]}
	st.Update(ev)
	ch, _ = st.Channel(channels[0])
	if got, exp := ch.Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateRplTopic(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.RPL_TOPIC,
		Sender: network,
		Args:   []string{st.selfUser.Nick(), channels[0], "topic topic"},
	}

	st.addChannel(channels[0])

	ch, _ := st.Channel(channels[0])
	if got, exp := ch.Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	ch, _ = st.Channel(channels[0])
	if got, exp := ch.Topic(), "topic topic"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateEmptyTopic(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.TOPIC,
		Sender: users[1],
		Args:   []string{channels[0], ""},
	}

	ch := st.addChannel(channels[0])
	ch.SetTopic("topic topic")

	channel, _ := st.Channel(channels[0])
	if got, exp := channel.Topic(), "topic topic"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	channel, _ = st.Channel(channels[0])
	if got, exp := channel.Topic(), ""; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdatePrivmsg(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:        irc.PRIVMSG,
		Sender:      users[0],
		Args:        []string{channels[0]},
		NetworkInfo: testNetInfo,
	}

	st.addChannel(channels[0])

	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if _, ok := st.UserModes(users[0], channels[0]); !ok {
		t.Error("Unexpected nil.")
	}

	ev.Sender = network
	size := len(st.users)
	st.Update(ev)
	if got, exp := len(st.users), size; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateNotice(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:        irc.NOTICE,
		Sender:      users[0],
		Args:        []string{channels[0]},
		NetworkInfo: testNetInfo,
	}

	st.addChannel(channels[0])

	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	if _, ok := st.UserModes(users[0], channels[0]); !ok {
		t.Error("Unexpected nil.")
	}

	ev.Sender = network
	size := len(st.users)
	st.Update(ev)
	if got, exp := len(st.users), size; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdatePrivmsgSelf(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:        irc.PRIVMSG,
		Sender:      users[0],
		Args:        []string{st.selfUser.Nick(), "msg"},
		NetworkInfo: testNetInfo,
	}

	u := st.Update(ev)
	if len(u.Seen) > 0 {
		t.Error("Expected no one to be seen, got:", u.Seen)
	}
}

func TestState_UpdateWelcome(t *testing.T) {
	t.Parallel()

	st := setupNewState()
	ev := &irc.Event{
		Name:   irc.RPL_WELCOME,
		Sender: network,
		Args:   []string{nicks[1], "Welcome to"},
	}

	st.Update(ev)
	if got, exp := string(st.selfUser.Host), nicks[1]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := st.users[nicks[1]].Host, st.selfUser.Host; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}

	ev = &irc.Event{
		Name:   irc.RPL_WELCOME,
		Sender: network,
		Args:   []string{nicks[1], "Welcome to " + users[1]},
	}

	st.Update(ev)
	if got, exp := string(st.selfUser.Host), users[1]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := st.users[nicks[1]].Host, st.selfUser.Host; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateRplNamereply(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.RPL_NAMREPLY,
		Sender: network,
		Args: []string{
			st.selfUser.Nick(), "=", channels[0],
			"@" + nicks[0] + " +" + nicks[1] + " " + st.selfUser.Nick(),
		},
	}

	st.addChannel(channels[0])

	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got, ok := st.UserModes(users[1], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got, ok := st.UserModes(st.selfUser.Nick(), channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	got, _ := st.UserModes(users[0], channels[0])
	if got.String() != "o" {
		t.Errorf(`Expected: "o", got: %q`, got.String())
	}
	got, _ = st.UserModes(users[1], channels[0])
	if got.String() != "v" {
		t.Errorf(`Expected: "v", got: %q`, got.String())
	}
	got, _ = st.UserModes(st.selfUser.Nick(), channels[0])
	if len(got.String()) > 0 {
		t.Errorf(`Expected: empty string, got: %q`, got.String())
	}
}

func TestState_RplWhoReply(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.RPL_WHOREPLY,
		Sender: network,
		Args: []string{
			st.selfUser.Nick(), channels[0], irc.Username(users[0]),
			irc.Hostname(users[0]), "*.network.net", nicks[0], "Hx@d",
			"3 real name",
		},
	}

	st.addChannel(channels[0])

	if got, ok := st.User(users[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got, ok := st.UserModes(users[0], channels[0]); ok {
		t.Errorf("Expected: %v to be nil.", got)
	}
	st.Update(ev)
	if _, ok := st.User(users[0]); !ok {
		t.Error("Unexpected nil.")
	}
	user, _ := st.User(users[0])
	if got, exp := string(user.Host), users[0]; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got, exp := user.Realname, "real name"; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	got, _ := st.UserModes(users[0], channels[0])
	if got.String() != "o" {
		t.Errorf(`Expected: "o", got: %q`, got.String())
	}

}

func TestState_UpdateRplMode(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.RPL_CHANNELMODEIS,
		Sender: network,
		Args:   []string{st.selfUser.Nick(), channels[0], "+ntzl", "10"},
	}

	st.addChannel(channels[0])
	ch, _ := st.Channel(channels[0])
	if got, exp := ch.IsSet("ntzl 10"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	ch, _ = st.Channel(channels[0])
	if got, exp := ch.IsSet("ntzl 10"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestState_UpdateRplBanlist(t *testing.T) {
	t.Parallel()

	st := setupNewState()

	ev := &irc.Event{
		Name:   irc.RPL_BANLIST,
		Sender: network,
		Args: []string{st.selfUser.Nick(), channels[0],
			nicks[0] + "!*@*", nicks[1], "1367197165",
		},
	}

	st.addChannel(channels[0])
	ch, _ := st.Channel(channels[0])
	if got, exp := ch.HasBan(nicks[0]+"!*@*"), false; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	st.Update(ev)
	ch, _ = st.Channel(channels[0])
	if got, exp := ch.HasBan(nicks[0]+"!*@*"), true; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}
