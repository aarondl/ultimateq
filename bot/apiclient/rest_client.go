package apiclient

import "net/http"

// RestClient can send requests and receive responses
type RestClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// NewDefaultRestClient creates a rest client from the http.Client type
func NewDefaultRestClient() RestClient {
	client := &http.Client{}
	return client
}
