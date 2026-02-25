package api

import (
	"fmt"
	"net/http"
	"net/url"
)

// GetPhone fetches the user's assigned Ravi phone number.
// Returns the first phone number associated with the authenticated user.
func (c *Client) GetPhone() (*Phone, error) {
	var result []Phone
	if err := c.doAuthenticatedRequest(http.MethodGet, PathPhone, nil, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no phone number assigned")
	}

	return &result[0], nil
}

// GetEmail fetches the user's assigned Ravi email address.
// Returns the first email address associated with the authenticated user.
func (c *Client) GetEmail() (*Email, error) {
	var result []Email
	if err := c.doAuthenticatedRequest(http.MethodGet, PathEmail, nil, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no email address assigned")
	}

	return &result[0], nil
}

// GetOwner fetches the account owner's profile information.
func (c *Client) GetOwner() (*Owner, error) {
	var result Owner
	if err := c.doAuthenticatedRequest(http.MethodGet, PathOwner, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListSMSMessages fetches all SMS messages (flat list, not grouped by conversation).
func (c *Client) ListSMSMessages(unreadOnly bool) ([]PhoneMessage, error) {
	params := url.Values{}
	if unreadOnly {
		params.Set("is_read", "false")
	}

	path := PathMessages
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result []PhoneMessage
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetSMSMessage fetches a specific SMS message by ID.
func (c *Client) GetSMSMessage(messageID string) (*PhoneMessage, error) {
	path := PathMessages + messageID + "/"

	var result PhoneMessage
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListEmailMessages fetches all email messages (flat list, not grouped by thread).
func (c *Client) ListEmailMessages(unreadOnly bool) ([]EmailMessageDetail, error) {
	params := url.Values{}
	if unreadOnly {
		params.Set("is_read", "false")
	}

	path := PathEmailMessages
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result []EmailMessageDetail
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetEmailMessage fetches a specific email message by ID.
func (c *Client) GetEmailMessage(messageID string) (*EmailMessageDetail, error) {
	path := PathEmailMessages + messageID + "/"

	var result EmailMessageDetail
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
