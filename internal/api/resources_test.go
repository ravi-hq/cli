package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetPhone_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathPhone {
			t.Errorf("Expected path %s, got %s", PathPhone, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Phone{
			{ID: 1, PhoneNumber: "+15551234567", Provider: "twilio"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	phone, err := client.GetPhone()
	if err != nil {
		t.Fatalf("GetPhone() error = %v", err)
	}
	if phone.PhoneNumber != "+15551234567" {
		t.Errorf("PhoneNumber = %q, want +15551234567", phone.PhoneNumber)
	}
}

func TestGetPhone_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Phone{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetPhone()
	if err == nil {
		t.Fatal("GetPhone() error = nil, want error for empty result")
	}
	if !strings.Contains(err.Error(), "no phone number") {
		t.Errorf("Error = %q, want to contain 'no phone number'", err.Error())
	}
}

func TestGetEmail_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Email{
			{ID: 42, Email: "user@ravi.id"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	email, err := client.GetEmail()
	if err != nil {
		t.Fatalf("GetEmail() error = %v", err)
	}
	if email.Email != "user@ravi.id" {
		t.Errorf("Email = %q, want user@ravi.id", email.Email)
	}
}

func TestGetEmail_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Email{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetEmail()
	if err == nil {
		t.Fatal("GetEmail() error = nil, want error for empty result")
	}
	if !strings.Contains(err.Error(), "no email address") {
		t.Errorf("Error = %q, want to contain 'no email address'", err.Error())
	}
}

func TestGetOwner_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathOwner {
			t.Errorf("Expected path %s, got %s", PathOwner, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Owner{FirstName: "John", LastName: "Doe"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	owner, err := client.GetOwner()
	if err != nil {
		t.Fatalf("GetOwner() error = %v", err)
	}
	if owner.FirstName != "John" {
		t.Errorf("FirstName = %q, want John", owner.FirstName)
	}
}

func TestListSMSMessages_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathMessages {
			t.Errorf("Expected path %s, got %s", PathMessages, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PhoneMessage{
			{ID: 1, Body: "Hello", Direction: "inbound"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	messages, err := client.ListSMSMessages(false)
	if err != nil {
		t.Fatalf("ListSMSMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len = %d, want 1", len(messages))
	}
}

func TestListSMSMessages_UnreadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("is_read") != "false" {
			t.Errorf("Expected is_read=false, got %q", r.URL.Query().Get("is_read"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]PhoneMessage{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListSMSMessages(true)
	if err != nil {
		t.Fatalf("ListSMSMessages(true) error = %v", err)
	}
}

func TestGetSMSMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, PathMessages) {
			t.Errorf("Expected path prefix %s", PathMessages)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PhoneMessage{ID: 42, Body: "Test message"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.GetSMSMessage("42")
	if err != nil {
		t.Fatalf("GetSMSMessage() error = %v", err)
	}
	if msg.Body != "Test message" {
		t.Errorf("Body = %q, want 'Test message'", msg.Body)
	}
}

func TestListEmailMessages_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathEmailMessages {
			t.Errorf("Expected path %s, got %s", PathEmailMessages, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EmailMessageDetail{
			{ID: 1, Subject: "Test email", CreatedDt: time.Now()},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	messages, err := client.ListEmailMessages(false)
	if err != nil {
		t.Fatalf("ListEmailMessages() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len = %d, want 1", len(messages))
	}
}

func TestListEmailMessages_UnreadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("is_read") != "false" {
			t.Errorf("Expected is_read=false, got %q", r.URL.Query().Get("is_read"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EmailMessageDetail{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListEmailMessages(true)
	if err != nil {
		t.Fatalf("ListEmailMessages(true) error = %v", err)
	}
}

func TestGetEmailMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EmailMessageDetail{ID: 99, Subject: "Important"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.GetEmailMessage("99")
	if err != nil {
		t.Fatalf("GetEmailMessage() error = %v", err)
	}
	if msg.Subject != "Important" {
		t.Errorf("Subject = %q, want Important", msg.Subject)
	}
}

func TestGetPhone_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Server error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetPhone()
	if err == nil {
		t.Fatal("GetPhone() error = nil, want error")
	}
}

func TestGetEmail_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Error{Detail: "Forbidden"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetEmail()
	if err == nil {
		t.Fatal("GetEmail() error = nil, want error")
	}
}

func TestGetOwner_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Error{Detail: "Unauthorized"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetOwner()
	if err == nil {
		t.Fatal("GetOwner() error = nil, want error")
	}
}

func TestGetSMSMessage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetSMSMessage("999")
	if err == nil {
		t.Fatal("GetSMSMessage() error = nil, want error")
	}
}

func TestGetEmailMessage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetEmailMessage("999")
	if err == nil {
		t.Fatal("GetEmailMessage() error = nil, want error")
	}
}

func TestListSMSMessages_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListSMSMessages(false)
	if err == nil {
		t.Fatal("ListSMSMessages() error = nil, want error")
	}
}

func TestListEmailMessages_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{Detail: "Error"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListEmailMessages(false)
	if err == nil {
		t.Fatal("ListEmailMessages() error = nil, want error")
	}
}
