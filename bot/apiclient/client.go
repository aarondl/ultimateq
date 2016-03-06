package apiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/aarondl/ultimateq/data"
)

var (
	ErrNotFound = errors.New("entity not found")
)

// ServerErr occurs when there's an internal server error with a description
type ServerErr struct {
	Err string `json:"error"`
}

// Error displays the server error
func (s ServerErr) Error() string {
	return s.Err
}

// Client can use
type Client struct {
	Token      string
	Endpoint   string
	RestClient RestClient
}

// NewClient creates a new client for the bot's remote API. Panics if given
// an unparseable endpoint.
func NewClient(token, endpoint string) *Client {
	ep, err := url.Parse(endpoint)
	if err != nil {
		panic("failed to parse endpoint: " + endpoint)
	}
	epURL := getEndpointURL(ep)
	if len(epURL) == 0 {
		panic(epURL)
	}

	return &Client{
		Token:      token,
		Endpoint:   epURL,
		RestClient: NewDefaultRestClient(),
	}
}

func getEndpointURL(u *url.URL) string {
	b := &bytes.Buffer{}
	if len(u.Scheme) != 0 {
		b.WriteString(u.Scheme)
		b.WriteString("://")
	}
	if u.User != nil {
		b.WriteString(u.User.String())
		b.WriteByte('@')
	}
	b.WriteString(u.Host)

	return b.String()
}

func (c *Client) doRequest(verb, path string, request, response interface{}, query ...string) error {
	var body io.Reader
	if request != nil {
		byt, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("failed to serialize request: %v", err)
		}

		body = bytes.NewReader(byt)
	}

	var queryString string
	if l := len(query); l > 0 && l%2 != 0 {
		panic("query must be an even number of key-value pairs")
	} else if l > 0 {
		q := make(url.Values)
		for i := 0; i < len(query); i += 2 {
			q.Set(query[i], query[i+1])
		}
		queryString = "?" + q.Encode()
	}

	endpoint := c.Endpoint + path + queryString
	req, err := http.NewRequest(verb, endpoint, body)
	if err != nil {
		fmt.Errorf("could not create request: %v", err)
	}

	req.Header.Set("User-Agent", "ultimateq apiclient 0.1")
	req.Header.Set("Accept", "text/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	resp, err := c.RestClient.Do(req)
	if err != nil {
		fmt.Errorf("failed to complete request: %v", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusInternalServerError:
		var servErr ServerErr
		if err = unmarshalResponse(resp, &servErr); err != nil {
			return fmt.Errorf("internal server error occurred but failed to retrieve the description: %v", err)
		}
		return servErr
	}

	if response != nil {
		if err = unmarshalResponse(resp, response); err != nil {
			return fmt.Errorf("failed to marshal response: %v", err)
		}
	}

	return nil
}

func unmarshalResponse(response *http.Response, responseObj interface{}) error {
	byt, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err = response.Body.Close(); err != nil {
		return err
	}

	return json.Unmarshal(byt, responseObj)
}

// Self gets the bot's current user information
func (c *Client) Self(network string) (data.User, data.ChannelModes, error) {
	path := fmt.Sprintf("/api/v1/state/net/%s/self", network)

	var resp = struct {
		User  data.User         `json:"user"`
		Modes data.ChannelModes `json:"modes"`
	}{}

	err := c.doRequest("GET", path, nil, &resp)
	return resp.User, resp.Modes, err
}

// StateUser gets a user from the state store
func (c *Client) StateUser(network, nickOrHost string) (data.User, error) {
	path := fmt.Sprintf("/api/v1/state/net/%s/user/%s", network, nickOrHost)

	var u data.User
	err := c.doRequest("GET", path, nil, &u)
	return u, err
}

// StateUsers gets the list of users from the state store optionally
// filtering by a channel name, leave it empty for all users.
func (c *Client) StateUsers(channelFilter string) ([]string, error) {
	return nil, nil
}

// StateNUsers gets the number of users from the state store optionally
// filtering by a channel name, leave it empty for all users.
func (c *Client) StateNUsers(channelFilter string) (int, error) {
	return 0, nil
}

// StateUserModes gets the user modes from a given nick or hostname on
// a channel.
func (c *Client) StateUserModes(channel string, nickOrHost string) (data.ChannelModes, error) {
	var m data.ChannelModes
	return m, nil
}

// StateChannel gets a channel.
func (c *Client) StateChannel(channel string) (data.Channel, error) {
	var ch data.Channel
	return ch, nil
}

// StateChannels gets the list of channels from the state store optionally
// filtering by a user nick/host, leave it empty for all channels.
func (c *Client) StateChannels(userFilter string) ([]string, error) {
	return nil, nil
}

// StateNChannels gets the number of channels from the state store optionally
// filtering by a user nick/host, leave it empty for all channels.
func (c *Client) StateNChannels(userFilter string) (int, error) {
	return 0, nil
}

// StateIsOn checks to see if nickOrHost is on channel.
func (c *Client) StateIsOn(channel string, nickOrHost string) (bool, error) {
	return false, nil
}

// StoreAuthUser authenticates a user.
func (c *Client) StoreAuthUser(network, host, username, password string, permanent bool) error {
	return nil
}

// StoreAuthedUser retrieves an authenticated user.
func (c *Client) StoreAuthedUser(network, host string) (*data.StoredUser, error) {
	return nil, nil
}

// StoreUser gets a stored user.
func (c *Client) StoreUser(username string) (*data.StoredUser, error) {
	return nil, nil
}

// StoreUsers gets all the users with global access.
func (c *Client) StoreUsers() ([]*data.StoredUser, error) {
	return nil, nil
}

// StoreNetworkUsers gets all the users with network access.
func (c *Client) StoreNetworkUsers(network string) ([]*data.StoredUser, error) {
	return nil, nil
}

// StoreChannelUsers gets all the users with channel access.
func (c *Client) StoreChannelUsers(network, channel string) ([]*data.StoredUser, error) {
	return nil, nil
}

// StoreChannel gets a stored channel.
func (c *Client) StoreChannel(network, channel string) ([]*data.StoredChannel, error) {
	return nil, nil
}

// StoreChannels gets all the stored channels.
func (c *Client) StoreChannels() ([]*data.StoredChannel, error) {
	return nil, nil
}

// StorePutUser saves a stored user to the database.
func (c *Client) StorePutUser(user *data.StoredUser) error {
	return nil
}

// StorePutChannel saves a stored channel to the database.
func (c *Client) StorePutChannel(channel *data.StoredChannel) error {
	return nil
}

// StoreRemoveUser removes a user from the database.
func (c *Client) StoreRemoveUser(username string) error {
	return nil
}

// StoreRemoveChannel removes a channel from the database.
func (c *Client) StoreRemoveChannel(network, channel string) error {
	return nil
}

// StoreLogout logs a user out from a network.
func (c *Client) StoreLogout(network, host string) error {
	return nil
}

// StoreLogoutByUsername logs a user out of all networks he's authenticated to.
func (c *Client) StoreLogoutByUsername(username string) error {
	return nil
}
