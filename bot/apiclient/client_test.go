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
}

func (m *mockNotFoundClient) Do(r *http.Request) (*http.Response, error) {
	resp := &http.Response{}
	resp.StatusCode = http.StatusNotFound

	return resp, nil
}

type mockJSONClient struct {
	path     string
	response interface{}
}

func (m *mockJSONClient) Do(r *http.Request) (*http.Response, error) {
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

func mkNotFoundClient() *Client {
	client := NewClient("a", "http://127.0.0.1:5000")
	client.RestClient = &mockNotFoundClient{}

	return client
}

func TestDoGetRequest(t *testing.T) {
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

}

func TestStateNUsers(t *testing.T) {
	t.Parallel()

}

func TestStateUserModes(t *testing.T) {
	t.Parallel()

}

func TestStateChannel(t *testing.T) {
	t.Parallel()

}

func TestStateChannels(t *testing.T) {
	t.Parallel()

}

func TestStateNChannels(t *testing.T) {
	t.Parallel()

}

func TestStateIsOn(t *testing.T) {
	t.Parallel()

}

func TestStoreAuthUser(t *testing.T) {
	t.Parallel()

}

func TestStoreAuthedUser(t *testing.T) {
	t.Parallel()

}

func TestStoreUser(t *testing.T) {
	t.Parallel()

}

func TestStoreUsers(t *testing.T) {
	t.Parallel()

}

func TestStoreNetworkUsers(t *testing.T) {
	t.Parallel()

}

func TestStoreChannelUsers(t *testing.T) {
	t.Parallel()

}

func TestStoreChannel(t *testing.T) {
	t.Parallel()

}

func TestStoreChannels(t *testing.T) {
	t.Parallel()

}

func TestStorePutUser(t *testing.T) {
	t.Parallel()

}

func TestStorePutChannel(t *testing.T) {
	t.Parallel()

}

func TestStoreRemoveUser(t *testing.T) {
	t.Parallel()

}

func TestStoreRemoveChannel(t *testing.T) {
	t.Parallel()

}

func TestStoreLogout(t *testing.T) {
	t.Parallel()

}

func TestStoreLogoutByUsername(t *testing.T) {
	t.Parallel()

}
