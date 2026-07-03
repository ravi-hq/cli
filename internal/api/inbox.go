package api

import (
	"net/http"
	"net/url"
)

// ListEmailThreads fetches email threads
func (c *Client) ListEmailThreads(unreadOnly bool) ([]EmailThread, error) {
	params := url.Values{}
	if unreadOnly {
		params.Set("has_unread", "true")
	}

	path := c.scopedPath(PathEmailInbox, params)

	var result []EmailThread
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetEmailThread fetches a specific email thread by ID
func (c *Client) GetEmailThread(threadID string) (*EmailThreadDetail, error) {
	// URL encode the thread ID (it may contain special chars like < > @)
	encodedID := url.PathEscape(threadID)
	path := c.scopedPath(PathEmailInbox+encodedID+"/", nil)

	var result EmailThreadDetail
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListSMSConversations fetches SMS conversations
func (c *Client) ListSMSConversations(unreadOnly bool) ([]SMSConversation, error) {
	params := url.Values{}
	if unreadOnly {
		params.Set("has_unread", "true")
	}

	path := c.scopedPath(PathSMSInbox, params)

	var result []SMSConversation
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetSMSConversation fetches a specific SMS conversation by ID
func (c *Client) GetSMSConversation(conversationID string) (*SMSConversationDetail, error) {
	// URL encode the conversation ID (it may contain + in phone numbers)
	encodedID := url.PathEscape(conversationID)
	path := c.scopedPath(PathSMSInbox+encodedID+"/", nil)

	var result SMSConversationDetail
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
