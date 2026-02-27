package api

import "time"

const (
	// TokenExpiryBuffer is the time before actual expiry to trigger refresh.
	// Backend issues 1-hour tokens; we refresh 5 minutes before expiry for safety.
	TokenExpiryBuffer = 55 * time.Minute
)

const (
	// API endpoint paths
	PathDeviceCode    = "/api/auth/device/"
	PathDeviceToken   = "/api/auth/device/token/"
	PathTokenRefresh  = "/api/auth/token/refresh/"
	PathEmailInbox    = "/api/email-inbox/"
	PathSMSInbox      = "/api/sms-inbox/"
	PathPhone         = "/api/phone/"
	PathEmail         = "/api/email/"
	PathMessages      = "/api/messages/"
	PathEmailMessages = "/api/email-messages/"
	PathEncryption    = "/api/encryption/"
	PathOwner         = "/api/master/"
	PathPasswords     = "/api/passwords/"
	PathSecrets       = "/api/vault/"
	PathIdentities    = "/api/identities/"

	PathEmailAttachmentPresign = "/api/email-attachments/presign/"
	PathEmailCompose           = "/api/email-messages/compose/"
)
