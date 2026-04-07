package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListContacts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != PathContacts {
			t.Errorf("Expected path %s, got %s", PathContacts, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ContactEntry{
			{UUID: "c1", DisplayName: "Alice", Email: "alice@example.com"},
			{UUID: "c2", DisplayName: "Bob", PhoneNumber: "+1234567890"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	contacts, err := client.ListContacts()
	if err != nil {
		t.Fatalf("ListContacts() error = %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("ListContacts() len = %d, want 2", len(contacts))
	}
	if contacts[0].DisplayName != "Alice" {
		t.Errorf("contacts[0].DisplayName = %q, want Alice", contacts[0].DisplayName)
	}
}

func TestListContacts_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Server error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListContacts()
	if err == nil {
		t.Fatal("ListContacts() error = nil, want error")
	}
}

func TestGetContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, PathContacts) {
			t.Errorf("Expected path prefix %s, got %s", PathContacts, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactEntry{UUID: "c1", DisplayName: "Alice"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	contact, err := client.GetContact("c1")
	if err != nil {
		t.Fatalf("GetContact() error = %v", err)
	}
	if contact.UUID != "c1" {
		t.Errorf("UUID = %q, want c1", contact.UUID)
	}
}

func TestCreateContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var entry ContactEntry
		json.NewDecoder(r.Body).Decode(&entry)
		entry.UUID = "new-uuid"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.CreateContact(ContactEntry{DisplayName: "Charlie", Email: "charlie@example.com"})
	if err != nil {
		t.Fatalf("CreateContact() error = %v", err)
	}
	if result.UUID != "new-uuid" {
		t.Errorf("UUID = %q, want new-uuid", result.UUID)
	}
}

func TestUpdateContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactEntry{UUID: "c1", DisplayName: "Updated"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.UpdateContact("c1", map[string]interface{}{"display_name": "Updated"})
	if err != nil {
		t.Fatalf("UpdateContact() error = %v", err)
	}
	if result.DisplayName != "Updated" {
		t.Errorf("DisplayName = %q, want Updated", result.DisplayName)
	}
}

func TestDeleteContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	err := client.DeleteContact("c1")
	if err != nil {
		t.Fatalf("DeleteContact() error = %v", err)
	}
}

func TestFindContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "find") {
			t.Errorf("Expected path to contain 'find', got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactEntry{UUID: "c1", Email: "alice@example.com"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.FindContact("alice@example.com", "")
	if err != nil {
		t.Fatalf("FindContact() error = %v", err)
	}
	if result.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", result.Email)
	}
}

func TestFindContact_WithPhone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("phone_number") == "" {
			t.Error("Expected phone_number query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ContactEntry{UUID: "c2", PhoneNumber: "+1234567890"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.FindContact("", "+1234567890")
	if err != nil {
		t.Fatalf("FindContact() error = %v", err)
	}
	if result.PhoneNumber != "+1234567890" {
		t.Errorf("PhoneNumber = %q, want +1234567890", result.PhoneNumber)
	}
}

func TestGetContact_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetContact("bad-uuid")
	if err == nil {
		t.Fatal("GetContact() error = nil, want error")
	}
}

func TestCreateContact_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Error{Detail: "Invalid"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.CreateContact(ContactEntry{})
	if err == nil {
		t.Fatal("CreateContact() error = nil, want error")
	}
}

func TestUpdateContact_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.UpdateContact("bad", map[string]interface{}{})
	if err == nil {
		t.Fatal("UpdateContact() error = nil, want error")
	}
}

func TestSearchContacts_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SearchContacts("test")
	if err == nil {
		t.Fatal("SearchContacts() error = nil, want error")
	}
}

func TestFindContact_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.FindContact("nobody@example.com", "")
	if err == nil {
		t.Fatal("FindContact() error = nil, want error")
	}
}

func TestSearchContacts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "search") {
			t.Errorf("Expected path to contain 'search', got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ContactEntry{
			{UUID: "c1", DisplayName: "Alice"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	results, err := client.SearchContacts("alice")
	if err != nil {
		t.Fatalf("SearchContacts() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("SearchContacts() len = %d, want 1", len(results))
	}
}
