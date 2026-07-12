package incidentrelay

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotFound = errors.New("incidentrelay resource not found")

type ClientConfig struct {
	BaseURL   string
	Token     string
	Username  string
	Password  string
	UserAgent string
	Insecure  bool
}

type Client struct {
	baseURL    string
	token      string
	username   string
	password   string
	userAgent  string
	httpClient *http.Client
}

func NewClient(config ClientConfig) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if config.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}

	return &Client{
		baseURL:   baseURL,
		token:     strings.TrimSpace(config.Token),
		username:  strings.TrimSpace(config.Username),
		password:  config.Password,
		userAgent: config.UserAgent,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
	}, nil
}

func (c *Client) Configure(ctx context.Context) error {
	if c.token != "" {
		return nil
	}
	if c.username == "" || c.password == "" {
		return fmt.Errorf("configure either token or username/password")
	}
	return c.authenticate(ctx)
}

func (c *Client) authenticate(ctx context.Context) error {
	var result map[string]interface{}
	if err := c.Do(ctx, http.MethodPost, "/api/auth/login", map[string]interface{}{
		"username": c.username,
		"password": c.password,
	}, &result); err != nil {
		return err
	}

	token, ok := result["access_token"].(string)
	if !ok || token == "" {
		return fmt.Errorf("authentication response does not contain access_token")
	}

	c.token = token
	return nil
}

func (c *Client) Do(ctx context.Context, method, endpoint string, body interface{}, out interface{}) error {
	req, err := c.newRequest(ctx, method, endpoint, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %d: %s", method, endpoint, resp.StatusCode, string(respBody))
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, endpoint, err)
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Request, error) {
	requestURL, err := joinURL(c.baseURL, endpoint)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return req, nil
}

func joinURL(baseURL, endpoint string) (string, error) {
	if endpoint == "" {
		return baseURL, nil
	}

	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		_, err := url.Parse(endpoint)
		return endpoint, err
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	endpoint = "/" + strings.TrimLeft(endpoint, "/")
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	return parsedBase.ResolveReference(parsedEndpoint).String(), nil
}
