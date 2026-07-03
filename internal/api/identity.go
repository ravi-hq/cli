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

// CreateIdentity creates a new identity with the given name and optional email
// address. The address is specified as emailIdentifier (the local part, e.g.
// "shopping") plus an optional domain; both empty auto-generates the address.
// A domain without an emailIdentifier is rejected by the server.
// When provisionPhone is true, the server provisions a phone number and links it
// to the identity (requires an active paid subscription).
func (c *Client) CreateIdentity(name, emailIdentifier, domain string, provisionPhone bool) (*Identity, error) {
	req := map[string]any{}
	if name != "" {
		req["name"] = name
	}
	if emailIdentifier != "" {
		req["email_identifier"] = emailIdentifier
	}
	if domain != "" {
		req["domain"] = domain
	}
	if provisionPhone {
		req["provision_phone"] = true
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
