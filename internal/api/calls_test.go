package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendSMS_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != PathMessagesSend {
			t.Errorf("Expected path %s, got %s", PathMessagesSend, r.URL.Path)
		}
		var body SmsSendRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.ToNumber != "+14155559876" || body.Body != "hello" {
			t.Errorf("unexpected body = %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PhoneMessage{
			ID: 15, FromNumber: "+15551234567", ToNumber: "+14155559876", Body: "hello", Direction: "outgoing",
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	msg, err := client.SendSMS(SmsSendRequest{ToNumber: "+14155559876", Body: "hello"})
	if err != nil {
		t.Fatalf("SendSMS() error = %v", err)
	}
	if msg.ID != 15 || msg.ToNumber != "+14155559876" {
		t.Errorf("msg = %+v, want id 15 to +14155559876", msg)
	}
}

func TestSendSMS_ScopedByIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("identity"); got != "id-uuid-1" {
			t.Errorf("identity param = %q, want id-uuid-1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(PhoneMessage{ID: 1})
	}))
	defer server.Close()

	client := newTestClient(server.URL).WithIdentity("id-uuid-1")
	if _, err := client.SendSMS(SmsSendRequest{ToNumber: "+1", Body: "x"}); err != nil {
		t.Fatalf("SendSMS() error = %v", err)
	}
}

func TestSendSMS_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Error{Detail: "Identity is required."})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.SendSMS(SmsSendRequest{ToNumber: "+1", Body: "x"})
	if err == nil {
		t.Fatal("SendSMS() error = nil, want error")
	}
}

func TestStartCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != PathCalls+"start/" {
			t.Errorf("Expected path %sstart/, got %s", PathCalls, r.URL.Path)
		}
		var body CallStartRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.ToNumber != "+14155559876" {
			t.Errorf("to_number = %q, want +14155559876", body.ToNumber)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Call{ID: 7, ToNumber: "+14155559876", Status: "queued"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	call, err := client.StartCall(CallStartRequest{ToNumber: "+14155559876"})
	if err != nil {
		t.Fatalf("StartCall() error = %v", err)
	}
	if call.ID != 7 || call.Status != "queued" {
		t.Errorf("call = %+v, want id 7 status queued", call)
	}
}

func TestStartCall_ScopedByIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("identity"); got != "id-uuid-2" {
			t.Errorf("identity param = %q, want id-uuid-2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Call{ID: 1})
	}))
	defer server.Close()

	client := newTestClient(server.URL).WithIdentity("id-uuid-2")
	if _, err := client.StartCall(CallStartRequest{ToNumber: "+1"}); err != nil {
		t.Fatalf("StartCall() error = %v", err)
	}
}

func TestListCalls_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathCalls {
			t.Errorf("Expected path %s, got %s", PathCalls, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Call{{ID: 1}, {ID: 2}})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	calls, err := client.ListCalls()
	if err != nil {
		t.Fatalf("ListCalls() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("len = %d, want 2", len(calls))
	}
}

func TestGetCallTranscript_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/transcript/") {
			t.Errorf("Expected transcript path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]CallTranscriptSegment{
			{ID: 1, Speaker: "agent", Text: "Hello"},
			{ID: 2, Speaker: "caller", Text: "Hi"},
		})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	segments, err := client.GetCallTranscript("7")
	if err != nil {
		t.Fatalf("GetCallTranscript() error = %v", err)
	}
	if len(segments) != 2 || segments[0].Speaker != "agent" {
		t.Errorf("segments = %+v", segments)
	}
}

func TestHangupCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/hangup/") {
			t.Errorf("Expected hangup path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Call{ID: 7, Status: "completed"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	call, err := client.HangupCall("7")
	if err != nil {
		t.Fatalf("HangupCall() error = %v", err)
	}
	if call.Status != "completed" {
		t.Errorf("status = %q, want completed", call.Status)
	}
}

func TestGetCall_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Error{Detail: "Not found"})
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	if _, err := client.GetCall("999"); err == nil {
		t.Fatal("GetCall() error = nil, want error")
	}
}
