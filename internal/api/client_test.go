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

// withTempHome is a test helper that temporarily changes the HOME environment variable.
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

// setupTestConfig saves config to disk in the temp home directory.
func setupTestConfig(t *testing.T, cfg *config.Config) {
	t.Helper()

	if err := config.SaveGlobalConfig(cfg); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}
}

// newTestClient creates a Client wired to the given httptest server URL.
func newTestClient(serverURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    strings.TrimSuffix(serverURL, "/"),
		apiKey:     "test-token",
	}
}

// clientFromConfig creates a Client with specific config pointed at a test server.
func clientFromConfig(serverURL string, apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    strings.TrimSuffix(serverURL, "/"),
		apiKey:     apiKey,
	}
}

// TestNewClient_Success verifies that NewClient loads config from disk and creates a valid client.
func TestNewClient_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cfg := &config.Config{
		ManagementKey: "ravi_mgmt_test123",
		IdentityKey:   "ravi_id_test456",
		UserEmail:     "test@example.com",
	}
	setupTestConfig(t, cfg)

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Should prefer identity key
	if client.apiKey != cfg.IdentityKey {
		t.Errorf("client.apiKey = %v, want %v", client.apiKey, cfg.IdentityKey)
	}
	if client.userEmail != cfg.UserEmail {
		t.Errorf("client.userEmail = %v, want %v", client.userEmail, cfg.UserEmail)
	}

	expectedBaseURL := strings.TrimSuffix(server.URL, "/")
	if client.baseURL != expectedBaseURL {
		t.Errorf("client.baseURL = %v, want %v", client.baseURL, expectedBaseURL)
	}
}

// TestNewClient_FallsBackToManagementKey verifies that NewClient uses management key when no identity key.
func TestNewClient_FallsBackToManagementKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cleanupURL := withAPIBaseURL(t, server.URL)
	defer cleanupURL()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cfg := &config.Config{
		ManagementKey: "ravi_mgmt_test123",
		UserEmail:     "test@example.com",
	}
	setupTestConfig(t, cfg)

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client.apiKey != cfg.ManagementKey {
		t.Errorf("client.apiKey = %v, want %v (management key fallback)", client.apiKey, cfg.ManagementKey)
	}
}

// TestNewClient_NoAPIURL verifies that NewClient returns an error when API URL is not configured.
func TestNewClient_NoAPIURL(t *testing.T) {
	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	client, err := NewClient()
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

	client := clientFromConfig(server.URL, "")

	requestBody := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	resp, err := client.doRequest(http.MethodPost, "/test", requestBody, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	if receivedContentType != "application/json" {
		t.Errorf("Content-Type = %v, want application/json", receivedContentType)
	}

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

	client := clientFromConfig(server.URL, "ravi_id_test-key-12345")

	resp, err := client.doRequest(http.MethodGet, "/test", nil, true)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	expectedAuth := "Bearer ravi_id_test-key-12345"
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

	client := clientFromConfig(server.URL, "ravi_id_should-not-be-sent")

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

	client := clientFromConfig(server.URL, "")

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
			responseBody: nil,
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

			client := clientFromConfig(server.URL, "")

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

	client := clientFromConfig(server.URL, "")

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

// TestIsAuthenticated_True verifies that IsAuthenticated returns true when API key is present.
func TestIsAuthenticated_True(t *testing.T) {
	client := clientFromConfig("http://localhost", "ravi_id_valid-key")

	if !client.IsAuthenticated() {
		t.Error("IsAuthenticated() = false, want true when API key is present")
	}
}

// TestIsAuthenticated_False verifies that IsAuthenticated returns false when API key is missing.
func TestIsAuthenticated_False(t *testing.T) {
	client := clientFromConfig("http://localhost", "")

	if client.IsAuthenticated() {
		t.Error("IsAuthenticated() = true, want false when API key is missing")
	}
}

// TestBuildURL verifies that BuildURL correctly builds URLs with query parameters.
func TestBuildURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := clientFromConfig(server.URL, "")

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
			client := &Client{
				httpClient: &http.Client{Timeout: 5 * time.Second},
				baseURL:    "http://localhost",
				userEmail:  tc.userEmail,
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
		if r.ContentLength > 0 {
			t.Errorf("ContentLength = %v, want 0 for nil body", r.ContentLength)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := clientFromConfig(server.URL, "")

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

	client := clientFromConfig(server.URL, "")

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

	cleanupURL := withAPIBaseURL(t, server.URL+"/")
	defer cleanupURL()

	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if strings.HasSuffix(client.baseURL, "/") {
		t.Errorf("client.baseURL = %v, should not end with /", client.baseURL)
	}
}

// TestParseResponse_EmptyBody verifies that parseResponse handles empty response body.
func TestParseResponse_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := clientFromConfig(server.URL, "")

	resp, err := client.doRequest(http.MethodDelete, "/test", nil, false)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = client.parseResponse(resp, &result)

	if err != nil {
		t.Errorf("parseResponse() error = %v, want nil for empty body with 200", err)
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

			client := clientFromConfig(server.URL, "")

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

// TestNewClient_NoConfigFile verifies NewClient returns empty auth when no config file exists.
func TestNewClient_NoConfigFile(t *testing.T) {
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

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// API key should be empty
	if client.apiKey != "" {
		t.Errorf("client.apiKey = %v, want empty", client.apiKey)
	}
}
