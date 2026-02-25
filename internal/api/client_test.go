package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/version"
)

// withTempHome is a test helper that temporarily changes the HOME environment variable
// to allow testing functions that use config.Load() and config.Save().
func withTempHome(t *testing.T) (tmpDir string, cleanup func()) {
	t.Helper()

	tmpDir = t.TempDir()

	var homeEnvVar string
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	} else {
		homeEnvVar = "HOME"
	}
	originalHome := os.Getenv(homeEnvVar)

	if err := os.Setenv(homeEnvVar, tmpDir); err != nil {
		t.Fatalf("Failed to set %s: %v", homeEnvVar, err)
	}

	cleanup = func() {
		os.Setenv(homeEnvVar, originalHome)
	}

	return tmpDir, cleanup
}

// withAPIBaseURL is a test helper that temporarily sets the version.APIBaseURL.
func withAPIBaseURL(t *testing.T, url string) func() {
	t.Helper()

	original := version.APIBaseURL
	version.APIBaseURL = url

	return func() {
		version.APIBaseURL = original
	}
}

// setupTestConfig creates a config file in the temp home directory with the given config.
func setupTestConfig(t *testing.T, cfg *config.Config) {
	t.Helper()

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}
}

// newTestClient creates a Client wired to the given httptest server URL.
// Used by inbox_test.go and passwords_test.go for quick test setup.
func newTestClient(serverURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    strings.TrimSuffix(serverURL, "/"),
		config:     &config.Config{AccessToken: "test-token"},
	}
}

// TestNewClient_Success verifies that NewClient creates a client with the provided config.
func TestNewClient_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		UserEmail:    "test@example.com",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Verify the client was configured with the provided config
	if client.config.AccessToken != cfg.AccessToken {
		t.Errorf("client.config.AccessToken = %v, want %v", client.config.AccessToken, cfg.AccessToken)
	}
	if client.config.RefreshToken != cfg.RefreshToken {
		t.Errorf("client.config.RefreshToken = %v, want %v", client.config.RefreshToken, cfg.RefreshToken)
	}
	if client.config.UserEmail != cfg.UserEmail {
		t.Errorf("client.config.UserEmail = %v, want %v", client.config.UserEmail, cfg.UserEmail)
	}

	// Verify the base URL was set (without trailing slash)
	expectedBaseURL := strings.TrimSuffix(server.URL, "/")
	if client.baseURL != expectedBaseURL {
		t.Errorf("client.baseURL = %v, want %v", client.baseURL, expectedBaseURL)
	}
}

// TestNewClient_NilConfig verifies that NewClient loads config from disk when nil is passed.
func TestNewClient_NilConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	// Create a config file on disk
	diskConfig := &config.Config{
		AccessToken:  "disk-access-token",
		RefreshToken: "disk-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		UserEmail:    "disk@example.com",
	}
	setupTestConfig(t, diskConfig)

	// Create client with nil config - should load from disk
	client, err := NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient(nil) error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient(nil) returned nil client")
	}

	// Verify the config was loaded from disk
	if client.config.AccessToken != diskConfig.AccessToken {
		t.Errorf("client.config.AccessToken = %v, want %v", client.config.AccessToken, diskConfig.AccessToken)
	}
	if client.config.UserEmail != diskConfig.UserEmail {
		t.Errorf("client.config.UserEmail = %v, want %v", client.config.UserEmail, diskConfig.UserEmail)
	}
}

// TestNewClient_NoAPIURL verifies that NewClient returns an error when API URL is not configured.
func TestNewClient_NoAPIURL(t *testing.T) {
	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken: "test-token",
	}

	client, err := NewClient(cfg)
	if err == nil {
		t.Fatal("NewClient() error = nil, want error when API URL not configured")
	}

	if client != nil {
		t.Errorf("NewClient() client = %v, want nil on error", client)
	}

	if !strings.Contains(err.Error(), "API URL not configured") {
		t.Errorf("NewClient() error = %v, want error containing 'API URL not configured'", err)
	}
}

// TestDoRequest_JSON verifies that doRequest properly marshals JSON request body.
func TestDoRequest_JSON(t *testing.T) {
	var receivedBody map[string]interface{}
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	requestBody := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	resp, err := client.doRequest(http.MethodPost, "/test", requestBody, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	// Verify Content-Type header
	if receivedContentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", receivedContentType)
	}

	// Verify request body was properly marshaled
	if receivedBody["key1"] != "value1" {
		t.Errorf("request body key1 = %v, want value1", receivedBody["key1"])
	}
	if receivedBody["key2"] != "value2" {
		t.Errorf("request body key2 = %v, want value2", receivedBody["key2"])
	}
}

// TestDoRequest_Auth verifies that doRequest adds Bearer token when auth=true.
func TestDoRequest_Auth(t *testing.T) {
	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken: "test-access-token-12345",
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodGet, "/test", nil, true)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	expectedAuth := "Bearer test-access-token-12345"
	if receivedAuthHeader != expectedAuth {
		t.Errorf("Authorization header = %v, want %v", receivedAuthHeader, expectedAuth)
	}
}

// TestDoRequest_NoAuth verifies that doRequest omits auth header when auth=false.
func TestDoRequest_NoAuth(t *testing.T) {
	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken: "test-access-token-should-not-be-sent",
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodGet, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	if receivedAuthHeader != "" {
		t.Errorf("Authorization header = %v, want empty (no auth)", receivedAuthHeader)
	}
}

// TestParseResponse_Success verifies that parseResponse parses JSON response body correctly.
func TestParseResponse_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    123,
			"name":  "Test User",
			"email": "test@example.com",
		})
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodGet, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	err = client.parseResponse(resp, &result)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}

	if result.ID != 123 {
		t.Errorf("result.ID = %v, want 123", result.ID)
	}
	if result.Name != "Test User" {
		t.Errorf("result.Name = %v, want 'Test User'", result.Name)
	}
	if result.Email != "test@example.com" {
		t.Errorf("result.Email = %v, want 'test@example.com'", result.Email)
	}
}

// TestParseResponse_Error400 verifies that parseResponse returns API error for 400 status.
func TestParseResponse_Error400(t *testing.T) {
	testCases := []struct {
		name         string
		responseBody interface{}
		wantContains string
	}{
		{
			name: "with detail field",
			responseBody: Error{
				Detail: "Invalid request parameters",
			},
			wantContains: "Invalid request parameters",
		},
		{
			name:         "without detail field",
			responseBody: map[string]string{"error": "bad request"},
			wantContains: "status 400",
		},
		{
			name:         "plain text body",
			responseBody: nil, // Will send plain text
			wantContains: "status 400",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if tc.responseBody != nil {
					json.NewEncoder(w).Encode(tc.responseBody)
				} else {
					w.Write([]byte("Bad Request"))
				}
			}))
			defer server.Close()

			cleanupURL := withAPIBaseURL(t, server.URL)
			defer cleanupURL()

			cfg := &config.Config{}
			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			resp, err := client.doRequest(http.MethodGet, "/test", nil, false)
			if err != nil {
				t.Fatalf("doRequest() error = %v", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			err = client.parseResponse(resp, &result)

			if err == nil {
				t.Fatal("parseResponse() error = nil, want error for 400 status")
			}

			if !strings.Contains(err.Error(), tc.wantContains) {
				t.Errorf("parseResponse() error = %v, want error containing %q", err, tc.wantContains)
			}
		})
	}
}

// TestParseResponse_Error500 verifies that parseResponse returns server error for 500 status.
func TestParseResponse_Error500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodGet, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = client.parseResponse(resp, &result)

	if err == nil {
		t.Fatal("parseResponse() error = nil, want error for 500 status")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("parseResponse() error = %v, want error containing '500'", err)
	}
}

// TestRefreshAccessToken_Success verifies that RefreshAccessToken updates tokens correctly.
func TestRefreshAccessToken_Success(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	newAccessToken := "new-access-token-after-refresh"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path
		if r.URL.Path != PathTokenRefresh {
			t.Errorf("Request path = %v, want %v", r.URL.Path, PathTokenRefresh)
		}

		// Verify the request body contains the refresh token
		var req RefreshRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Refresh != "original-refresh-token" {
			t.Errorf("Refresh token in request = %v, want original-refresh-token", req.Refresh)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RefreshResponse{
			Access: newAccessToken,
		})
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken:  "old-access-token",
		RefreshToken: "original-refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // Expired
	}
	setupTestConfig(t, cfg)

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	err = client.RefreshAccessToken()
	if err != nil {
		t.Fatalf("RefreshAccessToken() error = %v", err)
	}

	// Verify the client's config was updated
	if client.config.AccessToken != newAccessToken {
		t.Errorf("client.config.AccessToken = %v, want %v", client.config.AccessToken, newAccessToken)
	}

	// Verify the config was saved to disk
	loadedConfig, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if loadedConfig.AccessToken != newAccessToken {
		t.Errorf("saved config AccessToken = %v, want %v", loadedConfig.AccessToken, newAccessToken)
	}
}

// TestRefreshAccessToken_Failure verifies that RefreshAccessToken handles refresh errors.
func TestRefreshAccessToken_Failure(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	testCases := []struct {
		name           string
		statusCode     int
		responseBody   interface{}
		wantErrContain string
	}{
		{
			name:       "invalid refresh token",
			statusCode: http.StatusUnauthorized,
			responseBody: Error{
				Detail: "Token is invalid or expired",
			},
			wantErrContain: "Token is invalid or expired",
		},
		{
			name:           "server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   nil,
			wantErrContain: "500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				if tc.responseBody != nil {
					json.NewEncoder(w).Encode(tc.responseBody)
				}
			}))
			defer server.Close()

			cleanupURL := withAPIBaseURL(t, server.URL)
			defer cleanupURL()

			cfg := &config.Config{
				AccessToken:  "old-access-token",
				RefreshToken: "invalid-refresh-token",
			}
			setupTestConfig(t, cfg)

			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			err = client.RefreshAccessToken()
			if err == nil {
				t.Fatal("RefreshAccessToken() error = nil, want error")
			}

			if !strings.Contains(err.Error(), tc.wantErrContain) {
				t.Errorf("RefreshAccessToken() error = %v, want error containing %q", err, tc.wantErrContain)
			}
		})
	}
}

// TestDoAuthenticatedRequest_AutoRefresh verifies pre-emptive refresh when token is expired.
func TestDoAuthenticatedRequest_AutoRefresh(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	refreshCalled := false
	newAccessToken := "refreshed-access-token"
	var lastAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case PathTokenRefresh:
			refreshCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(RefreshResponse{
				Access: newAccessToken,
			})
		case "/api/protected":
			lastAuthHeader = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	// Create config with expired token
	cfg := &config.Config{
		AccessToken:  "expired-access-token",
		RefreshToken: "valid-refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // Already expired
	}
	setupTestConfig(t, cfg)

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var result map[string]string
	err = client.doAuthenticatedRequest(http.MethodGet, "/api/protected", nil, &result)
	if err != nil {
		t.Fatalf("doAuthenticatedRequest() error = %v", err)
	}

	// Verify refresh was called
	if !refreshCalled {
		t.Error("Expected RefreshAccessToken to be called for expired token")
	}

	// Verify the new token was used in the request
	expectedAuth := "Bearer " + newAccessToken
	if lastAuthHeader != expectedAuth {
		t.Errorf("Authorization header = %v, want %v", lastAuthHeader, expectedAuth)
	}
}

// TestDoAuthenticatedRequest_401Retry verifies retry on 401 status.
func TestDoAuthenticatedRequest_401Retry(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	requestCount := 0
	refreshCalled := false
	newAccessToken := "new-token-after-401"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case PathTokenRefresh:
			refreshCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(RefreshResponse{
				Access: newAccessToken,
			})
		case "/api/protected":
			requestCount++
			authHeader := r.Header.Get("Authorization")

			if requestCount == 1 {
				// First request: return 401 to trigger refresh
				if authHeader != "Bearer original-access-token" {
					t.Errorf("First request auth = %v, want Bearer original-access-token", authHeader)
				}
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(Error{Detail: "Token expired"})
				return
			}

			// Second request: should have refreshed token
			if authHeader != "Bearer "+newAccessToken {
				t.Errorf("Second request auth = %v, want Bearer %s", authHeader, newAccessToken)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		default:
			t.Errorf("Unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	// Create config with non-expired token (so pre-emptive refresh doesn't trigger)
	cfg := &config.Config{
		AccessToken:  "original-access-token",
		RefreshToken: "valid-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour), // Not expired
	}
	setupTestConfig(t, cfg)

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var result map[string]string
	err = client.doAuthenticatedRequest(http.MethodGet, "/api/protected", nil, &result)
	if err != nil {
		t.Fatalf("doAuthenticatedRequest() error = %v", err)
	}

	// Verify refresh was called after 401
	if !refreshCalled {
		t.Error("Expected RefreshAccessToken to be called after 401")
	}

	// Verify there were 2 requests to the protected endpoint
	if requestCount != 2 {
		t.Errorf("Request count = %v, want 2 (initial + retry)", requestCount)
	}

	// Verify the result
	if result["status"] != "success" {
		t.Errorf("result status = %v, want 'success'", result["status"])
	}
}

// TestIsAuthenticated_True verifies that IsAuthenticated returns true when both tokens are present.
func TestIsAuthenticated_True(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken:  "valid-access-token",
		RefreshToken: "valid-refresh-token",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if !client.IsAuthenticated() {
		t.Error("IsAuthenticated() = false, want true when both tokens are present")
	}
}

// TestIsAuthenticated_False verifies that IsAuthenticated returns false when tokens are missing.
func TestIsAuthenticated_False(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	testCases := []struct {
		name   string
		config *config.Config
	}{
		{
			name: "no access token",
			config: &config.Config{
				AccessToken:  "",
				RefreshToken: "valid-refresh-token",
			},
		},
		{
			name: "no refresh token",
			config: &config.Config{
				AccessToken:  "valid-access-token",
				RefreshToken: "",
			},
		},
		{
			name: "no tokens at all",
			config: &config.Config{
				AccessToken:  "",
				RefreshToken: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.config)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if client.IsAuthenticated() {
				t.Error("IsAuthenticated() = true, want false when tokens are missing")
			}
		})
	}
}

// TestBuildURL verifies that BuildURL correctly builds URLs with query parameters.
func TestBuildURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	testCases := []struct {
		name     string
		path     string
		params   map[string]string
		wantPath string
	}{
		{
			name:     "no params",
			path:     "/api/inbox",
			params:   nil,
			wantPath: server.URL + "/api/inbox",
		},
		{
			name:     "single param",
			path:     "/api/inbox",
			params:   map[string]string{"page": "1"},
			wantPath: server.URL + "/api/inbox?page=1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var params map[string][]string
			if tc.params != nil {
				params = make(map[string][]string)
				for k, v := range tc.params {
					params[k] = []string{v}
				}
			}

			result := client.BuildURL(tc.path, params)
			if result != tc.wantPath {
				t.Errorf("BuildURL() = %v, want %v", result, tc.wantPath)
			}
		})
	}
}

// TestGetUserEmail verifies that GetUserEmail returns the stored user email.
func TestGetUserEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	testCases := []struct {
		name      string
		userEmail string
	}{
		{
			name:      "with email",
			userEmail: "user@example.com",
		},
		{
			name:      "empty email",
			userEmail: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				UserEmail: tc.userEmail,
			}

			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if got := client.GetUserEmail(); got != tc.userEmail {
				t.Errorf("GetUserEmail() = %v, want %v", got, tc.userEmail)
			}
		})
	}
}

// TestDoRequest_NilBody verifies that doRequest handles nil body correctly.
func TestDoRequest_NilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no body was sent for GET request with nil body
		if r.ContentLength > 0 {
			t.Errorf("ContentLength = %v, want 0 for nil body", r.ContentLength)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodGet, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

// TestDoRequest_AcceptHeader verifies that doRequest sets Accept header.
func TestDoRequest_AcceptHeader(t *testing.T) {
	var receivedAcceptHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAcceptHeader = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodGet, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	if receivedAcceptHeader != "application/json" {
		t.Errorf("Accept header = %v, want application/json", receivedAcceptHeader)
	}
}

// TestNewClient_BaseURLTrailingSlash verifies that trailing slashes are trimmed from base URL.
func TestNewClient_BaseURLTrailingSlash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set URL with trailing slash
	cleanupURL := withAPIBaseURL(t, server.URL+"/")
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// The base URL should not have a trailing slash
	if strings.HasSuffix(client.baseURL, "/") {
		t.Errorf("client.baseURL = %v, should not end with /", client.baseURL)
	}
}

// TestParseResponse_EmptyBody verifies that parseResponse handles empty response body.
func TestParseResponse_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body written
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.doRequest(http.MethodDelete, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = client.parseResponse(resp, &result)

	// Should not error on empty body with 200 status
	if err != nil {
		t.Errorf("parseResponse() error = %v, want nil for empty body with 200", err)
	}
}

// TestDoAuthenticatedRequest_NoRefreshToken verifies behavior when no refresh token is available.
func TestDoAuthenticatedRequest_NoRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == PathTokenRefresh {
			t.Error("Refresh endpoint should not be called when no refresh token")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Error{Detail: "Token expired"})
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	cfg := &config.Config{
		AccessToken:  "expired-token",
		RefreshToken: "", // No refresh token
		ExpiresAt:    time.Now().Add(-time.Hour),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	var result map[string]interface{}
	err = client.doAuthenticatedRequest(http.MethodGet, "/api/protected", nil, &result)

	// Should return 401 error without attempting refresh
	if err == nil {
		t.Fatal("doAuthenticatedRequest() error = nil, want error for 401 without refresh token")
	}
}

// TestDoRequest_HTTPMethods verifies that doRequest works with different HTTP methods.
func TestDoRequest_HTTPMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			cleanupURL := withAPIBaseURL(t, server.URL)
			defer cleanupURL()

			cfg := &config.Config{}
			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			resp, err := client.doRequest(method, "/test", nil, false)
			if err != nil {
				t.Fatalf("doRequest() error = %v", err)
			}
			defer resp.Body.Close()

			if receivedMethod != method {
				t.Errorf("Request method = %v, want %v", receivedMethod, method)
			}
		})
	}
}

// TestNewClient_NilConfig_NoConfigFile verifies NewClient creates empty config when no file exists.
func TestNewClient_NilConfig_NoConfigFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	tmpDir, cleanupHome := withTempHome(t)
	defer cleanupHome()

	// Ensure no config file exists
	configPath := filepath.Join(tmpDir, ".ravi", "config.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("Config file should not exist in fresh temp dir")
	}

	// Create client with nil config - should load empty config
	client, err := NewClient(nil)
	if err != nil {
		t.Fatalf("NewClient(nil) error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient(nil) returned nil client")
	}

	// Config should be empty (loaded from non-existent file)
	if client.config.AccessToken != "" {
		t.Errorf("client.config.AccessToken = %v, want empty", client.config.AccessToken)
	}
	if client.config.RefreshToken != "" {
		t.Errorf("client.config.RefreshToken = %v, want empty", client.config.RefreshToken)
	}
}
