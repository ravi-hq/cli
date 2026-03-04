package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/version"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	auth       *config.AuthConfig
	bound      bool // true when operating with identity-scoped (bound) tokens
}

// NewClient creates a scoped API client: loads auth + uses bound tokens from
// the config chain when available. Most commands should use this.
func NewClient() (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	auth, err := config.LoadAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth: %w", err)
	}

	// Load bound tokens from config if available.
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Use bound tokens for identity-scoped operations.
	effectiveAuth := auth
	isBound := false
	if cfg.BoundAccessToken != "" {
		effectiveAuth = &config.AuthConfig{
			AccessToken:  cfg.BoundAccessToken,
			RefreshToken: cfg.BoundRefreshToken,
			UserEmail:    auth.UserEmail,
			PINSalt:      auth.PINSalt,
			PublicKey:     auth.PublicKey,
			PrivateKey:    auth.PrivateKey,
		}
		isBound = true
	} else if cfg.IdentityUUID != "" {
		fmt.Fprintf(os.Stderr, "Warning: identity %q (%s) has no scoped token — using global token; re-run 'ravi identity use' to fix\n", cfg.IdentityName, cfg.IdentityUUID)
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		auth:       effectiveAuth,
		bound:      isBound,
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

	return c.httpClient.Do(req)
}

// doAuthenticatedRequest performs a request with authentication and auto token refresh.
func (c *Client) doAuthenticatedRequest(method, path string, body interface{}, result interface{}) error {
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

	// Note: there is a narrow TOCTOU race if two CLI processes refresh bound
	// tokens concurrently. The last writer wins. Acceptable for a CLI tool.

	// Persist BEFORE updating in-memory state
	if c.bound {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config to save refreshed tokens: %w", err)
		}
		cfg.BoundAccessToken = result.Access
		if result.Refresh != "" {
			cfg.BoundRefreshToken = result.Refresh
		}
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("saving refreshed bound tokens: %w", err)
		}
	} else {
		newAuth := *c.auth // copy
		newAuth.AccessToken = result.Access
		if result.Refresh != "" {
			newAuth.RefreshToken = result.Refresh
		}
		if err := config.SaveAuth(&newAuth); err != nil {
			return fmt.Errorf("saving refreshed tokens: %w", err)
		}
	}

	// Only update in-memory state after successful persist
	c.auth.AccessToken = result.Access
	if result.Refresh != "" {
		c.auth.RefreshToken = result.Refresh
	}
	return nil
}

// IsAuthenticated checks if the client has valid auth tokens.
func (c *Client) IsAuthenticated() bool {
	return c.auth.AccessToken != "" && c.auth.RefreshToken != ""
}

// GetUserEmail returns the stored user email.
func (c *Client) GetUserEmail() string {
	return c.auth.UserEmail
}

// BindIdentity calls the bind-identity endpoint to get identity-scoped tokens.
func (c *Client) BindIdentity(identityUUID string) (*BindIdentityResponse, error) {
	req := map[string]string{
		"identity": identityUUID,
	}
	var result BindIdentityResponse
	if err := c.doAuthenticatedRequest(http.MethodPost, PathBindIdentity, req, &result); err != nil {
		return nil, err
	}
	if result.Access == "" {
		return nil, fmt.Errorf("bind-identity returned empty access token")
	}
	if result.Refresh == "" {
		return nil, fmt.Errorf("bind-identity returned empty refresh token")
	}
	return &result, nil
}

// BuildURL builds a full URL with query parameters.
func (c *Client) BuildURL(path string, params url.Values) string {
	if len(params) == 0 {
		return c.baseURL + path
	}
	return c.baseURL + path + "?" + params.Encode()
}
