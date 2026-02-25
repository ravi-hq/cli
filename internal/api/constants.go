package api

import "time"

const (
	// TokenExpiryBuffer is the time before actual expiry to trigger refresh.
	// Backend issues 5-minute tokens; we refresh at 4 minutes for safety.
	TokenExpiryBuffer = 4 * time.Minute
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
	PathVault         = "/api/vault/"
	PathIdentities    = "/api/identities/"
	PathBindIdentity  = "/api/auth/bind-identity/"

	PathEmailAttachmentPresign = "/api/email-attachments/presign/"
	PathEmailCompose           = "/api/email-messages/compose/"
)
