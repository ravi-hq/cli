package api

import "net/http"

// RequestSSOToken requests a short-lived SSO token from the server.
// Requires an identity-scoped API key and an active subscription.
// The returned token has an "rvt_" prefix and a 5-minute TTL.
func (c *Client) RequestSSOToken() (*SSOTokenResponse, error) {
	var result SSOTokenResponse
	if err := c.doAuthenticatedRequest(http.MethodPost, PathSSOToken, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
