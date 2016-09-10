package bot

import (
	"net"
	"sync"

	"github.com/aarondl/ultimateq/api"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/irc"
	"github.com/aarondl/ultimateq/registrar"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type msgPipe struct {
	Events    <-chan *irc.Event
	CmdEvents <-chan *cmd.Event
}

// api provides a REST api around a bot
type apiServer struct {
	bot   *Bot
	proxy *registrar.Proxy

	mut   sync.RWMutex
	pipes map[string]msgPipe
}

func (a apiServer) HandleRaw(w irc.Writer, ev *irc.Event) {
}

func (a apiServer) Cmd(command string, w irc.Writer, ev *cmd.Event) {
}

func newAPIServer(b *Bot) apiServer {
	server := apiServer{
		bot:   b,
		proxy: registrar.NewProxy(b),
	}

	return server
}

func (a apiServer) Pipe(pipe api.Ext_PipeServer) error {
	return nil
}

func (a apiServer) Register(ctx context.Context, in *api.RegisterRequest) (*api.Empty, error) {
	proxy := a.proxy.Get(in.Name)
	if proxy == nil {
		return nil, grpc.Errorf(codes.NotFound, "extension not found")
	}

	ext := in.Name
	for _, cmd := range in.Cmds {
	}
}

func (a apiServer) Unregister(ctx context.Context, in *api.UnregisterRequest) (*api.Empty, error) {
	a.proxy.Unregister(in.Name)
	return nil, nil
}

func (a apiServer) start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	api.RegisterExtServer(grpcServer, a)

	//TODO(aarondl): TLS
	return grpcServer.Serve(lis)
}

func (a apiServer) getState(network string) (*data.State, error) {
	state := a.bot.State(network)
	if state == nil {
		return nil, grpc.Errorf(codes.NotFound, "state not found")
	}

	return state, nil
}

func (a apiServer) getStore() (*data.Store, error) {
	store := a.bot.Store()
	if store == nil {
		return nil, grpc.Errorf(codes.Unavailable, "store not enabled")
	}

	return store, nil
}

func (a apiServer) StateSelf(ctx context.Context, in *api.Query) (*api.SelfResponse, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	self := state.Self()
	ret := &api.SelfResponse{}
	ret.User = &api.SimpleUser{
		Host: string(self.User.Host),
		Name: self.User.Realname,
	}
	ret.Modes = self.ChannelModes.ToProto()

	return ret, nil
}

func (a apiServer) StateUser(ctx context.Context, in *api.NetworkQuery) (*api.SimpleUser, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	user, ok := state.User(in.Query)
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "user not found")
	}

	return &api.SimpleUser{
		Host: string(user.Host),
		Name: user.Realname,
	}, nil
}

func (a apiServer) StateUsersByChan(ctx context.Context, in *api.NetworkQuery) (*api.ListResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	var users []string
	if len(in.Query) != 0 {
		users = state.UsersByChannel(in.Query)
	} else {
		users = state.Users()
	}

	return &api.ListResponse{List: users}, nil
}

func (a apiServer) StateUsersByChanCount(ctx context.Context, in *api.NetworkQuery) (*api.CountResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	var users int
	var ok bool
	if len(in.Query) != 0 {
		users, ok = state.NUsersByChannel(in.Query)
		if !ok {
			return nil, grpc.Errorf(codes.NotFound, "channel not found")
		}
	} else {
		users = state.NUsers()
	}

	return &api.CountResponse{Count: int32(users)}, nil
}

func (a apiServer) StateUserModes(ctx context.Context, in *api.ChannelQuery) (*api.UserModes, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	umodes, ok := state.UserModes(in.Query, in.Channel)
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "user or channel not found")
	}

	return &api.UserModes{
		Kinds: umodes.ModeKinds.ToProto(),
		Modes: int32(umodes.Modes),
	}, nil
}

func (a apiServer) StateChannel(ctx context.Context, in *api.NetworkQuery) (*api.ChannelResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	channel, ok := state.Channel(in.Query)
	if !ok {
		return nil, grpc.Errorf(codes.NotFound, "channel not found")
	}

	return &api.ChannelResponse{
		Name:  channel.Name,
		Topic: channel.Topic,
		Modes: channel.Modes.ToProto(),
	}, nil
}

func (a apiServer) StateChannels(ctx context.Context, in *api.Query) (*api.ListResponse, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	chans := state.Channels()
	return &api.ListResponse{
		List: chans,
	}, nil
}

func (a apiServer) StateChannelCount(ctx context.Context, in *api.Query) (*api.CountResponse, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	count := state.NChannels()

	return &api.CountResponse{
		Count: int32(count),
	}, nil
}

func (a apiServer) StateIsOn(ctx context.Context, in *api.ChannelQuery) (*api.Result, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	is := state.IsOn(in.Query, in.Channel)

	return &api.Result{Ok: is}, nil
}

func (a apiServer) StoreAuthUser(ctx context.Context, in *api.AuthUserRequest) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	if in.Permanent {
		_, err = store.AuthUserPerma(in.Network, in.Host, in.Username, in.Password)
	} else {
		_, err = store.AuthUserTmp(in.Network, in.Host, in.Username, in.Password)
	}

	return nil, err
}

func (a apiServer) StoreAuthedUser(ctx context.Context, in *api.NetworkQuery) (*api.StoredUser, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	user := store.AuthedUser(in.Network, in.Query)
	if user == nil {
		return nil, nil
	}

	return user.ToProto(), nil
}

func (a apiServer) StoreUser(ctx context.Context, in *api.Query) (*api.StoredUser, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	user, err := store.FindUser(in.Query)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, grpc.Errorf(codes.NotFound, "user not found")
	}

	return user.ToProto(), nil
}

func (a apiServer) StoreUsers(ctx context.Context, _ *api.Empty) (*api.StoredUsersResponse, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	users, err := store.GlobalUsers()
	if err != nil {
		return nil, err
	}

	return makeUsersResponse(users), nil
}

func (a apiServer) StoreUsersByNetwork(ctx context.Context, in *api.Query) (*api.StoredUsersResponse, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	users, err := store.NetworkUsers(in.Query)
	if err != nil {
		return nil, err
	}

	return makeUsersResponse(users), nil
}

func (a apiServer) StoreUsersByChannel(ctx context.Context, in *api.NetworkQuery) (*api.StoredUsersResponse, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	users, err := store.ChanUsers(in.Network, in.Query)
	if err != nil {
		return nil, err
	}

	return makeUsersResponse(users), nil
}

func makeUsersResponse(users []*data.StoredUser) *api.StoredUsersResponse {
	if len(users) == 0 {
		return &api.StoredUsersResponse{}
	}

	var resp api.StoredUsersResponse
	resp.Users = make([]*api.StoredUser, len(users))

	for i, u := range users {
		resp.Users[i] = u.ToProto()
	}

	return &resp
}

func (a apiServer) StoreChannel(ctx context.Context, in *api.NetworkQuery) (*api.StoredChannel, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	channel, err := store.FindChannel(in.Network, in.Query)
	if err != nil {
		return nil, err
	}

	if channel == nil {
		return nil, grpc.Errorf(codes.NotFound, "channel not found")
	}

	return channel.ToProto(), nil
}

func (a apiServer) StoreChannels(ctx context.Context, in *api.Empty) (*api.StoredChannelsResponse, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	channels, err := store.Channels()
	if err != nil {
		return nil, err
	}

	if len(channels) == 0 {
		return &api.StoredChannelsResponse{}, nil
	}

	var resp api.StoredChannelsResponse
	resp.Channels = make([]*api.StoredChannel, len(channels))

	for i, u := range channels {
		resp.Channels[i] = u.ToProto()
	}

	return &resp, nil
}

func (a apiServer) StorePutUser(ctx context.Context, in *api.StoredUser) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	user := new(data.StoredUser)
	user.FromProto(in)

	return nil, store.SaveUser(user)
}

func (a apiServer) StorePutChannel(ctx context.Context, in *api.StoredChannel) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	channel := new(data.StoredChannel)
	channel.FromProto(in)

	return nil, store.SaveChannel(channel)
}

func (a apiServer) StoreDeleteUser(ctx context.Context, in *api.Query) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	ok, err := store.RemoveUser(in.Query)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, grpc.Errorf(codes.NotFound, "user not found")
	}

	return nil, nil
}

func (a apiServer) StoreDeleteChannel(ctx context.Context, in *api.NetworkQuery) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	ok, err := store.RemoveChannel(in.Network, in.Query)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, grpc.Errorf(codes.NotFound, "user not found")
	}

	return nil, nil
}

func (a apiServer) StoreLogout(ctx context.Context, in *api.NetworkQuery) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	store.Logout(in.Network, in.Query)
	return nil, nil
}

func (a apiServer) StoreLogoutByUser(context.Context, *Query) (*Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	store.LogoutByUsername(in.Query)
	return nil, nil
}
