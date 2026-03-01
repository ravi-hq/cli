package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ravi-hq/cli/internal/config"
)

// TestGetEncryptionMeta_Success verifies that GetEncryptionMeta parses a
// successful 200 JSON response into an EncryptionMeta struct.
func TestGetEncryptionMeta_Success(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	expectedMeta := EncryptionMeta{
		ID:               42,
		Salt:             "dGVzdC1zYWx0LXZhbHVl",
		Verifier:         "dGVzdC12ZXJpZmllcg==",
		PublicKey:        "dGVzdC1wdWJsaWMta2V5",
		ManagedMasterKey: "dGVzdC1tYXN0ZXIta2V5",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request properties.
		if r.URL.Path != PathEncryption {
			t.Errorf("Request path = %v, want %v", r.URL.Path, PathEncryption)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Request method = %v, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedMeta)
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	meta, err := client.GetEncryptionMeta()
	if err != nil {
		t.Fatalf("GetEncryptionMeta() error = %v", err)
	}
	if meta == nil {
		t.Fatal("GetEncryptionMeta() returned nil")
	}

	if meta.ID != expectedMeta.ID {
		t.Errorf("ID = %v, want %v", meta.ID, expectedMeta.ID)
	}
	if meta.Salt != expectedMeta.Salt {
		t.Errorf("Salt = %v, want %v", meta.Salt, expectedMeta.Salt)
	}
	if meta.Verifier != expectedMeta.Verifier {
		t.Errorf("Verifier = %v, want %v", meta.Verifier, expectedMeta.Verifier)
	}
	if meta.PublicKey != expectedMeta.PublicKey {
		t.Errorf("PublicKey = %v, want %v", meta.PublicKey, expectedMeta.PublicKey)
	}
	if meta.ManagedMasterKey != expectedMeta.ManagedMasterKey {
		t.Errorf("ManagedMasterKey = %v, want %v", meta.ManagedMasterKey, expectedMeta.ManagedMasterKey)
	}
}

// TestGetEncryptionMeta_EmptyPublicKey verifies that GetEncryptionMeta correctly
// handles a response where the public_key field is an empty string.
func TestGetEncryptionMeta_EmptyPublicKey(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(EncryptionMeta{
			ID:        1,
			Salt:      "c29tZS1zYWx0",
			Verifier:  "",
			PublicKey: "",
		})
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	meta, err := client.GetEncryptionMeta()
	if err != nil {
		t.Fatalf("GetEncryptionMeta() error = %v", err)
	}

	if meta.PublicKey != "" {
		t.Errorf("PublicKey = %q, want empty string", meta.PublicKey)
	}
}

// TestGetEncryptionMeta_ServerError verifies that GetEncryptionMeta returns
// an error when the server responds with 500.
func TestGetEncryptionMeta_ServerError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	_, err := client.GetEncryptionMeta()
	if err == nil {
		t.Fatal("GetEncryptionMeta() error = nil, want error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("GetEncryptionMeta() error = %v, want error containing '500'", err)
	}
}

// TestUpdateEncryptionMeta_Success verifies that UpdateEncryptionMeta sends a
// PATCH request and returns no error on a 200 response.
func TestUpdateEncryptionMeta_Success(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathEncryption {
			t.Errorf("Request path = %v, want %v", r.URL.Path, PathEncryption)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("Request method = %v, want PATCH", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	data := map[string]string{
		"salt":       "bmV3LXNhbHQ=",
		"verifier":   "bmV3LXZlcmlmaWVy",
		"public_key": "bmV3LXB1YmxpYy1rZXk=",
	}

	err := client.UpdateEncryptionMeta(data)
	if err != nil {
		t.Fatalf("UpdateEncryptionMeta() error = %v, want nil", err)
	}
}

// TestUpdateEncryptionMeta_ValidationError verifies that UpdateEncryptionMeta
// returns an error when the server responds with 400.
func TestUpdateEncryptionMeta_ValidationError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Error{Detail: "Invalid salt format"})
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	err := client.UpdateEncryptionMeta(map[string]string{"salt": "bad"})
	if err == nil {
		t.Fatal("UpdateEncryptionMeta() error = nil, want error for 400 status")
	}
	if !strings.Contains(err.Error(), "Invalid salt format") {
		t.Errorf("UpdateEncryptionMeta() error = %v, want error containing 'Invalid salt format'", err)
	}
}

// TestUpdateEncryptionMeta_ServerError verifies that UpdateEncryptionMeta
// returns an error when the server responds with 500.
func TestUpdateEncryptionMeta_ServerError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	err := client.UpdateEncryptionMeta(map[string]string{"salt": "value"})
	if err == nil {
		t.Fatal("UpdateEncryptionMeta() error = nil, want error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("UpdateEncryptionMeta() error = %v, want error containing '500'", err)
	}
}

// TestUpdateEncryptionMeta_SendsCorrectBody verifies that UpdateEncryptionMeta
// sends the expected JSON fields in the request body.
func TestUpdateEncryptionMeta_SendsCorrectBody(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(bodyBytes, &receivedBody); err != nil {
			t.Fatalf("Failed to parse request body: %v", err)
		}

		// Verify Content-Type.
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", contentType)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	})

	data := map[string]string{
		"salt":       "dGVzdC1zYWx0",
		"verifier":   "dGVzdC12ZXJpZmllcg==",
		"public_key": "dGVzdC1wdWJsaWMta2V5",
	}

	err := client.UpdateEncryptionMeta(data)
	if err != nil {
		t.Fatalf("UpdateEncryptionMeta() error = %v", err)
	}

	// Verify each field was sent correctly.
	for key, want := range data {
		got, ok := receivedBody[key]
		if !ok {
			t.Errorf("Request body missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("Request body[%q] = %q, want %q", key, got, want)
		}
	}

	// Verify no extra fields were sent.
	if len(receivedBody) != len(data) {
		t.Errorf("Request body has %d fields, want %d", len(receivedBody), len(data))
	}
}

// TestGetEncryptionMeta_AuthorizationHeader verifies that GetEncryptionMeta
// sends the Bearer token in the Authorization header.
func TestGetEncryptionMeta_AuthorizationHeader(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	var receivedAuthHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(EncryptionMeta{ID: 1})
	}))
	defer server.Close()

	client := clientFromAuth(server.URL, &config.AuthConfig{
		AccessToken:  "my-secret-token",
		RefreshToken: "my-refresh-token",
	})

	_, err := client.GetEncryptionMeta()
	if err != nil {
		t.Fatalf("GetEncryptionMeta() error = %v", err)
	}

	expectedAuth := "Bearer my-secret-token"
	if receivedAuthHeader != expectedAuth {
		t.Errorf("Authorization header = %q, want %q", receivedAuthHeader, expectedAuth)
	}
}
