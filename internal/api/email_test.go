package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComposeEmail(t *testing.T) {
	var receivedPath string
	var receivedBody ComposeRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.RequestURI()
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EmailMessageDetail{
			ID:      1,
			Subject: "Test",
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	req := ComposeRequest{
		ToEmail: "user@example.com",
		Subject: "Test",
		Content: "<p>Hello</p>",
	}

	result, err := client.ComposeEmail(42, req)
	if err != nil {
		t.Fatalf("ComposeEmail() error = %v", err)
	}
	if result.ID != 1 {
		t.Errorf("result.ID = %v, want 1", result.ID)
	}
	if !strings.Contains(receivedPath, "inbox=42") {
		t.Errorf("path = %v, want to contain inbox=42", receivedPath)
	}
	if receivedBody.ToEmail != "user@example.com" {
		t.Errorf("body.ToEmail = %v, want user@example.com", receivedBody.ToEmail)
	}
}

func TestReplyEmail(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EmailMessageDetail{ID: 2})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ReplyEmail("123", ReplyRequest{
		Content: "reply body",
		Subject: "Re: Test",
	})
	if err != nil {
		t.Fatalf("ReplyEmail() error = %v", err)
	}
	if result.ID != 2 {
		t.Errorf("result.ID = %v, want 2", result.ID)
	}
	if receivedPath != "/api/email-messages/123/reply/" {
		t.Errorf("path = %v, want /api/email-messages/123/reply/", receivedPath)
	}
}

func TestReplyAllEmail(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EmailMessageDetail{ID: 3})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ReplyAllEmail("456", ReplyRequest{
		Content: "reply all body",
		Subject: "Re: Test",
	})
	if err != nil {
		t.Fatalf("ReplyAllEmail() error = %v", err)
	}
	if result.ID != 3 {
		t.Errorf("result.ID = %v, want 3", result.ID)
	}
	if receivedPath != "/api/email-messages/456/reply-all/" {
		t.Errorf("path = %v, want /api/email-messages/456/reply-all/", receivedPath)
	}
}

func TestPresignAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathEmailAttachmentPresign {
			t.Errorf("path = %v, want %v", r.URL.Path, PathEmailAttachmentPresign)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PresignResponse{
			UUID:       "test-uuid-123",
			UploadURL:  "https://r2.example.com/upload",
			StorageKey: "attachments/1/test-uuid-123/doc.pdf",
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.PresignAttachment(PresignRequest{
		Filename:    "doc.pdf",
		ContentType: "application/pdf",
		Size:        1024,
	})
	if err != nil {
		t.Fatalf("PresignAttachment() error = %v", err)
	}
	if result.UUID != "test-uuid-123" {
		t.Errorf("result.UUID = %v, want test-uuid-123", result.UUID)
	}
}

func TestGetInboxID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Email{{ID: 42, Email: "test@ravi.app"}})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	id, err := client.GetInboxID()
	if err != nil {
		t.Fatalf("GetInboxID() error = %v", err)
	}
	if id != 42 {
		t.Errorf("GetInboxID() = %v, want 42", id)
	}
}

func TestRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "42")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"detail":              "Request was throttled.",
			"retry_after_seconds": 42,
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ComposeEmail(1, ComposeRequest{
		ToEmail: "test@example.com",
		Subject: "Test",
		Content: "body",
	})
	if err == nil {
		t.Fatal("ComposeEmail() error = nil, want RateLimitError")
	}

	rlErr, ok := err.(*RateLimitError)
	if !ok {
		t.Fatalf("error type = %T, want *RateLimitError", err)
	}
	if rlErr.RetryAfterSeconds != 42 {
		t.Errorf("RetryAfterSeconds = %v, want 42", rlErr.RetryAfterSeconds)
	}
	if !strings.Contains(rlErr.Error(), "retry in 42s") {
		t.Errorf("Error() = %v, want to contain 'retry in 42s'", rlErr.Error())
	}
}
