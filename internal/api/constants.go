package api

const (
	// API endpoint paths
	PathDeviceCode    = "/api/auth/device/"
	PathDeviceToken   = "/api/auth/device/token/"
	PathEmailInbox    = "/api/email-inbox/"
	PathSMSInbox      = "/api/sms-inbox/"
	PathPhone         = "/api/phone/"
	PathEmail         = "/api/email/"
	PathMessages      = "/api/messages/"
	PathEmailMessages = "/api/email-messages/"
	PathOwner         = "/api/master/"
	PathPasswords     = "/api/passwords/"
	PathSecrets       = "/api/secrets/"
	PathIdentities    = "/api/identities/"
	PathContacts      = "/api/contacts/"
	PathDomains       = "/api/domains/"

	PathEmailAttachmentPresign = "/api/email-attachments/presign/"
	PathEmailCompose           = "/api/email-messages/compose/"

	// Key management endpoints
	PathManagementKeys = "/api/auth/keys/management/"
	PathIdentityKeys   = "/api/auth/keys/identity/"

	// SSO endpoints
	PathSSOToken = "/api/sso/token/"
)
