package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestReplyEmailWithCC(t *testing.T) {
	var receivedBody ReplyRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EmailMessageDetail{ID: 10})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ReplyEmail("789", ReplyRequest{
		Content: "reply with cc",
		CC:      []string{"alice@example.com", "bob@example.com"},
		BCC:     []string{"secret@example.com"},
	})
	if err != nil {
		t.Fatalf("ReplyEmail() error = %v", err)
	}
	if len(receivedBody.CC) != 2 {
		t.Errorf("CC length = %d, want 2", len(receivedBody.CC))
	}
	if receivedBody.CC[0] != "alice@example.com" {
		t.Errorf("CC[0] = %v, want alice@example.com", receivedBody.CC[0])
	}
	if len(receivedBody.BCC) != 1 {
		t.Errorf("BCC length = %d, want 1", len(receivedBody.BCC))
	}
	if receivedBody.BCC[0] != "secret@example.com" {
		t.Errorf("BCC[0] = %v, want secret@example.com", receivedBody.BCC[0])
	}
}

func TestForwardEmail(t *testing.T) {
	var receivedPath string
	var receivedBody ForwardRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(EmailMessageDetail{ID: 5})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	result, err := client.ForwardEmail("100", ForwardRequest{
		ToEmail: "forward@example.com",
		Content: "<p>FYI</p>",
		CC:      []string{"cc@example.com"},
	})
	if err != nil {
		t.Fatalf("ForwardEmail() error = %v", err)
	}
	if result.ID != 5 {
		t.Errorf("result.ID = %v, want 5", result.ID)
	}
	if receivedPath != "/api/email-messages/100/forward/" {
		t.Errorf("path = %v, want /api/email-messages/100/forward/", receivedPath)
	}
	if receivedBody.ToEmail != "forward@example.com" {
		t.Errorf("body.ToEmail = %v, want forward@example.com", receivedBody.ToEmail)
	}
	if len(receivedBody.CC) != 1 || receivedBody.CC[0] != "cc@example.com" {
		t.Errorf("body.CC = %v, want [cc@example.com]", receivedBody.CC)
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
		json.NewEncoder(w).Encode([]Email{{ID: 42, Email: "test@ravi.id"}})
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

func TestGetInboxID_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Email{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetInboxID()
	if err == nil {
		t.Fatal("GetInboxID() error = nil, want error when no emails")
	}
}

func TestUploadToPresignedURL_Success(t *testing.T) {
	var receivedContentType string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "upload-test-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test file content")
	tmpFile.Close()

	client := newTestClient(server.URL)
	err = client.UploadToPresignedURL(server.URL+"/upload", tmpFile.Name(), "text/plain")
	if err != nil {
		t.Fatalf("UploadToPresignedURL() error = %v", err)
	}
	if receivedMethod != http.MethodPut {
		t.Errorf("Method = %s, want PUT", receivedMethod)
	}
	if receivedContentType != "text/plain" {
		t.Errorf("Content-Type = %s, want text/plain", receivedContentType)
	}
}

func TestUploadToPresignedURL_FileNotFound(t *testing.T) {
	client := newTestClient("http://localhost")
	err := client.UploadToPresignedURL("http://localhost/upload", "/nonexistent/file.txt", "text/plain")
	if err == nil {
		t.Fatal("UploadToPresignedURL() error = nil, want error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Errorf("Error = %q, want to contain 'failed to open file'", err.Error())
	}
}

func TestUploadToPresignedURL_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Access denied"))
	}))
	defer server.Close()

	tmpFile, err := os.CreateTemp("", "upload-test-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test")
	tmpFile.Close()

	client := newTestClient(server.URL)
	err = client.UploadToPresignedURL(server.URL+"/upload", tmpFile.Name(), "text/plain")
	if err == nil {
		t.Fatal("UploadToPresignedURL() error = nil, want error for 403")
	}
	if !strings.Contains(err.Error(), "upload failed") {
		t.Errorf("Error = %q, want to contain 'upload failed'", err.Error())
	}
}

func TestPresignAttachment_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Error{Detail: "Invalid"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.PresignAttachment(PresignRequest{})
	if err == nil {
		t.Fatal("PresignAttachment() error = nil, want error")
	}
}

func TestReplyEmail_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ReplyEmail("999", ReplyRequest{Content: "test"})
	if err == nil {
		t.Fatal("ReplyEmail() error = nil, want error")
	}
}

func TestReplyAllEmail_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ReplyAllEmail("999", ReplyRequest{Content: "test"})
	if err == nil {
		t.Fatal("ReplyAllEmail() error = nil, want error")
	}
}

func TestForwardEmail_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ForwardEmail("999", ForwardRequest{ToEmail: "test@example.com"})
	if err == nil {
		t.Fatal("ForwardEmail() error = nil, want error")
	}
}

func TestUploadAttachment_BlockedExtension(t *testing.T) {
	blockedFile, err := os.CreateTemp("", "upload-test-*.exe")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer os.Remove(blockedFile.Name())
	blockedFile.WriteString("test")
	blockedFile.Close()

	client := newTestClient("http://localhost")
	_, err = client.UploadAttachment(blockedFile.Name())
	if err == nil {
		t.Fatal("UploadAttachment() error = nil, want error for .exe file")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("Error = %q, want to contain 'not allowed'", err.Error())
	}
}

func TestUploadAttachment_FileNotFound(t *testing.T) {
	client := newTestClient("http://localhost")
	_, err := client.UploadAttachment("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("UploadAttachment() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "cannot access") {
		t.Errorf("Error = %q, want to contain 'cannot access'", err.Error())
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
