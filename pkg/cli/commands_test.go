package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/auth"
	"github.com/ravi-hq/cli/internal/config"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/ravi-hq/cli/internal/version"
)

// Force usage of imports.
var _ = filepath.Join

func TestMain(m *testing.M) {
	// Never open a real browser during tests
	auth.OpenBrowser = func(url string) error { return nil }
	os.Exit(m.Run())
}

// withAPIBaseURL temporarily overrides version.APIBaseURL.
func withAPIBaseURL(t *testing.T, url string) func() {
	t.Helper()
	original := version.APIBaseURL
	version.APIBaseURL = url
	return func() { version.APIBaseURL = original }
}

// setupCLITest sets up a temp home with a config, a mock server, and returns cleanup.
func setupCLITest(t *testing.T, handler http.Handler) (server *httptest.Server, cleanups func()) {
	t.Helper()

	tmpDir, cleanupHome := withTempHome(t)

	server = httptest.NewServer(handler)
	cleanupURL := withAPIBaseURL(t, server.URL)

	// Write a config with test keys so api.NewClient() / api.NewManagementClient() work.
	raviDir := filepath.Join(tmpDir, ".ravi")
	os.MkdirAll(raviDir, 0700)
	cfg := config.Config{
		ManagementKey: "ravi_mgmt_test",
		IdentityKey:   "ravi_id_test",
		UserEmail:     "test@example.com",
		IdentityUUID:  "test-uuid",
		IdentityName:  "Test",
	}
	data, _ := json.Marshal(cfg)
	os.WriteFile(filepath.Join(raviDir, "config.json"), data, 0600)

	// Capture output.
	output.Current = &output.JSONFormatter{}

	return server, func() {
		server.Close()
		cleanupURL()
		cleanupHome()
	}
}

// --- Get commands ---

func TestGetPhoneCmd(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.Phone{{
			ID:          1,
			PhoneNumber: "+15551234567",
			Provider:    "twilio",
		}})
	}))
	_ = server
	defer cleanup()

	err := getPhoneCmd.RunE(getPhoneCmd, nil)
	if err != nil {
		t.Fatalf("getPhoneCmd.RunE() error = %v", err)
	}
}

func TestGetEmailCmd(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.Email{{ID: 1, Email: "test@ravi.id"}})
	}))
	_ = server
	defer cleanup()

	err := getEmailCmd.RunE(getEmailCmd, nil)
	if err != nil {
		t.Fatalf("getEmailCmd.RunE() error = %v", err)
	}
}

func TestGetOwnerCmd(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.Owner{FirstName: "John", LastName: "Doe"})
	}))
	_ = server
	defer cleanup()

	err := getOwnerCmd.RunE(getOwnerCmd, nil)
	if err != nil {
		t.Fatalf("getOwnerCmd.RunE() error = %v", err)
	}
}

// --- Inbox: Email threads ---

func TestListEmailThreads_Success(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailThread{
			{
				ThreadID:        "t-1",
				Subject:         "Hello",
				FromEmail:       "sender@example.com",
				MessageCount:    2,
				UnreadCount:     1,
				LatestMessageDt: time.Now(),
			},
		})
	}))
	_ = server
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listEmailThreads(client)
	if err != nil {
		t.Fatalf("listEmailThreads() error = %v", err)
	}
}

func TestListEmailThreads_Empty(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailThread{})
	}))
	_ = server
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listEmailThreads(client)
	if err != nil {
		t.Fatalf("listEmailThreads() error = %v", err)
	}
}

func TestShowEmailThread_Success(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.EmailThreadDetail{
			ThreadID:     "t-1",
			Subject:      "Hello",
			MessageCount: 2,
			Messages: []api.EmailMessage{
				{
					ID:          1,
					FromEmail:   "sender@example.com",
					ToEmail:     "test@ravi.id",
					Subject:     "Hello",
					TextContent: "Hi there",
					Direction:   "incoming",
					IsRead:      false,
					CreatedDt:   time.Now(),
				},
				{
					ID:          2,
					FromEmail:   "test@ravi.id",
					ToEmail:     "sender@example.com",
					CC:          "cc@example.com",
					Subject:     "Re: Hello",
					TextContent: "",
					Direction:   "outgoing",
					IsRead:      true,
					CreatedDt:   time.Now(),
				},
			},
		})
	}))
	_ = server
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = showEmailThread(client, "t-1")
	if err != nil {
		t.Fatalf("showEmailThread() error = %v", err)
	}
}

// --- Inbox: SMS conversations ---

func TestListSMSConversations_Success(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SMSConversation{
			{
				ConversationID:  "c-1",
				FromNumber:      "+15559876543",
				PhoneNumber:     "+15551234567",
				Preview:         "Hey!",
				MessageCount:    3,
				UnreadCount:     1,
				LatestMessageDt: time.Now(),
			},
		})
	}))
	_ = server
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listSMSConversations(client)
	if err != nil {
		t.Fatalf("listSMSConversations() error = %v", err)
	}
}

func TestListSMSConversations_Empty(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SMSConversation{})
	}))
	_ = server
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listSMSConversations(client)
	if err != nil {
		t.Fatalf("listSMSConversations() error = %v", err)
	}
}

func TestShowSMSConversation_Success(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.SMSConversationDetail{
			ConversationID: "c-1",
			FromNumber:     "+15559876543",
			Phone:          "+15551234567",
			MessageCount:   2,
			Messages: []api.SMSMessage{
				{
					ID:        1,
					Body:      "Hello from outside",
					Direction: "incoming",
					IsRead:    false,
					CreatedDt: time.Now(),
				},
				{
					ID:        2,
					Body:      "Hello back",
					Direction: "outgoing",
					IsRead:    true,
					CreatedDt: time.Now(),
				},
			},
		})
	}))
	_ = server
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = showSMSConversation(client, "c-1")
	if err != nil {
		t.Fatalf("showSMSConversation() error = %v", err)
	}
}

// --- Message commands ---

func TestMessageSMSCmd_List(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.PhoneMessage{
			{ID: 1, Body: "test sms", FromNumber: "+1555", ToNumber: "+1666", Direction: "incoming"},
		})
	}))
	defer cleanup()

	err := messageSMSCmd.RunE(messageSMSCmd, nil)
	if err != nil {
		t.Fatalf("messageSMSCmd.RunE() error = %v", err)
	}
}

func TestMessageSMSCmd_Get(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.PhoneMessage{ID: 42, Body: "specific sms"})
	}))
	defer cleanup()

	err := messageSMSCmd.RunE(messageSMSCmd, []string{"42"})
	if err != nil {
		t.Fatalf("messageSMSCmd.RunE() error = %v", err)
	}
}

func TestMessageEmailCmd_List(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailMessageDetail{
			{ID: 1, Subject: "test email"},
		})
	}))
	defer cleanup()

	err := messageEmailCmd.RunE(messageEmailCmd, nil)
	if err != nil {
		t.Fatalf("messageEmailCmd.RunE() error = %v", err)
	}
}

func TestMessageEmailCmd_Get(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 42, Subject: "specific email"})
	}))
	defer cleanup()

	err := messageEmailCmd.RunE(messageEmailCmd, []string{"42"})
	if err != nil {
		t.Fatalf("messageEmailCmd.RunE() error = %v", err)
	}
}

// --- Identity commands ---

func TestIdentityListCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.Identity{
			{UUID: "id-1", Name: "Personal", Email: "personal@ravi.id"},
		})
	}))
	defer cleanup()

	err := identityListCmd.RunE(identityListCmd, nil)
	if err != nil {
		t.Fatalf("identityListCmd.RunE() error = %v", err)
	}
}

func TestIdentityCreateCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.Identity{UUID: "new-id", Name: "NewIdentity", Email: "new@ravi.id"})
	}))
	defer cleanup()

	identityNameFlag = "NewIdentity"
	identityEmailFlag = ""
	defer func() { identityNameFlag = ""; identityEmailFlag = "" }()

	err := identityCreateCmd.RunE(identityCreateCmd, nil)
	if err != nil {
		t.Fatalf("identityCreateCmd.RunE() error = %v", err)
	}
}

func TestIdentityUseCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/identities/":
			json.NewEncoder(w).Encode([]api.Identity{
				{UUID: "target-uuid", Name: "Target", Email: "target@ravi.id"},
			})
		case "/api/auth/keys/identity/":
			json.NewEncoder(w).Encode(api.CreateIdentityKeyResponse{
				Key:          "ravi_id_switched",
				IdentityUUID: "target-uuid",
				Label:        "cli",
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer cleanup()

	err := identityUseCmd.RunE(identityUseCmd, []string{"target-uuid"})
	if err != nil {
		t.Fatalf("identityUseCmd.RunE() error = %v", err)
	}

	cfg, loadErr := config.LoadConfig()
	if loadErr != nil {
		t.Fatalf("LoadConfig() error = %v", loadErr)
	}
	if cfg.IdentityKey != "ravi_id_switched" {
		t.Errorf("IdentityKey = %q, want ravi_id_switched", cfg.IdentityKey)
	}
}

func TestIdentityUseCmd_NotFound(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.Identity{
			{UUID: "other-uuid", Name: "Other"},
		})
	}))
	defer cleanup()

	err := identityUseCmd.RunE(identityUseCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("identityUseCmd.RunE() error = nil, want error for not-found identity")
	}
}

// --- Domains command ---

func TestDomainsCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailDomain{
			{UUID: "d-1", Domain: "ravi.id", IsPlatform: true},
		})
	}))
	defer cleanup()

	err := domainsCmd.RunE(domainsCmd, nil)
	if err != nil {
		t.Fatalf("domainsCmd.RunE() error = %v", err)
	}
}

// --- Contacts commands ---

func TestContactsListCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{
			{UUID: "c-1", Email: "alice@example.com", DisplayName: "Alice", IsTrusted: true, Source: "manual"},
		})
	}))
	defer cleanup()

	// Test with JSON output.
	humanOutput = false
	defer func() { humanOutput = false }()

	err := ctListCmd.RunE(ctListCmd, nil)
	if err != nil {
		t.Fatalf("ctListCmd.RunE() error = %v", err)
	}
}

func TestContactsListCmd_Table(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{
			{UUID: "c-1", Email: "alice@example.com", DisplayName: "Alice"},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := ctListCmd.RunE(ctListCmd, nil)
	if err != nil {
		t.Fatalf("ctListCmd.RunE() error = %v", err)
	}
}

func TestContactsListCmd_Empty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{})
	}))
	defer cleanup()

	humanOutput = false
	err := ctListCmd.RunE(ctListCmd, nil)
	if err != nil {
		t.Fatalf("ctListCmd.RunE() error = %v", err)
	}
}

func TestContactsGetCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{
			UUID: "c-1", Email: "alice@example.com", DisplayName: "Alice",
			Nickname: "Ali", IsTrusted: true, Source: "manual", CreatedDt: "2026-01-01",
		})
	}))
	defer cleanup()

	// Test human format.
	humanOutput = false
	err := ctGetCmd.RunE(ctGetCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("ctGetCmd.RunE() error = %v", err)
	}
}

func TestContactsGetCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{UUID: "c-1", Email: "alice@example.com"})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := ctGetCmd.RunE(ctGetCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("ctGetCmd.RunE() error = %v", err)
	}
}

func TestContactsCreateCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.ContactEntry{UUID: "new-c", Email: "new@example.com"})
	}))
	defer cleanup()

	ctEmail = "new@example.com"
	ctDisplayName = "New"
	defer func() { ctEmail = ""; ctDisplayName = "" }()

	humanOutput = false
	err := ctCreateCmd.RunE(ctCreateCmd, nil)
	if err != nil {
		t.Fatalf("ctCreateCmd.RunE() error = %v", err)
	}
}

func TestContactsCreateCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.ContactEntry{UUID: "new-c"})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := ctCreateCmd.RunE(ctCreateCmd, nil)
	if err != nil {
		t.Fatalf("ctCreateCmd.RunE() error = %v", err)
	}
}

func TestContactsDeleteCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = false
	err := ctDeleteCmd.RunE(ctDeleteCmd, []string{"del-c"})
	if err != nil {
		t.Fatalf("ctDeleteCmd.RunE() error = %v", err)
	}
}

func TestContactsDeleteCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := ctDeleteCmd.RunE(ctDeleteCmd, []string{"del-c"})
	if err != nil {
		t.Fatalf("ctDeleteCmd.RunE() error = %v", err)
	}
}

func TestContactsSearchCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{
			{UUID: "c-1", Email: "alice@example.com"},
		})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := ctSearchCmd.RunE(ctSearchCmd, []string{"alice"})
	if err != nil {
		t.Fatalf("ctSearchCmd.RunE() error = %v", err)
	}
}

func TestContactsSearchCmd_Table(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{
			{UUID: "c-1", Email: "alice@example.com"},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := ctSearchCmd.RunE(ctSearchCmd, []string{"alice"})
	if err != nil {
		t.Fatalf("ctSearchCmd.RunE() error = %v", err)
	}
}

func TestContactsSearchCmd_Empty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{})
	}))
	defer cleanup()

	humanOutput = false
	err := ctSearchCmd.RunE(ctSearchCmd, []string{"nobody"})
	if err != nil {
		t.Fatalf("ctSearchCmd.RunE() error = %v", err)
	}
}

// --- Passwords commands ---

func TestPasswordsListCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.PasswordEntry{
			{UUID: "p-1", Domain: "example.com", Username: "user"},
		})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := pwListCmd.RunE(pwListCmd, nil)
	if err != nil {
		t.Fatalf("pwListCmd.RunE() error = %v", err)
	}
}

func TestPasswordsListCmd_Table(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.PasswordEntry{
			{UUID: "p-1", Domain: "example.com", Username: "user"},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := pwListCmd.RunE(pwListCmd, nil)
	if err != nil {
		t.Fatalf("pwListCmd.RunE() error = %v", err)
	}
}

func TestPasswordsListCmd_Empty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.PasswordEntry{})
	}))
	defer cleanup()

	humanOutput = false
	err := pwListCmd.RunE(pwListCmd, nil)
	if err != nil {
		t.Fatalf("pwListCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGetCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.PasswordEntry{
			UUID: "p-1", Domain: "example.com", Username: "user", Password: "pass",
			Notes: "some notes", CreatedDt: "2026-01-01",
		})
	}))
	defer cleanup()

	humanOutput = false
	err := pwGetCmd.RunE(pwGetCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwGetCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGetCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.PasswordEntry{UUID: "p-1", Domain: "example.com"})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := pwGetCmd.RunE(pwGetCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwGetCmd.RunE() error = %v", err)
	}
}

func TestPasswordsCreateCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/passwords/generate-password/":
			json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "gen-pass-123"})
		default:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.PasswordEntry{UUID: "new-p", Domain: "example.com"})
		}
	}))
	defer cleanup()

	pwUsername = "user"
	pwPassword = "pass123"
	humanOutput = false
	defer func() { pwUsername = ""; pwPassword = "" }()

	err := pwCreateCmd.RunE(pwCreateCmd, []string{"example.com"})
	if err != nil {
		t.Fatalf("pwCreateCmd.RunE() error = %v", err)
	}
}

func TestPasswordsCreateCmd_AutoGenerate(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/passwords/generate-password/":
			json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "auto-gen-pw"})
		default:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.PasswordEntry{UUID: "new-p", Domain: "auto.com"})
		}
	}))
	defer cleanup()

	pwPassword = "" // empty triggers auto-generate
	pwUsername = "auto-user"
	humanOutput = false
	defer func() { pwPassword = ""; pwUsername = "" }()

	err := pwCreateCmd.RunE(pwCreateCmd, []string{"auto.com"})
	if err != nil {
		t.Fatalf("pwCreateCmd.RunE() error = %v", err)
	}
}

func TestPasswordsDeleteCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = false
	err := pwDeleteCmd.RunE(pwDeleteCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwDeleteCmd.RunE() error = %v", err)
	}
}

func TestPasswordsDeleteCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := pwDeleteCmd.RunE(pwDeleteCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwDeleteCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGenerateCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "s3cur3p@ss"})
	}))
	defer cleanup()

	humanOutput = false
	err := pwGenerateCmd.RunE(pwGenerateCmd, nil)
	if err != nil {
		t.Fatalf("pwGenerateCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGenerateCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "s3cur3p@ss"})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := pwGenerateCmd.RunE(pwGenerateCmd, nil)
	if err != nil {
		t.Fatalf("pwGenerateCmd.RunE() error = %v", err)
	}
}

// --- Secrets commands ---

func TestSecretsListCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "e2e::abc"},
		})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := secretListCmd.RunE(secretListCmd, nil)
	if err != nil {
		t.Fatalf("secretListCmd.RunE() error = %v", err)
	}
}

func TestSecretsListCmd_Table(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "e2e::abc"},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := secretListCmd.RunE(secretListCmd, nil)
	if err != nil {
		t.Fatalf("secretListCmd.RunE() error = %v", err)
	}
}

func TestSecretsListCmd_Empty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{})
	}))
	defer cleanup()

	humanOutput = false
	err := secretListCmd.RunE(secretListCmd, nil)
	if err != nil {
		t.Fatalf("secretListCmd.RunE() error = %v", err)
	}
}

func TestSecretsGetCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// GetSecret returns a list filtered by key.
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "e2e::abc"},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := secretGetCmd.RunE(secretGetCmd, []string{"API_KEY"})
	if err != nil {
		t.Fatalf("secretGetCmd.RunE() error = %v", err)
	}
}

func TestSecretsGetCmd_NotFound(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{})
	}))
	defer cleanup()

	err := secretGetCmd.RunE(secretGetCmd, []string{"NONEXISTENT"})
	if err == nil {
		t.Fatal("secretGetCmd.RunE() error = nil, want error for not-found key")
	}
}

func TestSecretsSetCmd_Create(t *testing.T) {
	callCount := 0
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			// GetSecret returns empty list (key doesn't exist yet).
			json.NewEncoder(w).Encode([]api.SecretEntry{})
		} else {
			// CreateSecret
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.SecretEntry{UUID: "new-s", Key: "NEW_KEY", Value: "secret-value"})
		}
	}))
	defer cleanup()

	humanOutput = false
	err := secretSetCmd.RunE(secretSetCmd, []string{"NEW_KEY", "secret-value"})
	if err != nil {
		t.Fatalf("secretSetCmd.RunE() error = %v", err)
	}
}

func TestSecretsSetCmd_Update(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			// GetSecret returns existing entry.
			json.NewEncoder(w).Encode([]api.SecretEntry{
				{UUID: "existing-s", Key: "EXISTING_KEY", Value: "old-value"},
			})
		} else {
			// UpdateSecret
			json.NewEncoder(w).Encode(api.SecretEntry{UUID: "existing-s", Key: "EXISTING_KEY", Value: "new-value"})
		}
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := secretSetCmd.RunE(secretSetCmd, []string{"EXISTING_KEY", "new-value"})
	if err != nil {
		t.Fatalf("secretSetCmd.RunE() error = %v", err)
	}
}

func TestSecretsDeleteCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = false
	err := secretDeleteCmd.RunE(secretDeleteCmd, []string{"s-1"})
	if err != nil {
		t.Fatalf("secretDeleteCmd.RunE() error = %v", err)
	}
}

func TestSecretsDeleteCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := secretDeleteCmd.RunE(secretDeleteCmd, []string{"s-1"})
	if err != nil {
		t.Fatalf("secretDeleteCmd.RunE() error = %v", err)
	}
}

// --- Feedback command ---

func TestFeedbackCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/email/" && r.Method == "GET":
			json.NewEncoder(w).Encode([]api.Email{{ID: 1, Email: "test@ravi.id"}})
		default:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 1, Subject: "Feedback"})
		}
	}))
	defer cleanup()

	err := feedbackCmd.RunE(feedbackCmd, []string{"Great product!"})
	if err != nil {
		t.Fatalf("feedbackCmd.RunE() error = %v", err)
	}
}

// --- uploadAttachments ---

func TestUploadAttachments_EmptyPaths(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	uuids, err := uploadAttachments(client, nil)
	if err != nil {
		t.Fatalf("uploadAttachments() error = %v", err)
	}
	if uuids != nil {
		t.Errorf("Expected nil, got %v", uuids)
	}
}

func TestUploadAttachments_FileError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = uploadAttachments(client, []string{"/nonexistent/file.txt"})
	if err == nil {
		t.Fatal("uploadAttachments() error = nil, want error for missing file")
	}
}

// --- Execute / root ---

func TestRootCmd_SubcommandRegistration(t *testing.T) {
	// Verify that key subcommands are registered on rootCmd.
	subNames := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		subNames[cmd.Name()] = true
	}

	expected := []string{"get", "inbox", "message", "passwords", "secrets", "contacts", "identity", "domains", "auth", "email", "feedback", "sso"}
	for _, name := range expected {
		if !subNames[name] {
			t.Errorf("rootCmd missing subcommand %q", name)
		}
	}
}

// --- Auth commands ---

func TestLoginCmd(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/auth/device/":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"device_code":      "test-device-code",
				"user_code":        "TEST-1234",
				"verification_uri": "http://127.0.0.1:0/verify",
				"expires_in":       300,
				"interval":         0,
			})
		case "/api/auth/device/token/":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"management_key": "ravi_mgmt_login",
				"identity_key":   "ravi_id_login",
				"identity":       map[string]interface{}{"uuid": "id-1", "name": "Test"},
				"user":           map[string]interface{}{"email": "test@example.com"},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	_ = server
	defer cleanup()

	err := loginCmd.RunE(loginCmd, nil)
	if err != nil {
		t.Fatalf("loginCmd.RunE() error = %v", err)
	}
}

func TestLogoutCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cleanup()

	err := logoutCmd.RunE(logoutCmd, nil)
	if err != nil {
		t.Fatalf("logoutCmd.RunE() error = %v", err)
	}
}

func TestStatusCmd_Authenticated(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cleanup()

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("statusCmd.RunE() error = %v", err)
	}
}

func TestStatusCmd_NotAuthenticated(t *testing.T) {
	tmpDir, cleanupHome := withTempHome(t)
	cleanupURL := withAPIBaseURL(t, "http://localhost")
	defer func() { cleanupURL(); cleanupHome() }()

	// Write empty config.
	raviDir := filepath.Join(tmpDir, ".ravi")
	os.MkdirAll(raviDir, 0700)
	os.WriteFile(filepath.Join(raviDir, "config.json"), []byte(`{}`), 0600)

	output.Current = &output.JSONFormatter{}

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("statusCmd.RunE() error = %v", err)
	}
}

// --- Email inbox command (top-level RunE) ---

func TestEmailCmd_ListThreads(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailThread{
			{
				ThreadID:        "t-1",
				Subject:         "Hello",
				FromEmail:       "sender@example.com",
				MessageCount:    1,
				LatestMessageDt: time.Now(),
			},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := emailCmd.RunE(emailCmd, nil)
	if err != nil {
		t.Fatalf("emailCmd.RunE(nil) error = %v", err)
	}
}

func TestEmailCmd_ShowThread(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.EmailThreadDetail{
			ThreadID:     "t-1",
			Subject:      "Hello",
			MessageCount: 1,
			Messages:     []api.EmailMessage{{ID: 1, TextContent: "Hi", Direction: "incoming", CreatedDt: time.Now()}},
		})
	}))
	defer cleanup()

	err := emailCmd.RunE(emailCmd, []string{"t-1"})
	if err != nil {
		t.Fatalf("emailCmd.RunE(t-1) error = %v", err)
	}
}

func TestEmailCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailThread{})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := emailCmd.RunE(emailCmd, nil)
	if err != nil {
		t.Fatalf("emailCmd.RunE() error = %v", err)
	}
}

func TestEmailCmd_ThreadJSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.EmailThreadDetail{
			ThreadID: "t-1",
			Messages: []api.EmailMessage{},
		})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := emailCmd.RunE(emailCmd, []string{"t-1"})
	if err != nil {
		t.Fatalf("emailCmd.RunE(t-1) error = %v", err)
	}
}

// --- SMS inbox command (top-level RunE) ---

func TestSMSCmd_ListConversations(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SMSConversation{
			{
				ConversationID:  "c-1",
				FromNumber:      "+1555",
				PhoneNumber:     "+1666",
				MessageCount:    1,
				LatestMessageDt: time.Now(),
			},
		})
	}))
	defer cleanup()

	humanOutput = false
	err := smsCmd.RunE(smsCmd, nil)
	if err != nil {
		t.Fatalf("smsCmd.RunE(nil) error = %v", err)
	}
}

func TestSMSCmd_ShowConversation(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.SMSConversationDetail{
			ConversationID: "c-1",
			FromNumber:     "+1555",
			Phone:          "+1666",
			MessageCount:   1,
			Messages:       []api.SMSMessage{{ID: 1, Body: "Hi", Direction: "incoming", CreatedDt: time.Now()}},
		})
	}))
	defer cleanup()

	err := smsCmd.RunE(smsCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("smsCmd.RunE(c-1) error = %v", err)
	}
}

func TestSMSCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SMSConversation{})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := smsCmd.RunE(smsCmd, nil)
	if err != nil {
		t.Fatalf("smsCmd.RunE() error = %v", err)
	}
}

func TestSMSCmd_ConversationJSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.SMSConversationDetail{
			ConversationID: "c-1",
			Messages:       []api.SMSMessage{},
		})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := smsCmd.RunE(smsCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("smsCmd.RunE(c-1) error = %v", err)
	}
}

// --- Compose, Reply, ReplyAll, Forward commands ---

func TestComposeCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/email/" && r.Method == "GET":
			json.NewEncoder(w).Encode([]api.Email{{ID: 1, Email: "test@ravi.id"}})
		default:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 1, Subject: "Test"})
		}
	}))
	defer cleanup()

	composeCmd.Flags().Set("to", "user@example.com")
	composeCmd.Flags().Set("subject", "Test")
	composeCmd.Flags().Set("body", "<p>Hello</p>")

	err := composeCmd.RunE(composeCmd, nil)
	if err != nil {
		t.Fatalf("composeCmd.RunE() error = %v", err)
	}
}

func TestComposeCmd_WithCC(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/email/" && r.Method == "GET":
			json.NewEncoder(w).Encode([]api.Email{{ID: 1, Email: "test@ravi.id"}})
		default:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 1})
		}
	}))
	defer cleanup()

	composeCmd.Flags().Set("to", "user@example.com")
	composeCmd.Flags().Set("subject", "Test")
	composeCmd.Flags().Set("body", "body")
	composeCmd.Flags().Set("cc", "cc@example.com")
	composeCmd.Flags().Set("bcc", "bcc@example.com")

	err := composeCmd.RunE(composeCmd, nil)
	if err != nil {
		t.Fatalf("composeCmd.RunE() error = %v", err)
	}
}

func TestReplyCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 2})
	}))
	defer cleanup()

	replyCmd.Flags().Set("body", "reply body")

	err := replyCmd.RunE(replyCmd, []string{"123"})
	if err != nil {
		t.Fatalf("replyCmd.RunE() error = %v", err)
	}
}

func TestReplyAllCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 3})
	}))
	defer cleanup()

	replyAllCmd.Flags().Set("body", "reply all body")

	err := replyAllCmd.RunE(replyAllCmd, []string{"456"})
	if err != nil {
		t.Fatalf("replyAllCmd.RunE() error = %v", err)
	}
}

func TestForwardCmd(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 4})
	}))
	defer cleanup()

	forwardCmd.Flags().Set("to", "fwd@example.com")
	forwardCmd.Flags().Set("body", "fwd body")

	err := forwardCmd.RunE(forwardCmd, []string{"789"})
	if err != nil {
		t.Fatalf("forwardCmd.RunE() error = %v", err)
	}
}

func TestReplyCmd_WithCC(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 2})
	}))
	defer cleanup()

	replyCmd.Flags().Set("body", "reply body")
	replyCmd.Flags().Set("cc", "cc1@example.com, cc2@example.com")
	replyCmd.Flags().Set("bcc", "bcc@example.com")

	err := replyCmd.RunE(replyCmd, []string{"123"})
	if err != nil {
		t.Fatalf("replyCmd.RunE() error = %v", err)
	}
}

func TestForwardCmd_WithCC(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(api.EmailMessageDetail{ID: 4})
	}))
	defer cleanup()

	forwardCmd.Flags().Set("to", "fwd@example.com")
	forwardCmd.Flags().Set("body", "fwd body")
	forwardCmd.Flags().Set("cc", "cc@example.com")
	forwardCmd.Flags().Set("bcc", "bcc@example.com")

	err := forwardCmd.RunE(forwardCmd, []string{"789"})
	if err != nil {
		t.Fatalf("forwardCmd.RunE() error = %v", err)
	}
}

// --- Secrets get JSON format ---

func TestSecretsGetCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "e2e::abc", Notes: "some notes"},
		})
	}))
	defer cleanup()

	humanOutput = false
	defer func() { humanOutput = false }()

	err := secretGetCmd.RunE(secretGetCmd, []string{"API_KEY"})
	if err != nil {
		t.Fatalf("secretGetCmd.RunE() error = %v", err)
	}
}

// --- Password edit command ---

func TestPasswordsEditCmd_NoFields(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cleanup()

	// Since no flags are Changed(), this should return an error.
	err := pwEditCmd.RunE(pwEditCmd, []string{"some-uuid"})
	if err == nil {
		t.Fatal("pwEditCmd.RunE() error = nil, want error for no fields")
	}
}

// --- API client error tests (covering all NewClient() error branches) ---
// When version.APIBaseURL is empty, api.NewClient() and api.NewManagementClient() fail.
// This covers the `if err != nil { return err }` branches in every command.

func TestCommands_ClientError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	// Set APIBaseURL to empty to trigger NewClient() error.
	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	commands := []struct {
		name string
		fn   func() error
	}{
		{"ctListCmd", func() error { return ctListCmd.RunE(ctListCmd, nil) }},
		{"ctSearchCmd", func() error { return ctSearchCmd.RunE(ctSearchCmd, []string{"q"}) }},
		{"ctGetCmd", func() error { return ctGetCmd.RunE(ctGetCmd, []string{"uuid"}) }},
		{"ctCreateCmd", func() error { return ctCreateCmd.RunE(ctCreateCmd, nil) }},
		{"ctEditCmd", func() error { return ctEditCmd.RunE(ctEditCmd, []string{"uuid"}) }},
		{"ctDeleteCmd", func() error { return ctDeleteCmd.RunE(ctDeleteCmd, []string{"uuid"}) }},
		{"getPhoneCmd", func() error { return getPhoneCmd.RunE(getPhoneCmd, nil) }},
		{"getEmailCmd", func() error { return getEmailCmd.RunE(getEmailCmd, nil) }},
		{"getOwnerCmd", func() error { return getOwnerCmd.RunE(getOwnerCmd, nil) }},
		{"messageSMSCmd", func() error { return messageSMSCmd.RunE(messageSMSCmd, nil) }},
		{"messageEmailCmd", func() error { return messageEmailCmd.RunE(messageEmailCmd, nil) }},
		{"pwListCmd", func() error { return pwListCmd.RunE(pwListCmd, nil) }},
		{"pwGetCmd", func() error { return pwGetCmd.RunE(pwGetCmd, []string{"uuid"}) }},
		{"pwCreateCmd", func() error { return pwCreateCmd.RunE(pwCreateCmd, []string{"domain"}) }},
		{"pwDeleteCmd", func() error { return pwDeleteCmd.RunE(pwDeleteCmd, []string{"uuid"}) }},
		{"pwGenerateCmd", func() error { return pwGenerateCmd.RunE(pwGenerateCmd, nil) }},
		{"secretListCmd", func() error { return secretListCmd.RunE(secretListCmd, nil) }},
		{"secretGetCmd", func() error { return secretGetCmd.RunE(secretGetCmd, []string{"key"}) }},
		{"secretSetCmd", func() error { return secretSetCmd.RunE(secretSetCmd, []string{"k", "v"}) }},
		{"secretDeleteCmd", func() error { return secretDeleteCmd.RunE(secretDeleteCmd, []string{"uuid"}) }},
		{"emailCmd_list", func() error { return emailCmd.RunE(emailCmd, nil) }},
		{"smsCmd_list", func() error { return smsCmd.RunE(smsCmd, nil) }},
		{"domainsCmd", func() error { return domainsCmd.RunE(domainsCmd, nil) }},
		{"identityListCmd", func() error { return identityListCmd.RunE(identityListCmd, nil) }},
		{"identityCreateCmd", func() error { return identityCreateCmd.RunE(identityCreateCmd, nil) }},
		{"identityUseCmd", func() error { return identityUseCmd.RunE(identityUseCmd, []string{"uuid"}) }},
		{"feedbackCmd", func() error { return feedbackCmd.RunE(feedbackCmd, []string{"msg"}) }},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Errorf("%s: expected error when API URL not configured, got nil", tc.name)
			}
		})
	}
}

// --- Compose/Reply/ReplyAll/Forward client error ---

func TestEmailSendCommands_ClientError(t *testing.T) {
	_, cleanupHome := withTempHome(t)
	defer cleanupHome()

	cleanupURL := withAPIBaseURL(t, "")
	defer cleanupURL()

	commands := []struct {
		name string
		fn   func() error
	}{
		{"composeCmd", func() error { return composeCmd.RunE(composeCmd, nil) }},
		{"replyCmd", func() error { return replyCmd.RunE(replyCmd, []string{"123"}) }},
		{"replyAllCmd", func() error { return replyAllCmd.RunE(replyAllCmd, []string{"123"}) }},
		{"forwardCmd", func() error { return forwardCmd.RunE(forwardCmd, []string{"123"}) }},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if err == nil {
				t.Errorf("%s: expected error when API URL not configured, got nil", tc.name)
			}
		})
	}
}

// --- Contacts update with actual fields ---

func TestContactsEditCmd_WithFields(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{UUID: "up-c", Email: "updated@example.com"})
	}))
	defer cleanup()

	// Mark the "email" flag as changed.
	ctEditCmd.Flags().Set("email", "updated@example.com")
	humanOutput = false

	err := ctEditCmd.RunE(ctEditCmd, []string{"up-c"})
	if err != nil {
		t.Fatalf("ctEditCmd.RunE() error = %v", err)
	}
}

func TestContactsEditCmd_JSON(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{UUID: "up-c"})
	}))
	defer cleanup()

	ctEditCmd.Flags().Set("phone", "+1555")
	humanOutput = false
	defer func() { humanOutput = false }()

	err := ctEditCmd.RunE(ctEditCmd, []string{"up-c"})
	if err != nil {
		t.Fatalf("ctEditCmd.RunE() error = %v", err)
	}
}

// --- API call failure tests (server returns errors) ---

func TestCommands_APIError(t *testing.T) {
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "server error"})
	})

	// Each test covers the API call error branch within the command.
	tests := []struct {
		name string
		fn   func() error
	}{
		{"getPhoneCmd", func() error { return getPhoneCmd.RunE(getPhoneCmd, nil) }},
		{"getEmailCmd", func() error { return getEmailCmd.RunE(getEmailCmd, nil) }},
		{"getOwnerCmd", func() error { return getOwnerCmd.RunE(getOwnerCmd, nil) }},
		{"messageSMSCmd_list", func() error { return messageSMSCmd.RunE(messageSMSCmd, nil) }},
		{"messageSMSCmd_get", func() error { return messageSMSCmd.RunE(messageSMSCmd, []string{"1"}) }},
		{"messageEmailCmd_list", func() error { return messageEmailCmd.RunE(messageEmailCmd, nil) }},
		{"messageEmailCmd_get", func() error { return messageEmailCmd.RunE(messageEmailCmd, []string{"1"}) }},
		{"identityListCmd", func() error { return identityListCmd.RunE(identityListCmd, nil) }},
		{"identityCreateCmd", func() error { return identityCreateCmd.RunE(identityCreateCmd, nil) }},
		{"domainsCmd", func() error { return domainsCmd.RunE(domainsCmd, nil) }},
		{"emailCmd_list", func() error { return emailCmd.RunE(emailCmd, nil) }},
		{"emailCmd_thread", func() error { return emailCmd.RunE(emailCmd, []string{"t-1"}) }},
		{"smsCmd_list", func() error { return smsCmd.RunE(smsCmd, nil) }},
		{"smsCmd_conv", func() error { return smsCmd.RunE(smsCmd, []string{"c-1"}) }},
		{"pwListCmd", func() error { return pwListCmd.RunE(pwListCmd, nil) }},
		{"pwGetCmd", func() error { return pwGetCmd.RunE(pwGetCmd, []string{"uuid"}) }},
		{"pwDeleteCmd", func() error { return pwDeleteCmd.RunE(pwDeleteCmd, []string{"uuid"}) }},
		{"pwGenerateCmd", func() error { return pwGenerateCmd.RunE(pwGenerateCmd, nil) }},
		{"secretListCmd", func() error { return secretListCmd.RunE(secretListCmd, nil) }},
		{"secretGetCmd", func() error { return secretGetCmd.RunE(secretGetCmd, []string{"key"}) }},
		{"secretSetCmd", func() error { return secretSetCmd.RunE(secretSetCmd, []string{"k", "v"}) }},
		{"secretDeleteCmd", func() error { return secretDeleteCmd.RunE(secretDeleteCmd, []string{"uuid"}) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, cleanup := setupCLITest(t, errorHandler)
			defer cleanup()

			err := tc.fn()
			if err == nil {
				t.Errorf("%s: expected error from API, got nil", tc.name)
			}
		})
	}
}

// --- Compose/feedback error paths ---

func TestFeedbackCmd_InboxError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	err := feedbackCmd.RunE(feedbackCmd, []string{"msg"})
	if err == nil {
		t.Fatal("feedbackCmd.RunE() expected error, got nil")
	}
}

func TestComposeCmd_InboxError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	composeCmd.Flags().Set("to", "test@example.com")
	composeCmd.Flags().Set("subject", "Test")
	composeCmd.Flags().Set("body", "body")

	err := composeCmd.RunE(composeCmd, nil)
	if err == nil {
		t.Fatal("composeCmd.RunE() expected error, got nil")
	}
}

func TestReplyCmd_Error(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	replyCmd.Flags().Set("body", "body")

	err := replyCmd.RunE(replyCmd, []string{"123"})
	if err == nil {
		t.Fatal("replyCmd.RunE() expected error, got nil")
	}
}

func TestReplyAllCmd_Error(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	replyAllCmd.Flags().Set("body", "body")

	err := replyAllCmd.RunE(replyAllCmd, []string{"123"})
	if err == nil {
		t.Fatal("replyAllCmd.RunE() expected error, got nil")
	}
}

func TestForwardCmd_Error(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	forwardCmd.Flags().Set("to", "test@example.com")
	forwardCmd.Flags().Set("body", "body")

	err := forwardCmd.RunE(forwardCmd, []string{"123"})
	if err == nil {
		t.Fatal("forwardCmd.RunE() expected error, got nil")
	}
}

// --- IdentityUseCmd error paths ---

func TestIdentityUseCmd_ListError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	err := identityUseCmd.RunE(identityUseCmd, []string{"uuid"})
	if err == nil {
		t.Fatal("identityUseCmd.RunE() expected error, got nil")
	}
}

func TestIdentityUseCmd_CreateKeyError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/identities/":
			json.NewEncoder(w).Encode([]api.Identity{{UUID: "t-uuid", Name: "Test"}})
		case "/api/auth/keys/identity/":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.Error{Detail: "key error"})
		}
	}))
	defer cleanup()

	err := identityUseCmd.RunE(identityUseCmd, []string{"t-uuid"})
	if err == nil {
		t.Fatal("identityUseCmd.RunE() expected error, got nil")
	}
}

// --- Password create with generate error ---

func TestPasswordsCreateCmd_GenerateError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
	}))
	defer cleanup()

	pwPassword = ""
	defer func() { pwPassword = "" }()

	err := pwCreateCmd.RunE(pwCreateCmd, []string{"example.com"})
	if err == nil {
		t.Fatal("pwCreateCmd.RunE() expected error for generate failure, got nil")
	}
}

func TestPasswordsCreateCmd_CreateError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/passwords/generate-password/" {
			json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "gen"})
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.Error{Detail: "create failed"})
		}
	}))
	defer cleanup()

	pwPassword = ""
	defer func() { pwPassword = "" }()

	err := pwCreateCmd.RunE(pwCreateCmd, []string{"example.com"})
	if err == nil {
		t.Fatal("pwCreateCmd.RunE() expected error for create failure, got nil")
	}
}

// --- Compose after getting inbox ---

func TestComposeCmd_ComposeError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/email/" {
			json.NewEncoder(w).Encode([]api.Email{{ID: 1, Email: "test@ravi.id"}})
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.Error{Detail: "compose failed"})
		}
	}))
	defer cleanup()

	composeCmd.Flags().Set("to", "test@example.com")
	composeCmd.Flags().Set("subject", "Test")
	composeCmd.Flags().Set("body", "body")

	err := composeCmd.RunE(composeCmd, nil)
	if err == nil {
		t.Fatal("composeCmd.RunE() expected error, got nil")
	}
}

// --- Feedback compose error ---

func TestFeedbackCmd_ComposeError(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/email/" {
			json.NewEncoder(w).Encode([]api.Email{{ID: 1, Email: "test@ravi.id"}})
		} else {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.Error{Detail: "fail"})
		}
	}))
	defer cleanup()

	err := feedbackCmd.RunE(feedbackCmd, []string{"msg"})
	if err == nil {
		t.Fatal("feedbackCmd.RunE() expected error, got nil")
	}
}

// --- Human output tests ---
// These tests exercise the humanOutput=true branches which format data as tables/text.

func TestContactsListCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{
			{UUID: "c-1", Email: "alice@example.com", PhoneNumber: "+15551234567", DisplayName: "Alice", IsTrusted: true, Source: "manual"},
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctListCmd.RunE(ctListCmd, nil)
	if err != nil {
		t.Fatalf("ctListCmd.RunE() error = %v", err)
	}
}

func TestContactsListCmd_HumanEmpty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctListCmd.RunE(ctListCmd, nil)
	if err != nil {
		t.Fatalf("ctListCmd.RunE() error = %v", err)
	}
}

func TestContactsGetCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{
			UUID: "c-1", Email: "alice@example.com", DisplayName: "Alice",
			Nickname: "Ali", IsTrusted: true, Source: "manual", CreatedDt: "2026-01-01",
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctGetCmd.RunE(ctGetCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("ctGetCmd.RunE() error = %v", err)
	}
}

func TestContactsGetCmd_HumanNoNickname(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{
			UUID: "c-1", Email: "alice@example.com", DisplayName: "Alice",
			IsTrusted: false, Source: "auto", CreatedDt: "2026-01-01",
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctGetCmd.RunE(ctGetCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("ctGetCmd.RunE() error = %v", err)
	}
}

func TestContactsCreateCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.ContactEntry{UUID: "c-new", Email: "new@example.com"})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctCreateCmd.RunE(ctCreateCmd, nil)
	if err != nil {
		t.Fatalf("ctCreateCmd.RunE() error = %v", err)
	}
}

func TestContactsDeleteCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctDeleteCmd.RunE(ctDeleteCmd, []string{"c-1"})
	if err != nil {
		t.Fatalf("ctDeleteCmd.RunE() error = %v", err)
	}
}

func TestContactsSearchCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{
			{UUID: "c-1", Email: "alice@example.com", DisplayName: "Alice", Source: "manual"},
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctSearchCmd.RunE(ctSearchCmd, []string{"alice"})
	if err != nil {
		t.Fatalf("ctSearchCmd.RunE() error = %v", err)
	}
}

func TestContactsSearchCmd_HumanEmpty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.ContactEntry{})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := ctSearchCmd.RunE(ctSearchCmd, []string{"nobody"})
	if err != nil {
		t.Fatalf("ctSearchCmd.RunE() error = %v", err)
	}
}

// --- Passwords human output ---

func TestPasswordsListCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.PasswordEntry{
			{UUID: "p-1", Domain: "example.com", Username: "admin", CreatedDt: "2026-01-01"},
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := pwListCmd.RunE(pwListCmd, nil)
	if err != nil {
		t.Fatalf("pwListCmd.RunE() error = %v", err)
	}
}

func TestPasswordsListCmd_HumanEmpty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.PasswordEntry{})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := pwListCmd.RunE(pwListCmd, nil)
	if err != nil {
		t.Fatalf("pwListCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGetCmd_HumanOutput(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.PasswordEntry{
			UUID: "p-1", Domain: "example.com", Username: "admin",
			Password: "secret123", Notes: "prod creds", CreatedDt: "2026-01-01",
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := pwGetCmd.RunE(pwGetCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwGetCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGetCmd_HumanOutputNoNotes(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.PasswordEntry{
			UUID: "p-1", Domain: "example.com", Username: "admin",
			Password: "secret123", CreatedDt: "2026-01-01",
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := pwGetCmd.RunE(pwGetCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwGetCmd.RunE() error = %v", err)
	}
}

func TestPasswordsCreateCmd_HumanOutput(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/passwords/generate/" {
			json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "gen-pass-123"})
		} else {
			json.NewEncoder(w).Encode(api.PasswordEntry{UUID: "p-new", Domain: "example.com"})
		}
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	pwPassword = "mypass"
	err := pwCreateCmd.RunE(pwCreateCmd, []string{"example.com"})
	pwPassword = ""
	if err != nil {
		t.Fatalf("pwCreateCmd.RunE() error = %v", err)
	}
}

func TestPasswordsDeleteCmd_HumanOutput(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := pwDeleteCmd.RunE(pwDeleteCmd, []string{"p-1"})
	if err != nil {
		t.Fatalf("pwDeleteCmd.RunE() error = %v", err)
	}
}

func TestPasswordsGenerateCmd_HumanOutput(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.GeneratedPassword{Password: "gen-pass-456"})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := pwGenerateCmd.RunE(pwGenerateCmd, nil)
	if err != nil {
		t.Fatalf("pwGenerateCmd.RunE() error = %v", err)
	}
}

// --- Secrets human output ---

func TestSecretsListCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "hidden", CreatedDt: "2026-01-01"},
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := secretListCmd.RunE(secretListCmd, nil)
	if err != nil {
		t.Fatalf("secretListCmd.RunE() error = %v", err)
	}
}

func TestSecretsListCmd_HumanEmpty(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := secretListCmd.RunE(secretListCmd, nil)
	if err != nil {
		t.Fatalf("secretListCmd.RunE() error = %v", err)
	}
}

func TestSecretsGetCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "secret-val", Notes: "prod key", CreatedDt: "2026-01-01"},
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := secretGetCmd.RunE(secretGetCmd, []string{"API_KEY"})
	if err != nil {
		t.Fatalf("secretGetCmd.RunE() error = %v", err)
	}
}

func TestSecretsGetCmd_HumanNoNotes(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SecretEntry{
			{UUID: "s-1", Key: "API_KEY", Value: "secret-val", CreatedDt: "2026-01-01"},
		})
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := secretGetCmd.RunE(secretGetCmd, []string{"API_KEY"})
	if err != nil {
		t.Fatalf("secretGetCmd.RunE() error = %v", err)
	}
}

func TestSecretsSetCmd_Human(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			// GetSecret returns empty list (key not found) -> create path
			json.NewEncoder(w).Encode([]api.SecretEntry{})
		} else {
			json.NewEncoder(w).Encode(api.SecretEntry{UUID: "s-new", Key: "NEW_KEY", Value: "val"})
		}
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := secretSetCmd.RunE(secretSetCmd, []string{"NEW_KEY", "val"})
	if err != nil {
		t.Fatalf("secretSetCmd.RunE() error = %v", err)
	}
}

func TestSecretsDeleteCmd_HumanOutput(t *testing.T) {
	_, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	err := secretDeleteCmd.RunE(secretDeleteCmd, []string{"s-1"})
	if err != nil {
		t.Fatalf("secretDeleteCmd.RunE() error = %v", err)
	}
}

// --- Inbox human output ---

func TestListEmailThreads_Human(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailThread{
			{
				ThreadID:        "t-1",
				Subject:         "Hello",
				FromEmail:       "sender@example.com",
				MessageCount:    2,
				UnreadCount:     1,
				LatestMessageDt: time.Now(),
			},
		})
	}))
	_ = server
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listEmailThreads(client)
	if err != nil {
		t.Fatalf("listEmailThreads() error = %v", err)
	}
}

func TestListEmailThreads_HumanEmpty(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.EmailThread{})
	}))
	_ = server
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listEmailThreads(client)
	if err != nil {
		t.Fatalf("listEmailThreads() error = %v", err)
	}
}

func TestShowEmailThread_Human(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.EmailThreadDetail{
			ThreadID:     "t-1",
			Subject:      "Hello",
			MessageCount: 2,
			Messages: []api.EmailMessage{
				{
					ID:          1,
					FromEmail:   "sender@example.com",
					ToEmail:     "test@ravi.id",
					Subject:     "Hello",
					TextContent: "Hi there",
					Direction:   "incoming",
					IsRead:      false,
					CreatedDt:   time.Now(),
				},
				{
					ID:          2,
					FromEmail:   "test@ravi.id",
					ToEmail:     "sender@example.com",
					CC:          "cc@example.com",
					Subject:     "Re: Hello",
					TextContent: "",
					Direction:   "outgoing",
					IsRead:      true,
					CreatedDt:   time.Now(),
				},
			},
		})
	}))
	_ = server
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = showEmailThread(client, "t-1")
	if err != nil {
		t.Fatalf("showEmailThread() error = %v", err)
	}
}

func TestListSMSConversations_Human(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SMSConversation{
			{
				ConversationID:  "c-1",
				FromNumber:      "+15551234567",
				PhoneNumber:     "+15559876543",
				Preview:         "Hello!",
				MessageCount:    3,
				UnreadCount:     1,
				LatestMessageDt: time.Now(),
			},
		})
	}))
	_ = server
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listSMSConversations(client)
	if err != nil {
		t.Fatalf("listSMSConversations() error = %v", err)
	}
}

func TestListSMSConversations_HumanEmpty(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]api.SMSConversation{})
	}))
	_ = server
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = listSMSConversations(client)
	if err != nil {
		t.Fatalf("listSMSConversations() error = %v", err)
	}
}

func TestShowSMSConversation_Human(t *testing.T) {
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.SMSConversationDetail{
			ConversationID: "c-1",
			FromNumber:     "+15551234567",
			Phone:          "+15559876543",
			MessageCount:   2,
			Messages: []api.SMSMessage{
				{ID: 1, Body: "Hello", Direction: "incoming", IsRead: false, CreatedDt: time.Now()},
				{ID: 2, Body: "Hi back", Direction: "outgoing", IsRead: true, CreatedDt: time.Now()},
			},
		})
	}))
	_ = server
	defer cleanup()

	humanOutput = true
	defer func() { humanOutput = false }()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = showSMSConversation(client, "c-1")
	if err != nil {
		t.Fatalf("showSMSConversation() error = %v", err)
	}
}

// --- Upload attachments success path ---

func TestUploadAttachments_Success(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("hello"), 0600)

	var serverURL string
	server, cleanup := setupCLITest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "PUT" {
			w.WriteHeader(http.StatusOK)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":       "att-123",
			"upload_url": serverURL + "/upload",
		})
	}))
	serverURL = server.URL
	defer cleanup()

	client, err := api.NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	uuids, err := uploadAttachments(client, []string{tmpFile})
	if err != nil {
		t.Fatalf("uploadAttachments() error = %v", err)
	}
	if len(uuids) != 1 {
		t.Fatalf("uploadAttachments() returned %d uuids, want 1", len(uuids))
	}
}

// --- Execute and PersistentPreRun ---

func TestExecute(t *testing.T) {
	// Execute with --help to avoid actually running a command that needs auth.
	rootCmd.SetArgs([]string{"--help"})
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestPersistentPreRun(t *testing.T) {
	// Trigger PersistentPreRun by calling it directly.
	humanOutput = true
	rootCmd.PersistentPreRun(rootCmd, nil)

	// After PersistentPreRun with humanOutput=true, SetJSON(false) is called,
	// so Current should be HumanFormatter.
	humanOutput = false
	rootCmd.PersistentPreRun(rootCmd, nil)
}

// Suppress unused import warnings - these are used by setupCLITest.
var _ = fmt.Sprintf
var _ = runtime.GOOS
var _ = os.Stdin
