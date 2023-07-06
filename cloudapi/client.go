package cloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"time"
)

// defaultURL for Atlas Cloud.
const defaultURL = "https://api.atlasgo.cloud/query"

// roundTripper is a http.RoundTripper that adds the Authorization header.
type roundTripper struct {
	token string
}

// RoundTrip implements http.RoundTripper.
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("User-Agent", "gh-atlas-cli")
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultTransport.RoundTrip(req)
}

// Client is a client for the Atlas Cloud API.
type Client struct {
	client   *http.Client
	endpoint string
	token    string
}

// New creates a new Client for the Atlas Cloud API.
func New(endpoint, token string) *Client {
	if endpoint == "" {
		endpoint = defaultURL
	}
	return &Client{
		endpoint: endpoint,
		client: &http.Client{
			Transport: &roundTripper{
				token: token,
			},
			Timeout: time.Second * 30,
		},
		token: token,
	}
}

// sends a POST request to the Atlas Cloud API.
func (c *Client) post(ctx context.Context, query string, vars, data any) error {
	body, err := json.Marshal(struct {
		Query     string `json:"query"`
		Variables any    `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: vars,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer req.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	var scan = struct {
		Data   any           `json:"data"`
		Errors gqlerror.List `json:"errors,omitempty"`
	}{
		Data: data,
	}
	if err := json.NewDecoder(res.Body).Decode(&scan); err != nil {
		return err
	}
	if len(scan.Errors) > 0 {
		return scan.Errors
	}
	return nil
}

// ValidateToken validates the token inside the client with the Atlas Cloud API.
func (c *Client) ValidateToken(ctx context.Context) error {
	var (
		payload struct {
			ValidateToken struct {
				Success bool `json:"success"`
			} `json:"validateToken"`
		}
		query = `mutation validateToken($token: String!) {
		validateToken(token: $token) {
			success
		}
	}`
		vars = struct {
			Token string `json:"token"`
		}{
			Token: c.token,
		}
	)
	return c.post(ctx, query, vars, &payload)
}
