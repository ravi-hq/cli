package api

import "time"

// DeviceCodeRequest represents the request body for initiating the OAuth device code flow.
// It is empty as no parameters are required to start the flow.
type DeviceCodeRequest struct{}

// DeviceCodeResponse contains the device code and user code returned by the server
// when initiating the OAuth device code flow. The user must visit VerificationURI
// and enter the UserCode to authorize the device.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceTokenRequest represents the polling request to exchange a device code
// for access and refresh tokens after the user has authorized the device.
type DeviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

// DeviceTokenResponse contains the access token, refresh token, and user information
// returned after successful device authorization.
type DeviceTokenResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
	User    User   `json:"user"`
}

// DeviceTokenError represents an error response during device token polling,
// typically indicating the user has not yet authorized or the request was denied.
type DeviceTokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// User represents the authenticated user's profile information
// returned after successful authentication.
type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// RefreshRequest represents the request body for refreshing an expired access token
// using a valid refresh token.
type RefreshRequest struct {
	Refresh string `json:"refresh"`
}

// RefreshResponse contains the new access token (and optionally a rotated
// refresh token) returned after a successful token refresh operation.
type RefreshResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh,omitempty"`
}

// EmailThread represents an email conversation thread summary from the /api/email-inbox/ endpoint.
// It contains metadata about the thread including message counts and timestamps.
type EmailThread struct {
	ThreadID        string    `json:"thread_id"`
	Subject         string    `json:"subject"`
	Preview         string    `json:"preview"`
	FromEmail       string    `json:"from_email"`
	Email           string    `json:"inbox"`
	MessageCount    int       `json:"message_count"`
	UnreadCount     int       `json:"unread_count"`
	LatestMessageDt time.Time `json:"latest_message_dt"`
	OldestMessageDt time.Time `json:"oldest_message_dt"`
}

// EmailThreadDetail represents a complete email thread with all its messages,
// returned when viewing a specific thread by ID.
type EmailThreadDetail struct {
	ThreadID     string         `json:"thread_id"`
	Subject      string         `json:"subject"`
	MessageCount int            `json:"message_count"`
	Messages     []EmailMessage `json:"messages"`
}

// EmailMessage represents a single email within a thread, containing the full
// email content including text and HTML versions.
type EmailMessage struct {
	ID          int       `json:"id"`
	FromEmail   string    `json:"from_email"`
	ToEmail     string    `json:"to_email"`
	CC          string    `json:"cc"`
	Subject     string    `json:"subject"`
	TextContent string    `json:"text_content"`
	HTMLContent string    `json:"html_content"`
	Direction   string    `json:"direction"`
	IsRead      bool      `json:"is_read"`
	CreatedDt   time.Time `json:"created_dt"`
}

// SMSConversation represents an SMS conversation summary from the /api/sms-inbox/ endpoint.
// It groups messages between a Ravi phone number and an external number.
type SMSConversation struct {
	ConversationID  string    `json:"conversation_id"`
	FromNumber      string    `json:"from_number"`
	Phone           string    `json:"phone"`
	PhoneNumber     string    `json:"phone_number"`
	Preview         string    `json:"preview"`
	MessageCount    int       `json:"message_count"`
	UnreadCount     int       `json:"unread_count"`
	LatestMessageDt time.Time `json:"latest_message_dt"`
}

// SMSConversationDetail represents a complete SMS conversation with all its messages,
// returned when viewing a specific conversation by ID.
type SMSConversationDetail struct {
	ConversationID string       `json:"conversation_id"`
	FromNumber     string       `json:"from_number"`
	Phone          string       `json:"phone"`
	MessageCount   int          `json:"message_count"`
	Messages       []SMSMessage `json:"messages"`
}

// SMSMessage represents a single SMS message within a conversation.
type SMSMessage struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	Direction string    `json:"direction"`
	IsRead    bool      `json:"is_read"`
	CreatedDt time.Time `json:"created_dt"`
}

// Owner represents the account owner's profile information.
type Owner struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Error represents an error response from the API, containing a human-readable
// error message in the Detail field.
type Error struct {
	Detail string `json:"detail"`
}

// EncryptionMeta holds the user's E2E encryption metadata from the server.
type EncryptionMeta struct {
	ID               int    `json:"id"`
	Salt             string `json:"salt"`
	Verifier         string `json:"verifier"`
	PublicKey        string `json:"public_key"`
	ManagedMasterKey string `json:"managed_master_key"`
}

// Phone represents the user's assigned Ravi phone number.
type Phone struct {
	ID          int       `json:"id"`
	PhoneNumber string    `json:"phone_number"`
	Provider    string    `json:"provider"`
	CreatedDt   time.Time `json:"created_dt"`
}

// Email represents the user's assigned Ravi email address.
type Email struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedDt time.Time `json:"created_dt"`
}

// PhoneMessage represents an individual SMS message.
type PhoneMessage struct {
	ID         int       `json:"id"`
	URL        string    `json:"url"`
	FromNumber string    `json:"from_number"`
	ToNumber   string    `json:"to_number"`
	Body       string    `json:"body"`
	MessageSID string    `json:"message_sid"`
	Phone      string    `json:"phone"`
	Direction  string    `json:"direction"`
	IsRead     bool      `json:"is_read"`
	CreatedDt  time.Time `json:"created_dt"`
}

// EmailMessageDetail represents an individual email message (standalone, from /api/email-messages/).
type EmailMessageDetail struct {
	ID          int       `json:"id"`
	URL         string    `json:"url"`
	FromEmail   string    `json:"from_email"`
	ToEmail     string    `json:"to_email"`
	CC          string    `json:"cc"`
	Subject     string    `json:"subject"`
	TextContent string    `json:"text_content"`
	HTMLContent string    `json:"html_content"`
	Direction   string    `json:"direction"`
	IsRead      bool      `json:"is_read"`
	MessageID   string    `json:"message_id"`
	ThreadID    string    `json:"thread_id"`
	CreatedDt   time.Time `json:"created_dt"`
}

// PasswordEntry represents a stored website credential.
type PasswordEntry struct {
	UUID      string `json:"uuid"`
	Identity  int    `json:"identity,omitempty"`
	Domain    string `json:"domain"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Notes     string `json:"notes"`
	CreatedDt string `json:"created_dt"`
	UpdatedDt string `json:"updated_dt"`
}

// GeneratedPassword is the response from the password generator endpoint.
type GeneratedPassword struct {
	Password string `json:"password"`
}

// PasswordGenOpts holds query parameters for the password generator.
// The No* fields disable specific character categories. The zero value
// means all categories are enabled (server defaults apply).
type PasswordGenOpts struct {
	Length       int
	NoUppercase  bool
	NoLowercase  bool
	NoDigits     bool
	NoSpecial    bool
	ExcludeChars string
}

// Identity represents a user's named identity grouping (email + phone).
type Identity struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Email     string `json:"inbox"`
	Phone     string `json:"phone"`
	CreatedDt string `json:"created_dt"`
	UpdatedDt string `json:"updated_dt"`
}

// BindIdentityRequest is the request body for binding an identity to a JWT session.
type BindIdentityRequest struct {
	Identity string `json:"identity"`
}

// BindIdentityResponse contains the new token pair after binding an identity.
type BindIdentityResponse struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}
