package api

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
	PathSecrets       = "/api/secrets/"
	PathIdentities    = "/api/identities/"
	PathBindIdentity  = "/api/auth/bind-identity/"
	PathContacts      = "/api/contacts/"

	PathEmailAttachmentPresign = "/api/email-attachments/presign/"
	PathEmailCompose           = "/api/email-messages/compose/"
)
