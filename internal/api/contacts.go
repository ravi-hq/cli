package api

import (
	"net/http"
	"net/url"
)

// ListContacts fetches all contacts for the authenticated user.
func (c *Client) ListContacts() ([]ContactEntry, error) {
	var result []ContactEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, c.scopedPath(PathContacts, nil), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetContact fetches a single contact by UUID.
func (c *Client) GetContact(uuid string) (*ContactEntry, error) {
	path := c.scopedPath(PathContacts+uuid+"/", nil)
	var result ContactEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateContact creates a new contact.
func (c *Client) CreateContact(entry ContactEntry) (*ContactEntry, error) {
	var result ContactEntry
	if err := c.doAuthenticatedRequest(http.MethodPost, c.scopedPath(PathContacts, nil), entry, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateContact partially updates a contact by UUID.
func (c *Client) UpdateContact(uuid string, fields map[string]interface{}) (*ContactEntry, error) {
	path := c.scopedPath(PathContacts+uuid+"/", nil)
	var result ContactEntry
	if err := c.doAuthenticatedRequest(http.MethodPatch, path, fields, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteContact deletes a contact by UUID.
func (c *Client) DeleteContact(uuid string) error {
	path := c.scopedPath(PathContacts+uuid+"/", nil)
	return c.doAuthenticatedRequest(http.MethodDelete, path, nil, nil)
}

// FindContact finds a contact by email and/or phone number.
func (c *Client) FindContact(email, phone string) (*ContactEntry, error) {
	params := url.Values{}
	if email != "" {
		params.Set("email", email)
	}
	if phone != "" {
		params.Set("phone_number", phone)
	}

	path := c.scopedPath(PathContacts+"find/", params)
	var result ContactEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SearchContacts searches contacts by a query string.
func (c *Client) SearchContacts(query string) ([]ContactEntry, error) {
	params := url.Values{}
	params.Set("q", query)
	path := c.scopedPath(PathContacts+"search/", params)
	var result []ContactEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}
