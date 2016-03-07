package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/aarondl/ultimateq/data"
	"github.com/aarondl/ultimateq/irc"
)

type mockRestClient struct {
	Status   int
	Verb     string
	Path     string
	Query    url.Values
	Headers  http.Header
	Response string
	Request  string
	Error    string
}

func (m *mockRestClient) Do(r *http.Request) (*http.Response, error) {
	resp := &http.Response{}
	resp.StatusCode = m.Status
	m.Verb = r.Method
	m.Path = r.URL.Path
	m.Query = r.URL.Query()
	m.Headers = r.Header

	if len(m.Response) > 0 {
		resp.Body = ioutil.NopCloser(strings.NewReader(m.Response))
	}
	if len(m.Error) > 0 {
		e := fmt.Sprintf(`{"error":"%s"}`, m.Error)
		resp.Body = ioutil.NopCloser(strings.NewReader(e))
	}

	if r.Body != nil {
		byt, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		if err := r.Body.Close(); err != nil {
			panic(err)
		}

		m.Request = string(byt)
	}

	return resp, nil
}

type mockNotFoundClient struct {
	path string
}

func (m *mockNotFoundClient) Do(r *http.Request) (*http.Response, error) {
	if m.path != r.URL.Path {
		panic(fmt.Sprintf("Path was wrong got %q wanted %q", r.URL.Path, m.path))
	}

	resp := &http.Response{}
	resp.StatusCode = http.StatusNotFound

	return resp, nil
}

type mockJSONClient struct {
	path     string
	response interface{}
	request  []byte
	query    url.Values
}

func (m *mockJSONClient) Do(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		byt, err := ioutil.ReadAll(r.Body)
		if len(byt) > 0 {
			m.request = byt
		}
		if err = r.Body.Close(); err != nil {
			panic(err)
		}
	}

	byt, err := json.Marshal(m.response)
	if err != nil {
		return nil, err
	}

	if r.URL.Path != m.path {
		panic(fmt.Sprintf("Path was wrong got %q wanted %q", r.URL.Path, m.path))
	}

	resp := &http.Response{}
	resp.StatusCode = http.StatusOK
	resp.Body = ioutil.NopCloser(bytes.NewReader(byt))

	m.query = r.URL.Query()

	return resp, nil
}

func mkJSONClient(path string, response interface{}) *Client {
	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockJSONClient{
		path:     path,
		response: response,
	}
	client.RestClient = mock

	return client
}

func mkNotFoundClient(path string) *Client {
	client := NewClient("a", "http://127.0.0.1:5000")
	client.RestClient = &mockNotFoundClient{
		path: path,
	}

	return client
}

func TestDoGetRequest(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockRestClient{
		Status:   http.StatusOK,
		Response: `{"yep":true}`,
	}
	client.RestClient = mock

	reqResp := []struct {
		Yep bool `json:"yep"`
	}{
		{Yep: true},
		{Yep: false},
	}

	err := client.doRequest("PUT", "/hello/friend", &reqResp[0], &reqResp[1], "a", "b")
	if err != nil {
		t.Error(err)
	}

	if mock.Status != http.StatusOK {
		t.Error("Wrong status:", mock.Status)
	}
	if mock.Verb != "PUT" {
		t.Error("Wrong verb:", mock.Verb)
	}
	// Just re-using the magic string for ezness, response should not truly equal request
	if mock.Request != mock.Response {
		t.Error("Request was not serialized properly:", mock.Request)
	}
	if mock.Query.Get("a") != "b" {
		t.Error("Query string was not added properly:", mock.Query)
	}
	if reqResp[1].Yep != true {
		t.Error("Did not deserialize response properly")
	}
	if mock.Path != "/hello/friend" {
		t.Error("Path was wrong:", mock.Path)
	}
	if mock.Headers.Get("Accept") != "text/json" {
		t.Error("Accept header was wrong:", mock.Headers.Get("Accept"))
	}
	if mock.Headers.Get("Authorization") != "Bearer a" {
		t.Error("Authorization header was wrong:", mock.Headers.Get("Authorization"))
	}
}

func TestDoGetRequestError(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockRestClient{
		Status: http.StatusInternalServerError,
		Error:  "hello",
	}
	client.RestClient = mock

	err := client.doRequest("GET", "/hello/friend", nil, nil)
	if s, ok := err.(ServerErr); !ok {
		t.Error("Expected an error back")
	} else if s.Err != "hello" {
		t.Error("Error message was encoded wrong:", s.Err)
	}
}

func TestDoGetRequestNotFound(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockRestClient{
		Status: http.StatusNotFound,
	}
	client.RestClient = mock

	err := client.doRequest("GET", "/hello/friend", nil, nil)
	if err != ErrNotFound {
		t.Error("Expected not found error:", err)
	}
}

func TestGetEndpointURL(t *testing.T) {
	t.Parallel()

	ep, err := url.Parse("http://rofl:clown@127.0.0.1:5000/hello/there")
	if err != nil {
		t.Fatal(err)
	}

	if got := getEndpointURL(ep); got != "http://rofl:clown@127.0.0.1:5000" {
		t.Error("Endpoint was wrong:", got)
	}
}

func TestSelf(t *testing.T) {
	t.Parallel()

	var self = struct {
		User  data.User         `json:"user"`
		Modes data.ChannelModes `json:"modes"`
	}{}

	self.User.Host = "fish!fish@fish.com"

	client := mkJSONClient("/api/v1/state/net/network/self", self)
	user, modes, err := client.Self("network")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(self.User, user) {
		t.Errorf("User was wrong: %#v", user)
	}
	if !reflect.DeepEqual(self.Modes, modes) {
		t.Errorf("Modes were wrong: %#v", modes)
	}
}

func TestStateUser(t *testing.T) {
	t.Parallel()

	var u data.User
	u.Host = irc.Host("fish!fish@fish.com")

	client := mkJSONClient("/api/v1/state/net/network/user/fish", u)

	user, err := client.StateUser("network", "fish")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(u, user) {
		t.Errorf("Wrong user: %#v", user)
	}
}

func TestStateUsers(t *testing.T) {
	t.Parallel()

	users1 := []string{"fish!fish@fish.com", "zamn!zamn@zamn.com"}
	users2 := []string{"fish!fish@fish.com"}

	client := mkJSONClient("/api/v1/state/net/network/users", users1)

	users, err := client.StateUsers("network", "")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(users1, users) {
		t.Errorf("User list was wrong: %#v", users)
	}

	client = mkJSONClient("/api/v1/state/net/network/users", users2)

	users, err = client.StateUsers("network", "channel")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(users2, users) {
		t.Errorf("User list was wrong: %#v", users)
	}
}

func TestStateNUsers(t *testing.T) {
	t.Parallel()

	count1 := countResponse{5}
	count2 := countResponse{2}

	client := mkJSONClient("/api/v1/state/net/network/users/count", count1)

	count, err := client.StateNUsers("network", "")
	if err != nil {
		t.Error(err)
	}

	if count != count1.Count {
		t.Errorf("Count was wrong: %#v", count)
	}

	client = mkJSONClient("/api/v1/state/net/network/users/count", count2)

	count, err = client.StateNUsers("network", "channel")
	if err != nil {
		t.Error(err)
	}

	if count != count2.Count {
		t.Errorf("Count was wrong: %#v", count)
	}
}

func TestStateUserModes(t *testing.T) {
	t.Parallel()

	var m data.ChannelModes

	client := mkJSONClient("/api/v1/state/net/network/user_modes/channel/fish!fish@fish", m)

	modes, err := client.StateUserModes("network", "channel", "fish!fish@fish")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(modes, m) {
		t.Errorf("Modes were wrong: %#v", modes)
	}
}

func TestStateChannel(t *testing.T) {
	t.Parallel()

	var c data.Channel
	c.Name = "#channel"

	client := mkJSONClient("/api/v1/state/net/network/channel/#channel", c)

	channel, err := client.StateChannel("network", "#channel")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(channel, c) {
		t.Errorf("Wrong channel: %#v", channel)
	}
}

func TestStateChannels(t *testing.T) {
	t.Parallel()

	channels1 := []string{"fish!fish@fish.com", "zamn!zamn@zamn.com"}
	channels2 := []string{"fish!fish@fish.com"}

	client := mkJSONClient("/api/v1/state/net/network/channels", channels1)

	channels, err := client.StateChannels("network", "")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(channels1, channels) {
		t.Errorf("Channel list was wrong: %#v", channels)
	}

	client = mkJSONClient("/api/v1/state/net/network/channels", channels2)

	channels, err = client.StateChannels("network", "channel")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(channels2, channels) {
		t.Errorf("Channel list was wrong: %#v", channels)
	}
}

func TestStateNChannels(t *testing.T) {
	t.Parallel()

	count1 := countResponse{5}
	count2 := countResponse{2}

	client := mkJSONClient("/api/v1/state/net/network/channels/count", count1)

	count, err := client.StateNChannels("network", "")
	if err != nil {
		t.Error(err)
	}

	if count != count1.Count {
		t.Errorf("Count was wrong: %#v", count)
	}

	client = mkJSONClient("/api/v1/state/net/network/channels/count", count2)

	count, err = client.StateNChannels("network", "channel")
	if err != nil {
		t.Error(err)
	}

	if count != count2.Count {
		t.Errorf("Count was wrong: %#v", count)
	}
}

func TestStateIsOn(t *testing.T) {
	t.Parallel()

	client := mkNotFoundClient("/api/v1/state/net/network/is_on/#channel/fish!fish@fish")
	is, err := client.StateIsOn("network", "#channel", "fish!fish@fish")
	if err != nil {
		t.Error(err)
	}

	if is {
		t.Error("Is is wrong:", is)
	}

	client = mkJSONClient("/api/v1/state/net/network/is_on/#channel/fish!fish@fish", nil)
	is, err = client.StateIsOn("network", "#channel", "fish!fish@fish")
	if err != nil {
		t.Error(err)
	}

	if !is {
		t.Error("Is is wrong:", is)
	}
}

func TestStoreAuthUser(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockJSONClient{
		path:     "/api/v1/store/auth_user",
		response: nil,
	}
	client.RestClient = mock

	err := client.StoreAuthUser("network", "fish!fish@fish", "fish", "pwd", false)
	if err != nil {
		t.Error(err)
	}

	exp := `{"network":"network","host":"fish!fish@fish","username":"fish","password":"pwd","permanent":false}`

	if got := string(mock.request); got != exp {
		t.Error("Request was wrong:", got)
	}
}

func TestStoreAuthedUser(t *testing.T) {
	t.Parallel()

	u := &data.StoredUser{}
	u.Username = "fish"

	client := mkJSONClient("/api/v1/store/net/network/authed_user/fish!fish@fish", u)
	user, err := client.StoreAuthedUser("network", "fish!fish@fish")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(user, u) {
		t.Errorf("User was wrong: %#v", user)
	}
}

func TestStoreUser(t *testing.T) {
	t.Parallel()

	u := &data.StoredUser{}
	u.Username = "fish"

	client := mkJSONClient("/api/v1/store/user/fish", u)
	user, err := client.StoreUser("fish")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(user, u) {
		t.Errorf("User was wrong: %#v", user)
	}
}

func TestStoreUsers(t *testing.T) {
	t.Parallel()

	u := []*data.StoredUser{
		&data.StoredUser{Username: "fish"},
		&data.StoredUser{Username: "zamn"},
	}

	client := mkJSONClient("/api/v1/store/users", u)
	users, err := client.StoreUsers()
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(users, u) {
		t.Errorf("User list was wrong: %#v", users)
	}
}

func TestStoreNetworkUsers(t *testing.T) {
	t.Parallel()

	u := []*data.StoredUser{
		&data.StoredUser{Username: "fish"},
		&data.StoredUser{Username: "zamn"},
	}

	client := mkJSONClient("/api/v1/store/net/network/users", u)
	users, err := client.StoreNetworkUsers("network")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(users, u) {
		t.Errorf("User list was wrong: %#v", users)
	}
}

func TestStoreChannelUsers(t *testing.T) {
	t.Parallel()

	u := []*data.StoredUser{
		&data.StoredUser{Username: "fish"},
		&data.StoredUser{Username: "zamn"},
	}

	client := mkJSONClient("/api/v1/store/net/network/channel/#channel/users", u)
	users, err := client.StoreChannelUsers("network", "#channel")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(users, u) {
		t.Errorf("User list was wrong: %#v", users)
	}
}

func TestStoreChannel(t *testing.T) {
	t.Parallel()

	c := &data.StoredChannel{}
	c.Name = "#channel"

	client := mkJSONClient("/api/v1/store/net/network/channel/#channel", c)
	channel, err := client.StoreChannel("network", "#channel")
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(channel, c) {
		t.Errorf("Channel was wrong: %#v", channel)
	}
}

func TestStoreChannels(t *testing.T) {
	t.Parallel()

	c := []*data.StoredChannel{
		&data.StoredChannel{Name: "#channel1"},
		&data.StoredChannel{Name: "#channel2"},
	}

	client := mkJSONClient("/api/v1/store/channels", c)
	channels, err := client.StoreChannels()
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(channels, c) {
		t.Errorf("Channel was wrong: %#v", channels)
	}
}

func TestStorePutUser(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockJSONClient{
		path: "/api/v1/store/user",
	}
	client.RestClient = mock

	u := &data.StoredUser{Username: "fish"}

	err := client.StorePutUser(u)
	if err != nil {
		t.Error(err)
	}

	exp := `{"username":"fish","password":null,"masks":null,"access":null,"data":null}`

	if got := string(mock.request); got != exp {
		t.Error("Request was wrong:", got)
	}
}

func TestStorePutChannel(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockJSONClient{
		path: "/api/v1/store/channel",
	}
	client.RestClient = mock

	c := &data.StoredChannel{Name: "#channel"}

	err := client.StorePutChannel(c)
	if err != nil {
		t.Error(err)
	}

	exp := `{"netid":"","name":"#channel","data":null}`

	if got := string(mock.request); got != exp {
		t.Error("Request was wrong:", got)
	}
}

func TestStoreRemoveUser(t *testing.T) {
	t.Parallel()

	client := mkJSONClient("/api/v1/store/users/fish", nil)
	err := client.StoreRemoveUser("fish")
	if err != nil {
		t.Error(err)
	}
}

func TestStoreRemoveChannel(t *testing.T) {
	t.Parallel()

	client := mkJSONClient("/api/v1/store/net/network/channel/#channel", nil)
	err := client.StoreRemoveChannel("network", "#channel")
	if err != nil {
		t.Error(err)
	}
}

func TestStoreLogout(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockJSONClient{
		path: "/api/v1/store/logout",
	}
	client.RestClient = mock

	err := client.StoreLogout("network", "fish!fish@fish")
	if err != nil {
		t.Error(err)
	}

	if q := mock.query.Get("network"); q != "network" {
		t.Error("Network param wrong:", q)
	}
	if q := mock.query.Get("host"); q != "fish!fish@fish" {
		t.Error("Host param wrong:", q)
	}
}

func TestStoreLogoutByUsername(t *testing.T) {
	t.Parallel()

	client := NewClient("a", "http://127.0.0.1:5000")
	mock := &mockJSONClient{
		path: "/api/v1/store/logout",
	}
	client.RestClient = mock

	err := client.StoreLogoutByUsername("fish")
	if err != nil {
		t.Error(err)
	}

	if q := mock.query.Get("username"); q != "fish" {
		t.Error("Username param wrong:", q)
	}
}
