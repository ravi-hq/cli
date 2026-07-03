package api

import (
	"net/http"
)

// StartCall places an outbound call from the identity's phone number.
// Management-scoped keys must scope the client to an identity (WithIdentity).
func (c *Client) StartCall(req CallStartRequest) (*Call, error) {
	path := c.scopedPath(PathCalls+"start/", nil)
	var result Call
	if err := c.doAuthenticatedRequest(http.MethodPost, path, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListCalls fetches calls for the authenticated user, newest first.
func (c *Client) ListCalls() ([]Call, error) {
	path := c.scopedPath(PathCalls, nil)
	var result []Call
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetCall fetches a single call by ID.
func (c *Client) GetCall(id string) (*Call, error) {
	path := c.scopedPath(PathCalls+id+"/", nil)
	var result Call
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCallTranscript fetches the transcript segments for a call.
func (c *Client) GetCallTranscript(id string) ([]CallTranscriptSegment, error) {
	path := c.scopedPath(PathCalls+id+"/transcript/", nil)
	var result []CallTranscriptSegment
	if err := c.doAuthenticatedRequest(http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// HangupCall hangs up an active call.
func (c *Client) HangupCall(id string) (*Call, error) {
	path := c.scopedPath(PathCalls+id+"/hangup/", nil)
	var result Call
	if err := c.doAuthenticatedRequest(http.MethodPost, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
