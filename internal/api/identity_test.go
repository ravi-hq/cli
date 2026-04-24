package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListIdentities_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != PathIdentities {
			t.Errorf("Expected path %s, got %s", PathIdentities, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Identity{
			{UUID: "id-1", Name: "Personal", Email: "user@ravi.id"},
			{UUID: "id-2", Name: "Work", Email: "work@ravi.id"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	identities, err := client.ListIdentities()
	if err != nil {
		t.Fatalf("ListIdentities() error = %v", err)
	}
	if len(identities) != 2 {
		t.Fatalf("ListIdentities() len = %d, want 2", len(identities))
	}
	if identities[0].Name != "Personal" {
		t.Errorf("identities[0].Name = %q, want Personal", identities[0].Name)
	}
}

func TestListIdentities_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Error{Detail: "Authentication required"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListIdentities()
	if err == nil {
		t.Fatal("ListIdentities() error = nil, want error")
	}
}

func TestCreateIdentity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["name"] != "Research" {
			t.Errorf("name = %q, want Research", req["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Identity{UUID: "new-id", Name: "Research", Email: "research@ravi.id"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	identity, err := client.CreateIdentity("Research", "", false)
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	if identity.UUID != "new-id" {
		t.Errorf("UUID = %q, want new-id", identity.UUID)
	}
	if identity.Name != "Research" {
		t.Errorf("Name = %q, want Research", identity.Name)
	}
}

func TestCreateIdentity_WithEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if req["email"] != "shopping@acme.com" {
			t.Errorf("email = %q, want shopping@acme.com", req["email"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Identity{UUID: "id-3", Name: "Shopping", Email: "shopping@acme.com"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	identity, err := client.CreateIdentity("Shopping", "shopping@acme.com", false)
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	if identity.Email != "shopping@acme.com" {
		t.Errorf("Email = %q, want shopping@acme.com", identity.Email)
	}
}

func TestCreateIdentity_EmptyNameAndEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if _, ok := req["name"]; ok {
			t.Error("Expected name to be omitted when empty")
		}
		if _, ok := req["email"]; ok {
			t.Error("Expected email to be omitted when empty")
		}
		if _, ok := req["provision_phone"]; ok {
			t.Error("Expected provision_phone to be omitted when false")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Identity{UUID: "id-auto", Name: "auto-generated"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.CreateIdentity("", "", false)
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
}

func TestCreateIdentity_WithProvisionPhone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if req["provision_phone"] != true {
			t.Errorf("provision_phone = %v, want true", req["provision_phone"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Identity{UUID: "id-4", Name: "WithPhone", Phone: "+15551234567"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	identity, err := client.CreateIdentity("WithPhone", "", true)
	if err != nil {
		t.Fatalf("CreateIdentity() error = %v", err)
	}
	if identity.Phone != "+15551234567" {
		t.Errorf("Phone = %q, want +15551234567", identity.Phone)
	}
}

func TestCreateIdentity_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Error{Detail: "Validation error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.CreateIdentity("Bad", "", false)
	if err == nil {
		t.Fatal("CreateIdentity() error = nil, want error")
	}
}

func TestListDomains_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListDomains()
	if err == nil {
		t.Fatal("ListDomains() error = nil, want error")
	}
}

func TestListDomains_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathDomains {
			t.Errorf("Expected path %s, got %s", PathDomains, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EmailDomain{
			{UUID: "d1", Domain: "ravi.id", IsPlatform: true, IsVerified: true},
			{UUID: "d2", Domain: "acme.com", IsPlatform: false, IsVerified: true},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	domains, err := client.ListDomains()
	if err != nil {
		t.Fatalf("ListDomains() error = %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("ListDomains() len = %d, want 2", len(domains))
	}
	if domains[0].Domain != "ravi.id" {
		t.Errorf("domains[0].Domain = %q, want ravi.id", domains[0].Domain)
	}
}
