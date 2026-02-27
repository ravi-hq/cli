package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListSecrets_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathSecrets {
			t.Errorf("Expected path %s, got %s", PathSecrets, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		entries := []SecretEntry{
			{UUID: "uuid-1", Key: "OPENAI_API_KEY", Value: "e2e::abc", CreatedDt: "2026-02-10"},
			{UUID: "uuid-2", Key: "STRIPE_KEY", Value: "e2e::def", CreatedDt: "2026-02-09"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entries, err := client.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	if entries[0].Key != "OPENAI_API_KEY" {
		t.Errorf("entries[0].Key = %s, want OPENAI_API_KEY", entries[0].Key)
	}
}

func TestListSecrets_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]SecretEntry{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entries, err := client.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestGetSecret_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "OPENAI_API_KEY" {
			t.Errorf("Expected key query param OPENAI_API_KEY, got %s", r.URL.Query().Get("key"))
		}

		entries := []SecretEntry{
			{UUID: "uuid-1", Key: "OPENAI_API_KEY", Value: "e2e::abc"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entry, err := client.GetSecret("OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if entry == nil {
		t.Fatal("GetSecret() returned nil, want entry")
	}
	if entry.Key != "OPENAI_API_KEY" {
		t.Errorf("Key = %s, want OPENAI_API_KEY", entry.Key)
	}
}

func TestGetSecret_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]SecretEntry{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entry, err := client.GetSecret("NONEXISTENT_KEY")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if entry != nil {
		t.Errorf("GetSecret() = %v, want nil for missing key", entry)
	}
}

func TestGetSecretByUUID_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := PathSecrets + "test-uuid-123/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		entry := SecretEntry{UUID: "test-uuid-123", Key: "OPENAI_API_KEY", Value: "e2e::abc"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entry, err := client.GetSecretByUUID("test-uuid-123")
	if err != nil {
		t.Fatalf("GetSecretByUUID() error = %v", err)
	}
	if entry.UUID != "test-uuid-123" {
		t.Errorf("UUID = %s, want test-uuid-123", entry.UUID)
	}
	if entry.Key != "OPENAI_API_KEY" {
		t.Errorf("Key = %s, want OPENAI_API_KEY", entry.Key)
	}
}

func TestCreateSecret_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != PathSecrets {
			t.Errorf("Expected path %s, got %s", PathSecrets, r.URL.Path)
		}

		var input SecretEntry
		json.NewDecoder(r.Body).Decode(&input)
		if input.Key != "OPENAI_API_KEY" {
			t.Errorf("input.Key = %s, want OPENAI_API_KEY", input.Key)
		}

		result := SecretEntry{UUID: "new-uuid", Key: input.Key, Value: input.Value}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entry := SecretEntry{Key: "OPENAI_API_KEY", Value: "e2e::secret-value"}
	result, err := client.CreateSecret(entry)
	if err != nil {
		t.Fatalf("CreateSecret() error = %v", err)
	}
	if result.UUID != "new-uuid" {
		t.Errorf("UUID = %s, want new-uuid", result.UUID)
	}
}

func TestUpdateSecret_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		expectedPath := PathSecrets + "update-uuid/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		result := SecretEntry{UUID: "update-uuid", Key: "UPDATED_KEY", Value: "e2e::updated"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.UpdateSecret("update-uuid", map[string]interface{}{"value": "e2e::updated"})
	if err != nil {
		t.Fatalf("UpdateSecret() error = %v", err)
	}
	if result.Key != "UPDATED_KEY" {
		t.Errorf("Key = %s, want UPDATED_KEY", result.Key)
	}
}

func TestDeleteSecret_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		expectedPath := PathSecrets + "delete-uuid/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DeleteSecret("delete-uuid")
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DeleteSecret("nonexistent-uuid")
	if err == nil {
		t.Fatal("DeleteSecret() expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "Not found") {
		t.Errorf("Error should contain 'Not found', got: %v", err)
	}
}
