package api

import "net/http"

// ListIdentities returns all identities for the authenticated user.
func (c *Client) ListIdentities() ([]Identity, error) {
	var identities []Identity
	if err := c.doAuthenticatedRequest(http.MethodGet, PathIdentities, nil, &identities); err != nil {
		return nil, err
	}
	return identities, nil
}

// CreateIdentity creates a new identity with the given name.
func (c *Client) CreateIdentity(name string) (*Identity, error) {
	req := map[string]string{"name": name}
	var identity Identity
	if err := c.doAuthenticatedRequest(http.MethodPost, PathIdentities, req, &identity); err != nil {
		return nil, err
	}
	return &identity, nil
}

