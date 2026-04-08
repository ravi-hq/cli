package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestSSOToken_Success verifies that RequestSSOToken returns a valid token response.
func TestRequestSSOToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != PathSSOToken {
			t.Errorf("Expected path %s, got %s", PathSSOToken, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", r.Header.Get("Authorization"))
		}

		response := SSOTokenResponse{
			Token:     "rvt_test123",
			ExpiresAt: "2026-04-07T12:05:00Z",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	result, err := client.RequestSSOToken()
	if err != nil {
		t.Fatalf("RequestSSOToken() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("RequestSSOToken() result is nil, want non-nil")
	}
	if result.Token != "rvt_test123" {
		t.Errorf("Token = %q, want %q", result.Token, "rvt_test123")
	}
	if result.ExpiresAt != "2026-04-07T12:05:00Z" {
		t.Errorf("ExpiresAt = %q, want %q", result.ExpiresAt, "2026-04-07T12:05:00Z")
	}
	if !strings.HasPrefix(result.Token, "rvt_") {
		t.Errorf("Token = %q, want prefix 'rvt_'", result.Token)
	}
}

// TestRequestSSOToken_Unauthenticated verifies 401 response handling.
func TestRequestSSOToken_Unauthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Error{Detail: "Authentication credentials were not provided."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	result, err := client.RequestSSOToken()
	if err == nil {
		t.Fatal("RequestSSOToken() expected error for 401, got nil")
	}
	if result != nil {
		t.Errorf("RequestSSOToken() result = %v, want nil on error", result)
	}
	if !strings.Contains(err.Error(), "Authentication credentials") {
		t.Errorf("Error = %q, want to contain 'Authentication credentials'", err.Error())
	}
}

// TestRequestSSOToken_NoSubscription verifies 402 response handling.
func TestRequestSSOToken_NoSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(Error{Detail: "Active subscription required."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	result, err := client.RequestSSOToken()
	if err == nil {
		t.Fatal("RequestSSOToken() expected error for 402, got nil")
	}
	if result != nil {
		t.Errorf("RequestSSOToken() result = %v, want nil on error", result)
	}
	if !strings.Contains(err.Error(), "Active subscription required") {
		t.Errorf("Error = %q, want to contain 'Active subscription required'", err.Error())
	}
}

// TestRequestSSOToken_WrongKeyType verifies 403 response handling when using a management key.
func TestRequestSSOToken_WrongKeyType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Error{Detail: "Identity-scoped key required."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)

	result, err := client.RequestSSOToken()
	if err == nil {
		t.Fatal("RequestSSOToken() expected error for 403, got nil")
	}
	if result != nil {
		t.Errorf("RequestSSOToken() result = %v, want nil on error", result)
	}
	if !strings.Contains(err.Error(), "Identity-scoped key required") {
		t.Errorf("Error = %q, want to contain 'Identity-scoped key required'", err.Error())
	}
}
