package apiserver

import (
	"log"
	"net"
	"sync"

	"github.com/aarondl/ultimateq/api"
	"github.com/aarondl/ultimateq/bot"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/registrar"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
)

// apiServer provides a grpc api around a bot
type apiServer struct {
	bot *bot.Bot

	mut   sync.RWMutex
	proxy *registrar.Proxy
	pipes map[string]*msgPipe
}

func NewAPIServer(b *bot.Bot) apiServer {
	server := apiServer{
		bot:   b,
		proxy: registrar.NewProxy(b),
	}

	return server
}

func (a apiServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	// TODO(aarondl): Delete
	grpclog.SetLogger(log.Logger)

	grpcServer := grpc.NewServer()
	api.RegisterExtServer(grpcServer, a)

	// TODO(aarondl): TLS
	// TODO(aarondl): Disconnection handler, each Register/RegisterCmd leaves around
	// a msgPipe
	return grpcServer.Serve(lis)
}

func (a apiServer) getState(network string) (*data.State, error) {
	state := a.bot.State(network)
	if state == nil {
		return nil, grpc.Errorf(codes.NotFound, "state not found")
	}

	return state, nil
}

// getMsgPipe must be called under mut.Lock()
func (a apiServer) getMsgPipe(ext string) *msgPipe {
	pipe, ok := a.pipes[ext]
	if !ok {
		pipe = newMsgPipe(a.bot.Logger.New("ext", ext))
		a.pipes[ext] = pipe
	}

	return pipe
}

func (a apiServer) Pipe(pipe api.Ext_PipeServer) error {
	return nil
}

func (a apiServer) Register(ctx context.Context, in *api.RegisterRequest) (*api.Empty, error) {
	proxy := a.proxy.Get(in.Name)

	return nil, nil
}

func (a apiServer) Unregister(ctx context.Context, in *api.UnregisterRequest) (*api.Empty, error) {
	a.proxy.Unregister(in.Name)
	return nil, nil
}

func (a apiServer) getStore() (*data.Store, error) {
	store := a.bot.Store()
	if store == nil {
		return nil, grpc.Errorf(codes.Unavailable, "store not enabled")
	}

	return store, nil
}

func (a apiServer) Register(ctx context, in *api.RegisterRequest) (*api.RegisterResponse, error) {
	a.mut.Lock()
	proxy := a.proxy.Get(in.Ext)
	pipe := a.getMsgPipe(in.Ext)
	a.mut.Unlock()

	id := proxy.Register(in.Network, in.Channel, in.Event, pipe)
	return &api.RegisterResponse{
		id: id,
	}, nil
}

func (a apiServer) RegisterCmd(ctx context.Context, in *api.RegisterCmdRequest) (*api.Empty, error) {
	a.mut.Lock()
	proxy := a.proxy.Get(in.Ext)
	pipe := a.getMsgPipe(in.Ext)
	a.mut.Unlock()

	command := &cmd.Cmd{
		Cmd:         in.Cmd.Cmd,
		Extension:   in.Cmd.Ext,
		Description: in.Cmd.Desc,
		Kind:        cmd.MsgKind(in.Cmd.Kind),
		Scope:       cmd.MsgScope(in.Cmd.Scope),
		RequireAuth: in.Cmd.RequireAuth,
		ReqLevel:    uint8(in.Cmd.ReqLevel),
		ReqFlags:    in.Cmd.ReqFlags,
	}

	id := proxy.RegisterCmd(in.Network, in.Channel, command, pipe)
	return nil, nil
}

func (a apiServer) Unregister(ctx context.Context, in *api.UnregisterRequest) (*api.Empty, error) {
	a.mut.Lock()
	proxy := a.proxy.Get(in.Ext)
	a.mut.Unlock()

	proxy.Unregister(in.Id)
	return nil, nil
}

func (a apiServer) UnregisterCmd(ctx context.Context, in *api.UnregisterCmdRequest) (*api.Empty, error) {
	a.mut.Lock()
	proxy := a.proxy.Get(in.Ext)
	a.mut.Unlock()

	proxy.UnregisterCmd(in.Network, in.Channel, in.Ext, in.Command)
	return nil, nil
}

func (a apiServer) UnregisterAll(ctx context.Context, in *api.UnregisterAllRequest) (*api.Empty, error) {
	a.mut.Lock()
	proxy := a.proxy.Unregister(in.Ext)
	delete(a.pipes, in.Ext)
	a.mut.Unlock()

	return nil, nil
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

func (a apiServer) StoreLogoutByUser(ctx context.Context, in *api.Query) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	store.LogoutByUsername(in.Query)
	return nil, nil
}