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
	httpClient *http.Client
	baseURL    string
	config     *config.Config
}

// NewClient creates a new API client. If cfg is nil, attempts to load from disk.
func NewClient(cfg *config.Config) (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		cfg, err = config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		config:     cfg,
	}, nil
}

// doRequest performs an HTTP request with optional authentication
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

	if auth && c.config.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	}

	return c.httpClient.Do(req)
}

// doAuthenticatedRequest performs a request with authentication and auto token refresh
func (c *Client) doAuthenticatedRequest(method, path string, body interface{}, result interface{}) error {
	// Check if token is expired and refresh if needed
	if time.Now().After(c.config.ExpiresAt) && c.config.RefreshToken != "" {
		if err := c.RefreshAccessToken(); err != nil {
			return fmt.Errorf("session expired, run `ravi auth login` to re-authenticate: %w", err)
		}
	}

	resp, err := c.doRequest(method, path, body, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If 401, try to refresh token and retry once
	if resp.StatusCode == http.StatusUnauthorized && c.config.RefreshToken != "" {
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

// parseResponse parses the HTTP response into the result struct
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

// RefreshAccessToken refreshes the access token using the refresh token
func (c *Client) RefreshAccessToken() error {
	req := RefreshRequest{Refresh: c.config.RefreshToken}

	resp, err := c.doRequest(http.MethodPost, PathTokenRefresh, req, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result RefreshResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return err
	}

	c.config.AccessToken = result.Access
	if result.Refresh != "" {
		c.config.RefreshToken = result.Refresh
	}
	c.config.ExpiresAt = time.Now().Add(TokenExpiryBuffer) // Assume 5 min expiry, refresh at 4

	return config.Save(c.config)
}

// IsAuthenticated checks if the client has valid auth tokens.
// It attempts a token refresh if the access token is expired to verify
// the refresh token is still valid.
func (c *Client) IsAuthenticated() bool {
	if c.config.AccessToken == "" || c.config.RefreshToken == "" {
		return false
	}

	// If the access token is expired, try to refresh to verify tokens are valid
	if time.Now().After(c.config.ExpiresAt) {
		if err := c.RefreshAccessToken(); err != nil {
			return false
		}
	}

	return true
}

// GetUserEmail returns the stored user email
func (c *Client) GetUserEmail() string {
	return c.config.UserEmail
}

// GetIdentityName returns the stored identity name (empty if unbound)
func (c *Client) GetIdentityName() string {
	return c.config.IdentityName
}

// BuildURL builds a full URL with query parameters
func (c *Client) BuildURL(path string, params url.Values) string {
	if len(params) == 0 {
		return c.baseURL + path
	}
	return c.baseURL + path + "?" + params.Encode()
}
