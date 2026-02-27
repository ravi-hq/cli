package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListPasswords_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathPasswords {
			t.Errorf("Expected path %s, got %s", PathPasswords, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		entries := []PasswordEntry{
			{UUID: "uuid-1", Domain: "github.com", Username: "e2e::abc", Password: "e2e::def", CreatedDt: "2026-02-10"},
			{UUID: "uuid-2", Domain: "google.com", Username: "e2e::ghi", Password: "e2e::jkl", CreatedDt: "2026-02-09"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entries, err := client.ListPasswords()
	if err != nil {
		t.Fatalf("ListPasswords() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	if entries[0].Domain != "github.com" {
		t.Errorf("entries[0].Domain = %s, want github.com", entries[0].Domain)
	}
}

func TestListPasswords_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PasswordEntry{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entries, err := client.ListPasswords()
	if err != nil {
		t.Fatalf("ListPasswords() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestGetPassword_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := PathPasswords + "test-uuid-123/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		entry := PasswordEntry{UUID: "test-uuid-123", Domain: "github.com", Username: "e2e::abc"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entry, err := client.GetPassword("test-uuid-123")
	if err != nil {
		t.Fatalf("GetPassword() error = %v", err)
	}
	if entry.UUID != "test-uuid-123" {
		t.Errorf("UUID = %s, want test-uuid-123", entry.UUID)
	}
	if entry.Domain != "github.com" {
		t.Errorf("Domain = %s, want github.com", entry.Domain)
	}
}

func TestGetPassword_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetPassword("nonexistent-uuid")
	if err == nil {
		t.Fatal("GetPassword() expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "Not found") {
		t.Errorf("Error should contain 'Not found', got: %v", err)
	}
}

func TestCreatePassword_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != PathPasswords {
			t.Errorf("Expected path %s, got %s", PathPasswords, r.URL.Path)
		}

		var input PasswordEntry
		json.NewDecoder(r.Body).Decode(&input)
		if input.Domain != "github.com" {
			t.Errorf("input.Domain = %s, want github.com", input.Domain)
		}

		result := PasswordEntry{UUID: "new-uuid", Domain: input.Domain, Username: input.Username, Password: input.Password}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	entry := PasswordEntry{Domain: "github.com", Username: "e2e::user", Password: "e2e::pass"}
	result, err := client.CreatePassword(entry)
	if err != nil {
		t.Fatalf("CreatePassword() error = %v", err)
	}
	if result.UUID != "new-uuid" {
		t.Errorf("UUID = %s, want new-uuid", result.UUID)
	}
}

func TestUpdatePassword_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		expectedPath := PathPasswords + "update-uuid/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		result := PasswordEntry{UUID: "update-uuid", Domain: "updated.com"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.UpdatePassword("update-uuid", map[string]interface{}{"domain": "updated.com"})
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}
	if result.Domain != "updated.com" {
		t.Errorf("Domain = %s, want updated.com", result.Domain)
	}
}

func TestDeletePassword_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		expectedPath := PathPasswords + "delete-uuid/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DeletePassword("delete-uuid")
	if err != nil {
		t.Fatalf("DeletePassword() error = %v", err)
	}
}

func TestGeneratePassword_DefaultOpts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, PathPasswords+"generate-password/") {
			t.Errorf("Expected generate-password path, got %s", r.URL.Path)
		}

		result := GeneratedPassword{Password: "xK9mP2qR5sT7"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.GeneratePassword(PasswordGenOpts{})
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if result.Password != "xK9mP2qR5sT7" {
		t.Errorf("Password = %s, want xK9mP2qR5sT7", result.Password)
	}
}

func TestGeneratePassword_CustomOpts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("length") != "24" {
			t.Errorf("Expected length=24, got %s", query.Get("length"))
		}
		if query.Get("special") != "false" {
			t.Errorf("Expected special=false, got %s", query.Get("special"))
		}
		if query.Get("exclude_chars") != "abc" {
			t.Errorf("Expected exclude_chars=abc, got %s", query.Get("exclude_chars"))
		}
		// uppercase/lowercase/digits should not have params (they default true)
		if query.Get("uppercase") != "" {
			t.Errorf("Expected no uppercase param, got %s", query.Get("uppercase"))
		}

		result := GeneratedPassword{Password: "customGenerated123"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	opts := PasswordGenOpts{
		Length:       24,
		NoSpecial:    true,
		ExcludeChars: "abc",
	}
	result, err := client.GeneratePassword(opts)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if result.Password != "customGenerated123" {
		t.Errorf("Password = %s, want customGenerated123", result.Password)
	}
}
