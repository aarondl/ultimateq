package bot

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/aarondl/ultimateq/api"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/dispatch/cmd"
	"github.com/aarondl/ultimateq/registrar"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	broadcastTimeout = 500 * time.Millisecond
)

// apiServer provides a grpc api around a bot
type apiServer struct {
	bot *Bot

	mut       sync.RWMutex
	proxy     *registrar.Proxy
	nextSubID uint64
	subs      map[string]map[uint64]*sub
}

type sub struct {
	subID    uint64
	eventIDs []uint64

	eventChan   chan *api.IRCEventResponse
	commandChan chan *api.CmdEventResponse
}

var _ api.ExtServer = &apiServer{}

// NewAPIServer creates an api server
func NewAPIServer(b *Bot) *apiServer {
	server := &apiServer{
		bot:   b,
		proxy: registrar.NewProxy(b),

		nextSubID: 1,
		subs:      make(map[string]map[uint64]*sub),
	}

	return server
}

func (a *apiServer) Start() error {
	addr, ok := a.bot.conf.ExtGlobal().Listen()
	if !ok {
		return errors.New("no listen address configured")
	}

	proto := "tcp"
	if strings.Contains(addr, "/") {
		proto = "unix"
	}

	lis, err := net.Listen(proto, addr)
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption

	cert, certOk := a.bot.conf.ExtGlobal().TLSCert()
	key, keyOk := a.bot.conf.ExtGlobal().TLSKey()
	ca, caOk := a.bot.conf.ExtGlobal().TLSClientCA()
	insecure, _ := a.bot.conf.ExtGlobal().TLSInsecureSkipVerify()

	var config *tls.Config
	if certOk && keyOk {
		config = &tls.Config{}

		keypair, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return errors.Wrap(err, "failed to load server keypair")
		}

		config.Certificates = append(config.Certificates, keypair)

		config.ClientAuth = tls.RequireAndVerifyClientCert
		if insecure {
			config.ClientAuth = tls.RequireAnyClientCert
		}

		if caOk {
			clientCACert, err := ioutil.ReadFile(ca)
			if err != nil {
				return err
			}

			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM(clientCACert)
			config.ClientCAs = certPool
		} else {
			certPool, err := x509.SystemCertPool()
			if err != nil {
				return errors.Wrap(err, "failed to load system ca cert pool")
			}
			config.ClientCAs = certPool
		}
	}

	a.bot.Logger.Info("API Server Listening", "addr", addr)

	if config != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(config)))
	}

	grpcServer := grpc.NewServer(opts...)
	api.RegisterExtServer(grpcServer, a)

	return grpcServer.Serve(lis)
}

// subscribe is not goroutine safe, must lock the mut first
// this is so register and friends can lock a single mutex to mess with
// the proxy
func (a *apiServer) subscribe(ext string, isNormalEvent bool, ids []uint64) *sub {
	extSubs, ok := a.subs[ext]
	if !ok {
		extSubs = make(map[uint64]*sub)
		a.subs[ext] = extSubs
	}

	s := &sub{
		subID:    a.nextSubID,
		eventIDs: ids,
	}
	if isNormalEvent {
		s.eventChan = make(chan *api.IRCEventResponse)
	} else {
		s.commandChan = make(chan *api.CmdEventResponse)
	}
	a.nextSubID++

	extSubs[s.subID] = s
	return s
}

// unsubscribe is not goroutine safe, must lock the mut first
func (a *apiServer) unsubscribe(ext string, subID uint64) {
	extSubs, ok := a.subs[ext]
	if !ok {
		return
	}

	delete(extSubs, subID)

	if len(extSubs) == 0 {
		delete(a.subs, ext)
	}
}

func (a *apiServer) broadcastEvent(ext string, r *api.IRCEventResponse) bool {
	return a.broadcast(ext, r, nil)
}

func (a *apiServer) broadcastCmd(ext string, r *api.CmdEventResponse) bool {
	return a.broadcast(ext, nil, r)
}

func (a *apiServer) broadcast(ext string, rEvent *api.IRCEventResponse, rCmd *api.CmdEventResponse) bool {
	var subs []*sub
	var evID uint64
	if rEvent != nil {
		evID = rEvent.Id
	} else {
		evID = rCmd.Id
	}

	a.mut.RLock()

	// Create a list of subscribers we need to notify
	if extSubs, ok := a.subs[ext]; ok {
		for _, s := range extSubs {
			if s.eventChan != nil && rEvent == nil {
				continue
			} else if s.commandChan != nil && rCmd == nil {
				continue
			}

			has := len(s.eventIDs) == 0
			if !has {
				for _, i := range s.eventIDs {
					if i == evID {
						has = true
						break
					}
				}
			}

			if has {
				subs = append(subs, s)
			}
		}
	}
	a.mut.RUnlock()

	a.bot.Logger.Debug("publishing", "ext", ext, "id", evID, "n", len(subs))
	if len(subs) == 0 {
		return false
	}

	timer := time.NewTimer(broadcastTimeout)
	sent := false

	for _, s := range subs {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(broadcastTimeout)

		a.bot.Logger.Debug("publishing to", "ext", ext, "id", evID, "subid", s.subID)

		if rEvent != nil {
			select {
			case s.eventChan <- rEvent:
				a.bot.Logger.Debug("publish event success", "ext", ext, "id", evID, "subid", s.subID)
				sent = true
			case <-timer.C:
				a.bot.Logger.Debug("timeout to subscriber", "ext", ext, "id", evID, "subid", s.subID)
				// Timeout, do nothing
			}
		} else {
			select {
			case s.commandChan <- rCmd:
				a.bot.Logger.Debug("publish cmd success", "ext", ext, "id", evID, "subid", s.subID)
				sent = true
			case <-timer.C:
				a.bot.Logger.Debug("timeout to subscriber", "ext", ext, "id", evID, "subid", s.subID)
				// Timeout, do nothing
			}
		}
	}

	return sent
}

func (a *apiServer) makePipe(ext string) *pipeHandler {
	return &pipeHandler{
		logger: a.bot.Logger.New("ext", ext),
		ext:    ext,
		helper: a,
	}
}

func (a *apiServer) getState(network string) (*data.State, error) {
	state := a.bot.State(network)
	if state == nil {
		return nil, status.Errorf(codes.NotFound, "state not found")
	}

	return state, nil
}

func (a *apiServer) getStore() (*data.Store, error) {
	store := a.bot.Store()
	if store == nil {
		return nil, status.Errorf(codes.Unavailable, "store not enabled")
	}

	return store, nil
}

func (a *apiServer) unregEvent(ext string, id uint64) {
	a.bot.Logger.Debug("unregistering event", "ext", ext, "id", id)
	a.mut.Lock()
	proxy := a.proxy.Get(ext)
	proxy.Unregister(id)
	a.mut.Unlock()
}

func (a *apiServer) unregCmd(ext string, id uint64) {
	a.bot.Logger.Debug("unregistering command", "ext", ext, "id", id)
	a.mut.Lock()
	proxy := a.proxy.Get(ext)
	proxy.UnregisterCmd(id)
	a.mut.Unlock()
}

func (a *apiServer) Events(in *api.SubscriptionRequest, stream api.Ext_EventsServer) error {
	a.mut.Lock()
	s := a.subscribe(in.Ext, true, in.Ids)
	a.mut.Unlock()

	a.bot.Logger.Debug("event sub", "ext", in.Ext, "subid", s.subID)

	for {
		event := <-s.eventChan

		err := stream.Send(event)
		if err != nil {
			a.bot.Logger.Error("grpc event send err", "err", err, "id", event.Id, "subid", s.subID)
			break
		}
	}

	a.mut.Lock()
	a.unsubscribe(in.Ext, s.subID)
	a.mut.Unlock()

	// Drain channel
	for {
		select {
		case <-s.eventChan:
		default:
		}
		break
	}

	a.bot.Logger.Debug("event sub closed", "ext", in.Ext, "subid", s.subID)
	return nil
}

func (a *apiServer) Commands(in *api.SubscriptionRequest, stream api.Ext_CommandsServer) error {
	a.mut.Lock()
	s := a.subscribe(in.Ext, false, in.Ids)
	a.mut.Unlock()

	a.bot.Logger.Debug("command sub", "ext", in.Ext, "subid", s.subID)

	for {
		event := <-s.commandChan

		err := stream.Send(event)
		if err != nil {
			a.bot.Logger.Error("grpc cmd send err", "err", err, "id", event.Id, "subid", s.subID)
			break
		}
	}

	a.mut.Lock()
	a.unsubscribe(in.Ext, s.subID)
	a.mut.Unlock()

	// Drain channel
	for {
		select {
		case <-s.commandChan:
		default:
		}
		break
	}

	a.bot.Logger.Debug("command sub closed", "ext", in.Ext, "subid", s.subID)
	return nil
}

func (a *apiServer) Write(ctx context.Context, in *api.WriteRequest) (*api.Empty, error) {
	net := a.bot.NetworkWriter(in.Net)
	if net == nil {
		return nil, status.Errorf(codes.NotFound, "network id %q not found", in.Net)
	}

	a.bot.Logger.Debug("ext write", "ext", in.Ext, "net", in.Net, "msg", string(in.Msg.Msg))

	_, err := net.Write(in.Msg.Msg)
	if err != nil {
		return nil, err
	}

	return new(api.Empty), nil
}

func (a *apiServer) Register(ctx context.Context, in *api.RegisterRequest) (*api.RegisterResponse, error) {
	a.mut.Lock()
	defer a.mut.Unlock()

	pipe := a.makePipe(in.Ext)

	proxy := a.proxy.Get(in.Ext)
	id := proxy.Register(in.Network, in.Channel, in.Event, pipe)
	pipe.setEventID(id)

	a.bot.Logger.Info("remote event register", "ext", in.Ext, "net", in.Network, "chan", in.Channel, "ev", in.Event, "id", id)

	return &api.RegisterResponse{Id: id}, nil
}

func (a *apiServer) RegisterCmd(ctx context.Context, in *api.RegisterCmdRequest) (*api.RegisterResponse, error) {
	a.mut.Lock()
	defer a.mut.Unlock()

	pipe := a.makePipe(in.Ext)

	var command *cmd.Command
	var err error
	if in.Cmd.RequireAuth {
		command, err = cmd.NewErr(
			in.Cmd.Ext,
			in.Cmd.Name,
			in.Cmd.Desc,
			pipe,
			cmd.Kind(in.Cmd.Kind),
			cmd.Scope(in.Cmd.Scope),
		)
	} else {
		command, err = cmd.NewAuthedErr(
			in.Cmd.Ext,
			in.Cmd.Name,
			in.Cmd.Desc,
			pipe,
			cmd.Kind(in.Cmd.Kind),
			cmd.Scope(in.Cmd.Scope),
			uint8(in.Cmd.ReqLevel),
			in.Cmd.ReqFlags,
		)
	}

	if err != nil {
		return nil, err
	}

	proxy := a.proxy.Get(in.Ext)
	id, err := proxy.RegisterCmd(in.Network, in.Channel, command)
	if err != nil {
		return nil, err
	}

	a.bot.Logger.Info("remote command register", "ext", in.Ext, "net", in.Network, "chan", in.Channel, "cmd", in.Cmd.Name, "id", id)
	pipe.setEventID(id)

	return &api.RegisterResponse{Id: id}, nil
}

func (a *apiServer) Unregister(ctx context.Context, in *api.UnregisterRequest) (*api.Result, error) {
	a.mut.Lock()
	defer a.mut.Unlock()

	a.bot.Logger.Info("remote event unregister", "ext", in.Ext, "id", in.Id)

	proxy := a.proxy.Get(in.Ext)
	ok := proxy.Unregister(in.Id)
	return &api.Result{Ok: ok}, nil
}

func (a *apiServer) UnregisterCmd(ctx context.Context, in *api.UnregisterRequest) (*api.Result, error) {
	a.mut.Lock()
	proxy := a.proxy.Get(in.Ext)
	a.mut.Unlock()

	a.bot.Logger.Info("remote command unregister", "ext", in.Ext, "id", in.Id)

	ok := proxy.UnregisterCmd(in.Id)
	return &api.Result{Ok: ok}, nil
}

func (a *apiServer) UnregisterAll(ctx context.Context, in *api.UnregisterAllRequest) (*api.Empty, error) {
	a.mut.Lock()
	a.proxy.Unregister(in.Ext)
	delete(a.subs, in.Ext)
	a.mut.Unlock()

	return nil, nil
}

func (a *apiServer) StateSelf(ctx context.Context, in *api.Query) (*api.SelfResponse, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	self := state.Self()
	ret := &api.SelfResponse{}
	ret.User = &api.StateUser{
		Host:     string(self.User.Host),
		Realname: self.User.Realname,
	}
	ret.Modes = self.ChannelModes.ToProto()

	return ret, nil
}

func (a *apiServer) StateUsers(ctx context.Context, in *api.NetworkQuery) (*api.StateUser, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	user, ok := state.User(in.Query)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return &api.StateUser{
		Host:     string(user.Host),
		Realname: user.Realname,
	}, nil
}

func (a *apiServer) StateUsersByChan(ctx context.Context, in *api.NetworkQuery) (*api.ListResponse, error) {
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

func (a *apiServer) StateUsersByChanCount(ctx context.Context, in *api.NetworkQuery) (*api.CountResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	var users int
	var ok bool
	if len(in.Query) != 0 {
		users, ok = state.NUsersByChannel(in.Query)
		if !ok {
			return nil, status.Errorf(codes.NotFound, "channel not found")
		}
	} else {
		users = state.NUsers()
	}

	return &api.CountResponse{Count: int32(users)}, nil
}

func (a *apiServer) StateUserModes(ctx context.Context, in *api.ChannelQuery) (*api.UserModes, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	umodes, ok := state.UserModes(in.Query, in.Channel)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "user or channel not found")
	}

	return &api.UserModes{
		Kinds: umodes.ModeKinds.ToProto(),
		Modes: int32(umodes.Modes),
	}, nil
}

func (a *apiServer) StateChannel(ctx context.Context, in *api.NetworkQuery) (*api.ChannelResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}

	channel, ok := state.Channel(in.Query)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "channel not found")
	}

	return &api.ChannelResponse{
		Name:  channel.Name,
		Topic: channel.Topic,
		Modes: channel.Modes.ToProto(),
	}, nil
}

func (a *apiServer) StateChannels(ctx context.Context, in *api.Query) (*api.ListResponse, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	chans := state.Channels()
	return &api.ListResponse{
		List: chans,
	}, nil
}

func (a *apiServer) StateChannelCount(ctx context.Context, in *api.Query) (*api.CountResponse, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	count := state.NChannels()

	return &api.CountResponse{
		Count: int32(count),
	}, nil
}

func (a *apiServer) StateIsOn(ctx context.Context, in *api.ChannelQuery) (*api.Result, error) {
	state, err := a.getState(in.Query)
	if err != nil {
		return nil, err
	}

	is := state.IsOn(in.Query, in.Channel)

	return &api.Result{Ok: is}, nil
}

func (a *apiServer) StoreAuthUser(ctx context.Context, in *api.AuthUserRequest) (*api.Empty, error) {
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

func (a *apiServer) StoreAuthedUser(ctx context.Context, in *api.NetworkQuery) (*api.StoredUser, error) {
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

func (a *apiServer) StoreUser(ctx context.Context, in *api.Query) (*api.StoredUser, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	user, err := store.FindUser(in.Query)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return user.ToProto(), nil
}

func (a *apiServer) StoreUsers(ctx context.Context, _ *api.Empty) (*api.StoredUsersResponse, error) {
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

func (a *apiServer) StoreUsersByNetwork(ctx context.Context, in *api.Query) (*api.StoredUsersResponse, error) {
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

func (a *apiServer) StoreUsersByChannel(ctx context.Context, in *api.NetworkQuery) (*api.StoredUsersResponse, error) {
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

func (a *apiServer) StoreChannel(ctx context.Context, in *api.NetworkQuery) (*api.StoredChannel, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	channel, err := store.FindChannel(in.Network, in.Query)
	if err != nil {
		return nil, err
	}

	if channel == nil {
		return nil, status.Errorf(codes.NotFound, "channel not found")
	}

	return channel.ToProto(), nil
}

func (a *apiServer) StoreChannels(ctx context.Context, in *api.Empty) (*api.StoredChannelsResponse, error) {
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

func (a *apiServer) StorePutUser(ctx context.Context, in *api.StoredUser) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	user := new(data.StoredUser)
	user.FromProto(in)

	return nil, store.SaveUser(user)
}

func (a *apiServer) StorePutChannel(ctx context.Context, in *api.StoredChannel) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	channel := new(data.StoredChannel)
	channel.FromProto(in)

	return nil, store.SaveChannel(channel)
}

func (a *apiServer) StoreDeleteUser(ctx context.Context, in *api.Query) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	ok, err := store.RemoveUser(in.Query)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return nil, nil
}

func (a *apiServer) StoreDeleteChannel(ctx context.Context, in *api.NetworkQuery) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	ok, err := store.RemoveChannel(in.Network, in.Query)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return nil, nil
}

func (a *apiServer) StoreLogout(ctx context.Context, in *api.NetworkQuery) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	store.Logout(in.Network, in.Query)
	return nil, nil
}

func (a *apiServer) StoreLogoutByUser(ctx context.Context, in *api.Query) (*api.Empty, error) {
	store, err := a.getStore()
	if err != nil {
		return nil, err
	}

	store.LogoutByUsername(in.Query)
	return nil, nil
}

func (a *apiServer) NetworkInformation(ctx context.Context, in *api.NetworkInfoRequest) (*api.NetworkInfo, error) {
	return nil, status.Error(codes.Unimplemented, "not yet implemented")
}
