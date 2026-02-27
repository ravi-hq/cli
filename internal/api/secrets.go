package api

import (
	"net/http"
	"net/url"
)

// ListSecrets fetches all secret entries for the authenticated user.
func (c *Client) ListSecrets() ([]SecretEntry, error) {
	var result []SecretEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, PathSecrets, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSecret fetches a single secret entry by key using the ?key= filter.
// Returns nil if no secret matches the given key.
func (c *Client) GetSecret(key string) (*SecretEntry, error) {
	params := url.Values{}
	params.Set("key", key)
	path := PathSecrets + "?" + params.Encode()

	var result []SecretEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return &result[0], nil
}

// GetSecretByUUID fetches a single secret entry by UUID.
func (c *Client) GetSecretByUUID(uuid string) (*SecretEntry, error) {
	path := PathSecrets + uuid + "/"
	var result SecretEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateSecret creates a new secret entry.
func (c *Client) CreateSecret(entry SecretEntry) (*SecretEntry, error) {
	var result SecretEntry
	if err := c.doAuthenticatedRequest(http.MethodPost, PathSecrets, entry, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateSecret partially updates a secret entry by UUID.
func (c *Client) UpdateSecret(uuid string, fields map[string]interface{}) (*SecretEntry, error) {
	path := PathSecrets + uuid + "/"
	var result SecretEntry
	if err := c.doAuthenticatedRequest(http.MethodPatch, path, fields, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteSecret deletes a secret entry by UUID.
func (c *Client) DeleteSecret(uuid string) error {
	path := PathSecrets + uuid + "/"
	return c.doAuthenticatedRequest(http.MethodDelete, path, nil, nil)
}
