package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/version"
)

type Client struct {
	httpClient   *http.Client
	baseURL      string
	auth         *config.AuthConfig
	identityUUID string // X-Ravi-Identity header value (empty = unscoped)
}

// NewClient creates a scoped API client: loads auth + resolves identity UUID
// from the config chain. Most commands should use this.
func NewClient() (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	auth, err := config.LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth: %w", err)
	}

	// Resolve identity — empty string means unscoped (no config file yet).
	identityUUID, err := config.ResolveIdentityUUID()
	if err != nil {
		return nil, fmt.Errorf("resolving identity: %w", err)
	}

	return &Client{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		auth:         auth,
		identityUUID: identityUUID,
	}, nil
}

// NewUnscopedClient creates an API client without identity scoping.
// Use this for account-level operations (listing identities, switching identity).
func NewUnscopedClient() (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	auth, err := config.LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth: %w", err)
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		auth:       auth,
	}, nil
}

// NewClientWithTokens creates a client with explicit tokens (for login flow).
func NewClientWithTokens(access, refresh string) (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		auth: &config.AuthConfig{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresAt:    time.Now().Add(TokenExpiryBuffer),
		},
	}, nil
}

// doRequest performs an HTTP request with optional authentication.
func (c *Client) doRequest(method, path string, body interface{}, auth bool) (*http.Response, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if auth && c.auth.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.auth.AccessToken)
	}

	if auth && c.identityUUID != "" {
		req.Header.Set("X-Ravi-Identity", c.identityUUID)
	}

	return c.httpClient.Do(req)
}

// doAuthenticatedRequest performs a request with authentication and auto token refresh.
func (c *Client) doAuthenticatedRequest(method, path string, body interface{}, result interface{}) error {
	// Check if token is expired and refresh if needed.
	if time.Now().After(c.auth.ExpiresAt) && c.auth.RefreshToken != "" {
		if err := c.RefreshAccessToken(); err != nil {
			return fmt.Errorf("session expired, run `ravi auth login` to re-authenticate: %w", err)
		}
	}

	resp, err := c.doRequest(method, path, body, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If 401, try to refresh token and retry once.
	if resp.StatusCode == http.StatusUnauthorized && c.auth.RefreshToken != "" {
		if err := c.RefreshAccessToken(); err != nil {
			return fmt.Errorf("session expired, run `ravi auth login` to re-authenticate: %w", err)
		}
		resp, err = c.doRequest(method, path, body, true)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	return c.parseResponse(resp, result)
}

// parseResponse parses the HTTP response into the result struct.
func (c *Client) parseResponse(resp *http.Response, result interface{}) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		var rlErr RateLimitError
		if json.Unmarshal(bodyBytes, &rlErr) != nil || rlErr.Detail == "" {
			rlErr.Detail = "Request was throttled."
		}
		if rlErr.RetryAfterSeconds == 0 {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if seconds, parseErr := strconv.Atoi(ra); parseErr == nil {
					rlErr.RetryAfterSeconds = seconds
				}
			}
		}
		return &rlErr
	}

	if resp.StatusCode == http.StatusNotFound {
		var nfErr NotFoundError
		if json.Unmarshal(bodyBytes, &nfErr) == nil && nfErr.Detail != "" {
			return &nfErr
		}
		return &NotFoundError{Detail: string(bodyBytes)}
	}

	if resp.StatusCode >= 400 {
		var apiErr Error
		if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Detail != "" {
			return fmt.Errorf("API error: %s", apiErr.Detail)
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// RefreshAccessToken refreshes the access token using the refresh token.
func (c *Client) RefreshAccessToken() error {
	req := RefreshRequest{Refresh: c.auth.RefreshToken}

	resp, err := c.doRequest(http.MethodPost, PathTokenRefresh, req, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result RefreshResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return err
	}

	c.auth.AccessToken = result.Access
	if result.Refresh != "" {
		c.auth.RefreshToken = result.Refresh
	}
	c.auth.ExpiresAt = time.Now().Add(TokenExpiryBuffer)

	return config.SaveAuth(c.auth)
}

// IsAuthenticated checks if the client has valid auth tokens.
func (c *Client) IsAuthenticated() bool {
	if c.auth.AccessToken == "" || c.auth.RefreshToken == "" {
		return false
	}

	if time.Now().After(c.auth.ExpiresAt) {
		if err := c.RefreshAccessToken(); err != nil {
			return false
		}
	}

	return true
}

// GetUserEmail returns the stored user email.
func (c *Client) GetUserEmail() string {
	return c.auth.UserEmail
}

// GetIdentityUUID returns the resolved identity UUID (empty if unscoped).
func (c *Client) GetIdentityUUID() string {
	return c.identityUUID
}

// BuildURL builds a full URL with query parameters.
func (c *Client) BuildURL(path string, params url.Values) string {
	if len(params) == 0 {
		return c.baseURL + path
	}
	return c.baseURL + path + "?" + params.Encode()
}
