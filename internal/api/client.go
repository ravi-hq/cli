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
	apiKey     string // management key or identity key
	userEmail  string
}

// NewClient creates an identity-scoped API client using the identity key.
// Falls back to management key if no identity key is configured.
func NewClient() (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Prefer identity key for resource-scoped operations.
	apiKey := cfg.IdentityKey
	if apiKey == "" {
		apiKey = cfg.ManagementKey
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		userEmail:  cfg.UserEmail,
	}, nil
}

// NewManagementClient creates an API client using the management key.
// Use this for account-level operations (listing identities, managing keys).
func NewManagementClient() (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     cfg.ManagementKey,
		userEmail:  cfg.UserEmail,
	}, nil
}

// NewUnauthenticatedClient creates an API client with no auth (for login flow).
func NewUnauthenticatedClient() (*Client, error) {
	baseURL, err := version.GetAPIBaseURL()
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
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

	if auth && c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.httpClient.Do(req)
}

// doAuthenticatedRequest performs a request with authentication.
func (c *Client) doAuthenticatedRequest(method, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(method, path, body, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

// IsAuthenticated checks if the client has an API key.
func (c *Client) IsAuthenticated() bool {
	return c.apiKey != ""
}

// GetUserEmail returns the stored user email.
func (c *Client) GetUserEmail() string {
	return c.userEmail
}

// BuildURL builds a full URL with query parameters.
func (c *Client) BuildURL(path string, params url.Values) string {
	if len(params) == 0 {
		return c.baseURL + path
	}
	return c.baseURL + path + "?" + params.Encode()
}
