package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
)

// Reusable structure with the required metadata for talking to a GraphQL API.
// Client should be reused (not reinstantiated) across multiple requests.
type Client struct {
	Endpoint *url.URL

	client *http.Client
	header *http.Header
}

// Request holds the common structure/syntax across all GraphQL requests.
type Request struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// Response wraps the common structure/syntax across all GraphQL responses.
type Response struct {
	Data   interface{}
	Errors []interface{}
}

// New instantiates Client for `endpoint`.
// If you're looking for a way to instantiate Client with custom headers, check
// out NewWithHeader.
func New(endpoint string) (*Client, error) {
	return NewWithHeader(endpoint, &http.Header{})
}

// NewWithHeader instantiates Client for `endpoint` with custom headers.
// If you don't need to specify headers, check out New.
func NewWithHeader(endpoint string, header *http.Header) (*Client, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}

	return &Client{
		Endpoint: url,
		client:   &http.Client{},
		header:   header,
	}, nil
}

// Query performs a single GraphQL query and returns its results as a Response.
//
// The returned error does not represent GraphQL errors, those are stored within
// the Response itself.
func (c *Client) Query(request *Request) (*Response, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest(http.MethodPost, c.Endpoint.String(), bytes.NewBuffer(requestBody))

	if err != nil {
		return nil, err
	}

	r.Header = *c.header

	httpResponse, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}

	responseBodyBuffer := new(bytes.Buffer)
	if _, err := responseBodyBuffer.ReadFrom(httpResponse.Body); err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(responseBodyBuffer.Bytes(), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// QueryMany performs multiple GraphQL queries concurrently and returns a
// channel that will hold their results as Response structures.
//
// The returned errors do not represent GraphQL errors, those are stored within
// the Responses themselves.
//
// QueryMany is non-blocking, but you should handle that by reading responses
// from the returned Response channel.
// The Response channel is closed automatically when all queries are done.
func (c *Client) QueryMany(requests []*Request) (chan *Response, chan error) {
	var wg sync.WaitGroup
	responses := make(chan *Response, len(requests))
	errors := make(chan error, len(requests))

	for _, request := range requests {
		wg.Add(1)
		go func(request *Request) {
			defer wg.Done()
			response, error := c.Query(request)
			if response != nil {
				responses <- response
			}
			if error != nil {
				errors <- error
			}
		}(request)
	}

	go func() {
		wg.Wait()
		close(responses)
		close(errors)
	}()

	return responses, errors
}
