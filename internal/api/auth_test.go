package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupTestClient creates a Client configured to use the mock server URL.
func setupTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()

	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    strings.TrimSuffix(serverURL, "/"),
	}
}

// TestRequestDeviceCode_Success verifies that RequestDeviceCode returns a valid response.
func TestRequestDeviceCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != PathDeviceCode {
			t.Errorf("Expected path %s, got %s", PathDeviceCode, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", ct)
		}

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

// TestRequestDeviceCode_Error verifies error handling.
func TestRequestDeviceCode_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Internal server error"})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, err := client.RequestDeviceCode()
	if err == nil {
		t.Fatal("RequestDeviceCode() expected error, got nil")
	}
	if result != nil {
		t.Errorf("RequestDeviceCode() result = %v, want nil on error", result)
	}
	if err.Error() != "API error: Internal server error" {
		t.Errorf("Error message = %q, want 'API error: Internal server error'", err.Error())
	}
}

// TestPollForToken_Pending verifies authorization_pending response.
func TestPollForToken_Pending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != PathDeviceToken {
			t.Errorf("Expected path %s, got %s", PathDeviceToken, r.URL.Path)
		}

		var reqBody DeviceTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if reqBody.DeviceCode != "test-device-code" {
			t.Errorf("DeviceCode in request = %q, want %q", reqBody.DeviceCode, "test-device-code")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DeviceTokenError{
			Error:            "authorization_pending",
			ErrorDescription: "User hasn't authorized yet",
		})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("test-device-code")

	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("PollForToken() result = %v, want nil for pending", result)
	}
	if errorCode != "authorization_pending" {
		t.Errorf("PollForToken() errorCode = %q, want %q", errorCode, "authorization_pending")
	}
}

// TestPollForToken_Success verifies successful token response with API keys.
func TestPollForToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathDeviceToken {
			t.Errorf("Expected path %s, got %s", PathDeviceToken, r.URL.Path)
		}

		var reqBody DeviceTokenRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.DeviceCode != "valid-device-code" {
			t.Errorf("DeviceCode = %q, want %q", reqBody.DeviceCode, "valid-device-code")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeviceTokenResponse{
			ManagementKey: "ravi_mgmt_test123",
			IdentityKey:   "ravi_id_test456",
			User: User{
				ID:        42,
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
		})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("valid-device-code")

	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}
	if errorCode != "" {
		t.Errorf("PollForToken() errorCode = %q, want empty string", errorCode)
	}
	if result == nil {
		t.Fatal("PollForToken() result is nil, want non-nil")
	}

	if result.ManagementKey != "ravi_mgmt_test123" {
		t.Errorf("ManagementKey = %q, want %q", result.ManagementKey, "ravi_mgmt_test123")
	}
	if result.IdentityKey != "ravi_id_test456" {
		t.Errorf("IdentityKey = %q, want %q", result.IdentityKey, "ravi_id_test456")
	}
	if result.User.ID != 42 {
		t.Errorf("User.ID = %d, want %d", result.User.ID, 42)
	}
	if result.User.Email != "user@example.com" {
		t.Errorf("User.Email = %q, want %q", result.User.Email, "user@example.com")
	}
}

// TestPollForToken_Expired verifies expired_token response.
func TestPollForToken_Expired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DeviceTokenError{
			Error:            "expired_token",
			ErrorDescription: "The device code has expired",
		})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("expired-device-code")

	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("PollForToken() result = %v, want nil for expired", result)
	}
	if errorCode != "expired_token" {
		t.Errorf("PollForToken() errorCode = %q, want %q", errorCode, "expired_token")
	}
}

// TestPollForToken_InvalidCode verifies invalid_grant response.
func TestPollForToken_InvalidCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody DeviceTokenRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.DeviceCode != "invalid-code-xyz" {
			t.Errorf("DeviceCode = %q, want %q", reqBody.DeviceCode, "invalid-code-xyz")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DeviceTokenError{
			Error:            "invalid_grant",
			ErrorDescription: "The device code is invalid or has been revoked",
		})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	result, errorCode, err := client.PollForToken("invalid-code-xyz")

	if err != nil {
		t.Fatalf("PollForToken() unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("PollForToken() result = %v, want nil for invalid code", result)
	}
	if errorCode != "invalid_grant" {
		t.Errorf("PollForToken() errorCode = %q, want %q", errorCode, "invalid_grant")
	}
}

// TestPollForToken_UnexpectedStatus verifies handling of unexpected HTTP status codes.
func TestPollForToken_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	_, _, err := client.PollForToken("some-code")
	if err == nil {
		t.Fatal("PollForToken() error = nil, want error for 500 status")
	}
	if !strings.Contains(err.Error(), "unexpected status code: 500") {
		t.Errorf("Error = %q, want to contain 'unexpected status code: 500'", err.Error())
	}
}

// TestPollForToken_BadRequestMalformedJSON verifies handling of malformed JSON in 400 response.
func TestPollForToken_BadRequestMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)

	_, _, err := client.PollForToken("some-code")
	if err == nil {
		t.Fatal("PollForToken() error = nil, want error for malformed 400 response")
	}
	if !strings.Contains(err.Error(), "polling failed") {
		t.Errorf("Error = %q, want to contain 'polling failed'", err.Error())
	}
}

// TestPollForToken_SendsWaitTrue verifies that the request body includes wait: true.
func TestPollForToken_SendsWaitTrue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody DeviceTokenRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if !reqBody.Wait {
			t.Errorf("Wait = %v, want true", reqBody.Wait)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeviceTokenResponse{
			ManagementKey: "ravi_mgmt_test",
			User:          User{Email: "test@example.com"},
		})
	}))
	defer server.Close()

	client := setupTestClient(t, server.URL)
	_, _, _ = client.PollForToken("test-code")
}

// TestCreateIdentityKey_Success verifies creating an identity key.
func TestCreateIdentityKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != PathIdentityKeys {
			t.Errorf("Expected path %s, got %s", PathIdentityKeys, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer ravi_mgmt_test" {
			t.Errorf("Expected management key in auth header, got %s", r.Header.Get("Authorization"))
		}

		var req CreateIdentityKeyRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.IdentityUUID != "uuid-123" {
			t.Errorf("IdentityUUID = %q, want %q", req.IdentityUUID, "uuid-123")
		}
		if req.Label != "cli" {
			t.Errorf("Label = %q, want %q", req.Label, "cli")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CreateIdentityKeyResponse{
			Key:          "ravi_id_new_key",
			IdentityUUID: "uuid-123",
			Label:        "cli",
		})
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    strings.TrimSuffix(server.URL, "/"),
		apiKey:     "ravi_mgmt_test",
	}

	result, err := client.CreateIdentityKey("uuid-123", "cli")
	if err != nil {
		t.Fatalf("CreateIdentityKey() error = %v", err)
	}
	if result.Key != "ravi_id_new_key" {
		t.Errorf("Key = %q, want %q", result.Key, "ravi_id_new_key")
	}
	if result.IdentityUUID != "uuid-123" {
		t.Errorf("IdentityUUID = %q, want %q", result.IdentityUUID, "uuid-123")
	}
}

// TestCreateIdentityKey_Error verifies error handling for CreateIdentityKey.
func TestCreateIdentityKey_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Error{Detail: "Identity not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	result, err := client.CreateIdentityKey("bad-uuid", "cli")
	if err == nil {
		t.Fatal("CreateIdentityKey() error = nil, want error")
	}
	if result != nil {
		t.Errorf("CreateIdentityKey() result = %v, want nil on error", result)
	}
}
