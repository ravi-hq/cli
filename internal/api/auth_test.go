package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ravi-hq/cli/internal/config"
)

// setupTestClient creates a Client configured to use the mock server URL.
// It sets version.APIBaseURL to the server URL and restores it after the test.
func setupTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()

	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    strings.TrimSuffix(serverURL, "/"),
		auth:       &config.AuthConfig{},
	}
}

// TestRequestDeviceCode_Success verifies that RequestDeviceCode returns a valid
// DeviceCodeResponse when the API returns a successful response.
func TestRequestDeviceCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Verify request path
		if r.URL.Path != PathDeviceCode {
			t.Errorf("Expected path %s, got %s", PathDeviceCode, r.URL.Path)
		}

		// Verify Content-Type header
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", ct)
		}

		// Return mock response
		response := DeviceCodeResponse{
			DeviceCode:      "test-device-code-12345",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://example.com/device",
			ExpiresIn:       1800,
			Interval:        5,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, err := client.RequestDeviceCode()
	if err != nil {
		t.Fatalf("RequestDeviceCode() unexpected error: %v", err)
	}

	// Verify all response fields
	if result.DeviceCode != "test-device-code-12345" {
		t.Errorf("DeviceCode = %q, want %q", result.DeviceCode, "test-device-code-12345")
	}
	if result.UserCode != "ABCD-1234" {
		t.Errorf("UserCode = %q, want %q", result.UserCode, "ABCD-1234")
	}
	if result.VerificationURI != "https://example.com/device" {
		t.Errorf("VerificationURI = %q, want %q", result.VerificationURI, "https://example.com/device")
	}
	if result.ExpiresIn != 1800 {
		t.Errorf("ExpiresIn = %d, want %d", result.ExpiresIn, 1800)
	}
	if result.Interval != 5 {
		t.Errorf("Interval = %d, want %d", result.Interval, 5)
	}
}

// TestRequestDeviceCode_Error verifies that RequestDeviceCode returns an error
// when the API returns an error response.
func TestRequestDeviceCode_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != PathDeviceCode {
			t.Errorf("Expected path %s, got %s", PathDeviceCode, r.URL.Path)
		}

		// Return error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := Error{
			Detail: "Internal server error",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, err := client.RequestDeviceCode()

	// Should return an error
	if err == nil {
		t.Fatal("RequestDeviceCode() expected error, got nil")
	}

	// Result should be nil on error
	if result != nil {
		t.Errorf("RequestDeviceCode() result = %v, want nil on error", result)
	}

	// Error message should contain the API error detail
	if err.Error() != "API error: Internal server error" {
		t.Errorf("Error message = %q, want to contain 'Internal server error'", err.Error())
	}
}

// TestPollForToken_Pending verifies that PollForToken returns the "authorization_pending"
// error code when the user has not yet authorized the device.
func TestPollForToken_Pending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		// Verify request path
		if r.URL.Path != PathDeviceToken {
			t.Errorf("Expected path %s, got %s", PathDeviceToken, r.URL.Path)
		}

		// Verify request body contains device_code
		var reqBody DeviceTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if reqBody.DeviceCode != "test-device-code" {
			t.Errorf("DeviceCode in request = %q, want %q", reqBody.DeviceCode, "test-device-code")
		}

		// Return authorization_pending error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := DeviceTokenError{
			Error:            "authorization_pending",
			ErrorDescription: "User hasn't authorized yet",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("test-device-code")

	// No error should be returned for pending status
	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}

	// Result should be nil when pending
	if result != nil {
		t.Errorf("PollForToken() result = %v, want nil for pending", result)
	}

	// Error code should be "authorization_pending"
	if errorCode != "authorization_pending" {
		t.Errorf("PollForToken() errorCode = %q, want %q", errorCode, "authorization_pending")
	}
}

// TestPollForToken_Success verifies that PollForToken returns the token response
// when the user has successfully authorized the device.
func TestPollForToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != PathDeviceToken {
			t.Errorf("Expected path %s, got %s", PathDeviceToken, r.URL.Path)
		}

		// Verify request body
		var reqBody DeviceTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if reqBody.DeviceCode != "valid-device-code" {
			t.Errorf("DeviceCode in request = %q, want %q", reqBody.DeviceCode, "valid-device-code")
		}

		// Return successful token response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := DeviceTokenResponse{
			Access:  "access-token-12345",
			Refresh: "refresh-token-67890",
			User: User{
				ID:        42,
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("valid-device-code")

	// No error should be returned
	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}

	// Error code should be empty on success
	if errorCode != "" {
		t.Errorf("PollForToken() errorCode = %q, want empty string", errorCode)
	}

	// Verify result is not nil
	if result == nil {
		t.Fatal("PollForToken() result is nil, want non-nil")
	}

	// Verify token fields
	if result.Access != "access-token-12345" {
		t.Errorf("Access = %q, want %q", result.Access, "access-token-12345")
	}
	if result.Refresh != "refresh-token-67890" {
		t.Errorf("Refresh = %q, want %q", result.Refresh, "refresh-token-67890")
	}

	// Verify user fields
	if result.User.ID != 42 {
		t.Errorf("User.ID = %d, want %d", result.User.ID, 42)
	}
	if result.User.Email != "user@example.com" {
		t.Errorf("User.Email = %q, want %q", result.User.Email, "user@example.com")
	}
	if result.User.FirstName != "John" {
		t.Errorf("User.FirstName = %q, want %q", result.User.FirstName, "John")
	}
	if result.User.LastName != "Doe" {
		t.Errorf("User.LastName = %q, want %q", result.User.LastName, "Doe")
	}
}

// TestPollForToken_Expired verifies that PollForToken returns the "expired_token"
// error code when the device code has expired.
func TestPollForToken_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != PathDeviceToken {
			t.Errorf("Expected path %s, got %s", PathDeviceToken, r.URL.Path)
		}

		// Return expired_token error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := DeviceTokenError{
			Error:            "expired_token",
			ErrorDescription: "The device code has expired",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("expired-device-code")

	// No error should be returned for expired status (it's an expected OAuth error)
	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}

	// Result should be nil when expired
	if result != nil {
		t.Errorf("PollForToken() result = %v, want nil for expired", result)
	}

	// Error code should be "expired_token"
	if errorCode != "expired_token" {
		t.Errorf("PollForToken() errorCode = %q, want %q", errorCode, "expired_token")
	}
}

// TestPollForToken_InvalidCode verifies that PollForToken returns an appropriate
// error code when the device code is invalid or not found.
func TestPollForToken_InvalidCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != PathDeviceToken {
			t.Errorf("Expected path %s, got %s", PathDeviceToken, r.URL.Path)
		}

		// Verify request body
		var reqBody DeviceTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if reqBody.DeviceCode != "invalid-code-xyz" {
			t.Errorf("DeviceCode in request = %q, want %q", reqBody.DeviceCode, "invalid-code-xyz")
		}

		// Return invalid_grant error (common OAuth error for invalid codes)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := DeviceTokenError{
			Error:            "invalid_grant",
			ErrorDescription: "The device code is invalid or has been revoked",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("invalid-code-xyz")

	// No error should be returned for invalid_grant status (it's an expected OAuth error)
	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}

	// Result should be nil when invalid
	if result != nil {
		t.Errorf("PollForToken() result = %v, want nil for invalid code", result)
	}

	// Error code should be "invalid_grant"
	if errorCode != "invalid_grant" {
		t.Errorf("PollForToken() errorCode = %q, want %q", errorCode, "invalid_grant")
	}
}
