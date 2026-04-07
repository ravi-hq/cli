package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestListEmailThreads_Success verifies that ListEmailThreads returns email threads.
func TestListEmailThreads_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path
		if r.URL.Path != PathEmailInbox {
			t.Errorf("Expected path %s, got %s", PathEmailInbox, r.URL.Path)
		}

		// Verify no has_unread param when not filtering
		if r.URL.Query().Get("has_unread") != "" {
			t.Errorf("Expected no has_unread param, got %s", r.URL.Query().Get("has_unread"))
		}

		threads := []EmailThread{
			{
				ThreadID:        "<thread-1@example.com>",
				Subject:         "First thread subject",
				Preview:         "Preview of first email...",
				FromEmail:       "alice@example.com",
				Email:     "user@ravi.id",
				MessageCount:    3,
				UnreadCount:     1,
				LatestMessageDt: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
				OldestMessageDt: time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC),
			},
			{
				ThreadID:        "<thread-2@example.com>",
				Subject:         "Second thread subject",
				Preview:         "Preview of second email...",
				FromEmail:       "bob@example.com",
				Email:     "user@ravi.id",
				MessageCount:    1,
				UnreadCount:     0,
				LatestMessageDt: time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
				OldestMessageDt: time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(threads)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	threads, err := client.ListEmailThreads(false)

	if err != nil {
		t.Fatalf("ListEmailThreads() error = %v", err)
	}

	if len(threads) != 2 {
		t.Fatalf("Expected 2 threads, got %d", len(threads))
	}

	// Verify first thread
	if threads[0].ThreadID != "<thread-1@example.com>" {
		t.Errorf("First thread ID = %s, want <thread-1@example.com>", threads[0].ThreadID)
	}
	if threads[0].Subject != "First thread subject" {
		t.Errorf("First thread Subject = %s, want 'First thread subject'", threads[0].Subject)
	}
	if threads[0].MessageCount != 3 {
		t.Errorf("First thread MessageCount = %d, want 3", threads[0].MessageCount)
	}
	if threads[0].UnreadCount != 1 {
		t.Errorf("First thread UnreadCount = %d, want 1", threads[0].UnreadCount)
	}

	// Verify second thread
	if threads[1].ThreadID != "<thread-2@example.com>" {
		t.Errorf("Second thread ID = %s, want <thread-2@example.com>", threads[1].ThreadID)
	}
	if threads[1].UnreadCount != 0 {
		t.Errorf("Second thread UnreadCount = %d, want 0", threads[1].UnreadCount)
	}
}

// TestListEmailThreads_UnreadOnly verifies that ListEmailThreads filters unread threads.
func TestListEmailThreads_UnreadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify has_unread param is set to true
		hasUnreadParam := r.URL.Query().Get("has_unread")
		if hasUnreadParam != "true" {
			t.Errorf("Expected has_unread=true, got %s", hasUnreadParam)
		}

		threads := []EmailThread{
			{
				ThreadID:     "<unread-thread@example.com>",
				Subject:      "Unread thread",
				Preview:      "This thread has unread messages",
				FromEmail:    "sender@example.com",
				Email:  "user@ravi.id",
				MessageCount: 2,
				UnreadCount:  2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(threads)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	threads, err := client.ListEmailThreads(true)

	if err != nil {
		t.Fatalf("ListEmailThreads() error = %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("Expected 1 thread, got %d", len(threads))
	}

	if threads[0].UnreadCount == 0 {
		t.Error("Expected thread with unread messages")
	}
}

// TestGetEmailThread_Success verifies that GetEmailThread returns thread detail.
func TestGetEmailThread_Success(t *testing.T) {
	threadID := "<test-thread-123@example.com>"
	encodedID := url.PathEscape(threadID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path contains encoded thread ID using EscapedPath()
		expectedPath := PathEmailInbox + encodedID + "/"
		if r.URL.EscapedPath() != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.EscapedPath())
		}

		threadDetail := EmailThreadDetail{
			ThreadID:     threadID,
			Subject:      "Test Thread Subject",
			MessageCount: 2,
			Messages: []EmailMessage{
				{
					ID:          1,
					FromEmail:   "alice@example.com",
					ToEmail:     "user@ravi.id",
					CC:          "",
					Subject:     "Test Thread Subject",
					TextContent: "First message in thread",
					HTMLContent: "<p>First message in thread</p>",
					Direction:   "inbound",
					IsRead:      true,
					CreatedDt:   time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC),
				},
				{
					ID:          2,
					FromEmail:   "user@ravi.id",
					ToEmail:     "alice@example.com",
					CC:          "",
					Subject:     "Re: Test Thread Subject",
					TextContent: "Reply to first message",
					HTMLContent: "<p>Reply to first message</p>",
					Direction:   "outbound",
					IsRead:      true,
					CreatedDt:   time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(threadDetail)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	detail, err := client.GetEmailThread(threadID)

	if err != nil {
		t.Fatalf("GetEmailThread() error = %v", err)
	}

	if detail == nil {
		t.Fatal("GetEmailThread() returned nil")
	}

	if detail.ThreadID != threadID {
		t.Errorf("ThreadID = %s, want %s", detail.ThreadID, threadID)
	}
	if detail.Subject != "Test Thread Subject" {
		t.Errorf("Subject = %s, want 'Test Thread Subject'", detail.Subject)
	}
	if detail.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", detail.MessageCount)
	}
	if len(detail.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(detail.Messages))
	}

	// Verify first message
	if detail.Messages[0].Direction != "inbound" {
		t.Errorf("First message Direction = %s, want inbound", detail.Messages[0].Direction)
	}

	// Verify second message
	if detail.Messages[1].Direction != "outbound" {
		t.Errorf("Second message Direction = %s, want outbound", detail.Messages[1].Direction)
	}
}

// TestGetEmailThread_NotFound verifies that GetEmailThread handles 404 error.
func TestGetEmailThread_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Thread not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetEmailThread("<nonexistent-thread@example.com>")

	if err == nil {
		t.Fatal("GetEmailThread() expected error for 404, got nil")
	}

	if !strings.Contains(err.Error(), "Thread not found") {
		t.Errorf("Error should contain 'Thread not found', got: %v", err)
	}
}

// TestListSMSConversations_Success verifies that ListSMSConversations returns SMS conversations.
func TestListSMSConversations_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path
		if r.URL.Path != PathSMSInbox {
			t.Errorf("Expected path %s, got %s", PathSMSInbox, r.URL.Path)
		}

		// Verify no has_unread param when not filtering
		if r.URL.Query().Get("has_unread") != "" {
			t.Errorf("Expected no has_unread param, got %s", r.URL.Query().Get("has_unread"))
		}

		conversations := []SMSConversation{
			{
				ConversationID:    "+15551234567:+15559876543",
				FromNumber:        "+15551234567",
				Phone:       "My Phone",
				PhoneNumber: "+15559876543",
				Preview:           "Latest message preview...",
				MessageCount:      5,
				UnreadCount:       2,
				LatestMessageDt:   time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC),
			},
			{
				ConversationID:    "+15552223333:+15559876543",
				FromNumber:        "+15552223333",
				Phone:       "My Phone",
				PhoneNumber: "+15559876543",
				Preview:           "Another conversation...",
				MessageCount:      10,
				UnreadCount:       0,
				LatestMessageDt:   time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conversations)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	conversations, err := client.ListSMSConversations(false)

	if err != nil {
		t.Fatalf("ListSMSConversations() error = %v", err)
	}

	if len(conversations) != 2 {
		t.Fatalf("Expected 2 conversations, got %d", len(conversations))
	}

	// Verify first conversation
	if conversations[0].ConversationID != "+15551234567:+15559876543" {
		t.Errorf("First conversation ID = %s, want '+15551234567:+15559876543'", conversations[0].ConversationID)
	}
	if conversations[0].FromNumber != "+15551234567" {
		t.Errorf("First conversation FromNumber = %s, want '+15551234567'", conversations[0].FromNumber)
	}
	if conversations[0].MessageCount != 5 {
		t.Errorf("First conversation MessageCount = %d, want 5", conversations[0].MessageCount)
	}
	if conversations[0].UnreadCount != 2 {
		t.Errorf("First conversation UnreadCount = %d, want 2", conversations[0].UnreadCount)
	}

	// Verify second conversation
	if conversations[1].UnreadCount != 0 {
		t.Errorf("Second conversation UnreadCount = %d, want 0", conversations[1].UnreadCount)
	}
}

// TestListSMSConversations_UnreadOnly verifies that ListSMSConversations filters unread conversations.
func TestListSMSConversations_UnreadOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify has_unread param is set to true
		hasUnreadParam := r.URL.Query().Get("has_unread")
		if hasUnreadParam != "true" {
			t.Errorf("Expected has_unread=true, got %s", hasUnreadParam)
		}

		conversations := []SMSConversation{
			{
				ConversationID:    "+15551234567:+15559876543",
				FromNumber:        "+15551234567",
				Phone:       "My Phone",
				PhoneNumber: "+15559876543",
				Preview:           "Unread message...",
				MessageCount:      3,
				UnreadCount:       1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conversations)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	conversations, err := client.ListSMSConversations(true)

	if err != nil {
		t.Fatalf("ListSMSConversations() error = %v", err)
	}

	if len(conversations) != 1 {
		t.Fatalf("Expected 1 conversation, got %d", len(conversations))
	}

	if conversations[0].UnreadCount == 0 {
		t.Error("Expected conversation with unread messages")
	}
}

// TestGetSMSConversation_Success verifies that GetSMSConversation returns conversation detail.
func TestGetSMSConversation_Success(t *testing.T) {
	conversationID := "+15551234567:+15559876543"
	encodedID := url.PathEscape(conversationID)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path contains encoded conversation ID
		expectedPath := PathSMSInbox + encodedID + "/"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		conversationDetail := SMSConversationDetail{
			ConversationID: conversationID,
			FromNumber:     "+15551234567",
			Phone:    "My Phone",
			MessageCount:   3,
			Messages: []SMSMessage{
				{
					ID:        1,
					Body:      "Hello!",
					Direction: "inbound",
					IsRead:    true,
					CreatedDt: time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC),
				},
				{
					ID:        2,
					Body:      "Hi there!",
					Direction: "outbound",
					IsRead:    true,
					CreatedDt: time.Date(2024, 1, 10, 9, 5, 0, 0, time.UTC),
				},
				{
					ID:        3,
					Body:      "How are you?",
					Direction: "inbound",
					IsRead:    false,
					CreatedDt: time.Date(2024, 1, 10, 9, 10, 0, 0, time.UTC),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conversationDetail)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	detail, err := client.GetSMSConversation(conversationID)

	if err != nil {
		t.Fatalf("GetSMSConversation() error = %v", err)
	}

	if detail == nil {
		t.Fatal("GetSMSConversation() returned nil")
	}

	if detail.ConversationID != conversationID {
		t.Errorf("ConversationID = %s, want %s", detail.ConversationID, conversationID)
	}
	if detail.FromNumber != "+15551234567" {
		t.Errorf("FromNumber = %s, want '+15551234567'", detail.FromNumber)
	}
	if detail.MessageCount != 3 {
		t.Errorf("MessageCount = %d, want 3", detail.MessageCount)
	}
	if len(detail.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(detail.Messages))
	}

	// Verify first message
	if detail.Messages[0].Body != "Hello!" {
		t.Errorf("First message Body = %s, want 'Hello!'", detail.Messages[0].Body)
	}
	if detail.Messages[0].Direction != "inbound" {
		t.Errorf("First message Direction = %s, want inbound", detail.Messages[0].Direction)
	}

	// Verify last message is unread
	if detail.Messages[2].IsRead != false {
		t.Errorf("Last message IsRead = %v, want false", detail.Messages[2].IsRead)
	}
}

// TestGetSMSConversation_NotFound verifies that GetSMSConversation handles 404 error.
func TestGetSMSConversation_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Conversation not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.GetSMSConversation("+15550000000:+15559999999")

	if err == nil {
		t.Fatal("GetSMSConversation() expected error for 404, got nil")
	}

	if !strings.Contains(err.Error(), "Conversation not found") {
		t.Errorf("Error should contain 'Conversation not found', got: %v", err)
	}
}

// TestGetEmailThread_URLEncoding verifies that special characters in thread ID are properly URL encoded.
func TestGetEmailThread_URLEncoding(t *testing.T) {
	testCases := []struct {
		name      string
		threadID  string
		encodedID string
	}{
		{
			name:      "thread ID with angle brackets",
			threadID:  "<CABx+y@mail.example.com>",
			encodedID: url.PathEscape("<CABx+y@mail.example.com>"),
		},
		{
			name:      "thread ID with spaces",
			threadID:  "thread with spaces@example.com",
			encodedID: url.PathEscape("thread with spaces@example.com"),
		},
		{
			name:      "simple thread ID",
			threadID:  "simple-thread-123",
			encodedID: "simple-thread-123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := PathEmailInbox + tc.encodedID + "/"
				// Use EscapedPath() to check the encoded URL path
				if r.URL.EscapedPath() != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.EscapedPath())
				}

				threadDetail := EmailThreadDetail{
					ThreadID:     tc.threadID,
					Subject:      "Test",
					MessageCount: 0,
					Messages:     []EmailMessage{},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(threadDetail)
			}))
			defer server.Close()

			client := newTestClient(server.URL)
			_, err := client.GetEmailThread(tc.threadID)

			if err != nil {
				t.Fatalf("GetEmailThread() error = %v", err)
			}
		})
	}
}

// TestGetSMSConversation_URLEncoding verifies that phone numbers with + are properly URL encoded.
func TestGetSMSConversation_URLEncoding(t *testing.T) {
	testCases := []struct {
		name           string
		conversationID string
		encodedID      string
	}{
		{
			name:           "conversation ID with plus signs",
			conversationID: "+15551234567:+15559876543",
			encodedID:      url.PathEscape("+15551234567:+15559876543"),
		},
		{
			name:           "conversation ID without plus signs",
			conversationID: "15551234567:15559876543",
			encodedID:      "15551234567:15559876543",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := PathSMSInbox + tc.encodedID + "/"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				conversationDetail := SMSConversationDetail{
					ConversationID: tc.conversationID,
					FromNumber:     "+15551234567",
					Phone:    "My Phone",
					MessageCount:   0,
					Messages:       []SMSMessage{},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(conversationDetail)
			}))
			defer server.Close()

			client := newTestClient(server.URL)
			_, err := client.GetSMSConversation(tc.conversationID)

			if err != nil {
				t.Fatalf("GetSMSConversation() error = %v", err)
			}
		})
	}
}

// TestListEmailThreads_EmptyResponse verifies handling of empty email threads list.
func TestListEmailThreads_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]EmailThread{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	threads, err := client.ListEmailThreads(false)

	if err != nil {
		t.Fatalf("ListEmailThreads() error = %v", err)
	}

	if threads == nil {
		t.Fatal("ListEmailThreads() returned nil, expected empty slice")
	}

	if len(threads) != 0 {
		t.Errorf("Expected 0 threads, got %d", len(threads))
	}
}

// TestListSMSConversations_EmptyResponse verifies handling of empty SMS conversations list.
func TestListSMSConversations_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]SMSConversation{})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	conversations, err := client.ListSMSConversations(false)

	if err != nil {
		t.Fatalf("ListSMSConversations() error = %v", err)
	}

	if conversations == nil {
		t.Fatal("ListSMSConversations() returned nil, expected empty slice")
	}

	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(conversations))
	}
}
