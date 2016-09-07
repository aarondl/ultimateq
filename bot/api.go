package bot

import (
	"net"

	"github.com/aarondl/ultimateq/api"
	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/registrar"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// api provides a REST api around a bot
type apiServer struct {
	bot   *Bot
	proxy *registrar.Proxy
}

func newAPIServer(b *Bot) apiServer {
	server := apiServer{
		bot:   b,
		proxy: registrar.NewProxy(b),
	}

	return server
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
	return nil, nil
}

func (a apiServer) StateChannels(ctx context.Context, in *api.NetworkQuery) (*api.ListResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (a apiServer) StateChannelCount(ctx context.Context, in *api.NetworkQuery) (*api.CountResponse, error) {
	state, err := a.getState(in.Network)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

/*

func (a api) stateUserModes(e echo.Context) error {
	state, err := a.getNetState(e)
	if err != nil {
		return err
	}

	nickOrHost, err := getParam(e, "nick_or_host")
	if err != nil {
		return err
	}
	channel, err := getParam(e, "channel")
	if err != nil {
		return err
	}


	e.JSON(http.StatusOK, umodes)

	return nil
}

func (a api) stateChannel(e echo.Context) error {
	state, err := a.getNetState(e)
	if err != nil {
		return err
	}

	chanName, err := getParam(e, "channel")
	if err != nil {
		return err
	}

	channel, ok := state.Channel(chanName)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("channel %q not found", chanName))
	}

	e.JSON(http.StatusOK, channel)

	return nil
}

func (a api) stateChannels(e echo.Context) error {
	state, err := a.getNetState(e)
	if err != nil {
		return err
	}

	user := e.Query("user")

	var channels []string
	if len(user) > 0 {
		channels = state.ChannelsByUser(user)
		if channels == nil {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("nick or host %q not found", user))
		}
	} else {
		channels = state.Channels()
	}

	e.JSON(http.StatusOK, channels)

	return nil
}
*/

/*
func (a api) getNetState(e echo.Context) (*data.State, error) {
	value := e.Param("network")
	if len(value) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "missing network route param")
	}

	state := a.bot.State(value)
	if state == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "unknown network or state disabled")
	}

	return state, nil
}

func getParam(e echo.Context, key string) (string, error) {
	p := e.Param(key)
	if len(p) == 0 {
		return "", echo.NewHTTPError(http.StatusBadRequest, "missing route parameter: "+key)
	}

	return p, nil
}

func getQueryParam(e echo.Context, key string) (string, error) {
	p := e.Query(key)
	if len(p) == 0 {
		return "", echo.NewHTTPError(http.StatusBadRequest, "missing query parameter: "+key)
	}

	return p, nil
}

func getExtName(e echo.Context) (ext string) {
	return e.Get("ext").(string)
}

func (a api) connect(e echo.Context) error {
	ext := getExtName(e)

	stdRes := e.Response().(*standard.Response)
	w := stdRes.ResponseWriter.(http.Hijacker)

	conn, buffer, err := w.Hijack()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "could not hijack connection")
	}

	// Flush buffer and discard
	buffer.Flush()

	// Reset read/write deadlines
	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})

	a.dispatch.New(ext, conn, func(name string) {
		a.dispatch.Remove(name)
		a.proxy.Unregister(name)
	})

	return nil
}

type registerMessage struct {
	Network string `json:"network,omitempty"`
	Channel string `json:"channel,omitempty"`
	Event   string `json:"event,omitempty"`
}

type unregisterMessage struct {
	ID uint64 `json:"id"`
}

func (a api) register(e echo.Context) error {
	var r registerMessage

	if err := e.Bind(&r); err != nil {
		return err
	}

	ext := getExtName(e)
	proxy := a.proxy.Get(ext)
	handler := a.dispatch.Get(ext)
	proxy.Register(r.Network, r.Channel, r.Event, handler)

	return nil
}

func (a api) unregister(e echo.Context) error {
	var r unregisterMessage

	if err := e.Bind(&r); err != nil {
		return err
	}

	ext := getExtName(e)
	proxy := a.proxy.Get(ext)
	did := proxy.Unregister(r.ID)

	if !did {
		return e.NoContent(http.StatusNotFound)
	}

	return nil
}


func (a api) stateChannelCount(e echo.Context) error {
	state, err := a.getNetState(e)
	if err != nil {
		return err
	}

	user := e.Query("user")

	var channels int
	var ok bool
	if len(user) > 0 {
		channels, ok = state.NChannelsByUser(user)
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("user %q not found", user))
		}
	} else {
		channels = state.NChannels()
	}

	e.JSON(http.StatusOK, struct {
		Count int `json:"count"`
	}{
		Count: channels,
	})

	return nil
}

func (a api) stateIsOn(e echo.Context) error {
	state, err := a.getNetState(e)
	if err != nil {
		return err
	}

	channel, err := getParam(e, "channel")
	if err != nil {
		return err
	}
	user, err := getParam(e, "user")
	if err != nil {
		return err
	}

	ison := state.IsOn(user, channel)

	if ison {
		e.NoContent(http.StatusOK)
	} else {
		e.NoContent(http.StatusNotFound)
	}

	return nil
}

func (a api) storeAuthUser(e echo.Context) error {
	store := a.bot.Store()

	auth := struct {
		Network   string `json:"network"`
		Host      string `json:"host"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Permanent bool   `json:"permanent"`
	}{}

	if err := e.Bind(&auth); err != nil {
		return err
	}

	if len(auth.Host) == 0 || len(auth.Network) == 0 ||
		len(auth.Username) == 0 || len(auth.Password) == 0 {

		return echo.NewHTTPError(http.StatusBadRequest, "must supply all parameters (network, host, username, password)")
	}

	var err error
	if auth.Permanent {
		_, err = store.AuthUserPerma(auth.Network, auth.Host, auth.Username, auth.Password)
	} else {
		_, err = store.AuthUserTmp(auth.Network, auth.Host, auth.Username, auth.Password)
	}

	if _, ok := err.(data.AuthError); ok {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	} else if err != nil {
		return err
	}

	e.NoContent(http.StatusOK)

	return nil
}

func (a api) storeAuthedUser(e echo.Context) error {
	store := a.bot.Store()

	network, err := getParam(e, "network")
	if err != nil {
		return err
	}
	host, err := getParam(e, "host")
	if err != nil {
		return err
	}

	authedUser := store.AuthedUser(network, host)
	if authedUser == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	e.JSON(http.StatusOK, authedUser)

	return nil
}

func (a api) storeUser(e echo.Context) error {
	store := a.bot.Store()

	username, err := getParam(e, "username")
	if err != nil {
		return err
	}

	user, err := store.FindUser(username)
	if err != nil {
		return err
	} else if user == nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	e.JSON(http.StatusOK, user)

	return nil
}

func (a api) storeUsers(e echo.Context) error {
	store := a.bot.Store()

	users, err := store.GlobalUsers()
	if err != nil {
		return nil
	}

	e.JSON(http.StatusOK, users)

	return nil
}

func (a api) storeNetworkUsers(e echo.Context) error {
	store := a.bot.Store()

	network, err := getParam(e, "network")
	if err != nil {
		return err
	}

	users, err := store.NetworkUsers(network)
	if err != nil {
		return nil
	}

	e.JSON(http.StatusOK, users)

	return nil
}

func (a api) storeNetworkChannelUsers(e echo.Context) error {
	store := a.bot.Store()

	network, err := getParam(e, "network")
	if err != nil {
		return err
	}
	channel, err := getParam(e, "channel")
	if err != nil {
		return err
	}

	users, err := store.ChanUsers(network, channel)
	if err != nil {
		return nil
	}

	e.JSON(http.StatusOK, users)

	return nil
}

func (a api) storeChannel(e echo.Context) error {
	store := a.bot.Store()

	network, err := getParam(e, "network")
	if err != nil {
		return err
	}
	chanName, err := getParam(e, "channel")
	if err != nil {
		return err
	}

	channel, err := store.FindChannel(network, chanName)
	if err != nil {
		return nil
	}
	if channel == nil {
		return echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("channel %q not found", chanName))
	}

	e.JSON(http.StatusOK, channel)

	return nil
}

func (a api) storeChannels(e echo.Context) error {
	store := a.bot.Store()

	channels, err := store.Channels()
	if err != nil {
		return nil
	}

	e.JSON(http.StatusOK, channels)

	return nil
}

func (a api) storePutUser(e echo.Context) error {
	store := a.bot.Store()

	var user data.StoredUser
	if err := e.Bind(&user); err != nil {
		return err
	}

	if err := store.SaveUser(&user); err != nil {
		return err
	}

	e.NoContent(http.StatusCreated)

	return nil
}

func (a api) storePutChannel(e echo.Context) error {
	store := a.bot.Store()

	var channel data.StoredChannel
	if err := e.Bind(&channel); err != nil {
		return err
	}

	if err := store.SaveChannel(&channel); err != nil {
		return err
	}

	e.NoContent(http.StatusCreated)

	return nil
}

func (a api) storeDeleteUser(e echo.Context) error {
	store := a.bot.Store()

	username, err := getParam(e, "username")
	if err != nil {
		return err
	}

	ok, err := store.RemoveUser(username)
	if err != nil {
		return err
	}

	if ok {
		e.NoContent(http.StatusOK)
	} else {
		e.NoContent(http.StatusNotFound)
	}

	return nil
}

func (a api) storeDeleteChannel(e echo.Context) error {
	store := a.bot.Store()

	network, err := getParam(e, "network")
	if err != nil {
		return err
	}
	channel, err := getParam(e, "channel")
	if err != nil {
		return err
	}

	ok, err := store.RemoveChannel(network, channel)
	if err != nil {
		return err
	}

	if ok {
		e.NoContent(http.StatusOK)
	} else {
		e.NoContent(http.StatusNotFound)
	}

	return nil
}

func (a api) storeLogout(e echo.Context) error {
	store := a.bot.Store()

	network, host, username := e.Query("network"), e.Query("host"), e.Query("username")
	hasNet, hasHost, hasUname := len(network) > 0, len(host) > 0, len(username) > 0
	if (!hasUname && !hasNet && !hasHost) || (hasUname && (hasNet || hasHost)) {
		return echo.NewHTTPError(http.StatusBadRequest, "must supply username OR network and host")
	}

	if hasUname {
		store.LogoutByUsername(username)
	} else {
		store.Logout(network, host)
	}

	e.NoContent(http.StatusOK)

	return nil
}

type EchoLogger struct {
	logger log15.Logger
}

func (e EchoLogger) Write(b []byte) (int, error) {
	e.logger.Info(string(b))
	return len(b), nil
}

const (
	bearer = "Bearer"
)

func jwtAuth(key string) echo.MiddlewareFunc {
	return func(next echo.Handler) echo.Handler {
		return echo.HandlerFunc(func(c echo.Context) error {
			auth := c.Request().Header().Get("Authorization")
			l := len(bearer)
			he := echo.ErrUnauthorized

			if len(auth) > l+1 && auth[:l] == bearer {
				t, err := jwt.Parse(auth[l+1:], func(token *jwt.Token) (interface{}, error) {

					// Always check the signing method
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
					}

					// Return the key for validation
					return []byte(key), nil
				})
				if err == nil && t.Valid {
					// Store token claims in echo.Context
					if uqIntf, ok := t.Claims["uq"]; ok {
						if uq, ok := uqIntf.(string); ok {
							c.Set("uq", uq)
						}
					}
					if extNameIntf, ok := t.Claims["ext"]; ok {
						if extName, ok := extNameIntf.(string); ok {
							c.Set("ext", extName)
						}
					}

					return next.Handle(c)
				}
			}
			return he
		})
	}
}

func checkClaims(next echo.Handler) echo.Handler {
	return echo.HandlerFunc(func(c echo.Context) error {

		var errStr string
		if uqIntf := c.Get("uq"); uqIntf == nil {
			errStr = "missing uq claim in token"
		} else if uq, ok := uqIntf.(string); !ok {
			errStr = "uq claim in token wrong type"
		} else if uq != "extension" {
			errStr = `uq claim in token must be "extension"`
		}

		if extIntf := c.Get("ext"); extIntf == nil {
			errStr = `ext claim must exist`
		} else if ext, ok := extIntf.(string); !ok || len(ext) == 0 {
			errStr = `ext claim must be a non-empty string`
		}

		if len(errStr) > 0 {
			return echo.NewHTTPError(http.StatusBadRequest, errStr)
		}

		return next.Handle(c)
	})
}
*/

func (a apiServer) Pipe(pipe api.Ext_PipeServer) error {
	return nil
}

func (a apiServer) Register(ctx context.Context, in *api.RegisterRequest) (*api.Result, error) {
	return nil, nil
}

func (a apiServer) Unregister(ctx context.Context, in *api.UnregisterRequest) (*api.Result, error) {
	return nil, nil
}

func (a apiServer) StateIsOn(ctx context.Context, in *api.ChannelQuery) (*api.Result_OK, error) {
	return nil, nil
}

func (a apiServer) StoreAuthUser(ctx context.Context, in *api.AuthUserRequest) (*api.Result_OK, error) {
	return nil, nil
}

func (a apiServer) StoreAuthedUser(ctx context.Context, in *api.NetworkQuery) (*api.StoredUser, error) {
	return nil, nil
}

func (a apiServer) StoreUser(ctx context.Context, in *api.Query) (*api.StoredUser, error) {
	return nil, nil
}

func (a apiServer) StoreUsers(ctx context.Context, in *api.Empty) (*api.StoredUsersResponse, error) {
	return nil, nil
}

func (a apiServer) StoreUsersByNetwork(ctx context.Context, in *api.Query) (*api.StoredUsersResponse, error) {
	return nil, nil
}

func (a apiServer) StoreUsersByChannel(ctx context.Context, in *api.NetworkQuery) (*api.StoredUsersResponse, error) {
	return nil, nil
}

func (a apiServer) StoreChannel(ctx context.Context, in *api.NetworkQuery) (*api.StoredChannel, error) {
	return nil, nil
}

func (a apiServer) StoreChannels(ctx context.Context, in *api.Empty) (*api.StoredChannelsResponse, error) {
	return nil, nil
}

func (a apiServer) StorePutUser(ctx context.Context, in *api.StoredUser) (*api.Result, error) {
	return nil, nil
}

func (a apiServer) StorePutChannel(ctx context.Context, in *api.StoredChannel) (*api.Result, error) {
	return nil, nil
}

func (a apiServer) StoreDeleteUser(ctx context.Context, in *api.Query) (*api.Result, error) {
	return nil, nil
}

func (a apiServer) StoreDeleteChannel(ctx context.Context, in *api.NetworkQuery) (*api.Result, error) {
	return nil, nil
}
