package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProvisionPhone_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		wantPath := PathIdentities + "id-uuid-1/provision-phone/"
		if r.URL.Path != wantPath {
			t.Errorf("path = %s, want %s", r.URL.Path, wantPath)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["country_code"] != "CA" {
			t.Errorf("country_code = %v, want CA", body["country_code"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Identity{UUID: "id-uuid-1", Name: "Agent", Phone: "+15551230000"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	identity, err := client.ProvisionPhone("id-uuid-1", "CA")
	if err != nil {
		t.Fatalf("ProvisionPhone() error = %v", err)
	}
	if identity.Phone != "+15551230000" {
		t.Errorf("phone = %q, want +15551230000", identity.Phone)
	}
}

func TestProvisionPhone_DefaultsCountryCodeUS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["country_code"] != "US" {
			t.Errorf("country_code = %v, want US (default)", body["country_code"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Identity{UUID: "id-uuid-1", Phone: "+15551230000"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	if _, err := client.ProvisionPhone("id-uuid-1", ""); err != nil {
		t.Fatalf("ProvisionPhone() error = %v", err)
	}
}

func TestProvisionPhone_PaymentRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(Error{Detail: "A paid plan is required."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ProvisionPhone("id-uuid-1", "US")
	if err == nil {
		t.Fatal("ProvisionPhone() error = nil, want error on 402")
	}
	if !strings.Contains(err.Error(), "paid plan") {
		t.Errorf("error = %v, want it to surface the 402 detail", err)
	}
}
