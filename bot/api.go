package bot

import (
	"fmt"
	"net/http"

	"github.com/aarondl/ultimateq/data"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/inconshreveable/log15.v2"
)

// botAPI provides a REST api around a bot
type botAPI struct {
	b *Bot
	e *echo.Echo
}

func newBotAPI(b *Bot) botAPI {
	e := echo.New()
	e.SetLogger(EchoLogger{b.Logger})
	e.SetHTTPErrorHandler(errorHandler)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api := botAPI{
		b: b,
		e: e,
	}

	// State
	e.Get("/api/v1/state/net/:network/self", api.stateSelf)
	e.Get("/api/v1/state/net/:network/user/:user", api.stateUser)
	e.Get("/api/v1/state/net/:network/users", api.stateUsers)
	e.Get("/api/v1/state/net/:network/users/count", api.stateUserCount)
	e.Get("/api/v1/state/net/:network/user_modes/:channel/:nick_or_host", api.stateUserModes)

	e.Get("/api/v1/state/net/:network/channel/:channel", api.stateChannel)
	e.Get("/api/v1/state/net/:network/channels", api.stateChannels)
	e.Get("/api/v1/state/net/:network/channels/count", api.stateChannelCount)

	e.Get("/api/v1/state/net/:network/is_on/:channel/:user", api.stateIsOn)

	// Store
	e.Put("/api/v1/store/auth_user", api.storeAuthUser)

	e.Get("/api/v1/store/net/:network/authed_user/:host", api.storeAuthedUser)
	e.Get("/api/v1/store/user/:username", api.storeUser)
	e.Get("/api/v1/store/users", api.storeUsers)
	e.Get("/api/v1/store/net/:network/users", api.storeNetworkUsers)
	e.Get("/api/v1/store/net/:network/channel/:channel/users", api.storeNetworkChannelUsers)

	e.Get("/api/v1/store/net/:network/channel/:channel", api.storeChannel)
	e.Get("/api/v1/store/channels", api.storeChannels)

	e.Put("/api/v1/store/user", api.storePutUser)
	e.Put("/api/v1/store/channel", api.storePutChannel)
	e.Delete("/api/v1/store/user/:username", api.storeDeleteUser)
	e.Delete("/api/v1/store/net/:network/channel/:channel", api.storeDeleteChannel)

	e.Delete("/api/v1/store/logout", api.storeLogout)

	return api
}

// Start the server on the bind address
func (b botAPI) start(addr string) {
	b.e.Run(addr)
}

func errorHandler(err error, e *echo.Context) {
	status := http.StatusInternalServerError

	if httperr, ok := err.(*echo.HTTPError); ok {
		status = httperr.Code()
	}

	e.JSON(status, struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	})
}

func (b botAPI) getNetState(e *echo.Context) (*data.State, error) {
	value := e.Param("network")
	if len(value) == 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "missing network route param")
	}

	state := b.b.State(value)
	if state == nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "unknown network or state disabled")
	}

	return state, nil
}

func getParam(e *echo.Context, key string) (string, error) {
	p := e.Param(key)
	if len(p) == 0 {
		return "", echo.NewHTTPError(http.StatusBadRequest, "missing route parameter: "+key)
	}

	return p, nil
}

func getQueryParam(e *echo.Context, key string) (string, error) {
	p := e.Query(key)
	if len(p) == 0 {
		return "", echo.NewHTTPError(http.StatusBadRequest, "missing query parameter: "+key)
	}

	return p, nil
}

func (b botAPI) stateSelf(e *echo.Context) error {
	state, err := b.getNetState(e)
	if err != nil {
		return err
	}

	self := state.Self()
	e.JSON(http.StatusOK, struct {
		User         data.User         `json:"user"`
		ChannelModes data.ChannelModes `json:"user_modes"`
	}{
		self.User,
		self.ChannelModes,
	})

	return nil
}

func (b botAPI) stateUser(e *echo.Context) error {
	state, err := b.getNetState(e)
	if err != nil {
		return err
	}

	nickOrHost, err := getParam(e, "user")
	if err != nil {
		return err
	}

	user, ok := state.User(nickOrHost)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	e.JSON(http.StatusOK, user)

	return nil
}

func (b botAPI) stateUsers(e *echo.Context) error {
	state, err := b.getNetState(e)
	if err != nil {
		return err
	}

	channel := e.Param("channel")

	var users []string
	if len(channel) > 0 {
		users = state.UsersByChannel(channel)
		if users == nil {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("channel %q not found", channel))
		}
	} else {
		users = state.Users()
	}

	e.JSON(http.StatusOK, users)

	return nil
}

func (b botAPI) stateUserCount(e *echo.Context) error {
	state, err := b.getNetState(e)
	if err != nil {
		return err
	}

	channel := e.Query("channel")

	var users int
	var ok bool
	if len(channel) > 0 {
		users, ok = state.NUsersByChannel(channel)
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("channel %q not found", channel))
		}
	} else {
		users = state.NUsers()
	}

	e.JSON(http.StatusOK, struct {
		Count int `json:"count"`
	}{
		Count: users,
	})

	return nil
}

func (b botAPI) stateUserModes(e *echo.Context) error {
	state, err := b.getNetState(e)
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

	umodes, ok := state.UserModes(nickOrHost, channel)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound,
			fmt.Sprintf("host %q or channel %q not found", nickOrHost, channel))
	}

	e.JSON(http.StatusOK, umodes)

	return nil
}

func (b botAPI) stateChannel(e *echo.Context) error {
	state, err := b.getNetState(e)
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

func (b botAPI) stateChannels(e *echo.Context) error {
	state, err := b.getNetState(e)
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

func (b botAPI) stateChannelCount(e *echo.Context) error {
	state, err := b.getNetState(e)
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

func (b botAPI) stateIsOn(e *echo.Context) error {
	state, err := b.getNetState(e)
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

func (b botAPI) storeAuthUser(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeAuthedUser(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeUser(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeUsers(e *echo.Context) error {
	store := b.b.Store()

	users, err := store.GlobalUsers()
	if err != nil {
		return nil
	}

	e.JSON(http.StatusOK, users)

	return nil
}

func (b botAPI) storeNetworkUsers(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeNetworkChannelUsers(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeChannel(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeChannels(e *echo.Context) error {
	store := b.b.Store()

	channels, err := store.Channels()
	if err != nil {
		return nil
	}

	e.JSON(http.StatusOK, channels)

	return nil
}

func (b botAPI) storePutUser(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storePutChannel(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeDeleteUser(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeDeleteChannel(e *echo.Context) error {
	store := b.b.Store()

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

func (b botAPI) storeLogout(e *echo.Context) error {
	store := b.b.Store()

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

func (e EchoLogger) Debug(args ...interface{}) {
	e.logger.Debug(fmt.Sprintln(args...))
}
func (e EchoLogger) Debugf(format string, args ...interface{}) {
	e.logger.Debug(fmt.Sprintf(format, args...))
}
func (e EchoLogger) Info(args ...interface{}) {
	e.logger.Info(fmt.Sprintln(args...))
}
func (e EchoLogger) Infof(format string, args ...interface{}) {
	e.logger.Info(fmt.Sprintf(format, args...))
}
func (e EchoLogger) Warn(args ...interface{}) {
	e.logger.Warn(fmt.Sprintln(args...))
}
func (e EchoLogger) Warnf(format string, args ...interface{}) {
	e.logger.Warn(fmt.Sprintf(format, args...))
}
func (e EchoLogger) Error(args ...interface{}) {
	e.logger.Error(fmt.Sprintln(args...))
}
func (e EchoLogger) Errorf(format string, args ...interface{}) {
	e.logger.Error(fmt.Sprintf(format, args...))
}
func (e EchoLogger) Fatal(args ...interface{}) {
	e.logger.Crit(fmt.Sprintln(args...))
}
func (e EchoLogger) Fatalf(format string, args ...interface{}) {
	e.logger.Crit(fmt.Sprintf(format, args...))
}
