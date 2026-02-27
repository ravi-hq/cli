package api

import (
	"net/http"
	"net/url"
	"strconv"
)

// ListPasswords fetches all password entries for the authenticated user.
func (c *Client) ListPasswords() ([]PasswordEntry, error) {
	var result []PasswordEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, PathPasswords, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPassword fetches a single password entry by UUID.
func (c *Client) GetPassword(uuid string) (*PasswordEntry, error) {
	path := PathPasswords + uuid + "/"
	var result PasswordEntry
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePassword creates a new password entry.
func (c *Client) CreatePassword(entry PasswordEntry) (*PasswordEntry, error) {
	var result PasswordEntry
	if err := c.doAuthenticatedRequest(http.MethodPost, PathPasswords, entry, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePassword partially updates a password entry by UUID.
func (c *Client) UpdatePassword(uuid string, fields map[string]interface{}) (*PasswordEntry, error) {
	path := PathPasswords + uuid + "/"
	var result PasswordEntry
	if err := c.doAuthenticatedRequest(http.MethodPatch, path, fields, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePassword deletes a password entry by UUID.
func (c *Client) DeletePassword(uuid string) error {
	path := PathPasswords + uuid + "/"
	return c.doAuthenticatedRequest(http.MethodDelete, path, nil, nil)
}

// GeneratePassword calls the server-side password generator.
func (c *Client) GeneratePassword(opts PasswordGenOpts) (*GeneratedPassword, error) {
	params := url.Values{}
	if opts.Length > 0 {
		params.Set("length", strconv.Itoa(opts.Length))
	}
	// Only send params when explicitly disabling a category.
	// Zero-value (false) means "use server default" (all enabled).
	if opts.NoUppercase {
		params.Set("uppercase", "false")
	}
	if opts.NoLowercase {
		params.Set("lowercase", "false")
	}
	if opts.NoDigits {
		params.Set("digits", "false")
	}
	if opts.NoSpecial {
		params.Set("special", "false")
	}
	if opts.ExcludeChars != "" {
		params.Set("exclude_chars", opts.ExcludeChars)
	}

	path := PathPasswords + "generate-password/"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result GeneratedPassword
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
