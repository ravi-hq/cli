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

// CreateIdentity creates a new identity with the given name and optional email address.
// Email accepts three formats: local part only (e.g. "shopping"), full email
// (e.g. "shopping@acme.com"), or empty string for auto-generated.
func (c *Client) CreateIdentity(name string, email string) (*Identity, error) {
	req := map[string]string{}
	if name != "" {
		req["name"] = name
	}
	if email != "" {
		req["email"] = email
	}
	var identity Identity
	if err := c.doAuthenticatedRequest(http.MethodPost, PathIdentities, req, &identity); err != nil {
		return nil, err
	}
	return &identity, nil
}

// ListDomains returns all available email domains (platform + user's custom domains).
func (c *Client) ListDomains() ([]EmailDomain, error) {
	var domains []EmailDomain
	if err := c.doAuthenticatedRequest(http.MethodGet, PathDomains, nil, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

