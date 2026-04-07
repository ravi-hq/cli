package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RequestDeviceCode initiates the device code flow
func (c *Client) RequestDeviceCode() (*DeviceCodeResponse, error) {
	resp, err := c.doRequest(http.MethodPost, PathDeviceCode, nil, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DeviceCodeResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PollForToken polls for the device token with long polling.
// Returns (token_response, error_code, error)
// error_code is "authorization_pending" or "expired_token" on expected errors
func (c *Client) PollForToken(deviceCode string) (*DeviceTokenResponse, string, error) {
	req := DeviceTokenRequest{
		DeviceCode: deviceCode,
		Wait:       true,
	}

	resp, err := c.doRequest(http.MethodPost, PathDeviceToken, req, false)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response (400 status)
	if resp.StatusCode == http.StatusBadRequest {
		var tokenErr DeviceTokenError
		if err := json.Unmarshal(bodyBytes, &tokenErr); err == nil {
			return nil, tokenErr.Error, nil
		}
		return nil, "", fmt.Errorf("polling failed: %s", string(bodyBytes))
	}

	// Success response
	if resp.StatusCode == http.StatusOK {
		var result DeviceTokenResponse
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return nil, "", fmt.Errorf("failed to parse token response: %w", err)
		}
		return &result, "", nil
	}

	return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// CreateIdentityKey creates an identity-scoped API key for the given identity.
func (c *Client) CreateIdentityKey(identityUUID, label string) (*CreateIdentityKeyResponse, error) {
	req := CreateIdentityKeyRequest{
		IdentityUUID: identityUUID,
		Label:        label,
	}
	var result CreateIdentityKeyResponse
	if err := c.doAuthenticatedRequest(http.MethodPost, PathIdentityKeys, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
